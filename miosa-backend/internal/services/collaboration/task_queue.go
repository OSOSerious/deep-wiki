package collaboration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
)

// TaskQueue represents a distributed task queue for agent collaboration
type TaskQueue struct {
	redisClient *redis.Client
	logger      *zap.Logger
	subscribers map[agents.AgentType]*TaskSubscriber
	mu          sync.RWMutex
}

// CollaborativeTask represents a task that can be passed between agents
type CollaborativeTask struct {
	ID              uuid.UUID                `json:"id"`
	ParentID        *uuid.UUID               `json:"parent_id,omitempty"`
	Type            string                   `json:"type"`
	Priority        int                      `json:"priority"`
	Status          TaskStatus               `json:"status"`
	AssignedAgent   agents.AgentType         `json:"assigned_agent"`
	CreatedBy       agents.AgentType         `json:"created_by"`
	Input           string                   `json:"input"`
	Context         map[string]interface{}   `json:"context"`
	Dependencies    []uuid.UUID              `json:"dependencies"`
	Result          *agents.Result           `json:"result,omitempty"`
	ConfidenceScore float64                  `json:"confidence_score"`
	Feedback        []FeedbackEntry          `json:"feedback"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	Deadline        *time.Time               `json:"deadline,omitempty"`
	RetryCount      int                      `json:"retry_count"`
	MaxRetries      int                      `json:"max_retries"`
}

// TaskStatus represents the status of a collaborative task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// FeedbackEntry represents feedback from an agent about a task
type FeedbackEntry struct {
	AgentType   agents.AgentType       `json:"agent_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        FeedbackType           `json:"type"`
	Message     string                 `json:"message"`
	Confidence  float64                `json:"confidence"`
	Suggestions []string               `json:"suggestions"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// FeedbackType categorizes different types of feedback
type FeedbackType string

const (
	FeedbackTypeSuccess     FeedbackType = "success"
	FeedbackTypeImprovement FeedbackType = "improvement"
	FeedbackTypeError       FeedbackType = "error"
	FeedbackTypeHandoff     FeedbackType = "handoff"
	FeedbackTypeCollaborate FeedbackType = "collaborate"
)

// TaskSubscriber handles task subscriptions for an agent
type TaskSubscriber struct {
	AgentType    agents.AgentType
	Channel      string
	Handler      TaskHandler
	Capabilities []string
	Active       bool
}

// TaskHandler is a function that processes collaborative tasks
type TaskHandler func(ctx context.Context, task *CollaborativeTask) error

// NewTaskQueue creates a new distributed task queue
func NewTaskQueue(redisClient *redis.Client, logger *zap.Logger) *TaskQueue {
	return &TaskQueue{
		redisClient: redisClient,
		logger:      logger,
		subscribers: make(map[agents.AgentType]*TaskSubscriber),
	}
}

// PublishTask publishes a task to the queue for an agent to pick up
func (tq *TaskQueue) PublishTask(ctx context.Context, task *CollaborativeTask) error {
	task.ID = uuid.New()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = TaskStatusPending

	// Store task in Redis
	taskKey := fmt.Sprintf("task:%s", task.ID)
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Set task with expiration (24 hours)
	if err := tq.redisClient.Set(ctx, taskKey, taskData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store task: %w", err)
	}

	// Add to priority queue
	queueKey := fmt.Sprintf("queue:%s", task.AssignedAgent)
	score := float64(task.Priority)
	if task.Deadline != nil {
		// Higher score for tasks closer to deadline
		score += float64(time.Until(*task.Deadline).Seconds())
	}

	if err := tq.redisClient.ZAdd(ctx, queueKey, redis.Z{
		Score:  score,
		Member: task.ID.String(),
	}).Err(); err != nil {
		return fmt.Errorf("failed to add task to queue: %w", err)
	}

	// Publish event for real-time notification
	eventData, _ := json.Marshal(map[string]interface{}{
		"event": "task_created",
		"task":  task,
	})
	tq.redisClient.Publish(ctx, fmt.Sprintf("events:%s", task.AssignedAgent), eventData)

	tq.logger.Info("Task published",
		zap.String("task_id", task.ID.String()),
		zap.String("assigned_to", string(task.AssignedAgent)),
		zap.Int("priority", task.Priority))

	return nil
}

// SubscribeToTasks subscribes an agent to receive tasks
func (tq *TaskQueue) SubscribeToTasks(agentType agents.AgentType, handler TaskHandler) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if _, exists := tq.subscribers[agentType]; exists {
		return fmt.Errorf("agent %s already subscribed", agentType)
	}

	subscriber := &TaskSubscriber{
		AgentType: agentType,
		Channel:   fmt.Sprintf("queue:%s", agentType),
		Handler:   handler,
		Active:    true,
	}

	tq.subscribers[agentType] = subscriber

	// Start processing tasks for this agent
	go tq.processTasksForAgent(context.Background(), subscriber)

	return nil
}

// processTasksForAgent continuously processes tasks for a specific agent
func (tq *TaskQueue) processTasksForAgent(ctx context.Context, subscriber *TaskSubscriber) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !subscriber.Active {
				continue
			}

			// Get highest priority task
			task, err := tq.getNextTask(ctx, subscriber.AgentType)
			if err != nil || task == nil {
				continue
			}

			// Process task
			if err := tq.processTask(ctx, subscriber, task); err != nil {
				tq.logger.Error("Failed to process task",
					zap.String("task_id", task.ID.String()),
					zap.Error(err))
				
				// Handle retry logic
				tq.handleTaskFailure(ctx, task, err)
			}
		}
	}
}

// getNextTask retrieves the next task for an agent from the queue
func (tq *TaskQueue) getNextTask(ctx context.Context, agentType agents.AgentType) (*CollaborativeTask, error) {
	queueKey := fmt.Sprintf("queue:%s", agentType)
	
	// Get highest priority task
	result, err := tq.redisClient.ZPopMax(ctx, queueKey, 1).Result()
	if err != nil || len(result) == 0 {
		return nil, nil // No tasks available
	}

	taskID := result[0].Member.(string)
	
	// Retrieve task details
	taskKey := fmt.Sprintf("task:%s", taskID)
	taskData, err := tq.redisClient.Get(ctx, taskKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task CollaborativeTask
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Update task status
	task.Status = TaskStatusAssigned
	task.UpdatedAt = time.Now()
	tq.updateTask(ctx, &task)

	return &task, nil
}

// processTask processes a single task with an agent
func (tq *TaskQueue) processTask(ctx context.Context, subscriber *TaskSubscriber, task *CollaborativeTask) error {
	// Update status to in progress
	task.Status = TaskStatusInProgress
	task.UpdatedAt = time.Now()
	tq.updateTask(ctx, task)

	// Execute handler
	startTime := time.Now()
	err := subscriber.Handler(ctx, task)
	executionTime := time.Since(startTime)

	if err != nil {
		return err
	}

	// Update task with completion
	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()
	
	// Calculate confidence based on execution time and feedback
	task.ConfidenceScore = tq.calculateConfidence(task, executionTime)
	
	tq.updateTask(ctx, task)

	// Trigger self-improvement analysis if confidence is low
	if task.ConfidenceScore < 7.0 {
		tq.triggerImprovement(ctx, task)
	}

	return nil
}

// HandoffTask hands off a task from one agent to another
func (tq *TaskQueue) HandoffTask(ctx context.Context, taskID uuid.UUID, fromAgent, toAgent agents.AgentType, reason string) error {
	// Retrieve current task
	taskKey := fmt.Sprintf("task:%s", taskID)
	taskData, err := tq.redisClient.Get(ctx, taskKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	var task CollaborativeTask
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Add handoff feedback
	feedback := FeedbackEntry{
		AgentType:  fromAgent,
		Timestamp:  time.Now(),
		Type:       FeedbackTypeHandoff,
		Message:    fmt.Sprintf("Handing off to %s: %s", toAgent, reason),
		Confidence: task.ConfidenceScore,
	}
	task.Feedback = append(task.Feedback, feedback)

	// Update task assignment
	task.AssignedAgent = toAgent
	task.Status = TaskStatusPending
	task.UpdatedAt = time.Now()

	// Re-publish to new agent's queue
	return tq.PublishTask(ctx, &task)
}

// CreateSubtask creates a subtask for parallel processing
func (tq *TaskQueue) CreateSubtask(ctx context.Context, parentTask *CollaborativeTask, subtaskType string, assignTo agents.AgentType, input string) (*CollaborativeTask, error) {
	subtask := &CollaborativeTask{
		ParentID:      &parentTask.ID,
		Type:          subtaskType,
		Priority:      parentTask.Priority,
		AssignedAgent: assignTo,
		CreatedBy:     parentTask.AssignedAgent,
		Input:         input,
		Context:       parentTask.Context,
		MaxRetries:    3,
	}

	if err := tq.PublishTask(ctx, subtask); err != nil {
		return nil, err
	}

	return subtask, nil
}

// GetTaskStatus retrieves the current status of a task
func (tq *TaskQueue) GetTaskStatus(ctx context.Context, taskID uuid.UUID) (*CollaborativeTask, error) {
	taskKey := fmt.Sprintf("task:%s", taskID)
	taskData, err := tq.redisClient.Get(ctx, taskKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task CollaborativeTask
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// updateTask updates a task in Redis
func (tq *TaskQueue) updateTask(ctx context.Context, task *CollaborativeTask) error {
	taskKey := fmt.Sprintf("task:%s", task.ID)
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return tq.redisClient.Set(ctx, taskKey, taskData, 24*time.Hour).Err()
}

// calculateConfidence calculates confidence score based on various factors
func (tq *TaskQueue) calculateConfidence(task *CollaborativeTask, executionTime time.Duration) float64 {
	confidence := 5.0 // Base confidence

	// Adjust based on execution time
	if executionTime < 1*time.Second {
		confidence += 2.0
	} else if executionTime < 5*time.Second {
		confidence += 1.0
	} else if executionTime > 30*time.Second {
		confidence -= 1.0
	}

	// Adjust based on retry count
	confidence -= float64(task.RetryCount) * 0.5

	// Adjust based on feedback
	for _, feedback := range task.Feedback {
		if feedback.Type == FeedbackTypeSuccess {
			confidence += 0.5
		} else if feedback.Type == FeedbackTypeError {
			confidence -= 0.5
		}
	}

	// Cap between 0 and 10
	if confidence > 10 {
		confidence = 10
	}
	if confidence < 0 {
		confidence = 0
	}

	return confidence
}

// handleTaskFailure handles task failures with retry logic
func (tq *TaskQueue) handleTaskFailure(ctx context.Context, task *CollaborativeTask, err error) {
	task.RetryCount++
	
	if task.RetryCount >= task.MaxRetries {
		task.Status = TaskStatusFailed
		task.UpdatedAt = time.Now()
		
		// Add failure feedback
		feedback := FeedbackEntry{
			AgentType:  task.AssignedAgent,
			Timestamp:  time.Now(),
			Type:       FeedbackTypeError,
			Message:    fmt.Sprintf("Task failed after %d retries: %v", task.RetryCount, err),
			Confidence: 0,
		}
		task.Feedback = append(task.Feedback, feedback)
		
		tq.updateTask(ctx, task)
		return
	}

	// Re-queue with exponential backoff
	backoff := time.Duration(task.RetryCount) * 5 * time.Second
	time.AfterFunc(backoff, func() {
		task.Status = TaskStatusPending
		tq.PublishTask(context.Background(), task)
	})
}

// triggerImprovement triggers self-improvement analysis for low confidence tasks
func (tq *TaskQueue) triggerImprovement(ctx context.Context, task *CollaborativeTask) {
	// Create improvement analysis task
	improvementTask := &CollaborativeTask{
		Type:          "improvement_analysis",
		Priority:      task.Priority + 1, // Higher priority
		AssignedAgent: agents.OrchestratorAgent,
		CreatedBy:     task.AssignedAgent,
		Input:         fmt.Sprintf("Analyze and improve task execution: %s", task.ID),
		Context: map[string]interface{}{
			"original_task":    task,
			"confidence_score": task.ConfidenceScore,
			"feedback":         task.Feedback,
		},
		MaxRetries: 1,
	}

	tq.PublishTask(ctx, improvementTask)
	
	tq.logger.Info("Triggered improvement analysis",
		zap.String("task_id", task.ID.String()),
		zap.Float64("confidence", task.ConfidenceScore))
}