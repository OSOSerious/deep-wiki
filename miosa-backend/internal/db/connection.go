package db

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var (
	pgDB        *sql.DB
	redisClient *redis.Client
	once        sync.Once
)

type Config struct {
	PostgresURL string
	RedisURL    string
}

func Initialize(cfg *Config) error {
	var initErr error

	once.Do(func() {
		// PostgreSQL connection
		pgDB, initErr = sql.Open("postgres", cfg.PostgresURL)
		if initErr != nil {
			return
		}

		pgDB.SetMaxOpenConns(25)
		pgDB.SetMaxIdleConns(10)

		if initErr = pgDB.Ping(); initErr != nil {
			return
		}

		// Redis connection
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			initErr = err
			return
		}

		redisClient = redis.NewClient(opt)
	})

	return initErr
}

func GetDB() *sql.DB {
	return pgDB
}

func GetRedis() *redis.Client {
	return redisClient
}

func Close() error {
	if pgDB != nil {
		if err := pgDB.Close(); err != nil {
			return fmt.Errorf("failed to close postgres: %w", err)
		}
	}
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			return fmt.Errorf("failed to close redis: %w", err)
		}
	}
	return nil
}