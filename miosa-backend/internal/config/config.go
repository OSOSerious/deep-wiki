package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	LLM      LLMConfig
	Services ServicesConfig
	Security SecurityConfig
	Features FeatureFlags
}

type ServerConfig struct {
	Port               string
	Host               string
	Environment        string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	MaxRequestSize     int64
	EnableCORS         bool
	AllowedOrigins     []string
	EnableRateLimit    bool
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	ShutdownTimeout    time.Duration
}

type DatabaseConfig struct {
	PostgresURL      string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
	EnableMigrations bool
	MigrationsPath   string
	EnableReplicas   bool
	ReadReplicaURLs  []string
	VectorDimension  int
}

type RedisConfig struct {
	URL              string
	MaxRetries       int
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PoolSize         int
	MinIdleConns     int
	MaxConnAge       time.Duration
	IdleTimeout      time.Duration
	EnableCluster    bool
	ClusterAddresses []string
}

type AuthConfig struct {
	JWTSecret           string
	JWTExpiry           time.Duration
	RefreshTokenExpiry  time.Duration
	PasswordMinLength   int
	EnableOAuth         bool
	OAuthProviders      map[string]OAuthProvider
	SessionTimeout      time.Duration
	MaxLoginAttempts    int
	LockoutDuration     time.Duration
}

type OAuthProvider struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type LLMConfig struct {
	DefaultProvider string
	Providers       map[string]LLMProvider
	MaxTokens       int
	Temperature     float32
	TopP            float32
	StreamingEnabled bool
	CacheResponses   bool
	CacheTTL         time.Duration
}

type LLMProvider struct {
	APIKey          string
	BaseURL         string
	Model           string
	MaxConcurrency  int
	RequestTimeout  time.Duration
	RetryAttempts   int
	RetryDelay      time.Duration
}

type ServicesConfig struct {
	E2B        E2BConfig
	Render     RenderConfig
	Stripe     StripeConfig
	Temporal   TemporalConfig
	RabbitMQ   RabbitMQConfig
	Monitoring MonitoringConfig
}

type E2BConfig struct {
	APIKey           string
	BaseURL          string
	DefaultTemplate  string
	SessionTimeout   time.Duration
	MaxSessions      int
	EnablePreviews   bool
}

type RenderConfig struct {
	APIKey          string
	BaseURL         string
	DefaultRegion   string
	AutoScaling     bool
	MinInstances    int
	MaxInstances    int
}

type StripeConfig struct {
	SecretKey       string
	PublishableKey  string
	WebhookSecret   string
	DefaultCurrency string
	PricingPlans    map[string]PricingPlan
}

type PricingPlan struct {
	PriceID     string
	Name        string
	Features    []string
	Limits      map[string]int
}

type TemporalConfig struct {
	HostPort       string
	Namespace      string
	TaskQueue      string
	WorkerCount    int
	EnableMetrics  bool
}

type RabbitMQConfig struct {
	URL            string
	Exchange       string
	ExchangeType   string
	QueuePrefix    string
	PrefetchCount  int
	AutoAck        bool
	Durable        bool
	ReconnectDelay time.Duration
}

type MonitoringConfig struct {
	EnableMetrics     bool
	MetricsPort       string
	EnableTracing     bool
	TracingEndpoint   string
	EnableLogging     bool
	LogLevel          string
	LogFormat         string
	SentryDSN         string
	EnableProfiling   bool
	ProfilingPort     string
}

type SecurityConfig struct {
	EnableTLS          bool
	TLSCertPath        string
	TLSKeyPath         string
	EnableVault        bool
	VaultAddress       string
	VaultToken         string
	EncryptionKey      string
	CSRFSecret         string
	SecureHeaders      bool
	ContentSecurityPolicy string
}

type FeatureFlags struct {
	EnableMCP           bool
	EnableWebSockets    bool
	EnableCollaboration bool
	EnableAnalytics     bool
	EnableAIProviders   bool
	EnableCustomDomains bool
	EnableE2E           bool
	ExperimentalFeatures map[string]bool
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:               getEnv("PORT", "8080"),
			Host:               getEnv("HOST", "0.0.0.0"),
			Environment:        getEnv("ENVIRONMENT", "development"),
			ReadTimeout:        getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:       getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
			MaxRequestSize:     getInt64Env("MAX_REQUEST_SIZE", 10*1024*1024),
			EnableCORS:         getBoolEnv("ENABLE_CORS", true),
			AllowedOrigins:     getSliceEnv("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
			EnableRateLimit:    getBoolEnv("ENABLE_RATE_LIMIT", true),
			RateLimitRequests:  getIntEnv("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:    getDurationEnv("RATE_LIMIT_WINDOW", time.Minute),
			ShutdownTimeout:    getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			PostgresURL:      getEnvRequired("DATABASE_URL"),
			MaxOpenConns:     getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:     getIntEnv("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime:  getDurationEnv("DB_CONN_MAX_LIFETIME", time.Hour),
			EnableMigrations: getBoolEnv("ENABLE_MIGRATIONS", true),
			MigrationsPath:   getEnv("MIGRATIONS_PATH", "internal/db/migrations"),
			EnableReplicas:   getBoolEnv("ENABLE_READ_REPLICAS", false),
			ReadReplicaURLs:  getSliceEnv("READ_REPLICA_URLS", []string{}),
			VectorDimension:  getIntEnv("VECTOR_DIMENSION", 1536),
		},
		Redis: RedisConfig{
			URL:              getEnvRequired("REDIS_URL"),
			MaxRetries:       getIntEnv("REDIS_MAX_RETRIES", 3),
			DialTimeout:      getDurationEnv("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:      getDurationEnv("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:     getDurationEnv("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolSize:         getIntEnv("REDIS_POOL_SIZE", 10),
			MinIdleConns:     getIntEnv("REDIS_MIN_IDLE_CONNS", 5),
			MaxConnAge:       getDurationEnv("REDIS_MAX_CONN_AGE", 30*time.Minute),
			IdleTimeout:      getDurationEnv("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			EnableCluster:    getBoolEnv("REDIS_ENABLE_CLUSTER", false),
			ClusterAddresses: getSliceEnv("REDIS_CLUSTER_ADDRESSES", []string{}),
		},
		Auth: AuthConfig{
			JWTSecret:          getEnvRequired("JWT_SECRET"),
			JWTExpiry:          getDurationEnv("JWT_EXPIRY", 24*time.Hour),
			RefreshTokenExpiry: getDurationEnv("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
			PasswordMinLength:  getIntEnv("PASSWORD_MIN_LENGTH", 8),
			EnableOAuth:        getBoolEnv("ENABLE_OAUTH", false),
			OAuthProviders:     loadOAuthProviders(),
			SessionTimeout:     getDurationEnv("SESSION_TIMEOUT", 30*time.Minute),
			MaxLoginAttempts:   getIntEnv("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:    getDurationEnv("LOCKOUT_DURATION", 15*time.Minute),
		},
		LLM: LLMConfig{
			DefaultProvider:  getEnv("DEFAULT_LLM_PROVIDER", "groq"),
			Providers:        loadLLMProviders(),
			MaxTokens:        getIntEnv("LLM_MAX_TOKENS", 4000),
			Temperature:      getFloat32Env("LLM_TEMPERATURE", 0.7),
			TopP:             getFloat32Env("LLM_TOP_P", 0.9),
			StreamingEnabled: getBoolEnv("LLM_STREAMING_ENABLED", true),
			CacheResponses:   getBoolEnv("LLM_CACHE_RESPONSES", true),
			CacheTTL:         getDurationEnv("LLM_CACHE_TTL", 1*time.Hour),
		},
		Services: ServicesConfig{
			E2B: E2BConfig{
				APIKey:          getEnv("E2B_API_KEY", ""),
				BaseURL:         getEnv("E2B_BASE_URL", "https://api.e2b.dev"),
				DefaultTemplate: getEnv("E2B_DEFAULT_TEMPLATE", "base"),
				SessionTimeout:  getDurationEnv("E2B_SESSION_TIMEOUT", 30*time.Minute),
				MaxSessions:     getIntEnv("E2B_MAX_SESSIONS", 10),
				EnablePreviews:  getBoolEnv("E2B_ENABLE_PREVIEWS", true),
			},
			Render: RenderConfig{
				APIKey:        getEnv("RENDER_API_KEY", ""),
				BaseURL:       getEnv("RENDER_BASE_URL", "https://api.render.com"),
				DefaultRegion: getEnv("RENDER_DEFAULT_REGION", "oregon"),
				AutoScaling:   getBoolEnv("RENDER_AUTO_SCALING", true),
				MinInstances:  getIntEnv("RENDER_MIN_INSTANCES", 1),
				MaxInstances:  getIntEnv("RENDER_MAX_INSTANCES", 10),
			},
			Stripe: StripeConfig{
				SecretKey:       getEnv("STRIPE_SECRET_KEY", ""),
				PublishableKey:  getEnv("STRIPE_PUBLISHABLE_KEY", ""),
				WebhookSecret:   getEnv("STRIPE_WEBHOOK_SECRET", ""),
				DefaultCurrency: getEnv("STRIPE_DEFAULT_CURRENCY", "usd"),
				PricingPlans:    loadPricingPlans(),
			},
			Temporal: TemporalConfig{
				HostPort:      getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
				Namespace:     getEnv("TEMPORAL_NAMESPACE", "default"),
				TaskQueue:     getEnv("TEMPORAL_TASK_QUEUE", "miosa-workflows"),
				WorkerCount:   getIntEnv("TEMPORAL_WORKER_COUNT", 5),
				EnableMetrics: getBoolEnv("TEMPORAL_ENABLE_METRICS", true),
			},
			RabbitMQ: RabbitMQConfig{
				URL:            getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
				Exchange:       getEnv("RABBITMQ_EXCHANGE", "miosa"),
				ExchangeType:   getEnv("RABBITMQ_EXCHANGE_TYPE", "topic"),
				QueuePrefix:    getEnv("RABBITMQ_QUEUE_PREFIX", "miosa."),
				PrefetchCount:  getIntEnv("RABBITMQ_PREFETCH_COUNT", 10),
				AutoAck:        getBoolEnv("RABBITMQ_AUTO_ACK", false),
				Durable:        getBoolEnv("RABBITMQ_DURABLE", true),
				ReconnectDelay: getDurationEnv("RABBITMQ_RECONNECT_DELAY", 5*time.Second),
			},
			Monitoring: MonitoringConfig{
				EnableMetrics:   getBoolEnv("ENABLE_METRICS", true),
				MetricsPort:     getEnv("METRICS_PORT", "9090"),
				EnableTracing:   getBoolEnv("ENABLE_TRACING", false),
				TracingEndpoint: getEnv("TRACING_ENDPOINT", ""),
				EnableLogging:   getBoolEnv("ENABLE_LOGGING", true),
				LogLevel:        getEnv("LOG_LEVEL", "info"),
				LogFormat:       getEnv("LOG_FORMAT", "json"),
				SentryDSN:       getEnv("SENTRY_DSN", ""),
				EnableProfiling: getBoolEnv("ENABLE_PROFILING", false),
				ProfilingPort:   getEnv("PROFILING_PORT", "6060"),
			},
		},
		Security: SecurityConfig{
			EnableTLS:             getBoolEnv("ENABLE_TLS", false),
			TLSCertPath:           getEnv("TLS_CERT_PATH", ""),
			TLSKeyPath:            getEnv("TLS_KEY_PATH", ""),
			EnableVault:           getBoolEnv("ENABLE_VAULT", false),
			VaultAddress:          getEnv("VAULT_ADDRESS", ""),
			VaultToken:            getEnv("VAULT_TOKEN", ""),
			EncryptionKey:         getEnv("ENCRYPTION_KEY", ""),
			CSRFSecret:            getEnv("CSRF_SECRET", generateRandomSecret()),
			SecureHeaders:         getBoolEnv("SECURE_HEADERS", true),
			ContentSecurityPolicy: getEnv("CSP", "default-src 'self'"),
		},
		Features: FeatureFlags{
			EnableMCP:            getBoolEnv("FEATURE_MCP", true),
			EnableWebSockets:     getBoolEnv("FEATURE_WEBSOCKETS", true),
			EnableCollaboration:  getBoolEnv("FEATURE_COLLABORATION", true),
			EnableAnalytics:      getBoolEnv("FEATURE_ANALYTICS", true),
			EnableAIProviders:    getBoolEnv("FEATURE_AI_PROVIDERS", true),
			EnableCustomDomains:  getBoolEnv("FEATURE_CUSTOM_DOMAINS", true),
			EnableE2E:            getBoolEnv("FEATURE_E2E", true),
			ExperimentalFeatures: loadExperimentalFeatures(),
		},
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.Database.PostgresURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Redis.URL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.LLM.DefaultProvider != "" && len(c.LLM.Providers) == 0 {
		return fmt.Errorf("no LLM providers configured")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return b
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return i
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return defaultValue
		}
		return i
	}
	return defaultValue
}

func getFloat32Env(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		f, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return defaultValue
		}
		return float32(f)
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return defaultValue
		}
		return d
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func loadOAuthProviders() map[string]OAuthProvider {
	providers := make(map[string]OAuthProvider)
	
	if googleID := os.Getenv("GOOGLE_CLIENT_ID"); googleID != "" {
		providers["google"] = OAuthProvider{
			ClientID:     googleID,
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Scopes:       getSliceEnv("GOOGLE_SCOPES", []string{"profile", "email"}),
		}
	}
	
	if githubID := os.Getenv("GITHUB_CLIENT_ID"); githubID != "" {
		providers["github"] = OAuthProvider{
			ClientID:     githubID,
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
			Scopes:       getSliceEnv("GITHUB_SCOPES", []string{"user:email"}),
		}
	}
	
	return providers
}

func loadLLMProviders() map[string]LLMProvider {
	providers := make(map[string]LLMProvider)
	
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		providers["groq"] = LLMProvider{
			APIKey:         groqKey,
			BaseURL:        getEnv("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
			Model:          getEnv("GROQ_MODEL", "mixtral-8x7b-32768"),
			MaxConcurrency: getIntEnv("GROQ_MAX_CONCURRENCY", 10),
			RequestTimeout: getDurationEnv("GROQ_REQUEST_TIMEOUT", 30*time.Second),
			RetryAttempts:  getIntEnv("GROQ_RETRY_ATTEMPTS", 3),
			RetryDelay:     getDurationEnv("GROQ_RETRY_DELAY", time.Second),
		}
	}
	
	if kimiKey := os.Getenv("KIMI_API_KEY"); kimiKey != "" {
		providers["kimi"] = LLMProvider{
			APIKey:         kimiKey,
			BaseURL:        getEnv("KIMI_BASE_URL", "https://api.moonshot.cn/v1"),
			Model:          getEnv("KIMI_MODEL", "moonshot-v1-8k"),
			MaxConcurrency: getIntEnv("KIMI_MAX_CONCURRENCY", 5),
			RequestTimeout: getDurationEnv("KIMI_REQUEST_TIMEOUT", 60*time.Second),
			RetryAttempts:  getIntEnv("KIMI_RETRY_ATTEMPTS", 3),
			RetryDelay:     getDurationEnv("KIMI_RETRY_DELAY", 2*time.Second),
		}
	}
	
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		providers["openai"] = LLMProvider{
			APIKey:         openaiKey,
			BaseURL:        getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			Model:          getEnv("OPENAI_MODEL", "gpt-4-turbo-preview"),
			MaxConcurrency: getIntEnv("OPENAI_MAX_CONCURRENCY", 5),
			RequestTimeout: getDurationEnv("OPENAI_REQUEST_TIMEOUT", 60*time.Second),
			RetryAttempts:  getIntEnv("OPENAI_RETRY_ATTEMPTS", 3),
			RetryDelay:     getDurationEnv("OPENAI_RETRY_DELAY", 2*time.Second),
		}
	}
	
	return providers
}

func loadPricingPlans() map[string]PricingPlan {
	plans := make(map[string]PricingPlan)
	
	plans["free"] = PricingPlan{
		PriceID: os.Getenv("STRIPE_PRICE_FREE"),
		Name:    "Free",
		Features: []string{
			"1 workspace",
			"Basic AI assistance",
			"Community support",
		},
		Limits: map[string]int{
			"workspaces":     1,
			"projects":       3,
			"ai_requests":    100,
			"storage_mb":     500,
		},
	}
	
	plans["pro"] = PricingPlan{
		PriceID: os.Getenv("STRIPE_PRICE_PRO"),
		Name:    "Pro",
		Features: []string{
			"Unlimited workspaces",
			"Advanced AI models",
			"Priority support",
			"Custom domains",
		},
		Limits: map[string]int{
			"workspaces":     -1,
			"projects":       -1,
			"ai_requests":    10000,
			"storage_mb":     50000,
		},
	}
	
	plans["enterprise"] = PricingPlan{
		PriceID: os.Getenv("STRIPE_PRICE_ENTERPRISE"),
		Name:    "Enterprise",
		Features: []string{
			"Everything in Pro",
			"Dedicated infrastructure",
			"SLA guarantee",
			"Custom integrations",
		},
		Limits: map[string]int{
			"workspaces":     -1,
			"projects":       -1,
			"ai_requests":    -1,
			"storage_mb":     -1,
		},
	}
	
	return plans
}

func loadExperimentalFeatures() map[string]bool {
	features := make(map[string]bool)
	
	if value := os.Getenv("EXPERIMENTAL_FEATURES"); value != "" {
		for _, feature := range strings.Split(value, ",") {
			features[strings.TrimSpace(feature)] = true
		}
	}
	
	return features
}

func generateRandomSecret() string {
	return "default-secret-change-in-production"
}