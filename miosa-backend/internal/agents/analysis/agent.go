package analysis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
)

// AnalysisAgent handles requirements analysis and system understanding
type AnalysisAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
	logger     *zap.Logger
}

// New creates a new analysis agent
func New(groqClient *groq.Client) agents.Agent {
	logger, _ := zap.NewProduction()
	return &AnalysisAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "llama-3.3-70b-versatile", // Powerful model for analysis
			MaxTokens:   4000,
			Temperature: 0.3, // Lower temp for analytical work
			TopP:        0.9,
		},
		logger: logger,
	}
}

// GetType returns the agent type
func (a *AnalysisAgent) GetType() agents.AgentType {
	return agents.AnalysisAgent
}

// GetDescription returns the agent description
func (a *AnalysisAgent) GetDescription() string {
	return "Analyzes requirements, breaks down problems, and provides insights"
}

// GetCapabilities returns the agent's capabilities
func (a *AnalysisAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "requirements_analysis", Description: "Break down and analyze requirements", Required: true},
		{Name: "feasibility_study", Description: "Assess technical feasibility", Required: true},
		{Name: "risk_assessment", Description: "Identify potential risks", Required: false},
		{Name: "dependency_mapping", Description: "Map system dependencies", Required: false},
	}
}

// Execute processes an analysis task
func (a *AnalysisAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	
	// Initialize tools
	tools := NewAnalysisTools()
	
	// Use tools to analyze requirements
	requirementsAnalysis, err := tools.AnalyzeRequirements(ctx, task.Input)
	if err != nil {
		a.logger.Warn("Tool analysis failed, falling back to LLM", zap.Error(err))
	}
	
	// Build analysis prompt with tool results
	prompt := fmt.Sprintf(`As a systems analyst, analyze the following request:

Request: %s

Tool Analysis Results:
%v

Provide a comprehensive analysis including:
1. Key requirements and objectives
2. Technical considerations
3. Potential challenges
4. Recommended approach
5. Success criteria

Be specific and actionable.`, task.Input, requirementsAnalysis)

	// Get analysis from LLM
	response, err := a.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: groq.ChatModel(a.config.Model),
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are an expert systems analyst specializing in breaking down complex requirements.",
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
			Error:       fmt.Errorf("analysis failed: %w", err),
			ExecutionMS: time.Since(startTime).Milliseconds(),
			Confidence:  0,
		}, err
	}
	
	if len(response.Choices) == 0 {
		return &agents.Result{
			Success:     false,
			Error:       fmt.Errorf("no analysis generated"),
			ExecutionMS: time.Since(startTime).Milliseconds(),
			Confidence:  0,
		}, fmt.Errorf("no response from model")
	}
	
	content := response.Choices[0].Message.Content
	
	// Calculate confidence based on analysis completeness
	confidence := a.calculateConfidence(content)
	
	// Determine next agent based on analysis
	nextAgent := a.determineNextAgent(content, task)
	
	result := &agents.Result{
		Success:     true,
		Output:      content,
		NextAgent:   nextAgent,
		Confidence:  confidence,
		ExecutionMS: time.Since(startTime).Milliseconds(),
		Data: map[string]interface{}{
			"model":      a.config.Model,
			"word_count": len(strings.Fields(content)),
		},
	}
	
	// Record execution for self-improvement
	agents.RecordExecution(a.GetType(), result)
	
	// If confidence is low, suggest improvements
	if confidence < 7.0 {
		result.Suggestions = []string{
			"Consider breaking down the requirements further",
			"May need additional context or clarification",
			"Review with domain expert for validation",
		}
	}
	
	return result, nil
}

// calculateConfidence assesses the quality of the analysis
func (a *AnalysisAgent) calculateConfidence(content string) float64 {
	confidence := 5.0 // Base confidence
	
	// Check for key analysis components
	if strings.Contains(strings.ToLower(content), "requirement") {
		confidence += 1.0
	}
	if strings.Contains(strings.ToLower(content), "technical") {
		confidence += 1.0
	}
	if strings.Contains(strings.ToLower(content), "challenge") || strings.Contains(strings.ToLower(content), "risk") {
		confidence += 1.0
	}
	if strings.Contains(strings.ToLower(content), "approach") || strings.Contains(strings.ToLower(content), "solution") {
		confidence += 1.0
	}
	if strings.Contains(strings.ToLower(content), "success") || strings.Contains(strings.ToLower(content), "criteria") {
		confidence += 1.0
	}
	
	// Cap at 10
	if confidence > 10 {
		confidence = 10
	}
	
	return confidence
}

// determineNextAgent decides which agent should handle the task next
func (a *AnalysisAgent) determineNextAgent(content string, task agents.Task) agents.AgentType {
	contentLower := strings.ToLower(content)
	
	// Route based on analysis results
	if strings.Contains(contentLower, "architecture") || strings.Contains(contentLower, "design") {
		return agents.ArchitectAgent
	}
	if strings.Contains(contentLower, "implement") || strings.Contains(contentLower, "code") {
		return agents.DevelopmentAgent
	}
	if strings.Contains(contentLower, "security") || strings.Contains(contentLower, "vulnerability") {
		return "security"
	}
	if strings.Contains(contentLower, "deploy") || strings.Contains(contentLower, "release") {
		return agents.DeploymentAgent
	}
	
	// Default to development for implementation
	return agents.DevelopmentAgent
}