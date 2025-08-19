package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
)

// Orchestrator is the main agent that routes tasks to other agents
type Orchestrator struct {
	groqClient *groq.Client
	config     AgentConfig
}

// NewOrchestrator creates a new orchestrator agent
func NewOrchestrator(groqClient *groq.Client) *Orchestrator {
	return &Orchestrator{
		groqClient: groqClient,
		config: AgentConfig{
			Model:       "moonshotai/kimi-k2-instruct",
			MaxTokens:   4000,
			Temperature: 0.3,
			TopP:        0.9,
		},
	}
}

// GetType returns the agent type
func (o *Orchestrator) GetType() AgentType {
	return OrchestratorAgent
}

// GetDescription returns the agent description
func (o *Orchestrator) GetDescription() string {
	return "Orchestrates and routes tasks to appropriate specialized agents"
}

// GetCapabilities returns the orchestrator's capabilities
func (o *Orchestrator) GetCapabilities() []Capability {
	return []Capability{
		{Name: "task_routing", Description: "Routes tasks to appropriate agents", Required: true},
		{Name: "task_planning", Description: "Breaks down complex tasks into steps", Required: true},
		{Name: "coordination", Description: "Coordinates multi-agent workflows", Required: true},
		{Name: "monitoring", Description: "Monitors agent execution", Required: false},
	}
}

// AgentRoutingDecision represents the routing decision
type AgentRoutingDecision struct {
	Agent      string   `json:"agent"`
	Reasoning  string   `json:"reasoning"`
	Confidence float64  `json:"confidence"`
	Subtasks   []string `json:"subtasks,omitempty"`
}

// Execute processes a task and routes it to the appropriate agent
func (o *Orchestrator) Execute(ctx context.Context, task Task) (*Result, error) {
	startTime := time.Now()
	
	// Analyze task to determine routing
	routing, err := o.analyzeAndRoute(ctx, task)
	if err != nil {
		return &Result{
			Success:     false,
			Error:       err,
			ExecutionMS: time.Since(startTime).Milliseconds(),
		}, err
	}
	
	// Get the target agent
	targetAgent, err := Get(AgentType(routing.Agent))
	if err != nil {
		// If specific agent not found, use communication agent as fallback
		log.Printf("Agent %s not found, falling back to communication agent: %v", routing.Agent, err)
		targetAgent, err = Get(CommunicationAgent)
		if err != nil {
			return &Result{
				Success:     false,
				Error:       fmt.Errorf("failed to get fallback agent: %w", err),
				ExecutionMS: time.Since(startTime).Milliseconds(),
			}, err
		}
	}
	
	// Execute with the selected agent
	result, err := targetAgent.Execute(ctx, task)
	if err != nil {
		return &Result{
			Success:     false,
			Error:       err,
			ExecutionMS: time.Since(startTime).Milliseconds(),
		}, err
	}
	
	// Add orchestration metadata
	if result.Data == nil {
		result.Data = make(map[string]interface{})
	}
	result.Data["orchestration"] = map[string]interface{}{
		"routed_to":  routing.Agent,
		"reasoning":  routing.Reasoning,
		"confidence": routing.Confidence,
	}
	
	result.ExecutionMS = time.Since(startTime).Milliseconds()
	return result, nil
}

// analyzeAndRoute uses AI to determine which agent should handle the task
func (o *Orchestrator) analyzeAndRoute(ctx context.Context, task Task) (*AgentRoutingDecision, error) {
	systemPrompt := `You are the MIOSA orchestrator agent. Analyze the given task and determine which specialized agent should handle it.

Available agents:
- communication: Handles user interactions, chat responses, UI/UX communications
- analysis: Performs deep business analysis, market research, strategic insights
- strategy: Creates business strategies, roadmaps, long-term planning
- development: Generates code, creates applications, technical implementation
- quality: Ensures code quality, runs tests, validates implementations
- deployment: Manages deployments, infrastructure, cloud resources
- monitoring: Monitors application health, performance metrics, system status
- integration: Handles integrations with external services, APIs, third-party tools

Respond with JSON format:
{
  "agent": "agent_name",
  "reasoning": "why this agent is best suited",
  "confidence": 0.0-1.0,
  "subtasks": ["optional", "list", "of", "subtasks"]
}`

	userPrompt := fmt.Sprintf("Task Type: %s\nInput: %s\nContext Phase: %s\nParameters: %v",
		task.Type, task.Input, task.Context.Phase, task.Parameters)
	
	response, err := o.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: groq.ChatModel(o.config.Model),
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MaxTokens:   o.config.MaxTokens,
		Temperature: o.config.Temperature,
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to analyze task: %w", err)
	}
	
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI model")
	}
	
	// Parse the JSON response
	var decision AgentRoutingDecision
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &decision); err != nil {
		// Fallback to communication agent if parsing fails
		return &AgentRoutingDecision{
			Agent:      "communication",
			Reasoning:  "Failed to parse routing decision, defaulting to communication",
			Confidence: 0.5,
		}, nil
	}
	
	return &decision, nil
}

// PlanWorkflow breaks down a complex task into steps
func (o *Orchestrator) PlanWorkflow(ctx context.Context, task Task) ([]Task, error) {
	systemPrompt := `Break down this complex task into a series of steps. Each step should be handled by a specific agent.
Respond with a JSON array of steps, each containing:
{
  "step_number": 1,
  "agent": "agent_name",
  "description": "what this step does",
  "dependencies": [0], // step numbers this depends on
  "input": "specific input for this step"
}`

	response, err := o.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: groq.ChatModel(o.config.Model),
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Task: %s", task.Input),
			},
		},
		MaxTokens:   o.config.MaxTokens,
		Temperature: 0.2, // Lower temperature for planning
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to plan workflow: %w", err)
	}
	
	// Parse and convert to tasks
	var steps []struct {
		StepNumber   int    `json:"step_number"`
		Agent        string `json:"agent"`
		Description  string `json:"description"`
		Dependencies []int  `json:"dependencies"`
		Input        string `json:"input"`
	}
	
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &steps); err != nil {
		return nil, fmt.Errorf("failed to parse workflow plan: %w", err)
	}
	
	// Convert to tasks
	tasks := make([]Task, len(steps))
	for i, step := range steps {
		tasks[i] = Task{
			ID:      uuid.New(),
			Type:    step.Agent,
			Input:   step.Input,
			Context: task.Context,
			Parameters: map[string]interface{}{
				"step_number":  step.StepNumber,
				"description":  step.Description,
				"dependencies": step.Dependencies,
			},
		}
	}
	
	return tasks, nil
}