package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

// Orchestrator is the main agent that routes tasks to other agents
type Orchestrator struct {
	groqClient           *groq.Client
	config               AgentConfig
	logger               *zap.Logger
	workflowAnalyzer     *WorkflowAnalyzer
	improvementEngine    *ImprovementEngine
	vectorStore          VectorStore
	confidenceThreshold  float64
	subtaskScores        map[uuid.UUID]*SubtaskScore
	mu                   sync.RWMutex
}

// WorkflowAnalyzer analyzes workflow patterns and stores insights
type WorkflowAnalyzer struct {
	vectorDB     VectorStore
	patterns     map[string]*WorkflowPattern
	mu           sync.RWMutex
}

// WorkflowPattern represents a learned workflow pattern
type WorkflowPattern struct {
	ID           uuid.UUID
	TaskType     string
	AgentSequence []AgentType
	SuccessRate  float64
	AvgDuration  time.Duration
	Confidence   float64
	Embedding    pgvector.Vector
	LastUpdated  time.Time
}

// ImprovementEngine suggests improvements based on workflow analysis
type ImprovementEngine struct {
	analyzer     *WorkflowAnalyzer
	suggestions  []ImprovementSuggestion
	mu           sync.RWMutex
}

// ImprovementSuggestion represents a suggested improvement
type ImprovementSuggestion struct {
	ID           uuid.UUID
	TargetAgent  AgentType
	Suggestion   string
	Confidence   float64  // 0-10 scale as discussed
	Priority     int
	PullRequest  *PullRequestDraft
	CreatedAt    time.Time
}

// PullRequestDraft represents an auto-generated PR
type PullRequestDraft struct {
	Title       string
	Description string
	Changes     []CodeChange
	Branch      string
}

// CodeChange represents a code modification
type CodeChange struct {
	File    string
	OldCode string
	NewCode string
	Reason  string
}

// SubtaskScore tracks individual subtask performance
type SubtaskScore struct {
	SubtaskID    uuid.UUID
	Agent        AgentType
	Description  string
	Score        float64  // 0-10 scale
	Confidence   float64
	ExecutionMS  int64
	Success      bool
	Timestamp    time.Time
}

// VectorStore interface for vector database operations
type VectorStore interface {
	Store(ctx context.Context, id uuid.UUID, embedding pgvector.Vector, metadata map[string]interface{}) error
	Search(ctx context.Context, query pgvector.Vector, limit int) ([]SearchResult, error)
	Update(ctx context.Context, id uuid.UUID, embedding pgvector.Vector, metadata map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SearchResult from vector store
type SearchResult struct {
	ID       uuid.UUID
	Distance float64
	Metadata map[string]interface{}
}

// NewOrchestrator creates a new orchestrator agent
func NewOrchestrator(groqClient *groq.Client, logger *zap.Logger, vectorStore VectorStore) *Orchestrator {
	o := &Orchestrator{
		groqClient: groqClient,
		config: AgentConfig{
			Model:       "moonshotai/kimi-k2-instruct",  // Kimi K2 for orchestration
			MaxTokens:   16384,  // K2 supports up to 16K output
			Temperature: 0.3,    // Low temp for consistent orchestration
			TopP:        0.9,
		},
		logger:              logger,
		vectorStore:         vectorStore,
		confidenceThreshold: 7.0,  // Out of 10 scale
		subtaskScores:       make(map[uuid.UUID]*SubtaskScore),
	}
	
	o.workflowAnalyzer = &WorkflowAnalyzer{
		vectorDB: vectorStore,
		patterns: make(map[string]*WorkflowPattern),
	}
	
	o.improvementEngine = &ImprovementEngine{
		analyzer:    o.workflowAnalyzer,
		suggestions: make([]ImprovementSuggestion, 0),
	}
	
	return o
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
	
	// Check for learned patterns first
	if pattern := o.workflowAnalyzer.FindBestPattern(ctx, task); pattern != nil {
		if pattern.Confidence >= o.confidenceThreshold {
			o.logger.Info("Using learned workflow pattern",
				zap.String("pattern_id", pattern.ID.String()),
				zap.Float64("confidence", pattern.Confidence))
			return o.executeLearnedPattern(ctx, task, pattern)
		}
	}
	
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
	
	// Score the execution (0-10 scale)
	score := o.scoreExecution(result)
	
	// Record subtask score
	subtaskScore := &SubtaskScore{
		SubtaskID:   task.ID,
		Agent:       AgentType(routing.Agent),
		Description: task.Input,
		Score:       score,
		Confidence:  routing.Confidence * 10, // Convert to 0-10 scale
		ExecutionMS: result.ExecutionMS,
		Success:     result.Success,
		Timestamp:   time.Now(),
	}
	
	o.mu.Lock()
	o.subtaskScores[task.ID] = subtaskScore
	o.mu.Unlock()
	
	// Store workflow in vector DB for learning
	go o.storeWorkflowPattern(ctx, task, routing, result, score)
	
	// Analyze for improvements if score is low
	if score < o.confidenceThreshold {
		go o.analyzeForImprovements(ctx, task, result, score)
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
	
	// Record execution for evaluation
	RecordExecution(OrchestratorAgent, result)
	
	return result, nil
}

// scoreExecution scores the execution result on a 0-10 scale
func (o *Orchestrator) scoreExecution(result *Result) float64 {
	score := 5.0 // Base score
	
	if result.Success {
		score += 2.0
	}
	
	// Adjust based on confidence
	score += (result.Confidence * 2) // Add up to 2 points for confidence
	
	// Adjust based on execution time (faster is better)
	if result.ExecutionMS < 1000 {
		score += 1.0
	} else if result.ExecutionMS > 10000 {
		score -= 1.0
	}
	
	// Cap at 10
	if score > 10 {
		score = 10
	}
	if score < 0 {
		score = 0
	}
	
	return score
}

// storeWorkflowPattern stores successful workflow patterns for learning
func (o *Orchestrator) storeWorkflowPattern(ctx context.Context, task Task, routing *AgentRoutingDecision, result *Result, score float64) {
	if o.vectorStore == nil {
		return
	}
	
	// Generate embedding from task description
	embedding := o.generateEmbedding(ctx, task.Input)
	
	metadata := map[string]interface{}{
		"task_type":    task.Type,
		"agent_used":   routing.Agent,
		"success":      result.Success,
		"score":        score,
		"confidence":   routing.Confidence,
		"execution_ms": result.ExecutionMS,
		"timestamp":    time.Now(),
	}
	
	if err := o.vectorStore.Store(ctx, task.ID, embedding, metadata); err != nil {
		if o.logger != nil {
			o.logger.Error("Failed to store workflow pattern", zap.Error(err))
		}
	}
}

// analyzeForImprovements triggers improvement analysis for low-scoring executions
func (o *Orchestrator) analyzeForImprovements(ctx context.Context, task Task, result *Result, score float64) {
	if o.improvementEngine == nil {
		return
	}
	
	suggestion := o.improvementEngine.GenerateSuggestion(ctx, task, result, score)
	if suggestion != nil {
		// Check if we should create a pull request
		if suggestion.Confidence >= 8.0 { // High confidence improvement
			pr := o.generatePullRequest(ctx, suggestion)
			if pr != nil {
				suggestion.PullRequest = pr
				o.notifyImprovement(ctx, suggestion)
			}
		}
		
		o.improvementEngine.mu.Lock()
		o.improvementEngine.suggestions = append(o.improvementEngine.suggestions, *suggestion)
		o.improvementEngine.mu.Unlock()
	}
}

// executeLearnedPattern executes a task using a learned workflow pattern
func (o *Orchestrator) executeLearnedPattern(ctx context.Context, task Task, pattern *WorkflowPattern) (*Result, error) {
	startTime := time.Now()
	var lastResult *Result
	
	for _, agentType := range pattern.AgentSequence {
		agent, err := Get(agentType)
		if err != nil {
			continue
		}
		
		result, err := agent.Execute(ctx, task)
		if err != nil {
			continue
		}
		
		lastResult = result
		if result.Success {
			break
		}
	}
	
	if lastResult == nil {
		lastResult = &Result{
			Success: false,
			Error:   fmt.Errorf("learned pattern execution failed"),
		}
	}
	
	lastResult.ExecutionMS = time.Since(startTime).Milliseconds()
	return lastResult, nil
}

// generateEmbedding generates a vector embedding for text
func (o *Orchestrator) generateEmbedding(ctx context.Context, text string) pgvector.Vector {
	// For now, return a simple embedding - in production, use an embedding model
	// Could use a small model (SML) as mentioned for subagents
	return pgvector.NewVector(make([]float32, 768)) // Standard embedding size
}

// generatePullRequest creates a PR draft for an improvement
func (o *Orchestrator) generatePullRequest(ctx context.Context, suggestion *ImprovementSuggestion) *PullRequestDraft {
	// Generate PR based on suggestion
	return &PullRequestDraft{
		Title:       fmt.Sprintf("Improve %s agent performance", suggestion.TargetAgent),
		Description: suggestion.Suggestion,
		Branch:      fmt.Sprintf("auto-improve-%s-%d", suggestion.TargetAgent, time.Now().Unix()),
		Changes:     []CodeChange{},
	}
}

// notifyImprovement sends notification about improvement suggestion
func (o *Orchestrator) notifyImprovement(ctx context.Context, suggestion *ImprovementSuggestion) {
	if o.logger != nil {
		o.logger.Info("Improvement suggestion generated",
			zap.String("target_agent", string(suggestion.TargetAgent)),
			zap.Float64("confidence", suggestion.Confidence),
			zap.String("suggestion", suggestion.Suggestion))
	}
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

// WorkflowAnalyzer methods

// FindBestPattern finds the best matching workflow pattern
func (wa *WorkflowAnalyzer) FindBestPattern(ctx context.Context, task Task) *WorkflowPattern {
	wa.mu.RLock()
	defer wa.mu.RUnlock()
	
	var bestPattern *WorkflowPattern
	var bestScore float64
	
	for _, pattern := range wa.patterns {
		if pattern.TaskType == task.Type {
			score := pattern.SuccessRate * pattern.Confidence
			if score > bestScore {
				bestScore = score
				bestPattern = pattern
			}
		}
	}
	
	return bestPattern
}

// ImprovementEngine methods

// GenerateSuggestion generates improvement suggestions
func (ie *ImprovementEngine) GenerateSuggestion(ctx context.Context, task Task, result *Result, score float64) *ImprovementSuggestion {
	return &ImprovementSuggestion{
		ID:          uuid.New(),
		TargetAgent: OrchestratorAgent,
		Suggestion:  fmt.Sprintf("Task scored %.1f/10. Consider optimizing execution path or using different agent.", score),
		Confidence:  (10 - score), // Higher improvement confidence for lower scores
		Priority:    int(10 - score),
		CreatedAt:   time.Now(),
	}
}

// GetSubtaskScores returns scores for all subtasks
func (o *Orchestrator) GetSubtaskScores() map[uuid.UUID]*SubtaskScore {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	scores := make(map[uuid.UUID]*SubtaskScore)
	for id, score := range o.subtaskScores {
		scores[id] = score
	}
	return scores
}

// GetImprovementSuggestions returns all improvement suggestions
func (o *Orchestrator) GetImprovementSuggestions() []ImprovementSuggestion {
	if o.improvementEngine == nil {
		return []ImprovementSuggestion{}
	}
	
	o.improvementEngine.mu.RLock()
	defer o.improvementEngine.mu.RUnlock()
	
	return o.improvementEngine.suggestions
}