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
	ArchitectAgent     AgentType = "architect"
	RecommenderAgent   AgentType = "recommender"
	AIProvidersAgent   AgentType = "ai_providers"
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

// Tool represents a tool available to agents
type Tool interface {
	GetName() string
	GetDescription() string
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
	Validate(input map[string]interface{}) error
}

// Workflow represents a multi-agent workflow
type Workflow struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Steps       []WorkflowStep  `json:"steps"`
	Status      ExecutionStatus `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// WorkflowStep represents a step in a workflow
type WorkflowStep struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	AgentType    AgentType              `json:"agent_type"`
	Task         Task                   `json:"task"`
	Dependencies []uuid.UUID            `json:"dependencies"`
	Status       ExecutionStatus        `json:"status"`
	Result       *Result                `json:"result,omitempty"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	Timeout      time.Duration          `json:"timeout"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AgentPool represents a pool of agents
type AgentPool interface {
	GetAgent(agentType AgentType) (Agent, error)
	RegisterAgent(agent Agent) error
	UnregisterAgent(agentType AgentType) error
	ListAgents() []AgentType
	HealthCheck(ctx context.Context) map[AgentType]bool
}

// MemoryStore represents a memory store for agents
type MemoryStore interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	GetHistory(ctx context.Context, sessionID uuid.UUID) ([]Message, error)
	AddMessage(ctx context.Context, sessionID uuid.UUID, message Message) error
}

// EventType represents the type of event
type EventType string

const (
	EventTaskCreated   EventType = "task_created"
	EventTaskStarted   EventType = "task_started"
	EventTaskCompleted EventType = "task_completed"
	EventTaskFailed    EventType = "task_failed"
	EventAgentStarted  EventType = "agent_started"
	EventAgentFinished EventType = "agent_finished"
	EventToolExecuted  EventType = "tool_executed"
)

// Event represents an event in the system
type Event struct {
	ID        uuid.UUID              `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
}

// EventHandler handles events
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

// Phase represents a phase in the MIOSA loop
type Phase string

const (
	PhaseOnboarding   Phase = "onboarding"
	PhaseConsultation Phase = "consultation"
	PhaseAnalysis     Phase = "analysis"
	PhaseStrategy     Phase = "strategy"
	PhaseDevelopment  Phase = "development"
	PhaseTesting      Phase = "testing"
	PhaseDeployment   Phase = "deployment"
	PhaseMonitoring   Phase = "monitoring"
	PhaseOptimization Phase = "optimization"
	PhaseExpansion    Phase = "expansion"
)