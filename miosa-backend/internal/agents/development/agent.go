package development

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

// DevelopmentAgent handles code generation and implementation
type DevelopmentAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

// New creates a new development agent
func New(groqClient *groq.Client) agents.Agent {
	return &DevelopmentAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "moonshotai/kimi-k2-instruct", // Best model for code generation
			MaxTokens:   8000,
			Temperature: 0.2, // Low temp for consistent code
			TopP:        0.95,
		},
	}
}

// GetType returns the agent type
func (a *DevelopmentAgent) GetType() agents.AgentType {
	return agents.DevelopmentAgent
}

// GetDescription returns the agent description
func (a *DevelopmentAgent) GetDescription() string {
	return "Generates high-quality code implementations with best practices"
}

// GetCapabilities returns the agent's capabilities
func (a *DevelopmentAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "code_generation", Description: "Generate production-ready code", Required: true},
		{Name: "refactoring", Description: "Refactor and optimize code", Required: true},
		{Name: "debugging", Description: "Debug and fix issues", Required: false},
		{Name: "documentation", Description: "Generate code documentation", Required: false},
	}
}

// Execute processes a development task
func (a *DevelopmentAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	
	// Build development prompt
	prompt := fmt.Sprintf(`As an expert software developer, implement the following:

Task: %s

Requirements:
- Write clean, production-ready code
- Follow best practices and design patterns
- Include error handling
- Add appropriate comments
- Make it maintainable and scalable

Provide complete, working code.`, task.Input)

	// Get code from LLM
	response, err := a.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: groq.ChatModel(a.config.Model),
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are an expert software engineer who writes clean, efficient, and maintainable code.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   a.config.MaxTokens,
		Temperature: float32(a.config.Temperature),
		TopP:        float32(a.config.TopP),
	})
	
	if err != nil {
		return &agents.Result{
			Success:     false,
			Error:       fmt.Errorf("code generation failed: %w", err),
			ExecutionMS: time.Since(startTime).Milliseconds(),
			Confidence:  0,
		}, err
	}
	
	if len(response.Choices) == 0 {
		return &agents.Result{
			Success:     false,
			Error:       fmt.Errorf("no code generated"),
			ExecutionMS: time.Since(startTime).Milliseconds(),
			Confidence:  0,
		}, fmt.Errorf("no response from model")
	}
	
	content := response.Choices[0].Message.Content
	
	// Calculate confidence based on code quality indicators
	confidence := a.calculateConfidence(content)
	
	result := &agents.Result{
		Success:     true,
		Output:      content,
		NextAgent:   agents.QualityAgent, // Always go to QA after development
		Confidence:  confidence,
		ExecutionMS: time.Since(startTime).Milliseconds(),
		Data: map[string]interface{}{
			"model":       a.config.Model,
			"line_count":  len(strings.Split(content, "\n")),
			"has_tests":   strings.Contains(content, "test") || strings.Contains(content, "Test"),
			"has_docs":    strings.Contains(content, "/**") || strings.Contains(content, "#"),
		},
	}
	
	// Record execution for self-improvement
	agents.RecordExecution(a.GetType(), result)
	
	// If confidence is low, suggest improvements
	if confidence < 7.0 {
		result.Suggestions = []string{
			"Code may need additional error handling",
			"Consider adding more comprehensive tests",
			"Review for performance optimizations",
		}
	}
	
	return result, nil
}

// calculateConfidence assesses code quality
func (a *DevelopmentAgent) calculateConfidence(content string) float64 {
	confidence := 6.0 // Base confidence for Kimi K2
	
	// Check for code quality indicators
	if strings.Contains(content, "```") {
		confidence += 0.5 // Has code blocks
	}
	if strings.Contains(content, "error") || strings.Contains(content, "Error") {
		confidence += 0.5 // Has error handling
	}
	if strings.Contains(content, "//") || strings.Contains(content, "#") || strings.Contains(content, "/*") {
		confidence += 0.5 // Has comments
	}
	if strings.Contains(content, "func ") || strings.Contains(content, "def ") || strings.Contains(content, "function ") {
		confidence += 1.0 // Has proper function definitions
	}
	if strings.Contains(content, "import") || strings.Contains(content, "require") {
		confidence += 0.5 // Has imports/dependencies
	}
	if strings.Contains(content, "test") || strings.Contains(content, "Test") {
		confidence += 1.0 // Has tests
	}
	
	// Cap at 10
	if confidence > 10 {
		confidence = 10
	}
	
	return confidence
}