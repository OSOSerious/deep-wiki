package architect

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

type ArchitectAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

func New(groqClient *groq.Client) agents.Agent {
	return &ArchitectAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "moonshotai/kimi-k2-instruct",
			MaxTokens:   4000,
			Temperature: 0.4,
			TopP:        0.95,
		},
	}
}

func (a *ArchitectAgent) GetType() agents.AgentType {
	return agents.ArchitectAgent
}

func (a *ArchitectAgent) GetDescription() string {
	return "Designs system architecture and technical solutions"
}

func (a *ArchitectAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "system_design", Description: "Design system architecture", Required: true},
		{Name: "tech_stack", Description: "Select technology stack", Required: true},
	}
}

func (a *ArchitectAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	result := &agents.Result{
		Success:     true,
		Output:      fmt.Sprintf("Architecture design for: %s", task.Input),
		Confidence:  9.0,
		ExecutionMS: time.Since(startTime).Milliseconds(),
		NextAgent:   agents.DevelopmentAgent,
	}
	agents.RecordExecution(a.GetType(), result)
	return result, nil
}
