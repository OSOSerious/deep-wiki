package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"io"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"github.com/sormind/OSA/miosa-backend/internal/agents/ai_providers"
	"github.com/sormind/OSA/miosa-backend/internal/agents/analysis"
	"github.com/sormind/OSA/miosa-backend/internal/agents/architect"
	"github.com/sormind/OSA/miosa-backend/internal/agents/communication"
	"github.com/sormind/OSA/miosa-backend/internal/agents/deployment"
	"github.com/sormind/OSA/miosa-backend/internal/agents/monitoring"
	"github.com/sormind/OSA/miosa-backend/internal/agents/quality"
	"github.com/sormind/OSA/miosa-backend/internal/agents/recommender"
	"github.com/sormind/OSA/miosa-backend/internal/agents/strategy"
	"github.com/conneroisu/groq-go"
	"go.uber.org/zap"
)

// EnhancedOrchestrator manages agents with proper file generation
type EnhancedOrchestrator struct {
	registry     map[agents.AgentType]agents.Agent
	groqClient   *groq.Client
	logger       *zap.Logger
	workspaceDir string
	mu           sync.RWMutex
}

// CodeFile represents a parsed code file
type CodeFile struct {
	Path     string
	Content  string
	Language string
}

// NewEnhancedOrchestrator creates orchestrator with enhanced file handling
func NewEnhancedOrchestrator(apiKey, workspaceDir string) (*EnhancedOrchestrator, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	groqClient, err := groq.NewClient(apiKey)
	if err != nil {
		return nil, err
	}

	o := &EnhancedOrchestrator{
		registry:     make(map[agents.AgentType]agents.Agent),
		groqClient:   groqClient,
		logger:       logger,
		workspaceDir: workspaceDir,
	}

	o.registerAllAgents()
	return o, nil
}

func (o *EnhancedOrchestrator) registerAllAgents() {
	// Create enhanced development agent that generates multiple files
	o.registry[agents.DevelopmentAgent] = &EnhancedDevelopmentAgent{
		groqClient: o.groqClient,
		config: agents.AgentConfig{
			Model:       "moonshotai/kimi-k2-instruct",
			MaxTokens:   8000,
			Temperature: 0.2,
			TopP:        0.95,
		},
	}

	// Register other agents
	o.registry[agents.AnalysisAgent] = analysis.New(o.groqClient)
	o.registry[agents.ArchitectAgent] = architect.New(o.groqClient)
	o.registry[agents.QualityAgent] = quality.New(o.groqClient)
	o.registry[agents.DeploymentAgent] = deployment.New(o.groqClient)
	o.registry[agents.MonitoringAgent] = monitoring.New(o.groqClient)
	o.registry[agents.StrategyAgent] = strategy.New(o.groqClient)
	o.registry[agents.CommunicationAgent] = communication.New(o.groqClient)
	o.registry[agents.RecommenderAgent] = recommender.New(o.groqClient)
	o.registry[agents.AIProvidersAgent] = ai_providers.New(o.groqClient)

	o.logger.Info("Registered enhanced agents", zap.Int("count", len(o.registry)))
}

// EnhancedDevelopmentAgent generates actual code files
type EnhancedDevelopmentAgent struct {
	groqClient *groq.Client
	config     agents.AgentConfig
}

func (a *EnhancedDevelopmentAgent) GetType() agents.AgentType {
	return agents.DevelopmentAgent
}

func (a *EnhancedDevelopmentAgent) GetDescription() string {
	return "Generates complete application code with multiple files"
}

func (a *EnhancedDevelopmentAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{Name: "multi_file_generation", Description: "Generate complete applications", Required: true},
		{Name: "language_support", Description: "Support multiple languages", Required: true},
	}
}

func (a *EnhancedDevelopmentAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()

	// Generate structured application code
	prompt := fmt.Sprintf(`Generate a complete application for: %s

Create a structured response with multiple files for a full application.
Format each file as:

=== FILE: path/to/file.ext ===
<file content here>
=== END FILE ===

Generate the following files:
1. Main application file (app.js/main.go/app.py etc)
2. Configuration file (config.json/config.yaml)
3. Docker file (Dockerfile)
4. Package file (package.json/go.mod/requirements.txt)
5. Database schema (schema.sql)
6. API routes (routes.js/routes.go/routes.py)
7. Models/types (models.js/models.go/models.py)
8. Utilities (utils.js/utils.go/utils.py)
9. Tests (test files)
10. README.md with setup instructions

Make it a complete, runnable application.`, task.Input)

	response, err := a.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: groq.ChatModel(a.config.Model),
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are an expert developer. Generate complete, production-ready applications with multiple files.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   a.config.MaxTokens,
		Temperature: float32(a.config.Temperature),
		TopP:        float32(a.config.TopP),
	})

	if err != nil {
		return &agents.Result{
			Success:     false,
			Error:       fmt.Errorf("generation failed: %w", err),
			ExecutionMS: time.Since(startTime).Milliseconds(),
		}, err
	}

	content := response.Choices[0].Message.Content

	return &agents.Result{
		Success:     true,
		Output:      content,
		NextAgent:   agents.QualityAgent,
		Confidence:  9.0,
		ExecutionMS: time.Since(startTime).Milliseconds(),
	}, nil
}

// ExecuteWorkflow runs complete multi-agent workflow with enhanced file generation
func (o *EnhancedOrchestrator) ExecuteWorkflow(ctx context.Context, description string) (*WorkflowResult, error) {
	workflowID := uuid.New()
	results := make([]AgentResult, 0)

	task := agents.Task{
		ID:    workflowID,
		Type:  "implementation",
		Input: description,
		Context: &agents.TaskContext{
			Phase:  "initialization",
			Memory: make(map[string]interface{}),
		},
	}

	// Execute agents
	agentSequence := []agents.AgentType{
		agents.StrategyAgent,
		agents.AnalysisAgent,
		agents.ArchitectAgent,
		agents.DevelopmentAgent, // This will generate actual code files
		agents.QualityAgent,
		agents.MonitoringAgent,
		agents.DeploymentAgent,
		agents.RecommenderAgent,
	}

	for _, agentType := range agentSequence {
		agent, exists := o.registry[agentType]
		if !exists {
			continue
		}

		o.logger.Info("Executing agent", zap.String("type", string(agentType)))
		task.Context.Phase = string(agentType)

		result, err := agent.Execute(ctx, task)
		if err != nil {
			o.logger.Error("Agent failed", zap.Error(err))
			continue
		}

		// Enhanced saving that parses and creates actual code files
		if err := o.saveEnhancedOutput(agentType, workflowID, result); err != nil {
			o.logger.Error("Failed to save output", zap.Error(err))
		}

		results = append(results, AgentResult{
			Agent:       agentType,
			Success:     result.Success,
			Output:      result.Output,
			Confidence:  result.Confidence,
			ExecutionMS: result.ExecutionMS,
		})

		if task.Context.Memory == nil {
			task.Context.Memory = make(map[string]interface{})
		}
		task.Context.Memory[string(agentType)] = result.Output
	}

	projectDir := filepath.Join(o.workspaceDir, workflowID.String()[:8])
	o.triggerE2BWorkflow(projectDir)

	return &WorkflowResult{
		WorkflowID: workflowID,
		Results:    results,
		Success:    true,
		Timestamp:  time.Now(),
	}, nil
}

// saveEnhancedOutput parses output and saves as appropriate file types
func (o *EnhancedOrchestrator) saveEnhancedOutput(agentType agents.AgentType, workflowID uuid.UUID, result *agents.Result) error {
	projectDir := filepath.Join(o.workspaceDir, workflowID.String()[:8])

	switch agentType {
	case agents.DevelopmentAgent:
		// Parse and save multiple code files
		files := o.parseCodeFiles(result.Output)
		for _, file := range files {
			filePath := filepath.Join(projectDir, file.Path)
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
				return err
			}
			o.logger.Info("Created code file", zap.String("path", filePath))
		}

	case agents.DeploymentAgent:
		// Save as Docker and Kubernetes files
		deployDir := filepath.Join(projectDir, "deployment")
		os.MkdirAll(deployDir, 0755)
		
		// Extract Docker content
		if dockerContent := o.extractSection(result.Output, "Dockerfile"); dockerContent != "" {
			dockerPath := filepath.Join(deployDir, "Dockerfile")
			os.WriteFile(dockerPath, []byte(dockerContent), 0644)
		}
		
		// Extract K8s manifests
		if k8sContent := o.extractSection(result.Output, "kubernetes"); k8sContent != "" {
			k8sPath := filepath.Join(deployDir, "k8s-deployment.yaml")
			os.WriteFile(k8sPath, []byte(k8sContent), 0644)
		}

		// Extract docker-compose
		if composeContent := o.extractSection(result.Output, "docker-compose"); composeContent != "" {
			composePath := filepath.Join(deployDir, "docker-compose.yml")
			os.WriteFile(composePath, []byte(composeContent), 0644)
		}

	case agents.MonitoringAgent:
		// Save monitoring configs
		monitorDir := filepath.Join(projectDir, "monitoring")
		os.MkdirAll(monitorDir, 0755)
		
		// Prometheus config
		if promContent := o.extractSection(result.Output, "prometheus"); promContent != "" {
			promPath := filepath.Join(monitorDir, "prometheus.yml")
			os.WriteFile(promPath, []byte(promContent), 0644)
		}
		
		// Grafana dashboards
		if grafanaContent := o.extractSection(result.Output, "grafana"); grafanaContent != "" {
			grafanaPath := filepath.Join(monitorDir, "grafana-dashboard.json")
			os.WriteFile(grafanaPath, []byte(grafanaContent), 0644)
		}

	case agents.QualityAgent:
		// Save test files
		testDir := filepath.Join(projectDir, "tests")
		os.MkdirAll(testDir, 0755)
		
		// Extract test code
		if testContent := o.extractCodeBlocks(result.Output); len(testContent) > 0 {
			for i, test := range testContent {
				testPath := filepath.Join(testDir, fmt.Sprintf("test_%d.js", i+1))
				os.WriteFile(testPath, []byte(test), 0644)
			}
		}

	default:
		// Save documentation for other agents
		docDir := filepath.Join(projectDir, "docs")
		os.MkdirAll(docDir, 0755)
		docPath := filepath.Join(docDir, fmt.Sprintf("%s.md", agentType))
		os.WriteFile(docPath, []byte(result.Output), 0644)
	}

	return nil
}

// parseCodeFiles extracts multiple files from structured output
func (o *EnhancedOrchestrator) parseCodeFiles(content string) []CodeFile {
	var files []CodeFile
	
	// Pattern to match file blocks
	filePattern := regexp.MustCompile(`=== FILE: (.+?) ===\n([\s\S]*?)(?:=== END FILE ===|$)`)
	matches := filePattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			files = append(files, CodeFile{
				Path:    strings.TrimSpace(match[1]),
				Content: strings.TrimSpace(match[2]),
			})
		}
	}
	
	// If no structured format, try to extract code blocks
	if len(files) == 0 {
		codeBlocks := o.extractCodeBlocks(content)
		for i, block := range codeBlocks {
			ext := o.detectLanguage(block)
			files = append(files, CodeFile{
				Path:    fmt.Sprintf("file_%d.%s", i+1, ext),
				Content: block,
			})
		}
	}
	
	return files
}

// extractCodeBlocks finds code blocks in markdown
func (o *EnhancedOrchestrator) extractCodeBlocks(content string) []string {
	var blocks []string
	codePattern := regexp.MustCompile("```[a-z]*\n([\\s\\S]*?)```")
	matches := codePattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			blocks = append(blocks, strings.TrimSpace(match[1]))
		}
	}
	return blocks
}

// extractSection extracts specific sections from content
func (o *EnhancedOrchestrator) extractSection(content, section string) string {
	lower := strings.ToLower(content)
	start := strings.Index(lower, strings.ToLower(section))
	if start == -1 {
		return ""
	}
	
	// Find the content after the section header
	subContent := content[start:]
	lines := strings.Split(subContent, "\n")
	
	var result []string
	inSection := false
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(section)) {
			inSection = true
			continue
		}
		if inSection {
			if strings.HasPrefix(line, "#") && !strings.Contains(line, section) {
				break // Next section started
			}
			result = append(result, line)
		}
	}
	
	return strings.Join(result, "\n")
}

// detectLanguage detects programming language from code
func (o *EnhancedOrchestrator) triggerE2BWorkflow(projectPath string) {
	e2bServerURL := "http://localhost:3000" // The Node.js server
	o.logger.Info("Triggering E2B workflow", zap.String("path", projectPath))

	payload := map[string]string{"path": projectPath}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		o.logger.Error("Error creating JSON payload for E2B server", zap.Error(err))
		return
	}

	resp, err := http.Post(e2bServerURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		o.logger.Error("Error calling E2B server", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		o.logger.Error("E2B server returned non-OK status", zap.String("status", resp.Status), zap.String("body", string(body)))
		return
	}

	o.logger.Info("Successfully triggered E2B workflow.")
}

// detectLanguage detects programming language from code
func (o *EnhancedOrchestrator) detectLanguage(code string) string {
	if strings.Contains(code, "package main") || strings.Contains(code, "func ") {
		return "go"
	}
	if strings.Contains(code, "const ") || strings.Contains(code, "function ") || strings.Contains(code, "=>") {
		return "js"
	}
	if strings.Contains(code, "def ") || strings.Contains(code, "import ") {
		return "py"
	}
	if strings.Contains(code, "FROM ") || strings.Contains(code, "RUN ") {
		return "dockerfile"
	}
	if strings.Contains(code, "apiVersion:") || strings.Contains(code, "kind:") {
		return "yaml"
	}
	if strings.Contains(code, "{") && strings.Contains(code, "}") {
		return "json"
	}
	return "txt"
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
	orchestrator *EnhancedOrchestrator
	router       *mux.Router
}

func NewServer(orchestrator *EnhancedOrchestrator) *Server {
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	var (
		port      = flag.String("port", "8092", "Server port")
		workspace = flag.String("workspace", "/Users/ososerious/OSA/agent-workspace", "Workspace directory")
	)
	flag.Parse()

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal("GROQ_API_KEY environment variable is required")
	}

	// Create enhanced orchestrator
	orchestrator, err := NewEnhancedOrchestrator(apiKey, *workspace)
	if err != nil {
		log.Fatal("Failed to create orchestrator:", err)
	}

	// Create server
	server := NewServer(orchestrator)

	log.Printf("[ENHANCED ORCHESTRATOR] Starting on port %s", *port)
	log.Printf("[WORKSPACE] %s", *workspace)
	log.Printf("[STATUS] Ready to generate complete applications!")

	if err := http.ListenAndServe(":"+*port, server.router); err != nil {
		log.Fatal(err)
	}
}