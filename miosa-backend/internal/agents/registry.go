package agents

import (
	"fmt"
	"sync"
)

// Registry manages all registered agents
type Registry struct {
	agents map[AgentType]Agent
	mu     sync.RWMutex
}

// Global registry instance
var defaultRegistry = &Registry{
	agents: make(map[AgentType]Agent),
}

// Register adds an agent to the registry
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