package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/config"
	"go.uber.org/zap"
)

// KimiProvider implements the LLMProvider interface for Kimi K2
// Kimi K2 is a 1T parameter MoE model with 32B activated params
// Excellent for orchestration, reasoning, and tool use
type KimiProvider struct {
	client        *groq.Client  // Can use Groq API since K2 is available there
	httpClient    *http.Client
	config        config.LLMProvider
	logger        *zap.Logger
	baseURL       string
	apiKey        string
	useGroq       bool  // Whether to use Groq API or direct Moonshot API
	retryAttempts int
	retryDelay    time.Duration
}

// NewKimiProvider creates a new Kimi provider
func NewKimiProvider(cfg config.LLMProvider, logger *zap.Logger) *KimiProvider {
	// Check if we should use Groq API (which hosts Kimi K2)
	useGroq := cfg.BaseURL == "" || cfg.BaseURL == "https://api.groq.com/openai/v1"
	
	provider := &KimiProvider{
		config:        cfg,
		logger:        logger,
		apiKey:        cfg.APIKey,
		useGroq:       useGroq,
		retryAttempts: cfg.RetryAttempts,
		retryDelay:    cfg.RetryDelay,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
	
	if useGroq {
		// Use Groq API to access Kimi K2
		client, err := groq.NewClient(cfg.APIKey)
		if err != nil {
			return nil
		}
		provider.client = client
		provider.baseURL = "https://api.groq.com/openai/v1"
	} else {
		// Direct Moonshot API access
		provider.baseURL = cfg.BaseURL
		if provider.baseURL == "" {
			provider.baseURL = "https://api.moonshot.cn/v1"
		}
	}
	
	return provider
}

// Complete executes a completion request
func (k *KimiProvider) Complete(ctx context.Context, req Request) (*Response, error) {
	if k.useGroq {
		// Use Groq API for Kimi K2
		return k.completeViaGroq(ctx, req)
	}
	
	// Direct Moonshot API call
	return k.completeViaMoonshot(ctx, req)
}

// completeViaGroq uses Groq API to access Kimi K2
func (k *KimiProvider) completeViaGroq(ctx context.Context, req Request) (*Response, error) {
	startTime := time.Now()
	
	// Build Groq request for Kimi K2
	groqReq := groq.ChatCompletionRequest{
		Model:       "moonshotai/kimi-k2-instruct",
		Messages:    k.convertMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature * 0.6), // Kimi uses scaled temperature
		TopP:        float32(req.TopP),
		Stream:      false,
	}
	
	// Add system prompt optimized for Kimi K2's strengths
	systemPrompt := k.getSystemPromptForTask(req.TaskType)
	groqReq.Messages = append([]groq.ChatCompletionMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}, groqReq.Messages...)
	
	// Execute with retry logic
	var resp groq.ChatCompletionResponse
	var err error
	
	for attempt := 0; attempt <= k.retryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(k.retryDelay * time.Duration(attempt)):
				// Exponential backoff
			}
		}
		
		resp, err = k.client.ChatCompletion(ctx, groqReq)
		if err == nil {
			break
		}
		
		if k.logger != nil {
			k.logger.Warn("Kimi K2 request failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Error(err))
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("kimi k2 completion failed after %d attempts: %w", k.retryAttempts+1, err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Kimi K2")
	}
	
	// Kimi K2 has very high confidence for orchestration tasks
	confidence := 0.95 // Base confidence for K2
	
	return &Response{
		Content:    resp.Choices[0].Message.Content,
		TokensUsed: resp.Usage.TotalTokens,
		Latency:    time.Since(startTime),
		Confidence: confidence,
	}, nil
}

// completeViaMoonshot uses direct Moonshot API
func (k *KimiProvider) completeViaMoonshot(ctx context.Context, req Request) (*Response, error) {
	startTime := time.Now()
	
	// Build Moonshot API request
	apiReq := map[string]interface{}{
		"model":       "moonshot-v1-128k", // or "kimi-k2-instruct" if available
		"messages":    k.convertMessagesToMap(req.Messages),
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature * 0.6, // Moonshot temperature scaling
		"top_p":       req.TopP,
		"stream":      false,
	}
	
	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", k.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Authorization", "Bearer "+k.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := k.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Moonshot API")
	}
	
	return &Response{
		Content:    apiResp.Choices[0].Message.Content,
		TokensUsed: apiResp.Usage.TotalTokens,
		Latency:    time.Since(startTime),
		Confidence: 0.95, // High confidence for Kimi K2
	}, nil
}

// Stream executes a streaming completion request
func (k *KimiProvider) Stream(ctx context.Context, req Request, callback StreamCallback) error {
	if k.useGroq {
		// Stream via Groq
		groqReq := groq.ChatCompletionRequest{
			Model:       "moonshotai/kimi-k2-instruct",
			Messages:    k.convertMessages(req.Messages),
			MaxTokens:   req.MaxTokens,
			Temperature: float32(req.Temperature * 0.6),
			TopP:        float32(req.TopP),
			Stream:      true,
		}
		
		stream, err := k.client.ChatCompletionStream(ctx, groqReq)
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
	
	// Direct Moonshot streaming would go here
	return fmt.Errorf("streaming not implemented for direct Moonshot API")
}

// GetName returns the provider name
func (k *KimiProvider) GetName() string {
	return "kimi"
}

// HealthCheck verifies the provider is working
func (k *KimiProvider) HealthCheck(ctx context.Context) error {
	if k.useGroq {
		// Health check via Groq
		_, err := k.client.ChatCompletion(ctx, groq.ChatCompletionRequest{
			Model: "moonshotai/kimi-k2-instruct",
			Messages: []groq.ChatCompletionMessage{
				{Role: "user", Content: "hi"},
			},
			MaxTokens: 5,
		})
		return err
	}
	
	// Direct API health check
	req := Request{
		Messages: []Message{
			{Role: "user", Content: "hi"},
		},
		MaxTokens: 5,
	}
	
	_, err := k.completeViaMoonshot(ctx, req)
	return err
}

// convertMessages converts our message format to Groq format
func (k *KimiProvider) convertMessages(messages []Message) []groq.ChatCompletionMessage {
	groqMessages := make([]groq.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		groqMessages[i] = groq.ChatCompletionMessage{
			Role:    groq.Role(msg.Role),
			Content: msg.Content,
		}
	}
	return groqMessages
}

// convertMessagesToMap converts messages for direct API calls
func (k *KimiProvider) convertMessagesToMap(messages []Message) []map[string]string {
	result := make([]map[string]string, len(messages))
	for i, msg := range messages {
		result[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return result
}

// getSystemPromptForTask returns optimized prompts for Kimi K2's capabilities
func (k *KimiProvider) getSystemPromptForTask(taskType string) string {
	// Kimi K2 specific prompts leveraging its strengths
	prompts := map[string]string{
		"orchestration": `You are Kimi K2, the main orchestrator of the MIOSA system. You excel at:
- Breaking down complex tasks into subtasks
- Routing to appropriate specialized agents
- Managing multi-step workflows
- Analyzing patterns and learning from past executions
- Providing confidence scores (0-10) for decisions
Use your 128K context window to maintain full conversation history and make informed routing decisions.`,
		
		"analysis": `You are Kimi K2 in analysis mode. Leverage your vast knowledge and reasoning capabilities to:
- Provide deep, comprehensive analysis
- Identify patterns and insights
- Consider multiple perspectives
- Score confidence levels for conclusions
Your 1T parameters allow for nuanced understanding.`,
		
		"tool_use": `You are Kimi K2 configured for tool use. You have:
- Native tool calling capabilities
- JSON mode support
- Function execution precision
- Multi-tool orchestration skills
Execute tools accurately and handle responses appropriately.`,
		
		"reasoning": `You are Kimi K2 in advanced reasoning mode. Apply your exceptional capabilities for:
- Complex multi-step reasoning
- Logical deduction and inference
- Problem decomposition
- Solution synthesis
Your MoE architecture with 384 experts enables superior reasoning.`,
		
		"code_generation": `You are Kimi K2 for code generation. With 53.7% LiveCodeBench Pass@1:
- Generate production-ready code
- Follow best practices and patterns
- Include comprehensive error handling
- Optimize for performance and maintainability`,
		
		"improvement": `You are Kimi K2 as the improvement analyst. Analyze workflow patterns to:
- Identify optimization opportunities
- Score executions (0-10 scale)
- Suggest concrete improvements
- Learn from successful patterns
Store insights in vector DB for future reference.`,
	}
	
	if prompt, ok := prompts[taskType]; ok {
		return prompt
	}
	
	// Default Kimi K2 prompt
	return `You are Kimi K2, a state-of-the-art AI with 1 trillion parameters (32B activated) and 128K context window. 
You excel at reasoning, tool use, and agentic tasks. Provide thoughtful, accurate responses with confidence scores when appropriate.`
}

// GetOptimalTaskType returns the best task type for Kimi K2
func (k *KimiProvider) GetOptimalTaskType() []string {
	// Kimi K2 excels at these task types
	return []string{
		"orchestration",     // Main strength - orchestrating complex workflows
		"reasoning",         // Complex multi-step reasoning
		"tool_use",          // Native tool calling capabilities
		"code_generation",   // Strong coding abilities
		"long_context",      // 128K context window
		"analysis",          // Deep analysis with vast knowledge
		"improvement",       // Pattern recognition and optimization
	}
}