package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

type MonitoringAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

func New(groqClient *groq.Client) agents.Agent {
	return &MonitoringAgent{
		groqClient: groqClient,
		config: agents.AgentConfig{
			Model:       "llama-3.1-8b-instant",
			MaxTokens:   1500,
			Temperature: 0.4,
			TopP:        0.9,
		},
	}
}

func (a *MonitoringAgent) GetType() agents.AgentType {
	return agents.MonitoringAgent
}

func (a *MonitoringAgent) GetDescription() string {
	return "Sets up monitoring, logging, and observability"
}

func (a *MonitoringAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "monitoring", Description: "Setup monitoring", Required: true},
		{Name: "alerts", Description: "Configure alerts", Required: false},
	}
}

func (a *MonitoringAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	result := &agents.Result{
		Success:     true,
		Output:      fmt.Sprintf("Monitoring setup for: %s", task.Input),
		Confidence:  8.5,
		ExecutionMS: time.Since(startTime).Milliseconds(),
	}
	agents.RecordExecution(a.GetType(), result)
	return result, nil
}
