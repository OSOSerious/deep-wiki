package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/config"
	"go.uber.org/zap"
)

// GroqProvider implements the LLMProvider interface for Groq
type GroqProvider struct {
	client        *groq.Client
	config        config.LLMProvider
	logger        *zap.Logger
	modelMapping  map[ModelSize]string
	retryAttempts int
	retryDelay    time.Duration
}

// ModelSize represents the size of the model
type ModelSize int

const (
	ModelSizeSmall ModelSize = iota
	ModelSizeMedium
	ModelSizeLarge
)

// NewGroqProvider creates a new Groq provider
func NewGroqProvider(cfg config.LLMProvider, logger *zap.Logger) (*GroqProvider, error) {
	client, err := groq.NewClient(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Groq client: %w", err)
	}

	return &GroqProvider{
		client: client,
		config: cfg,
		logger: logger,
		modelMapping: map[ModelSize]string{
			ModelSizeSmall:  "llama-3.1-8b-instant",   // Ultra-fast for subagents
			ModelSizeMedium: "mixtral-8x7b-32768",     // Balanced
			ModelSizeLarge:  "llama-3.3-70b-versatile", // High quality
		},
		retryAttempts: cfg.RetryAttempts,
		retryDelay:    cfg.RetryDelay,
	}, nil
}

// Complete executes a completion request
func (g *GroqProvider) Complete(ctx context.Context, req Request) (*Response, error) {
	startTime := time.Now()
	
	// Select model based on task type
	model := g.selectModel(req.TaskType)
	
	groqReq := groq.ChatCompletionRequest{
		Model:       groq.ChatModel(model),
		Messages:    g.convertMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
		TopP:        float32(req.TopP),
		Stream:      false,
	}
	
	// Execute with retry logic
	var resp groq.ChatCompletionResponse
	var err error
	
	for attempt := 0; attempt <= g.retryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(g.retryDelay * time.Duration(attempt)):
				// Exponential backoff
			}
		}
		
		resp, err = g.client.ChatCompletion(ctx, groqReq)
		if err == nil {
			break
		}
		
		if g.logger != nil {
			g.logger.Warn("Groq request failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Error(err))
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("groq completion failed after %d attempts: %w", g.retryAttempts+1, err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Groq")
	}
	
	// Calculate confidence based on model and response
	confidence := g.calculateConfidence(model, resp)
	
	return &Response{
		Content:    resp.Choices[0].Message.Content,
		TokensUsed: resp.Usage.TotalTokens,
		Latency:    time.Since(startTime),
		Confidence: confidence,
	}, nil
}

// Stream executes a streaming completion request
func (g *GroqProvider) Stream(ctx context.Context, req Request, callback StreamCallback) error {
	model := g.selectModel(req.TaskType)
	
	groqReq := groq.ChatCompletionRequest{
		Model:       groq.ChatModel(model),
		Messages:    g.convertMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
		TopP:        float32(req.TopP),
		Stream:      true,
	}
	
	stream, err := g.client.ChatCompletionStream(ctx, groqReq)
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}
	defer stream.Close()
	
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("stream error: %w", err)
		}
		
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if err := callback(chunk.Choices[0].Delta.Content); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// GetName returns the provider name
func (g *GroqProvider) GetName() string {
	return "groq"
}

// HealthCheck verifies the provider is working
func (g *GroqProvider) HealthCheck(ctx context.Context) error {
	_, err := g.client.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []groq.ChatCompletionMessage{
			{Role: "user", Content: "hi"},
		},
		MaxTokens: 5,
	})
	return err
}

// selectModel chooses the appropriate model based on task type
func (g *GroqProvider) selectModel(taskType string) string {
	switch taskType {
	case "quick_response", "extraction", "classification":
		return g.modelMapping[ModelSizeSmall] // Fast subagent tasks
	case "reasoning", "analysis", "code_review":
		return g.modelMapping[ModelSizeMedium]
	case "complex_reasoning", "code_generation", "architecture":
		return g.modelMapping[ModelSizeLarge]
	default:
		return g.modelMapping[ModelSizeSmall] // Default to fast model
	}
}

// convertMessages converts our message format to Groq format
func (g *GroqProvider) convertMessages(messages []Message) []groq.ChatCompletionMessage {
	groqMessages := make([]groq.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		groqMessages[i] = groq.ChatCompletionMessage{
			Role:    groq.Role(msg.Role),
			Content: msg.Content,
		}
	}
	return groqMessages
}

// calculateConfidence calculates confidence score based on model and response
func (g *GroqProvider) calculateConfidence(model string, resp groq.ChatCompletionResponse) float64 {
	baseConfidence := 0.7 // Base confidence for Groq
	
	// Adjust based on model size
	switch model {
	case g.modelMapping[ModelSizeLarge]:
		baseConfidence += 0.15
	case g.modelMapping[ModelSizeMedium]:
		baseConfidence += 0.1
	case g.modelMapping[ModelSizeSmall]:
		baseConfidence += 0.05
	}
	
	// Adjust based on token usage (more tokens = more detailed response)
	if resp.Usage.CompletionTokens > 100 {
		baseConfidence += 0.05
	}
	
	// Cap at 0.95 for Groq (Kimi K2 gets higher confidence)
	if baseConfidence > 0.95 {
		baseConfidence = 0.95
	}
	
	return baseConfidence
}

// GetOptimalTaskTypes returns task types this provider excels at
func (g *GroqProvider) GetOptimalTaskTypes() []string {
	return []string{
		"quick_response",    // Ultra-fast responses
		"extraction",        // Data extraction
		"classification",    // Text classification
		"subagent_tasks",    // Fast subagent operations
		"real_time",         // Real-time interactions
	}
}