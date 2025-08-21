package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"github.com/sormind/OSA/miosa-backend/internal/agents/ai_providers"
	"github.com/sormind/OSA/miosa-backend/internal/agents/analysis"
	"github.com/sormind/OSA/miosa-backend/internal/agents/architect"
	"github.com/sormind/OSA/miosa-backend/internal/agents/communication"
	"github.com/sormind/OSA/miosa-backend/internal/agents/deployment"
	"github.com/sormind/OSA/miosa-backend/internal/agents/development"
	"github.com/sormind/OSA/miosa-backend/internal/agents/monitoring"
	"github.com/sormind/OSA/miosa-backend/internal/agents/quality"
	"github.com/sormind/OSA/miosa-backend/internal/agents/recommender"
	"github.com/sormind/OSA/miosa-backend/internal/agents/strategy"
	"github.com/sormind/OSA/miosa-backend/internal/config"
	"github.com/sormind/OSA/miosa-backend/internal/middleware"
	"github.com/sormind/OSA/miosa-backend/internal/services/collaboration"
	"github.com/sormind/OSA/miosa-backend/internal/services/gateway"
	"go.uber.org/zap"
)

type Config struct {
	Port      string
	GroqKey   string
	FastModel string
	DeepModel string
	DBUrl     string
	RedisUrl  string
	JWTSecret string
	E2BKey    string
	RenderKey string
}

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

type AnalyzeRequest struct {
	Content string `json:"content" binding:"required"`
	Type    string `json:"type"` // business, technical, product
}

type ConsultationRequest struct {
	Topic   string `json:"topic" binding:"required"`
	Context string `json:"context"`
	Phase   string `json:"phase"` // initial, exploration, deep-dive
}

type GenerateRequest struct {
	Type        string            `json:"type" binding:"required"` // code, architecture, docs
	Description string            `json:"description" binding:"required"`
	Context     map[string]string `json:"context"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Model   string      `json:"model,omitempty"`
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func boolToEmoji(b bool) string {
	if b {
		return "✅"
	}
	return "⚠️"
}

func loadConfig() *Config {
	_ = godotenv.Load()

	config := &Config{
		Port:      getEnv("PORT", "8080"),
		GroqKey:   os.Getenv("GROQ_API_KEY"),
		FastModel: getEnv("FAST_MODEL", "llama-3.1-8b-instant"),
		DeepModel: getEnv("DEEP_MODEL", "moonshotai/kimi-k2-instruct"),
		DBUrl:     os.Getenv("DATABASE_URL"),
		RedisUrl:  os.Getenv("REDIS_URL"),
		JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-this"),
		E2BKey:    os.Getenv("E2B_API_KEY"),
		RenderKey: os.Getenv("RENDER_API_KEY"),
	}

	// Log what's configured
	log.Println("🔧 Configuration Status:")
	log.Printf("  %v Groq API: %v", boolToEmoji(config.GroqKey != ""), config.GroqKey != "")
	log.Printf("  %v Database: %v", boolToEmoji(config.DBUrl != ""), config.DBUrl != "")
	log.Printf("  %v Redis: %v", boolToEmoji(config.RedisUrl != ""), config.RedisUrl != "")
	log.Printf("  %v E2B: %v", boolToEmoji(config.E2BKey != ""), config.E2BKey != "")
	log.Printf("  %v Render: %v", boolToEmoji(config.RenderKey != ""), config.RenderKey != "")

	return config
}

func callGroq(client *groq.Client, model string, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: groq.ChatModel(model),
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})

	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from model")
}

func main() {
	log.Println("🚀 Starting MIOSA API Gateway with Full Integration")
	
	// Load configuration
	cfg := loadConfig()
	
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	
	// Initialize database connection (optional - will work without it)
	var db *sql.DB
	if cfg.DBUrl != "" {
		db, err = sql.Open("postgres", cfg.DBUrl)
		if err != nil {
			logger.Warn("Database connection failed, continuing without DB", zap.Error(err))
		} else {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(10)
			db.SetConnMaxLifetime(5 * time.Minute)
			if err := db.Ping(); err != nil {
				logger.Warn("Database ping failed", zap.Error(err))
				db = nil
			} else {
				logger.Info("✅ Connected to PostgreSQL")
				defer db.Close()
			}
		}
	}
	
	// Initialize Redis connection (optional - will work without it)
	var redisClient redis.UniversalClient
	if cfg.RedisUrl != "" {
		opts, err := redis.ParseURL(cfg.RedisUrl)
		if err != nil {
			logger.Warn("Redis URL parse failed", zap.Error(err))
		} else {
			redisClient = redis.NewClient(opts)
			ctx := context.Background()
			if err := redisClient.Ping(ctx).Err(); err != nil {
				logger.Warn("Redis connection failed", zap.Error(err))
				redisClient = nil
			} else {
				logger.Info("✅ Connected to Redis")
			}
		}
	}
	
	// Initialize Groq client
	var groqClient *groq.Client
	if cfg.GroqKey != "" && cfg.GroqKey != "gsk_YOUR_ACTUAL_KEY_HERE" {
		groqClient, err = groq.NewClient(cfg.GroqKey)
		if err != nil {
			logger.Error("Failed to create Groq client", zap.Error(err))
		} else {
			logger.Info("✅ Groq client initialized")
		}
	} else {
		logger.Warn("GROQ_API_KEY not configured - API features limited")
	}
	
	// Initialize agent orchestrator
	var orchestrator *agents.Orchestrator
	if groqClient != nil {
		// Register all agents from their packages
		agents.Register(communication.New(groqClient))
		agents.Register(analysis.New(groqClient))
		agents.Register(development.New(groqClient))
		agents.Register(quality.New(groqClient))
		agents.Register(deployment.New(groqClient))
		agents.Register(architect.New(groqClient))
		agents.Register(monitoring.New(groqClient))
		agents.Register(strategy.New(groqClient))
		
		// Register new agents with Redis support
		recommenderAgent := recommender.New(groqClient)
		if redisClient != nil {
			if rc, ok := redisClient.(*redis.Client); ok {
				recommenderAgent.SetRedis(rc)
			}
			recommenderAgent.SetLogger(logger)
		}
		agents.Register(recommenderAgent)
		
		aiProvidersAgent := ai_providers.New(groqClient)
		if redisClient != nil {
			if rc, ok := redisClient.(*redis.Client); ok {
				aiProvidersAgent.SetRedis(rc)
			}
			aiProvidersAgent.SetLogger(logger)
		}
		agents.Register(aiProvidersAgent)
		
		orchestrator = agents.NewOrchestrator(groqClient, logger, nil)
		logger.Info("✅ Agent orchestrator initialized with all agents")
	}
	
	// Setup Gin with production settings
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	
	// Recovery middleware (must be first)
	r.Use(gin.Recovery())
	
	// Initialize our middleware chain
	
	// 1. Security middleware
	securityConfig := middleware.DefaultSecurityConfig()
	securityMiddleware := middleware.NewSecurityMiddleware(logger, securityConfig)
	r.Use(securityMiddleware.Handle())
	
	// 2. Logging middleware
	loggingConfig := &middleware.LoggingConfig{
		SkipPaths:       []string{"/health", "/metrics"},
		SlowRequestTime: 2 * time.Second,
		Level:           "info",
		Environment:     "production",
	}
	loggingMiddleware, err := middleware.NewLoggingMiddleware(loggingConfig)
	if err != nil {
		logger.Error("Failed to create logging middleware", zap.Error(err))
	} else {
		r.Use(loggingMiddleware.Handle())
	}
	
	// 3. Auth middleware (skip for public endpoints)
	if db != nil && redisClient != nil {
		authConfig := &config.AuthConfig{
			JWTSecret: cfg.JWTSecret,
		}
		authMiddleware := middleware.NewAuthMiddleware(authConfig, db, redisClient, logger)
		// Apply selectively to protected routes
		r.Use(func(c *gin.Context) {
			// Skip auth for public endpoints
			publicPaths := []string{"/health", "/api/auth/login", "/api/auth/register"}
			for _, path := range publicPaths {
				if c.Request.URL.Path == path {
					c.Next()
					return
				}
			}
			authMiddleware.Handle()(c)
		})
	}
	
	// 4. Rate limiting middleware
	if redisClient != nil {
		rateLimitConfig := middleware.DefaultRateLimitConfig()
		rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, logger, rateLimitConfig)
		r.Use(rateLimitMiddleware.Handle())
	}

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize gateway handlers
	handlers := gateway.NewHandlers(orchestrator, groqClient, logger)
	
	// Initialize collaboration handlers (only if Redis is available)
	var collabHandlers *collaboration.Handlers
	if redisClient != nil {
		// Type assertion for Redis client
		if rc, ok := redisClient.(*redis.Client); ok {
			collabHandlers = collaboration.NewHandlers(orchestrator, rc, logger)
		}
	}
	
	// Health check
	r.GET("/health", handlers.HealthCheck)

	// API routes
	api := r.Group("/api")
	{
		// Main agent execution endpoint
		api.POST("/agents/execute", handlers.ExecuteAgent)
		
		// Legacy chat endpoint for backward compatibility
		api.POST("/chat", handlers.Chat)
		
		// Collaboration endpoints (only if handlers available)
		if collabHandlers != nil {
			api.POST("/collaboration/execute", collabHandlers.ExecuteCollaborativeTask)
		}

		// Additional endpoints can be added here as needed
		// All complex logic should go through the agent system
	}

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}
	
	// Start server in goroutine
	go func() {
		logger.Info("🚀 MIOSA API Gateway starting",
			zap.String("port", cfg.Port),
			zap.Bool("database", db != nil),
			zap.Bool("redis", redisClient != nil),
			zap.Bool("groq", groqClient != nil))
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Info("Shutting down server...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	
	logger.Info("Server exited properly")
}