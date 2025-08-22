package agents

import (
    "fmt"
    "sync"
    "time"
)

//
// ===== Registry definition =====
//

// Registry manages all registered agents
type Registry struct {
    agents       map[AgentType]Agent
    tools        map[string]Tool
    toolsByAgent map[AgentType][]string
    evaluations  map[AgentType]*AgentEvaluation
    mu           sync.RWMutex
}

// AgentEvaluation tracks agent performance metrics
type AgentEvaluation struct {
    TotalExecutions     int64
    SuccessfulExecutions int64
    FailedExecutions    int64
    AverageConfidence   float64
    AverageExecutionMS  int64
    LastEvaluated       time.Time
}

// Global registry instance
var defaultRegistry = &Registry{
    agents:       make(map[AgentType]Agent),
    tools:        make(map[string]Tool),
    toolsByAgent: make(map[AgentType][]string),
    evaluations:  make(map[AgentType]*AgentEvaluation),
}

//
// ===== Strategic Reasoning integration =====
//

// sharedStrategicReasoner is created once and reused for all wrapped agents.
var sharedStrategicReasoner StrategicReasoner

// InitStrategicReasoner configures the default LLM-based strategic reasoner.
// Call this ONCE at application startup BEFORE registering agents.
func InitStrategicReasoner(client ChatClient) {
    sharedStrategicReasoner = NewDefaultStrategicReasoner(DefaultStrategicReasonerConfig{
        Client:        client,
        Model:         "your-model-name", // TODO: move to config/env
        Temperature:   0.7,
        MaxTokens:     1200,
        MinCandidates: 3,
        MaxCandidates: 3,
        TaskText: func(t Task) string {
            // Adjust this if Task has a specific field with the natural language input
            return fmt.Sprintf("%+v", t)
        },
    })
}

// RegisterWithStrategic wraps the given agent with the StrategicExecutor before registering.
// Requires InitStrategicReasoner to have been called. If not, falls back to normal Register.
func RegisterWithStrategic(agent Agent, minCandidates int) error {
    if agent == nil {
        return fmt.Errorf("cannot register nil agent")
    }

    if sharedStrategicReasoner != nil {
        agent = NewStrategicExecutor(agent, sharedStrategicReasoner, minCandidates)
    }
    return Register(agent)
}

//
// ===== Agent registration and retrieval =====
//

// Register adds an agent to the registry without any wrapper.
func Register(agent Agent) error {
    if agent == nil {
        return fmt.Errorf("cannot register nil agent")
    }

    defaultRegistry.mu.Lock()
    defer defaultRegistry.mu.Unlock()

    agentType := agent.GetType()
    if _, exists := defaultRegistry.agents[agentType]; exists {
        return fmt.Errorf("agent %s already registered", agentType)
    }

    defaultRegistry.agents[agentType] = agent
    return nil
}

// Get retrieves an agent by type
func Get(agentType AgentType) (Agent, error) {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    agent, exists := defaultRegistry.agents[agentType]
    if !exists {
        return nil, fmt.Errorf("agent %s not registered", agentType)
    }
    return agent, nil
}

// MustGet retrieves an agent by type or panics if not found
func MustGet(agentType AgentType) Agent {
    agent, err := Get(agentType)
    if err != nil {
        panic(err)
    }
    return agent
}

// ListAgents returns all registered agent types
func ListAgents() []AgentType {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    types := make([]AgentType, 0, len(defaultRegistry.agents))
    for t := range defaultRegistry.agents {
        types = append(types, t)
    }
    return types
}

// GetAll returns all registered agents
func GetAll() map[AgentType]Agent {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    agents := make(map[AgentType]Agent)
    for k, v := range defaultRegistry.agents {
        agents[k] = v
    }
    return agents
}

// IsRegistered checks if an agent type is registered
func IsRegistered(agentType AgentType) bool {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    _, exists := defaultRegistry.agents[agentType]
    return exists
}

// Clear removes all agents from the registry (useful for testing)
func Clear() {
    defaultRegistry.mu.Lock()
    defer defaultRegistry.mu.Unlock()

    defaultRegistry.agents = make(map[AgentType]Agent)
}

//
// ===== Capabilities =====
//

// GetCapabilities returns all capabilities of a specific agent
func GetCapabilities(agentType AgentType) ([]Capability, error) {
    agent, err := Get(agentType)
    if err != nil {
        return nil, err
    }
    return agent.GetCapabilities(), nil
}

// GetAllCapabilities returns capabilities of all registered agents
func GetAllCapabilities() map[AgentType][]Capability {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    capabilities := make(map[AgentType][]Capability)
    for agentType, agent := range defaultRegistry.agents {
        capabilities[agentType] = agent.GetCapabilities()
    }
    return capabilities
}

//
// ===== Tool Management =====
//

// RegisterTool adds a tool to the registry
func RegisterTool(tool Tool) error {
    if tool == nil {
        return fmt.Errorf("cannot register nil tool")
    }

    defaultRegistry.mu.Lock()
    defer defaultRegistry.mu.Unlock()

    toolName := tool.GetName()
    if _, exists := defaultRegistry.tools[toolName]; exists {
        return fmt.Errorf("tool %s already registered", toolName)
    }

    defaultRegistry.tools[toolName] = tool
    return nil
}

// GetTool retrieves a tool by name
func GetTool(name string) (Tool, error) {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    tool, exists := defaultRegistry.tools[name]
    if !exists {
        return nil, fmt.Errorf("tool %s not registered", name)
    }
    return tool, nil
}

// RegisterToolForAgent associates a tool with an agent
func RegisterToolForAgent(agentType AgentType, toolName string) error {
    defaultRegistry.mu.Lock()
    defer defaultRegistry.mu.Unlock()

    if _, exists := defaultRegistry.agents[agentType]; !exists {
        return fmt.Errorf("agent %s not registered", agentType)
    }

    if _, exists := defaultRegistry.tools[toolName]; !exists {
        return fmt.Errorf("tool %s not registered", toolName)
    }

    // Avoid duplicate registration
    tools := defaultRegistry.toolsByAgent[agentType]
    for _, t := range tools {
        if t == toolName {
            return nil
        }
    }

    defaultRegistry.toolsByAgent[agentType] = append(tools, toolName)
    return nil
}

// GetToolsForAgent returns all tools available to a specific agent
func GetToolsForAgent(agentType AgentType) ([]Tool, error) {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    toolNames, exists := defaultRegistry.toolsByAgent[agentType]
    if !exists {
        return []Tool{}, nil
    }

    tools := make([]Tool, 0, len(toolNames))
    for _, name := range toolNames {
        if tool, exists := defaultRegistry.tools[name]; exists {
            tools = append(tools, tool)
        }
    }

    return tools, nil
}

//
// ===== Evaluation Methods =====
//

// RecordExecution records the result of an agent execution for evaluation
func RecordExecution(agentType AgentType, result *Result) {
    defaultRegistry.mu.Lock()
    defer defaultRegistry.mu.Unlock()

    eval, exists := defaultRegistry.evaluations[agentType]
    if !exists {
        eval = &AgentEvaluation{}
        defaultRegistry.evaluations[agentType] = eval
    }

    eval.TotalExecutions++
    if result.Success {
        eval.SuccessfulExecutions++
    } else {
        eval.FailedExecutions++
    }

    // Running average for confidence
    if eval.TotalExecutions == 1 {
        eval.AverageConfidence = result.Confidence
    } else {
        eval.AverageConfidence =
            (eval.AverageConfidence*float64(eval.TotalExecutions-1) + result.Confidence) /
                float64(eval.TotalExecutions)
    }

    // Running average for execution time
    if eval.TotalExecutions == 1 {
        eval.AverageExecutionMS = result.ExecutionMS
    } else {
        eval.AverageExecutionMS =
            (eval.AverageExecutionMS*(eval.TotalExecutions-1) + result.ExecutionMS) /
                eval.TotalExecutions
    }

    eval.LastEvaluated = time.Now()
}

// GetEvaluation returns the evaluation metrics for an agent
func GetEvaluation(agentType AgentType) (*AgentEvaluation, error) {
    defaultRegistry.mu.RLock()
    defer defaultRegistry.mu.RUnlock()

    eval, exists := defaultRegistry.evaluations[agentType]
    if !exists {
        return nil, fmt.Errorf("no evaluation data for agent %s", agentType)
    }

    return &AgentEvaluation{
        TotalExecutions:     eval.TotalExecutions,
        SuccessfulExecutions: eval.SuccessfulExecutions,
        FailedExecutions:    eval.FailedExecutions,
        AverageConfidence:   eval.AverageConfidence,
        AverageExecutionMS:  eval.AverageExecutionMS,
        LastEvaluated:       eval.LastEvaluated,
    }, nil
}
