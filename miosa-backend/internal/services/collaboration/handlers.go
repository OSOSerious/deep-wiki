package collaboration

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
)

// Handlers manages collaboration endpoints
type Handlers struct {
	taskQueue     *TaskQueue
	improvement   *SelfImprovementEngine
	orchestrator  *agents.Orchestrator
	redisClient   *redis.Client
	logger        *zap.Logger
}

// NewHandlers creates new collaboration handlers
func NewHandlers(orchestrator *agents.Orchestrator, redisClient *redis.Client, logger *zap.Logger) *Handlers {
	taskQueue := NewTaskQueue(redisClient, logger)
	improvement := NewSelfImprovementEngine(redisClient, logger)
	
	return &Handlers{
		taskQueue:     taskQueue,
		improvement:   improvement,
		orchestrator:  orchestrator,
		redisClient:   redisClient,
		logger:        logger,
	}
}

// ExecuteCollaborativeTask handles multi-agent collaboration requests
func (h *Handlers) ExecuteCollaborativeTask(c *gin.Context) {
	var req struct {
		Task        string                 `json:"task"`
		Type        string                 `json:"type"`
		Priority    int                    `json:"priority"`
		Context     map[string]interface{} `json:"context"`
		Agents      []string               `json:"agents,omitempty"`
		Parallel    bool                   `json:"parallel"`
		LearnFrom   bool                   `json:"learn_from"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx := context.Background()
	tasks := h.createCollaborativeTasks(req.Task, req.Type, req.Priority, req.Context, req.Agents)
	
	// Execute tasks
	var results []*agents.Result
	if req.Parallel {
		results = h.executeParallel(ctx, tasks)
	} else {
		results = h.executeSequential(ctx, tasks)
	}
	
	// Learn from collaboration if requested
	if req.LearnFrom {
		go h.improvement.AnalyzeCollaboration(ctx, tasks)
	}
	
	// Aggregate results
	finalResult := h.aggregateResults(results)
	
	c.JSON(http.StatusOK, gin.H{
		"success":    finalResult.Success,
		"output":     finalResult.Output,
		"confidence": finalResult.Confidence,
		"agents_used": len(req.Agents),
	})
}

func (h *Handlers) createCollaborativeTasks(task, taskType string, priority int, context map[string]interface{}, agentNames []string) []*CollaborativeTask {
	tasks := make([]*CollaborativeTask, 0)
	
	for _, agentName := range agentNames {
		t := &CollaborativeTask{
			Type:          taskType,
			Priority:      priority,
			AssignedAgent: agents.AgentType(agentName),
			CreatedBy:     agents.OrchestratorAgent,
			Input:         task,
			Context:       context,
			MaxRetries:    3,
		}
		tasks = append(tasks, t)
	}
	
	return tasks
}

func (h *Handlers) executeParallel(ctx context.Context, tasks []*CollaborativeTask) []*agents.Result {
	results := make([]*agents.Result, len(tasks))
	var wg sync.WaitGroup
	
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t *CollaborativeTask) {
			defer wg.Done()
			h.taskQueue.PublishTask(ctx, t)
			// Wait for completion
			time.Sleep(5 * time.Second) // Simplified
			results[idx] = &agents.Result{Success: true}
		}(i, task)
	}
	
	wg.Wait()
	return results
}

func (h *Handlers) executeSequential(ctx context.Context, tasks []*CollaborativeTask) []*agents.Result {
	results := make([]*agents.Result, 0)
	
	for _, task := range tasks {
		h.taskQueue.PublishTask(ctx, task)
		time.Sleep(2 * time.Second) // Simplified
		results = append(results, &agents.Result{Success: true})
	}
	
	return results
}

func (h *Handlers) aggregateResults(results []*agents.Result) *agents.Result {
	if len(results) == 0 {
		return &agents.Result{
			Success: false,
			Error:   fmt.Errorf("no results"),
		}
	}
	return results[len(results)-1]
}
