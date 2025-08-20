package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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
	config := loadConfig()

	// Initialize Groq client (will be nil if no key)
	var client *groq.Client
	var err error
	
	if config.GroqKey != "" && config.GroqKey != "gsk_YOUR_ACTUAL_KEY_HERE" {
		client, err = groq.NewClient(config.GroqKey)
		if err != nil {
			log.Printf("⚠️  Failed to create Groq client: %v", err)
		}
	} else {
		log.Println("⚠️  GROQ_API_KEY not configured. Running in demo mode.")
		log.Println("   Get your key from https://console.groq.com")
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "MIOSA API is running",
			Data: map[string]interface{}{
				"version": "1.0.0",
				"models": map[string]string{
					"fast": config.FastModel,
					"deep": config.DeepModel,
				},
			},
		})
	})

	// API routes
	api := r.Group("/api")
	{
		// Fast chat endpoint (Mixtral)
		api.POST("/chat", func(c *gin.Context) {
			var req ChatRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Error:   err.Error(),
				})
				return
			}

			if client == nil {
				c.JSON(http.StatusOK, Response{
					Success: true,
					Data:    fmt.Sprintf("Demo response: You said '%s'. Please add your GROQ_API_KEY to get real AI responses.", req.Message),
					Model:   "demo",
				})
				return
			}

			response, err := callGroq(client, config.FastModel, req.Message)
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Error:   fmt.Sprintf("Failed to get response: %v", err),
				})
				return
			}

			c.JSON(http.StatusOK, Response{
				Success: true,
				Data:    response,
				Model:   config.FastModel,
			})
		})

		// Deep analysis endpoint (Kimi K2)
		api.POST("/analyze", func(c *gin.Context) {
			var req AnalyzeRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Error:   err.Error(),
				})
				return
			}

			prompt := fmt.Sprintf("Analyze the following %s content:\n\n%s", req.Type, req.Content)
			response, err := callGroq(client, config.DeepModel, prompt)
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Error:   fmt.Sprintf("Failed to analyze: %v", err),
				})
				return
			}

			c.JSON(http.StatusOK, Response{
				Success: true,
				Data:    response,
				Model:   config.DeepModel,
			})
		})

		// Consultation endpoint (phase-based routing)
		api.POST("/consultation", func(c *gin.Context) {
			var req ConsultationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Error:   err.Error(),
				})
				return
			}

			// Choose model based on phase
			model := config.FastModel
			if req.Phase == "deep-dive" {
				model = config.DeepModel
			}

			prompt := fmt.Sprintf("Topic: %s\nContext: %s\nPhase: %s\n\nProvide consultation:", 
				req.Topic, req.Context, req.Phase)
			
			response, err := callGroq(client, model, prompt)
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Error:   fmt.Sprintf("Failed to consult: %v", err),
				})
				return
			}

			c.JSON(http.StatusOK, Response{
				Success: true,
				Data:    response,
				Model:   model,
			})
		})

		// Code generation endpoint (Kimi K2)
		api.POST("/generate", func(c *gin.Context) {
			var req GenerateRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Error:   err.Error(),
				})
				return
			}

			prompt := fmt.Sprintf("Generate %s:\nDescription: %s\nContext: %v", 
				req.Type, req.Description, req.Context)
			
			response, err := callGroq(client, config.DeepModel, prompt)
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Error:   fmt.Sprintf("Failed to generate: %v", err),
				})
				return
			}

			c.JSON(http.StatusOK, Response{
				Success: true,
				Data:    response,
				Model:   config.DeepModel,
			})
		})
	}

	// Start server
	log.Printf("🚀 MIOSA API starting on port %s", config.Port)
	log.Printf("✅ Models configured:")
	log.Printf("   📱 Communication (Fast): %s", config.FastModel)
	log.Printf("   🧠 Analysis (Deep): %s", config.DeepModel)
	
	if err := r.Run(":" + config.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}