package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

type StrategyAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

func New(groqClient *groq.Client) agents.Agent {
	return &StrategyAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "moonshotai/kimi-k2-instruct",
			MaxTokens:   4000,
			Temperature: 0.5,
			TopP:        0.95,
		},
	}
}

func (a *StrategyAgent) GetType() agents.AgentType {
	return agents.StrategyAgent
}

func (a *StrategyAgent) GetDescription() string {
	return "Develops strategic plans and roadmaps"
}

func (a *StrategyAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "planning", Description: "Strategic planning", Required: true},
		{Name: "roadmap", Description: "Create roadmaps", Required: true},
	}
}

func (a *StrategyAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	result := &agents.Result{
		Success:     true,
		Output:      fmt.Sprintf("Strategic plan for: %s", task.Input),
		Confidence:  9.0,
		ExecutionMS: time.Since(startTime).Milliseconds(),
		NextAgent:   agents.AnalysisAgent,
	}
	agents.RecordExecution(a.GetType(), result)
	return result, nil
}
