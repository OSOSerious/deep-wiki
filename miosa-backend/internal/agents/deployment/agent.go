package deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

type DeploymentAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

func New(groqClient *groq.Client) agents.Agent {
	return &DeploymentAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "llama-3.1-8b-instant",
			MaxTokens:   2000,
			Temperature: 0.3,
			TopP:        0.9,
		},
	}
}

func (a *DeploymentAgent) GetType() agents.AgentType {
	return agents.DeploymentAgent
}

func (a *DeploymentAgent) GetDescription() string {
	return "Handles deployment to various cloud platforms"
}

func (a *DeploymentAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "deploy", Description: "Deploy applications", Required: true},
		{Name: "ci_cd", Description: "Setup CI/CD pipelines", Required: false},
	}
}

func (a *DeploymentAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	result := &agents.Result{
		Success:     true,
		Output:      fmt.Sprintf("Deployment configuration for: %s", task.Input),
		Confidence:  8.0,
		ExecutionMS: time.Since(startTime).Milliseconds(),
		NextAgent:   agents.MonitoringAgent,
	}
	agents.RecordExecution(a.GetType(), result)
	return result, nil
}
