package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/config"
	"go.uber.org/zap"
)

var (
	pgDB          *sql.DB
	pgxPool       *pgxpool.Pool
	redisClient   *redis.Client
	redisCluster  *redis.ClusterClient
	readReplicas  []*sql.DB
	logger        *zap.Logger
	once          sync.Once
	currentReplica int
	replicaMutex   sync.Mutex
)

type Manager struct {
	config        *config.DatabaseConfig
	redisConfig   *config.RedisConfig
	primaryDB     *sql.DB
	pgxPool       *pgxpool.Pool
	redis         *redis.Client
	redisCluster  *redis.ClusterClient
	readReplicas  []*sql.DB
	logger        *zap.Logger
}

func Initialize(cfg *config.Config, log *zap.Logger) error {
	var initErr error

	once.Do(func() {
		logger = log
		
		initErr = initializePostgres(cfg.Database)
		if initErr != nil {
			return
		}
		
		initErr = initializeRedis(cfg.Redis)
		if initErr != nil {
			return
		}
		
		if cfg.Database.EnableReplicas {
			initErr = initializeReadReplicas(cfg.Database)
		}
	})

	return initErr
}

func initializePostgres(cfg config.DatabaseConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	db, err := sql.Open("postgres", cfg.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %w", err)
	}
	
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping postgres: %w", err)
	}
	
	pgDB = db
	
	poolConfig, err := pgxpool.ParseConfig(cfg.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to parse pgx config: %w", err)
	}
	
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns / 2)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create pgx pool: %w", err)
	}
	
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping pgx pool: %w", err)
	}
	
	pgxPool = pool
	
	if logger != nil {
		logger.Info("PostgreSQL connection established",
			zap.Int("max_open_conns", cfg.MaxOpenConns),
			zap.Int("max_idle_conns", cfg.MaxIdleConns),
		)
	}
	
	return nil
}

func initializeRedis(cfg config.RedisConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if cfg.EnableCluster && len(cfg.ClusterAddresses) > 0 {
		redisCluster = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.ClusterAddresses,
			MaxRetries:   cfg.MaxRetries,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			MaxConnAge:   cfg.MaxConnAge,
			IdleTimeout:  cfg.IdleTimeout,
		})
		
		if err := redisCluster.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("failed to connect to redis cluster: %w", err)
		}
		
		if logger != nil {
			logger.Info("Redis cluster connection established",
				zap.Strings("addresses", cfg.ClusterAddresses),
			)
		}
	} else {
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return fmt.Errorf("failed to parse redis URL: %w", err)
		}
		
		opt.MaxRetries = cfg.MaxRetries
		opt.DialTimeout = cfg.DialTimeout
		opt.ReadTimeout = cfg.ReadTimeout
		opt.WriteTimeout = cfg.WriteTimeout
		opt.PoolSize = cfg.PoolSize
		opt.MinIdleConns = cfg.MinIdleConns
		opt.MaxConnAge = cfg.MaxConnAge
		opt.ConnMaxIdleTime = cfg.IdleTimeout
		
		redisClient = redis.NewClient(opt)
		
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("failed to connect to redis: %w", err)
		}
		
		if logger != nil {
			logger.Info("Redis connection established",
				zap.String("url", cfg.URL),
			)
		}
	}
	
	return nil
}

func initializeReadReplicas(cfg config.DatabaseConfig) error {
	readReplicas = make([]*sql.DB, 0, len(cfg.ReadReplicaURLs))
	
	for i, replicaURL := range cfg.ReadReplicaURLs {
		db, err := sql.Open("postgres", replicaURL)
		if err != nil {
			return fmt.Errorf("failed to open read replica %d: %w", i, err)
		}
		
		db.SetMaxOpenConns(cfg.MaxOpenConns / 2)
		db.SetMaxIdleConns(cfg.MaxIdleConns / 2)
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := db.PingContext(ctx); err != nil {
			cancel()
			if logger != nil {
				logger.Warn("Failed to connect to read replica, skipping",
					zap.Int("replica", i),
					zap.Error(err),
				)
			}
			continue
		}
		cancel()
		
		readReplicas = append(readReplicas, db)
		
		if logger != nil {
			logger.Info("Read replica connected",
				zap.Int("replica", i),
			)
		}
	}
	
	if len(readReplicas) == 0 && len(cfg.ReadReplicaURLs) > 0 {
		return fmt.Errorf("failed to connect to any read replicas")
	}
	
	return nil
}

func GetDB() *sql.DB {
	return pgDB
}

func GetPgxPool() *pgxpool.Pool {
	return pgxPool
}

func GetReadDB() *sql.DB {
	if len(readReplicas) == 0 {
		return pgDB
	}
	
	replicaMutex.Lock()
	defer replicaMutex.Unlock()
	
	db := readReplicas[currentReplica]
	currentReplica = (currentReplica + 1) % len(readReplicas)
	
	return db
}

func WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := pgDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}
	
	return tx.Commit()
}

func GetRedis() redis.UniversalClient {
	if redisCluster != nil {
		return redisCluster
	}
	return redisClient
}

func GetRedisClient() *redis.Client {
	return redisClient
}

func GetRedisCluster() *redis.ClusterClient {
	return redisCluster
}

func HealthCheck(ctx context.Context) error {
	if err := pgDB.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}
	
	if redisClient != nil {
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis health check failed: %w", err)
		}
	}
	
	if redisCluster != nil {
		if err := redisCluster.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis cluster health check failed: %w", err)
		}
	}
	
	return nil
}

func Close() error {
	var errs []error
	
	if pgDB != nil {
		if err := pgDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close postgres: %w", err))
		}
	}
	
	if pgxPool != nil {
		pgxPool.Close()
	}
	
	for i, replica := range readReplicas {
		if replica != nil {
			if err := replica.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close replica %d: %w", i, err))
			}
		}
	}
	
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close redis: %w", err))
		}
	}
	
	if redisCluster != nil {
		if err := redisCluster.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close redis cluster: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	
	return nil
}

func NewManager(cfg *config.Config, log *zap.Logger) (*Manager, error) {
	if err := Initialize(cfg, log); err != nil {
		return nil, err
	}
	
	return &Manager{
		config:       &cfg.Database,
		redisConfig:  &cfg.Redis,
		primaryDB:    pgDB,
		pgxPool:      pgxPool,
		redis:        redisClient,
		redisCluster: redisCluster,
		readReplicas: readReplicas,
		logger:       log,
	}, nil
}

func (m *Manager) GetPrimary() *sql.DB {
	return m.primaryDB
}

func (m *Manager) GetReadReplica() *sql.DB {
	if len(m.readReplicas) == 0 {
		return m.primaryDB
	}
	
	replicaMutex.Lock()
	defer replicaMutex.Unlock()
	
	db := m.readReplicas[currentReplica]
	currentReplica = (currentReplica + 1) % len(m.readReplicas)
	
	return db
}

func (m *Manager) GetPgxPool() *pgxpool.Pool {
	return m.pgxPool
}

func (m *Manager) GetRedis() redis.UniversalClient {
	if m.redisCluster != nil {
		return m.redisCluster
	}
	return m.redis
}