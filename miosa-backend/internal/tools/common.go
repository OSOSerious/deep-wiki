package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FileSystemTool provides file system operations
type FileSystemTool struct {
	basePath string
}

func NewFileSystemTool() *FileSystemTool {
	return &FileSystemTool{
		basePath: "/tmp/miosa", // Sandboxed path
	}
}

func (t *FileSystemTool) GetName() string {
	return "filesystem"
}

func (t *FileSystemTool) GetDescription() string {
	return "Performs file system operations like read, write, list, and delete files"
}

func (t *FileSystemTool) Validate(input map[string]interface{}) error {
	operation, ok := input["operation"].(string)
	if !ok {
		return fmt.Errorf("operation is required")
	}
	
	validOps := []string{"read", "write", "list", "delete", "mkdir"}
	valid := false
	for _, op := range validOps {
		if operation == op {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid operation: %s", operation)
	}
	
	return nil
}

func (t *FileSystemTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	operation := input["operation"].(string)
	
	switch operation {
	case "read":
		path := input["path"].(string)
		content, err := os.ReadFile(filepath.Join(t.basePath, path))
		if err != nil {
			return nil, err
		}
		return string(content), nil
		
	case "write":
		path := input["path"].(string)
		content := input["content"].(string)
		fullPath := filepath.Join(t.basePath, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		return nil, os.WriteFile(fullPath, []byte(content), 0644)
		
	case "list":
		path := input["path"].(string)
		entries, err := os.ReadDir(filepath.Join(t.basePath, path))
		if err != nil {
			return nil, err
		}
		
		files := []string{}
		for _, entry := range entries {
			files = append(files, entry.Name())
		}
		return files, nil
		
	case "delete":
		path := input["path"].(string)
		return nil, os.Remove(filepath.Join(t.basePath, path))
		
	case "mkdir":
		path := input["path"].(string)
		return nil, os.MkdirAll(filepath.Join(t.basePath, path), 0755)
		
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// APICallTool makes HTTP API calls
type APICallTool struct {
	client *http.Client
}

func NewAPICallTool() *APICallTool {
	return &APICallTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *APICallTool) GetName() string {
	return "api_call"
}

func (t *APICallTool) GetDescription() string {
	return "Makes HTTP API calls to external services"
}

func (t *APICallTool) Validate(input map[string]interface{}) error {
	if _, ok := input["url"].(string); !ok {
		return fmt.Errorf("url is required")
	}
	if _, ok := input["method"].(string); !ok {
		return fmt.Errorf("method is required")
	}
	return nil
}

func (t *APICallTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	url := input["url"].(string)
	method := input["method"].(string)
	
	var body io.Reader
	if bodyData, ok := input["body"]; ok {
		bodyBytes, err := json.Marshal(bodyData)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(string(bodyBytes))
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	
	if headers, ok := input["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}
	}
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
		"headers":     resp.Header,
	}, nil
}

// DatabaseQueryTool executes database queries
type DatabaseQueryTool struct {
	db *sql.DB
}

func NewDatabaseQueryTool() *DatabaseQueryTool {
	// In production, this would be initialized with actual DB connection
	return &DatabaseQueryTool{}
}

func (t *DatabaseQueryTool) GetName() string {
	return "database_query"
}

func (t *DatabaseQueryTool) GetDescription() string {
	return "Executes database queries and returns results"
}

func (t *DatabaseQueryTool) Validate(input map[string]interface{}) error {
	if _, ok := input["query"].(string); !ok {
		return fmt.Errorf("query is required")
	}
	return nil
}

func (t *DatabaseQueryTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	query := input["query"].(string)
	
	// Safety check - only allow SELECT queries in this demo
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}
	
	// Mock response for demo
	return map[string]interface{}{
		"rows": []map[string]interface{}{
			{"id": 1, "name": "Example", "created_at": time.Now()},
		},
		"count": 1,
	}, nil
}

// SearchTool performs code and document searches
type SearchTool struct{}

func NewSearchTool() *SearchTool {
	return &SearchTool{}
}

func (t *SearchTool) GetName() string {
	return "search"
}

func (t *SearchTool) GetDescription() string {
	return "Searches for code patterns, files, and documentation"
}

func (t *SearchTool) Validate(input map[string]interface{}) error {
	if _, ok := input["query"].(string); !ok {
		return fmt.Errorf("query is required")
	}
	return nil
}

func (t *SearchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	query := input["query"].(string)
	searchType := "code"
	if st, ok := input["type"].(string); ok {
		searchType = st
	}
	
	// Mock search results
	return map[string]interface{}{
		"results": []map[string]interface{}{
			{
				"file":    "example.go",
				"line":    42,
				"content": fmt.Sprintf("// Found: %s", query),
				"score":   0.95,
			},
		},
		"total":  1,
		"type":   searchType,
		"query":  query,
	}, nil
}

// CodeAnalyzerTool analyzes code for quality and patterns
type CodeAnalyzerTool struct{}

func NewCodeAnalyzerTool() *CodeAnalyzerTool {
	return &CodeAnalyzerTool{}
}

func (t *CodeAnalyzerTool) GetName() string {
	return "code_analyzer"
}

func (t *CodeAnalyzerTool) GetDescription() string {
	return "Analyzes code for quality, patterns, and potential issues"
}

func (t *CodeAnalyzerTool) Validate(input map[string]interface{}) error {
	if _, ok := input["code"].(string); !ok {
		return fmt.Errorf("code is required")
	}
	return nil
}

func (t *CodeAnalyzerTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	code := input["code"].(string)
	language := "go"
	if lang, ok := input["language"].(string); ok {
		language = lang
	}
	
	// Mock analysis
	return map[string]interface{}{
		"language":    language,
		"lines":       len(strings.Split(code, "\n")),
		"complexity":  "moderate",
		"issues":      []string{},
		"suggestions": []string{"Consider adding error handling"},
		"score":       85,
	}, nil
}

// DocumentationTool generates and retrieves documentation
type DocumentationTool struct{}

func NewDocumentationTool() *DocumentationTool {
	return &DocumentationTool{}
}

func (t *DocumentationTool) GetName() string {
	return "documentation"
}

func (t *DocumentationTool) GetDescription() string {
	return "Generates and retrieves documentation for code and APIs"
}

func (t *DocumentationTool) Validate(input map[string]interface{}) error {
	if _, ok := input["action"].(string); !ok {
		return fmt.Errorf("action is required (generate/retrieve)")
	}
	return nil
}

func (t *DocumentationTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	action := input["action"].(string)
	
	switch action {
	case "generate":
		code := input["code"].(string)
		return map[string]interface{}{
			"documentation": fmt.Sprintf("Documentation for:\n%s", code[:min(100, len(code))]),
			"format":        "markdown",
		}, nil
		
	case "retrieve":
		topic := input["topic"].(string)
		return map[string]interface{}{
			"content": fmt.Sprintf("Documentation about %s", topic),
			"source":  "internal",
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// TestRunnerTool runs tests and returns results
type TestRunnerTool struct{}

func NewTestRunnerTool() *TestRunnerTool {
	return &TestRunnerTool{}
}

func (t *TestRunnerTool) GetName() string {
	return "test_runner"
}

func (t *TestRunnerTool) GetDescription() string {
	return "Runs tests and returns results with coverage information"
}

func (t *TestRunnerTool) Validate(input map[string]interface{}) error {
	if _, ok := input["path"].(string); !ok {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (t *TestRunnerTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	path := input["path"].(string)
	testType := "unit"
	if tt, ok := input["type"].(string); ok {
		testType = tt
	}
	
	// Mock test execution
	return map[string]interface{}{
		"passed":   10,
		"failed":   0,
		"skipped":  2,
		"coverage": 85.5,
		"duration": "2.3s",
		"type":     testType,
		"path":     path,
	}, nil
}

// GitTool performs git operations
type GitTool struct{}

func NewGitTool() *GitTool {
	return &GitTool{}
}

func (t *GitTool) GetName() string {
	return "git"
}

func (t *GitTool) GetDescription() string {
	return "Performs git operations like status, diff, commit, and branch management"
}

func (t *GitTool) Validate(input map[string]interface{}) error {
	if _, ok := input["command"].(string); !ok {
		return fmt.Errorf("command is required")
	}
	return nil
}

func (t *GitTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	command := input["command"].(string)
	
	// Safety: only allow safe git commands
	allowedCommands := []string{"status", "diff", "log", "branch", "show"}
	allowed := false
	for _, cmd := range allowedCommands {
		if strings.HasPrefix(command, cmd) {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return nil, fmt.Errorf("command not allowed: %s", command)
	}
	
	// Execute git command
	cmd := exec.CommandContext(ctx, "git", strings.Fields(command)...)
	output, err := cmd.CombinedOutput()
	
	return map[string]interface{}{
		"output": string(output),
		"error":  err,
	}, nil
}

// DockerTool manages Docker containers and images
type DockerTool struct{}

func NewDockerTool() *DockerTool {
	return &DockerTool{}
}

func (t *DockerTool) GetName() string {
	return "docker"
}

func (t *DockerTool) GetDescription() string {
	return "Manages Docker containers and images"
}

func (t *DockerTool) Validate(input map[string]interface{}) error {
	if _, ok := input["action"].(string); !ok {
		return fmt.Errorf("action is required")
	}
	return nil
}

func (t *DockerTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	action := input["action"].(string)
	
	// Mock Docker operations
	switch action {
	case "ps":
		return map[string]interface{}{
			"containers": []map[string]interface{}{
				{"id": "abc123", "name": "miosa-api", "status": "running"},
			},
		}, nil
		
	case "images":
		return map[string]interface{}{
			"images": []map[string]interface{}{
				{"repository": "miosa/api", "tag": "latest", "size": "150MB"},
			},
		}, nil
		
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// SchemaGeneratorTool generates database and API schemas
type SchemaGeneratorTool struct{}

func NewSchemaGeneratorTool() *SchemaGeneratorTool {
	return &SchemaGeneratorTool{}
}

func (t *SchemaGeneratorTool) GetName() string {
	return "schema_generator"
}

func (t *SchemaGeneratorTool) GetDescription() string {
	return "Generates database schemas, API schemas, and type definitions"
}

func (t *SchemaGeneratorTool) Validate(input map[string]interface{}) error {
	if _, ok := input["type"].(string); !ok {
		return fmt.Errorf("type is required (database/api/graphql)")
	}
	return nil
}

func (t *SchemaGeneratorTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	schemaType := input["type"].(string)
	name := "Entity"
	if n, ok := input["name"].(string); ok {
		name = n
	}
	
	switch schemaType {
	case "database":
		return map[string]interface{}{
			"sql": fmt.Sprintf(`CREATE TABLE %s (
	id UUID PRIMARY KEY,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);`, strings.ToLower(name)),
			"type": "postgresql",
		}, nil
		
	case "api":
		return map[string]interface{}{
			"openapi": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"id": map[string]string{"type": "string"}},
			},
		}, nil
		
	case "graphql":
		return map[string]interface{}{
			"schema": fmt.Sprintf("type %s {\n  id: ID!\n  createdAt: DateTime!\n}", name),
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown schema type: %s", schemaType)
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}