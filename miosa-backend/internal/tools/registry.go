package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

// ToolRegistry manages all available tools
type ToolRegistry struct {
	tools map[string]agents.Tool
	mu    sync.RWMutex
}

// GlobalRegistry is the singleton tool registry
var GlobalRegistry = &ToolRegistry{
	tools: make(map[string]agents.Tool),
}

// Register adds a tool to the registry
func Register(tool agents.Tool) error {
	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()
	
	name := tool.GetName()
	if _, exists := GlobalRegistry.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}
	
	GlobalRegistry.tools[name] = tool
	return nil
}

// Get retrieves a tool by name
func Get(name string) (agents.Tool, error) {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()
	
	tool, exists := GlobalRegistry.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	
	return tool, nil
}

// List returns all registered tool names
func List() []string {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()
	
	names := make([]string, 0, len(GlobalRegistry.tools))
	for name := range GlobalRegistry.tools {
		names = append(names, name)
	}
	
	return names
}

// Execute runs a tool with the given input
func Execute(ctx context.Context, toolName string, input map[string]interface{}) (interface{}, error) {
	tool, err := Get(toolName)
	if err != nil {
		return nil, err
	}
	
	// Validate input
	if err := tool.Validate(input); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Execute tool
	return tool.Execute(ctx, input)
}

// InitializeDefaultTools registers all default tools
func InitializeDefaultTools() {
	// Register common tools
	Register(NewFileSystemTool())
	Register(NewAPICallTool())
	Register(NewDatabaseQueryTool())
	Register(NewSearchTool())
	Register(NewCodeAnalyzerTool())
	Register(NewDocumentationTool())
	Register(NewTestRunnerTool())
	Register(NewGitTool())
	Register(NewDockerTool())
	Register(NewSchemaGeneratorTool())
}