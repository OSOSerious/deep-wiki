package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
)

// Handlers contains the gateway service handlers
type Handlers struct {
	orchestrator *agents.Orchestrator
	groqClient   *groq.Client
	logger       *zap.Logger
}

// NewHandlers creates new gateway handlers
func NewHandlers(orchestrator *agents.Orchestrator, groqClient *groq.Client, logger *zap.Logger) *Handlers {
	return &Handlers{
		orchestrator: orchestrator,
		groqClient:   groqClient,
		logger:       logger,
	}
}

// ExecuteAgentRequest represents a request to execute an agent task
type ExecuteAgentRequest struct {
	Task     string                 `json:"task" binding:"required"`
	Type     string                 `json:"type"`
	Phase    string                 `json:"phase"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ExecuteAgentResponse represents the response from agent execution
type ExecuteAgentResponse struct {
	Success     bool        `json:"success"`
	Output      string      `json:"output"`
	Confidence  float64     `json:"confidence"`
	NextAgent   string      `json:"next_agent,omitempty"`
	ExecutionMS int64       `json:"execution_ms"`
	TaskID      string      `json:"task_id"`
}

// ExecuteAgent handles agent execution requests
func (h *Handlers) ExecuteAgent(c *gin.Context) {
	if h.orchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent system not initialized"})
		return
	}

	var req ExecuteAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get task context from middleware
	var taskContext *agents.TaskContext
	if ctx, exists := c.Get("task_context"); exists {
		taskContext = ctx.(*agents.TaskContext)
	} else {
		// Create default context
		taskContext = &agents.TaskContext{
			UserID:      uuid.New(),
			TenantID:    uuid.New(),
			WorkspaceID: uuid.New(),
			Phase:       req.Phase,
			Metadata:    make(map[string]string),
		}
	}

	// Create task
	task := agents.Task{
		ID:       uuid.New(),
		Type:     req.Type,
		Input:    req.Task,
		Context:  taskContext,
		Priority: 5,
		Timeout:  30 * time.Second,
	}

	if req.Metadata != nil {
		task.Parameters = req.Metadata
	}

	// Execute through orchestrator
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := h.orchestrator.Execute(ctx, task)
	if err != nil {
		h.logger.Error("Agent execution failed",
			zap.Error(err),
			zap.String("task_id", task.ID.String()),
			zap.String("task_type", task.Type))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Execution failed: %v", err)})
		return
	}

	// Return response
	response := ExecuteAgentResponse{
		Success:     result.Success,
		Output:      result.Output,
		Confidence:  result.Confidence,
		ExecutionMS: result.ExecutionMS,
		TaskID:      task.ID.String(),
	}

	if result.NextAgent != "" {
		response.NextAgent = string(result.NextAgent)
	}

	c.JSON(http.StatusOK, response)
}

// ChatRequest represents a simple chat request
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
	Model   string `json:"model"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Model   string `json:"model"`
}

// Chat handles simple chat requests (backward compatibility)
func (h *Handlers) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.groqClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Chat service not available"})
		return
	}

	// Use specified model or default
	model := req.Model
	if model == "" {
		model = "llama-3.1-8b-instant"
	}

	// Make direct Groq call for simple chat
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: groq.ChatModel(model),
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "user",
				Content: req.Message,
			},
		},
		MaxTokens:   500,
		Temperature: 0.7,
	})

	if err != nil {
		h.logger.Error("Chat completion failed", zap.Error(err))
		c.JSON(500, gin.H{"error": "Failed to get response"})
		return
	}

	if len(resp.Choices) == 0 {
		c.JSON(500, gin.H{"error": "No response from model"})
		return
	}

	c.JSON(200, ChatResponse{
		Success: true,
		Message: resp.Choices[0].Message.Content,
		Model:   model,
	})
}

// HealthCheck returns service health status
func (h *Handlers) HealthCheck(c *gin.Context) {
	status := gin.H{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
		"services": gin.H{
			"orchestrator": h.orchestrator != nil,
			"groq":         h.groqClient != nil,
		},
	}
	c.JSON(200, status)
}