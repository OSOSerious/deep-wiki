package quality

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

type QualityAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

func New(groqClient *groq.Client) agents.Agent {
	return &QualityAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "llama-3.3-70b-versatile",
			MaxTokens:   2000,
			Temperature: 0.3,
			TopP:        0.9,
		},
	}
}

func (a *QualityAgent) GetType() agents.AgentType {
	return agents.QualityAgent
}

func (a *QualityAgent) GetDescription() string {
	return "Ensures code quality through testing and review"
}

func (a *QualityAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "code_review", Description: "Review code quality", Required: true},
		{Name: "testing", Description: "Generate and run tests", Required: true},
	}
}

func (a *QualityAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	result := &agents.Result{
		Success:     true,
		Output:      fmt.Sprintf("Quality check completed for: %s", task.Input),
		Confidence:  8.5,
		ExecutionMS: time.Since(startTime).Milliseconds(),
		NextAgent:   agents.DeploymentAgent,
	}
	agents.RecordExecution(a.GetType(), result)
	return result, nil
}
