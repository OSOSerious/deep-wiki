package communication

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

// CommunicationAgent handles all user interactions and chat responses
type CommunicationAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

// New creates a new communication agent
func New(groqClient *groq.Client) *CommunicationAgent {
	return &CommunicationAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "llama-3.1-8b-instant", // Fast model for user interactions
			MaxTokens:   2000,
			Temperature: 0.7,
			TopP:        0.9,
		},
	}
}

// GetType returns the agent type
func (a *CommunicationAgent) GetType() agents.AgentType {
	return agents.CommunicationAgent
}

// GetDescription returns the agent description
func (a *CommunicationAgent) GetDescription() string {
	return "Handles user interactions, chat responses, and UI/UX communications"
}

// GetCapabilities returns the agent's capabilities
func (a *CommunicationAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "chat", Description: "Natural conversation with users", Required: true},
		{Name: "consultation", Description: "Business consultation dialogue", Required: true},
		{Name: "support", Description: "User support and help", Required: true},
		{Name: "onboarding", Description: "User onboarding flow", Required: false},
		{Name: "feedback", Description: "Collect user feedback", Required: false},
	}
}

// Execute processes a communication task
func (a *CommunicationAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	
	// Build conversation context
	messages := a.buildConversationContext(task)
	
	// Add current message
	messages = append(messages, groq.ChatCompletionMessage{
		Role:    "user",
		Content: task.Input,
	})
	
	// Get response from LLM
	response, err := a.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model:       groq.ChatModel(a.config.Model),
		Messages:    messages,
		MaxTokens:   a.config.MaxTokens,
		Temperature: float32(a.config.Temperature),
		TopP:        float32(a.config.TopP),
	})
	
	if err != nil {
		return &agents.Result{
			Success:     false,
			Error:       fmt.Errorf("failed to get response: %w", err),
			ExecutionMS: time.Since(startTime).Milliseconds(),
		}, err
	}
	
	if len(response.Choices) == 0 {
		return &agents.Result{
			Success:     false,
			Error:       fmt.Errorf("no response from model"),
			ExecutionMS: time.Since(startTime).Milliseconds(),
		}, fmt.Errorf("no response from model")
	}
	
	content := response.Choices[0].Message.Content
	
	// Analyze response for next steps
	nextStep, suggestions := a.analyzeResponse(content, task)
	
	return &agents.Result{
		Success:     true,
		Output:      content,
		NextStep:    nextStep,
		Suggestions: suggestions,
		Confidence:  0.85,
		Data: map[string]interface{}{
			"model":       a.config.Model,
			"phase":       task.Context.Phase,
			"tokens_used": response.Usage.TotalTokens,
		},
		ExecutionMS: time.Since(startTime).Milliseconds(),
	}, nil
}

// buildConversationContext builds the conversation context from task history
func (a *CommunicationAgent) buildConversationContext(task agents.Task) []groq.ChatCompletionMessage {
	// System prompt based on phase
	phase := ""
	if task.Context != nil {
		phase = task.Context.Phase
	}
	systemPrompt := a.getSystemPrompt(phase)
	
	messages := []groq.ChatCompletionMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}
	
	// Add conversation history if context exists
	if task.Context != nil {
		for _, msg := range task.Context.History {
			role := "user"
			if msg.Role == "assistant" {
				role = "assistant"
			}
			messages = append(messages, groq.ChatCompletionMessage{
				Role:    groq.Role(role),
				Content: msg.Content,
			})
		}
	}
	
	return messages
}

// getSystemPrompt returns the appropriate system prompt based on phase
func (a *CommunicationAgent) getSystemPrompt(phase string) string {
	basePrompt := `You are MIOSA, an AI-powered business assistant. You help users with their business challenges, provide insights, and guide them through solutions.`
	
	phasePrompts := map[string]string{
		"initial": `
You're in the initial consultation phase. Focus on:
- Understanding the user's business and challenges
- Asking clarifying questions
- Building rapport and trust
- Gathering context for deeper analysis`,
		
		"exploration": `
You're in the exploration phase. Focus on:
- Diving deeper into specific challenges
- Exploring potential solutions
- Providing initial recommendations
- Identifying opportunities for improvement`,
		
		"deep-dive": `
You're in the deep-dive phase. Focus on:
- Providing detailed analysis and insights
- Offering specific, actionable recommendations
- Creating implementation plans
- Addressing technical details`,
		
		"implementation": `
You're in the implementation phase. Focus on:
- Guiding through execution steps
- Providing technical support
- Monitoring progress
- Adjusting plans as needed`,
	}
	
	if phasePrompt, exists := phasePrompts[phase]; exists {
		return basePrompt + phasePrompt
	}
	
	return basePrompt
}

// analyzeResponse analyzes the response to determine next steps
func (a *CommunicationAgent) analyzeResponse(content string, task agents.Task) (string, []string) {
	contentLower := strings.ToLower(content)
	suggestions := []string{}
	nextStep := ""
	
	// Detect if user needs deeper analysis
	if strings.Contains(contentLower, "analyze") || strings.Contains(contentLower, "investigate") {
		nextStep = "deep_analysis"
		suggestions = append(suggestions, "Consider running a detailed analysis")
	}
	
	// Detect if user needs code generation
	if strings.Contains(contentLower, "code") || strings.Contains(contentLower, "implement") {
		suggestions = append(suggestions, "Ready to generate code when needed")
	}
	
	// Detect if user needs strategy
	if strings.Contains(contentLower, "strategy") || strings.Contains(contentLower, "plan") {
		suggestions = append(suggestions, "Strategic planning might be helpful")
	}
	
	// Phase transition suggestions
	switch task.Context.Phase {
	case "initial":
		if len(task.Context.History) > 4 {
			nextStep = "exploration"
			suggestions = append(suggestions, "Ready to move to exploration phase")
		}
	case "exploration":
		if strings.Contains(contentLower, "specific") || strings.Contains(contentLower, "detail") {
			nextStep = "deep-dive"
			suggestions = append(suggestions, "Consider moving to deep-dive phase")
		}
	}
	
	return nextStep, suggestions
}

// Register registers the communication agent
func Register(groqClient *groq.Client) error {
	agent := New(groqClient)
	return agents.Register(agent)
}