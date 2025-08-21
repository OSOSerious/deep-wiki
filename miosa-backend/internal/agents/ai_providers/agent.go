package ai_providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
)

// AIProvidersAgent manages multi-model orchestration with Kimi K2 as primary
type AIProvidersAgent struct {
	id          uuid.UUID
	groqClient  *groq.Client
	redisClient *redis.Client
	logger      *zap.Logger
	modelStats  map[string]*ModelStats
	cache       *ResponseCache
	router      *ModelRouter
	mu          sync.RWMutex
}

// ModelStats tracks performance for each model
type ModelStats struct {
	ModelID      string
	TotalCalls   int
	SuccessRate  float64
	AvgLatency   time.Duration
	TokensUsed   int
	CostPerToken float64
	LastUsed     time.Time
	Capabilities []string
}

// ResponseCache implements intelligent caching
type ResponseCache struct {
	client     *redis.Client
	ttl        time.Duration
	hitRate    float64
	totalHits  int
	totalCalls int
	mu         sync.RWMutex
}

// ModelRouter intelligently routes to the best model
type ModelRouter struct {
	models       map[string]*ModelConfig
	routingRules map[string]RoutingRule
	fallbacks    map[string]string
}

// ModelConfig defines a model's capabilities
type ModelConfig struct {
	ID           string
	Provider     string
	Capabilities []string
	MaxTokens    int
	Temperature  float64
	CostPerToken float64
	Speed        string // "fast", "medium", "slow"
	Quality      string // "high", "medium", "low"
}

// RoutingRule determines model selection
type RoutingRule struct {
	TaskType    string
	Complexity  string
	PreferSpeed bool
	MinQuality  string
	MaxCost     float64
}

// Available models
var (
	KimiK2Model = &ModelConfig{
		ID:           "moonshotai/kimi-k2-instruct",
		Provider:     "moonshot",
		Capabilities: []string{"reasoning", "coding", "tools", "long_context"},
		MaxTokens:    16384,
		Temperature:  0.3,
		CostPerToken: 0.00002,
		Speed:        "medium",
		Quality:      "high",
	}
	
	GPTOSS20BModel = &ModelConfig{
		ID:           "openai/gpt-oss-20b",
		Provider:     "openai",
		Capabilities: []string{"general", "creative", "conversation"},
		MaxTokens:    8192,
		Temperature:  0.7,
		CostPerToken: 0.00001,
		Speed:        "medium",
		Quality:      "high",
	}
	
	Llama70BModel = &ModelConfig{
		ID:           "llama-3.3-70b-versatile",
		Provider:     "meta",
		Capabilities: []string{"general", "coding", "reasoning"},
		MaxTokens:    8192,
		Temperature:  0.5,
		CostPerToken: 0.000008,
		Speed:        "fast",
		Quality:      "high",
	}
	
	Llama8BModel = &ModelConfig{
		ID:           "llama-3.1-8b-instant",
		Provider:     "meta",
		Capabilities: []string{"general", "fast_response"},
		MaxTokens:    4096,
		Temperature:  0.5,
		CostPerToken: 0.000002,
		Speed:        "fast",
		Quality:      "medium",
	}
)

func New(groqClient *groq.Client) *AIProvidersAgent {
	router := &ModelRouter{
		models: map[string]*ModelConfig{
			"kimi-k2":     KimiK2Model,
			"gpt-oss-20b": GPTOSS20BModel,
			"llama-70b":   Llama70BModel,
			"llama-8b":    Llama8BModel,
		},
		routingRules: make(map[string]RoutingRule),
		fallbacks: map[string]string{
			"kimi-k2":     "llama-70b",
			"gpt-oss-20b": "llama-70b",
			"llama-70b":   "llama-8b",
		},
	}
	
	return &AIProvidersAgent{
		id:         uuid.New(),
		groqClient: groqClient,
		modelStats: make(map[string]*ModelStats),
		router:     router,
	}
}

// SetRedis adds Redis client for caching
func (a *AIProvidersAgent) SetRedis(client *redis.Client) {
	a.redisClient = client
	a.cache = &ResponseCache{
		client: client,
		ttl:    30 * time.Minute,
	}
}

// SetLogger adds logger
func (a *AIProvidersAgent) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

func (a *AIProvidersAgent) GetID() uuid.UUID {
	return a.id
}

func (a *AIProvidersAgent) GetType() agents.AgentType {
	return "ai_providers"
}

func (a *AIProvidersAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{
			Name:        "multi_model_routing",
			Description: "Intelligently route tasks to optimal LLM models",
			Required:    true,
		},
		{
			Name:        "response_caching",
			Description: "Cache LLM responses for efficiency",
			Required:    false,
		},
		{
			Name:        "model_fallback",
			Description: "Automatic fallback to alternative models on failure",
			Required:    true,
		},
		{
			Name:        "cost_optimization",
			Description: "Select models based on cost/performance tradeoffs",
			Required:    false,
		},
	}
}

func (a *AIProvidersAgent) GetDescription() string {
	return "Multi-model orchestration agent using Kimi K2 as primary, with intelligent routing and caching"
}

func (a *AIProvidersAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	
	// Check cache first
	if a.cache != nil {
		if cached := a.checkCache(ctx, task); cached != nil {
			if a.logger != nil {
				a.logger.Info("Cache hit for task", 
					zap.String("task_type", task.Type),
					zap.Float64("hit_rate", a.cache.hitRate))
			}
			return cached, nil
		}
	}
	
	// Determine best model for task
	selectedModel := a.selectModel(ctx, task)
	
	if a.logger != nil {
		a.logger.Info("Selected model for task",
			zap.String("model", selectedModel.ID),
			zap.String("task_type", task.Type),
			zap.String("speed", selectedModel.Speed),
			zap.String("quality", selectedModel.Quality))
	}
	
	// Execute with selected model
	result, err := a.executeWithModel(ctx, task, selectedModel)
	
	// Handle fallback if needed
	if err != nil && a.router.fallbacks[selectedModel.ID] != "" {
		fallbackID := a.router.fallbacks[selectedModel.ID]
		if a.logger != nil {
			a.logger.Warn("Model failed, using fallback",
				zap.String("failed_model", selectedModel.ID),
				zap.String("fallback", fallbackID),
				zap.Error(err))
		}
		
		for _, model := range a.router.models {
			if model.ID == a.router.fallbacks[selectedModel.ID] {
				result, err = a.executeWithModel(ctx, task, model)
				break
			}
		}
	}
	
	if err != nil {
		return &agents.Result{
			Success:     false,
			Error:       err,
			ExecutionMS: time.Since(startTime).Milliseconds(),
		}, err
	}
	
	// Update statistics
	a.updateModelStats(selectedModel.ID, result, startTime)
	
	// Cache successful result
	if a.cache != nil && result.Success {
		a.cacheResult(ctx, task, result)
	}
	
	// Add model info to result
	if result.Data == nil {
		result.Data = make(map[string]interface{})
	}
	result.Data["model_used"] = selectedModel.ID
	result.Data["model_speed"] = selectedModel.Speed
	result.Data["model_quality"] = selectedModel.Quality
	result.Data["cache_hit_rate"] = a.getCacheHitRate()
	
	return result, nil
}

// selectModel chooses the best model based on task requirements
func (a *AIProvidersAgent) selectModel(ctx context.Context, task agents.Task) *ModelConfig {
	// Analyze task to determine requirements
	requirements := a.analyzeTaskRequirements(task)
	
	// Complex reasoning tasks - use Kimi K2
	if requirements.complexity == "high" {
		return KimiK2Model
	}
	
	// Fast response needed - use Llama 8B
	if requirements.preferSpeed && requirements.complexity == "low" {
		return Llama8BModel
	}
	
	// Creative tasks - use GPT-OSS-20B
	if requirements.creative {
		return GPTOSS20BModel
	}
	
	// Default to Llama 70B for balance
	return Llama70BModel
}

// executeWithModel executes task with selected model
func (a *AIProvidersAgent) executeWithModel(ctx context.Context, task agents.Task, model *ModelConfig) (*agents.Result, error) {
	messages := []groq.ChatCompletionMessage{
		{
			Role:    "user",
			Content: task.Input,
		},
	}
	
	// Add context if available
	if task.Context != nil {
		for _, msg := range task.Context.History {
			messages = append(messages, groq.ChatCompletionMessage{
				Role:    groq.Role(msg.Role),
				Content: msg.Content,
			})
		}
	}
	
	// Add system message for Kimi K2
	if model.ID == KimiK2Model.ID {
		messages = append([]groq.ChatCompletionMessage{
			{
				Role: "system",
				Content: `You are Kimi K2, an advanced AI orchestrator managing a multi-agent system.
You excel at complex reasoning, coding, and tool use. Be precise and efficient.`,
			},
		}, messages...)
	}
	
	response, err := a.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model:       groq.ChatModel(model.ID),
		Messages:    messages,
		Temperature: float32(model.Temperature),
		MaxTokens:   model.MaxTokens,
	})
	
	if err != nil {
		return nil, err
	}
	
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from model %s", model.ID)
	}
	
	// Calculate confidence based on model quality
	confidence := 7.0
	if model.Quality == "high" {
		confidence = 8.5
		if model.ID == KimiK2Model.ID {
			confidence = 9.0 // Highest confidence for Kimi K2
		}
	} else if model.Quality == "medium" {
		confidence = 7.0
	}
	
	return &agents.Result{
		Success:    true,
		Output:     response.Choices[0].Message.Content,
		Confidence: confidence,
		Data: map[string]interface{}{
			"model": model.ID,
			"usage": response.Usage,
		},
	}, nil
}

// Cache implementation

func (a *AIProvidersAgent) checkCache(ctx context.Context, task agents.Task) *agents.Result {
	if a.cache == nil || a.redisClient == nil {
		return nil
	}
	
	cacheKey := a.generateCacheKey(task)
	data, err := a.redisClient.Get(ctx, cacheKey).Result()
	if err != nil {
		a.cache.mu.Lock()
		a.cache.totalCalls++
		a.cache.hitRate = float64(a.cache.totalHits) / float64(a.cache.totalCalls)
		a.cache.mu.Unlock()
		return nil
	}
	
	var result agents.Result
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil
	}
	
	a.cache.mu.Lock()
	a.cache.totalHits++
	a.cache.totalCalls++
	a.cache.hitRate = float64(a.cache.totalHits) / float64(a.cache.totalCalls)
	a.cache.mu.Unlock()
	
	if result.Data == nil {
		result.Data = make(map[string]interface{})
	}
	result.Data["from_cache"] = true
	return &result
}

func (a *AIProvidersAgent) cacheResult(ctx context.Context, task agents.Task, result *agents.Result) {
	if a.cache == nil || a.redisClient == nil {
		return
	}
	
	cacheKey := a.generateCacheKey(task)
	data, _ := json.Marshal(result)
	
	a.redisClient.Set(ctx, cacheKey, data, a.cache.ttl)
}

func (a *AIProvidersAgent) generateCacheKey(task agents.Task) string {
	// Generate semantic cache key
	hash := fmt.Sprintf("%x", task.Input)
	if len(hash) > 16 {
		hash = hash[:16]
	}
	return fmt.Sprintf("cache:ai:%s:%s", task.Type, hash)
}

func (a *AIProvidersAgent) getCacheHitRate() float64 {
	if a.cache == nil {
		return 0
	}
	a.cache.mu.RLock()
	defer a.cache.mu.RUnlock()
	return a.cache.hitRate
}

// Helper methods

func (a *AIProvidersAgent) analyzeTaskRequirements(task agents.Task) struct {
	complexity  string
	preferSpeed bool
	creative    bool
} {
	lower := strings.ToLower(task.Input)
	
	// Determine complexity
	complexity := "low"
	if len(task.Input) > 500 || strings.Contains(lower, "complex") || strings.Contains(lower, "analyze") {
		complexity = "high"
	} else if len(task.Input) > 200 {
		complexity = "medium"
	}
	
	return struct {
		complexity  string
		preferSpeed bool
		creative    bool
	}{
		complexity:  complexity,
		preferSpeed: strings.Contains(lower, "quick") || strings.Contains(lower, "fast") || strings.Contains(lower, "simple"),
		creative:    strings.Contains(lower, "creative") || strings.Contains(lower, "story") || strings.Contains(lower, "imagine"),
	}
}

func (a *AIProvidersAgent) updateModelStats(modelID string, result *agents.Result, startTime time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	stats, exists := a.modelStats[modelID]
	if !exists {
		stats = &ModelStats{
			ModelID: modelID,
		}
		a.modelStats[modelID] = stats
	}
	
	stats.TotalCalls++
	if result.Success {
		stats.SuccessRate = (stats.SuccessRate*float64(stats.TotalCalls-1) + 1.0) / float64(stats.TotalCalls)
	} else {
		stats.SuccessRate = (stats.SuccessRate * float64(stats.TotalCalls-1)) / float64(stats.TotalCalls)
	}
	
	latency := time.Since(startTime)
	stats.AvgLatency = (stats.AvgLatency*time.Duration(stats.TotalCalls-1) + latency) / time.Duration(stats.TotalCalls)
	stats.LastUsed = time.Now()
	
	// Track token usage if available
	if usage, ok := result.Data["usage"]; ok {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			if totalTokens, ok := usageMap["total_tokens"].(int); ok {
				stats.TokensUsed += totalTokens
			}
		}
	}
}