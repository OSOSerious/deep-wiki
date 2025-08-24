package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"github.com/sormind/OSA/miosa-backend/internal/agents/ai_providers"
	"github.com/sormind/OSA/miosa-backend/internal/agents/analysis"
	"github.com/sormind/OSA/miosa-backend/internal/agents/architect"
	"github.com/sormind/OSA/miosa-backend/internal/agents/communication"
	"github.com/sormind/OSA/miosa-backend/internal/agents/deployment"
	"github.com/sormind/OSA/miosa-backend/internal/agents/development"
	"github.com/sormind/OSA/miosa-backend/internal/agents/monitoring"
	"github.com/sormind/OSA/miosa-backend/internal/agents/quality"
	"github.com/sormind/OSA/miosa-backend/internal/agents/recommender"
	"github.com/sormind/OSA/miosa-backend/internal/agents/strategy"
	"github.com/conneroisu/groq-go"
	"go.uber.org/zap"
)

// FullOrchestrator manages ALL agents
type FullOrchestrator struct {
	registry    map[agents.AgentType]agents.Agent
	groqClient  *groq.Client
	logger      *zap.Logger
	workspaceDir string
	mu          sync.RWMutex
}

// NewFullOrchestrator creates orchestrator with ALL agents
func NewFullOrchestrator(apiKey, workspaceDir string) (*FullOrchestrator, error) {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	// Initialize Groq client
	groqClient, err := groq.NewClient(apiKey)
	if err != nil {
		return nil, err
	}

	// Create orchestrator
	o := &FullOrchestrator{
		registry:     make(map[agents.AgentType]agents.Agent),
		groqClient:   groqClient,
		logger:       logger,
		workspaceDir: workspaceDir,
	}

	// Register ALL agents
	o.registerAllAgents()

	return o, nil
}

func (o *FullOrchestrator) registerAllAgents() {
	// Register all agents using their New functions
	o.registry[agents.AnalysisAgent] = analysis.New(o.groqClient)
	o.registry[agents.ArchitectAgent] = architect.New(o.groqClient)
	o.registry[agents.DevelopmentAgent] = development.New(o.groqClient)
	o.registry[agents.QualityAgent] = quality.New(o.groqClient)
	o.registry[agents.DeploymentAgent] = deployment.New(o.groqClient)
	o.registry[agents.MonitoringAgent] = monitoring.New(o.groqClient)
	o.registry[agents.StrategyAgent] = strategy.New(o.groqClient)
	o.registry[agents.CommunicationAgent] = communication.New(o.groqClient)
	o.registry[agents.RecommenderAgent] = recommender.New(o.groqClient)
	o.registry[agents.AIProvidersAgent] = ai_providers.New(o.groqClient)

	o.logger.Info("Registered all agents", zap.Int("count", len(o.registry)))
}

// ExecuteWorkflow runs complete multi-agent workflow
func (o *FullOrchestrator) ExecuteWorkflow(ctx context.Context, description string) (*WorkflowResult, error) {
	workflowID := uuid.New()
	results := make([]AgentResult, 0)

	// Create base task
	task := agents.Task{
		ID:    workflowID,
		Type:  "implementation",
		Input: description,
		Context: &agents.TaskContext{
			Phase: "initialization",
			Memory: make(map[string]interface{}),
		},
	}

	// Define agent execution order for comprehensive solution
	agentSequence := []agents.AgentType{
		agents.StrategyAgent,      // Strategic planning
		agents.AnalysisAgent,      // Requirements analysis  
		agents.ArchitectAgent,     // System architecture
		agents.DevelopmentAgent,   // Implementation
		agents.QualityAgent,       // Quality assurance
		agents.MonitoringAgent,    // Monitoring setup
		agents.DeploymentAgent,    // Deployment config
		agents.RecommenderAgent,   // Recommendations
	}

	// Execute agents in sequence
	for _, agentType := range agentSequence {
		agent, exists := o.registry[agentType]
		if !exists {
			o.logger.Warn("Agent not found", zap.String("type", string(agentType)))
			continue
		}

		o.logger.Info("Executing agent", zap.String("type", string(agentType)))

		// Update task context
		task.Context.Phase = string(agentType)

		// Execute agent
		result, err := agent.Execute(ctx, task)
		if err != nil {
			o.logger.Error("Agent failed", 
				zap.String("type", string(agentType)),
				zap.Error(err))
			continue
		}

		// Save agent output
		if err := o.saveAgentOutput(agentType, workflowID, result); err != nil {
			o.logger.Error("Failed to save output", zap.Error(err))
		}

		// Record result
		results = append(results, AgentResult{
			Agent:       agentType,
			Success:     result.Success,
			Output:      result.Output,
			Confidence:  result.Confidence,
			ExecutionMS: result.ExecutionMS,
		})

		// Update task memory with result
		if task.Context.Memory == nil {
			task.Context.Memory = make(map[string]interface{})
		}
		task.Context.Memory[string(agentType)] = result.Output
	}

	return &WorkflowResult{
		WorkflowID: workflowID,
		Results:    results,
		Success:    true,
		Timestamp:  time.Now(),
	}, nil
}

// saveAgentOutput saves agent output to appropriate directory
func (o *FullOrchestrator) saveAgentOutput(agentType agents.AgentType, workflowID uuid.UUID, result *agents.Result) error {
	// Determine output directory based on agent type
	var outputDir string
	var fileName string
	var extension string

	switch agentType {
	case agents.AnalysisAgent:
		outputDir = "analysis"
		fileName = fmt.Sprintf("analysis_%s", workflowID.String()[:8])
		extension = ".md"
	case agents.ArchitectAgent:
		outputDir = "architecture"
		fileName = fmt.Sprintf("architecture_%s", workflowID.String()[:8])
		extension = ".md"
	case agents.DevelopmentAgent:
		outputDir = "code"
		fileName = fmt.Sprintf("implementation_%s", workflowID.String()[:8])
		extension = ".go"
	case agents.QualityAgent:
		outputDir = "quality"
		fileName = fmt.Sprintf("quality_report_%s", workflowID.String()[:8])
		extension = ".md"
	case agents.DeploymentAgent:
		outputDir = "deployment"
		fileName = fmt.Sprintf("deployment_%s", workflowID.String()[:8])
		extension = ".yaml"
	case agents.MonitoringAgent:
		outputDir = "monitoring"
		fileName = fmt.Sprintf("monitoring_%s", workflowID.String()[:8])
		extension = ".yaml"
	case agents.StrategyAgent:
		outputDir = "strategy"
		fileName = fmt.Sprintf("strategy_%s", workflowID.String()[:8])
		extension = ".md"
	case agents.RecommenderAgent:
		outputDir = "recommendations"
		fileName = fmt.Sprintf("recommendations_%s", workflowID.String()[:8])
		extension = ".md"
	default:
		outputDir = "output"
		fileName = fmt.Sprintf("%s_%s", agentType, workflowID.String()[:8])
		extension = ".txt"
	}

	// Create directory if it doesn't exist
	fullDir := filepath.Join(o.workspaceDir, outputDir)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return err
	}

	// Write file
	filePath := filepath.Join(fullDir, fileName+extension)
	return os.WriteFile(filePath, []byte(result.Output), 0644)
}

// WorkflowResult represents complete workflow execution
type WorkflowResult struct {
	WorkflowID uuid.UUID     `json:"workflow_id"`
	Results    []AgentResult `json:"results"`
	Success    bool          `json:"success"`
	Timestamp  time.Time     `json:"timestamp"`
}

// AgentResult represents individual agent result
type AgentResult struct {
	Agent       agents.AgentType `json:"agent"`
	Success     bool            `json:"success"`
	Output      string          `json:"output"`
	Confidence  float64         `json:"confidence"`
	ExecutionMS int64           `json:"execution_ms"`
}

// API Server
type Server struct {
	orchestrator *FullOrchestrator
	router       *mux.Router
}

func NewServer(orchestrator *FullOrchestrator) *Server {
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
	s.router.HandleFunc("/api/workflow/{id}", s.handleGetWorkflow).Methods("GET")
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
	result, err := s.orchestrator.ExecuteWorkflow(ctx, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := make([]map[string]interface{}, 0)
	
	for agentType, agent := range s.orchestrator.registry {
		agents = append(agents, map[string]interface{}{
			"type":        agentType,
			"description": agent.GetDescription(),
			"capabilities": agent.GetCapabilities(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement workflow retrieval
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	var (
		port      = flag.String("port", "8091", "Server port")
		workspace = flag.String("workspace", "/Users/ososerious/OSA/agent-workspace", "Workspace directory")
	)
	flag.Parse()

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal("GROQ_API_KEY environment variable is required")
	}

	// Create orchestrator with ALL agents
	orchestrator, err := NewFullOrchestrator(apiKey, *workspace)
	if err != nil {
		log.Fatal("Failed to create orchestrator:", err)
	}

	// Create directories
	dirs := []string{
		"analysis", "architecture", "code", "quality", 
		"deployment", "monitoring", "strategy", "recommendations",
		"tests", "documentation", "output",
	}
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(*workspace, dir), 0755)
	}

	// Create and start server
	server := NewServer(orchestrator)

	log.Printf("[FULL ORCHESTRATOR] Starting with ALL %d agents on port %s", 
		len(orchestrator.registry), *port)
	log.Printf("[WORKSPACE] %s", *workspace)
	log.Printf("[STATUS] Ready to orchestrate complete workflows!")

	if err := http.ListenAndServe(":"+*port, server.router); err != nil {
		log.Fatal(err)
	}
}