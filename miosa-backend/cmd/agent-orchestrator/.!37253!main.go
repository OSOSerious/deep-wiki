package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// AgentType represents different agent specializations
type AgentType string

const (
	OrchestratorAgent  AgentType = "orchestrator"
	AnalysisAgent      AgentType = "analysis"
	ArchitectAgent     AgentType = "architect"
	DevelopmentAgent   AgentType = "development"
	TestingAgent       AgentType = "testing"
	QualityAgent       AgentType = "quality"
	DeploymentAgent    AgentType = "deployment"
	MonitoringAgent    AgentType = "monitoring"
)

// Task represents a task for an agent
type Task struct {
	ID          uuid.UUID              `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Input       string                 `json:"input"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    int                    `json:"priority"`
	Context     *TaskContext           `json:"context"`
}

// TaskContext provides context for task execution
type TaskContext struct {
	ProjectID   uuid.UUID              `json:"project_id"`
	Phase       string                 `json:"phase"`
	Memory      map[string]interface{} `json:"memory"`
	History     []Message              `json:"history"`
	IDEEndpoint string                 `json:"ide_endpoint"`
}

// Message represents a conversation message
type Message struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Agent   AgentType `json:"agent"`
	Time    time.Time `json:"time"`
}

// Result represents agent execution result
type Result struct {
	Success     bool                   `json:"success"`
	Output      string                 `json:"output"`
	Files       []GeneratedFile        `json:"files,omitempty"`
	Data        map[string]interface{} `json:"data"`
	NextAgent   AgentType              `json:"next_agent,omitempty"`
	Confidence  float64                `json:"confidence"`
	ExecutionMS int64                  `json:"execution_ms"`
	Error       string                 `json:"error,omitempty"`
}

// GeneratedFile represents a file created by an agent
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

// Agent interface
type Agent interface {
	GetType() AgentType
	Execute(ctx context.Context, task Task) (*Result, error)
	GetCapabilities() []string
}

// BaseAgent provides common functionality
type BaseAgent struct {
	Type        AgentType
	Name        string
	Description string
	APIKey      string
	IDEClient   *IDEClient
}

// IDEClient handles IDE server communication
type IDEClient struct {
	BaseURL string
}

// SaveFile saves content to IDE
func (c *IDEClient) SaveFile(path string, content string) error {
	payload := map[string]string{
		"path":    path,
		"content": content,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/api/ide/file", c.BaseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to save file: %s", string(body))
	}

	return nil
}

// LLMClient handles LLM API calls
type LLMClient struct {
	APIKey string
}

// CallLLM makes a request to the LLM
func (l *LLMClient) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"
	
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	requestBody := map[string]interface{}{
		"model": "llama3-70b-8192",
		"messages": messages,
		"temperature": 0.7,
		"max_tokens": 3000,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if ok && len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		message := choice["message"].(map[string]interface{})
		return message["content"].(string), nil
	}

	return "", fmt.Errorf("no response from LLM")
}

// AnalysisAgentImpl analyzes requirements
type AnalysisAgentImpl struct {
	BaseAgent
	llm *LLMClient
}

func (a *AnalysisAgentImpl) GetType() AgentType {
	return AnalysisAgent
}

func (a *AnalysisAgentImpl) GetCapabilities() []string {
	return []string{"requirements_analysis", "specification_creation", "risk_assessment"}
}

func (a *AnalysisAgentImpl) Execute(ctx context.Context, task Task) (*Result, error) {
	start := time.Now()
	
	systemPrompt := `You are an Analysis Agent specializing in software requirements analysis.
Your role is to:
1. Analyze the given requirements thoroughly
2. Identify technical constraints and considerations
3. Create detailed specifications
4. Assess potential risks and challenges
5. Define success criteria

Format your response as a structured markdown document with clear sections.`

	response, err := a.llm.CallLLM(ctx, systemPrompt, task.Description)
	if err != nil {
		return &Result{
			Success:     false,
			Error:       err.Error(),
			ExecutionMS: time.Since(start).Milliseconds(),
		}, nil
	}

	// Save analysis document
	fileName := fmt.Sprintf("analysis_%s.md", task.ID.String()[:8])
	filePath := filepath.Join("/Users/ososerious/OSA/miosa-backend/internal/docs", fileName)
	
	if err := a.IDEClient.SaveFile(filePath, response); err != nil {
		log.Printf("Failed to save analysis: %v", err)
	}

	return &Result{
		Success:     true,
		Output:      response,
		Files:       []GeneratedFile{{Path: fileName, Content: response, Type: "markdown"}},
		Confidence:  0.85,
		NextAgent:   ArchitectAgent,
		ExecutionMS: time.Since(start).Milliseconds(),
	}, nil
}

// ArchitectAgentImpl designs system architecture
type ArchitectAgentImpl struct {
	BaseAgent
	llm *LLMClient
}

func (a *ArchitectAgentImpl) GetType() AgentType {
	return ArchitectAgent
}

func (a *ArchitectAgentImpl) GetCapabilities() []string {
	return []string{"system_design", "data_modeling", "api_design"}
}

func (a *ArchitectAgentImpl) Execute(ctx context.Context, task Task) (*Result, error) {
	start := time.Now()
	
	systemPrompt := `You are an Architecture Agent specializing in system design.
Your role is to:
1. Design data models and structures
2. Define interfaces and APIs
3. Create system architecture
4. Follow Go best practices

Generate complete, working Go code with proper struct definitions, interfaces, and JSON tags.
Include comments explaining design decisions.`

	response, err := a.llm.CallLLM(ctx, systemPrompt, task.Description)
	if err != nil {
		return &Result{
			Success:     false,
			Error:       err.Error(),
			ExecutionMS: time.Since(start).Milliseconds(),
		}, nil
	}

	// Extract code from response
	code := extractCode(response)
	
	// Save model file
	fileName := fmt.Sprintf("model_%s.go", task.ID.String()[:8])
	filePath := filepath.Join("/Users/ososerious/OSA/miosa-backend/internal/models", fileName)
	
	if err := a.IDEClient.SaveFile(filePath, code); err != nil {
		log.Printf("Failed to save model: %v", err)
	}

	return &Result{
		Success:     true,
		Output:      response,
		Files:       []GeneratedFile{{Path: fileName, Content: code, Type: "go"}},
		Confidence:  0.88,
		NextAgent:   DevelopmentAgent,
		ExecutionMS: time.Since(start).Milliseconds(),
	}, nil
}

// DevelopmentAgentImpl implements business logic
type DevelopmentAgentImpl struct {
	BaseAgent
	llm *LLMClient
}

func (a *DevelopmentAgentImpl) GetType() AgentType {
	return DevelopmentAgent
}

func (a *DevelopmentAgentImpl) GetCapabilities() []string {
	return []string{"api_implementation", "business_logic", "integration"}
}

func (a *DevelopmentAgentImpl) Execute(ctx context.Context, task Task) (*Result, error) {
	start := time.Now()
	
	systemPrompt := `You are a Development Agent specializing in Go implementation.
Your role is to:
1. Implement complete API handlers
2. Add proper error handling
3. Include input validation
4. Follow RESTful conventions
5. Use the models and interfaces defined

Generate production-ready Go code with all necessary imports.`

	response, err := a.llm.CallLLM(ctx, systemPrompt, task.Description)
	if err != nil {
		return &Result{
			Success:     false,
			Error:       err.Error(),
			ExecutionMS: time.Since(start).Milliseconds(),
		}, nil
	}

	// Extract code from response
	code := extractCode(response)
	
	// Save handler file
	fileName := fmt.Sprintf("handler_%s.go", task.ID.String()[:8])
	filePath := filepath.Join("/Users/ososerious/OSA/miosa-backend/internal/handlers", fileName)
	
	if err := a.IDEClient.SaveFile(filePath, code); err != nil {
		log.Printf("Failed to save handler: %v", err)
	}

	return &Result{
		Success:     true,
		Output:      response,
		Files:       []GeneratedFile{{Path: fileName, Content: code, Type: "go"}},
		Confidence:  0.90,
		NextAgent:   TestingAgent,
		ExecutionMS: time.Since(start).Milliseconds(),
	}, nil
}

// Orchestrator coordinates agent execution
type Orchestrator struct {
	agents      map[AgentType]Agent
	ideClient   *IDEClient
	taskHistory []Task
	mu          sync.RWMutex
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(apiKey string, ideEndpoint string) *Orchestrator {
	ideClient := &IDEClient{BaseURL: ideEndpoint}
	llmClient := &LLMClient{APIKey: apiKey}
	
	agents := make(map[AgentType]Agent)
	
	// Initialize agents
	agents[AnalysisAgent] = &AnalysisAgentImpl{
		BaseAgent: BaseAgent{
			Type:      AnalysisAgent,
			Name:      "Analysis Agent",
			IDEClient: ideClient,
		},
		llm: llmClient,
	}
	
	agents[ArchitectAgent] = &ArchitectAgentImpl{
		BaseAgent: BaseAgent{
			Type:      ArchitectAgent,
			Name:      "Architecture Agent",
			IDEClient: ideClient,
		},
		llm: llmClient,
	}
	
	agents[DevelopmentAgent] = &DevelopmentAgentImpl{
		BaseAgent: BaseAgent{
			Type:      DevelopmentAgent,
			Name:      "Development Agent",
			IDEClient: ideClient,
		},
		llm: llmClient,
	}
	
	return &Orchestrator{
		agents:    agents,
		ideClient: ideClient,
	}
}

// ExecuteTask orchestrates task execution across agents
func (o *Orchestrator) ExecuteTask(ctx context.Context, description string) (*WorkflowResult, error) {
	workflowID := uuid.New()
	results := make([]*Result, 0)
	
	// Create initial task
	task := Task{
		ID:          workflowID,
		Type:        "implementation",
		Description: description,
		Context: &TaskContext{
			ProjectID:   workflowID,
			Phase:       "analysis",
			Memory:      make(map[string]interface{}),
			IDEEndpoint: o.ideClient.BaseURL,
		},
	}
	
	// Start with analysis agent
	currentAgent := AnalysisAgent
	
	for i := 0; i < 5; i++ { // Max 5 agent hops
		agent, exists := o.agents[currentAgent]
		if !exists {
			break
		}
		
		log.Printf("> Executing %s agent...", currentAgent)
		
		result, err := agent.Execute(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("agent %s failed: %w", currentAgent, err)
		}
		
		results = append(results, result)
		
		// Update task context with result
		if task.Context.Memory == nil {
			task.Context.Memory = make(map[string]interface{})
		}
		task.Context.Memory[string(currentAgent)] = result.Output
		
		// Move to next agent if specified
		if result.NextAgent == "" {
			break
		}
		currentAgent = result.NextAgent
		
		// Update task for next agent
		task.Context.Phase = string(currentAgent)
	}
	
	return &WorkflowResult{
		WorkflowID: workflowID,
		Results:    results,
		Success:    true,
	}, nil
}

// WorkflowResult represents complete workflow execution
type WorkflowResult struct {
	WorkflowID uuid.UUID `json:"workflow_id"`
	Results    []*Result `json:"results"`
	Success    bool      `json:"success"`
}

// extractCode extracts code blocks from markdown
func extractCode(text string) string {
	if strings.Contains(text, "```go") {
		parts := strings.Split(text, "```go")
		if len(parts) >= 2 {
			codePart := strings.Split(parts[1], "```")[0]
			return strings.TrimSpace(codePart)
		}
	} else if strings.Contains(text, "```") {
		parts := strings.Split(text, "```")
		if len(parts) >= 2 {
			code := parts[1]
			if idx := strings.Index(code, "\n"); idx > 0 {
				code = code[idx+1:]
			}
			return strings.TrimSpace(code)
		}
	}
	return text
}

// API Server
type Server struct {
	orchestrator *Orchestrator
	router       *mux.Router
}

// NewServer creates a new API server
func NewServer(orchestrator *Orchestrator) *Server {
	s := &Server{
		orchestrator: orchestrator,
		router:       mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/api/orchestrate", s.handleOrchestrate).Methods("POST")
	s.router.HandleFunc("/api/agents", s.handleListAgents).Methods("GET")
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *Server) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string `json:"description"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	ctx := context.Background()
	result, err := s.orchestrator.ExecuteTask(ctx, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := []map[string]interface{}{}
	
	for agentType, agent := range s.orchestrator.agents {
		agents = append(agents, map[string]interface{}{
			"type":         agentType,
			"capabilities": agent.GetCapabilities(),
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	var (
		port   = flag.String("port", "8090", "Server port")
		ideURL = flag.String("ide", "http://localhost:8085", "IDE server URL")
	)
	flag.Parse()
	
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal("GROQ_API_KEY environment variable is required")
	}
	
	// Create orchestrator
	orchestrator := NewOrchestrator(apiKey, *ideURL)
	
	// Create and start server
	server := NewServer(orchestrator)
	
