package agents

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AgentType represents the type of agent
type AgentType string

const (
	OrchestratorAgent  AgentType = "orchestrator"
	CommunicationAgent AgentType = "communication"
	AnalysisAgent      AgentType = "analysis"
	DevelopmentAgent   AgentType = "development"
	StrategyAgent      AgentType = "strategy"
	DeploymentAgent    AgentType = "deployment"
	QualityAgent       AgentType = "quality"
	MonitoringAgent    AgentType = "monitoring"
	IntegrationAgent   AgentType = "integration"
)

// Agent is the interface that all agents must implement
type Agent interface {
	GetType() AgentType
	GetCapabilities() []Capability
	Execute(ctx context.Context, task Task) (*Result, error)
	GetDescription() string
}

// Task represents a task to be executed by an agent
type Task struct {
	ID         uuid.UUID              `json:"id"`
	Type       string                 `json:"type"`
	Input      string                 `json:"input"`
	Parameters map[string]interface{} `json:"parameters"`
	Context    *TaskContext           `json:"context"`
	Priority   int                    `json:"priority"`
	Timeout    time.Duration          `json:"timeout"`
}

// TaskContext provides context for task execution
type TaskContext struct {
	UserID         uuid.UUID              `json:"user_id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	WorkspaceID    uuid.UUID              `json:"workspace_id"`
	SessionID      uuid.UUID              `json:"session_id"`
	ConsultationID uuid.UUID              `json:"consultation_id,omitempty"`
	Phase          string                 `json:"phase"`
	Memory         map[string]interface{} `json:"memory"`
	History        []Message              `json:"history"`
	Metadata       map[string]string      `json:"metadata"`
}

// Result represents the result of an agent execution
type Result struct {
	Success      bool                   `json:"success"`
	Output       string                 `json:"output"`
	Data         map[string]interface{} `json:"data"`
	NextStep     string                 `json:"next_step,omitempty"`
	NextAgent    AgentType              `json:"next_agent,omitempty"`
	Confidence   float64                `json:"confidence"`
	ExecutionMS  int64                  `json:"execution_ms"`
	Error        error                  `json:"error,omitempty"`
	Suggestions  []string               `json:"suggestions,omitempty"`
}

// Capability represents a capability of an agent
type Capability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Version     string `json:"version"`
}

// Message represents a message in conversation history
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// AgentConfig holds configuration for an agent
type AgentConfig struct {
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
	TopP        float64                `json:"top_p"`
	Settings    map[string]interface{} `json:"settings"`
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	StatusPending    ExecutionStatus = "pending"
	StatusInProgress ExecutionStatus = "in_progress"
	StatusCompleted  ExecutionStatus = "completed"
	StatusFailed     ExecutionStatus = "failed"
	StatusCancelled  ExecutionStatus = "cancelled"
)