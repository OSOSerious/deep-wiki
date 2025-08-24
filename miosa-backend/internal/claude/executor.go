package claude

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
)

// CommandExecutor handles the execution of Claude Code commands
type CommandExecutor struct {
	registry    *CommandRegistry
	sessionID   uuid.UUID
	workspaceID uuid.UUID
	userID      uuid.UUID
	history     []CommandHistory
	mu          sync.RWMutex
}

// CommandHistory tracks command execution history
type CommandHistory struct {
	Command   CommandType
	Arguments map[string]interface{}
	Result    *agents.Result
	Timestamp time.Time
	Duration  time.Duration
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(sessionID, workspaceID, userID uuid.UUID) *CommandExecutor {
	return &CommandExecutor{
		registry:    NewCommandRegistry(),
		sessionID:   sessionID,
		workspaceID: workspaceID,
		userID:      userID,
		history:     make([]CommandHistory, 0),
	}
}

// ExecuteCommand executes a slash command
func (e *CommandExecutor) ExecuteCommand(ctx context.Context, input string) (*CommandResult, error) {
	// Parse the command
	cmdType, args, err := e.registry.ParseCommand(input)
	if err != nil {
		// Check if it's a help command
		if strings.HasPrefix(input, "/help") {
			return e.handleHelp(input)
		}
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	// Handle meta commands
	switch cmdType {
	case CommandHelp:
		return e.handleHelp(input)
	case CommandCapabilities:
		return e.handleCapabilities(args)
	case CommandStatus:
		return e.handleStatus(args)
	case CommandWorkflow:
		return e.handleWorkflow(ctx, args)
	}

	// Execute agent-based command
	return e.executeAgentCommand(ctx, cmdType, args)
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success    bool                   `json:"success"`
	Output     string                 `json:"output"`
	Data       map[string]interface{} `json:"data"`
	Agent      agents.AgentType       `json:"agent,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Error      error                  `json:"error,omitempty"`
}

// executeAgentCommand executes a command through its associated agent
func (e *CommandExecutor) executeAgentCommand(ctx context.Context, cmdType CommandType, args map[string]interface{}) (*CommandResult, error) {
	startTime := time.Now()

	// Get command definition
	cmd, exists := e.registry.Get(cmdType)
	if !exists {
		return &CommandResult{
			Success: false,
			Error:   fmt.Errorf("command %s not found", cmdType),
		}, nil
	}

	// Add context to arguments
	args["session_id"] = e.sessionID
	args["workspace_id"] = e.workspaceID
	args["user_id"] = e.userID

	// Execute through registry
	result, err := e.registry.Execute(ctx, string(cmdType), args)
	
	duration := time.Since(startTime)

	// Record in history
	e.recordHistory(cmdType, args, result, duration)

	if err != nil {
		return &CommandResult{
			Success:  false,
			Output:   fmt.Sprintf("Command execution failed: %v", err),
			Agent:    cmd.Agent,
			Duration: duration,
			Error:    err,
		}, nil
	}

	// Format the result
	cmdResult := &CommandResult{
		Success:  result.Success,
		Output:   result.Output,
		Data:     result.Data,
		Agent:    cmd.Agent,
		Duration: duration,
	}

	// Add suggestions if available
	if len(result.Suggestions) > 0 {
		cmdResult.Suggestion = strings.Join(result.Suggestions, "\n")
	}

	// Handle next steps
	if result.NextAgent != "" {
		cmdResult.Suggestion = fmt.Sprintf("Next suggested command: /%s %s", 
			strings.ToLower(string(result.NextAgent)), result.NextStep)
	}

	return cmdResult, nil
}

// handleHelp handles help commands
func (e *CommandExecutor) handleHelp(input string) (*CommandResult, error) {
	parts := strings.Fields(input)
	
	if len(parts) == 1 {
		// General help
		return &CommandResult{
			Success: true,
			Output:  e.registry.GetAllHelp(),
		}, nil
	}

	// Specific command help
	cmdName := parts[1]
	if !strings.HasPrefix(cmdName, "/") {
		cmdName = "/" + cmdName
	}

	cmd, exists := e.registry.GetByName(cmdName)
	if !exists {
		return &CommandResult{
			Success: false,
			Output:  fmt.Sprintf("Command %s not found", cmdName),
		}, nil
	}

	return &CommandResult{
		Success: true,
		Output:  e.registry.GetHelp(cmd.Type),
	}, nil
}

// handleCapabilities handles the capabilities command
func (e *CommandExecutor) handleCapabilities(args map[string]interface{}) (*CommandResult, error) {
	var output strings.Builder
	
	// Check if specific agent requested
	if agentName, ok := args["agent"].(string); ok {
		agentType := agents.AgentType(agentName)
		capabilities, err := agents.GetCapabilities(agentType)
		if err != nil {
			return &CommandResult{
				Success: false,
				Output:  fmt.Sprintf("Failed to get capabilities for %s: %v", agentName, err),
			}, nil
		}

		output.WriteString(fmt.Sprintf("Capabilities for %s:\n\n", agentName))
		for _, cap := range capabilities {
			required := ""
			if cap.Required {
				required = " (required)"
			}
			output.WriteString(fmt.Sprintf("• %s%s\n  %s\n  Version: %s\n\n", 
				cap.Name, required, cap.Description, cap.Version))
		}
	} else {
		// Show all capabilities
		allCaps := agents.GetAllCapabilities()
		output.WriteString("Agent Capabilities:\n\n")
		
		for agentType, caps := range allCaps {
			output.WriteString(fmt.Sprintf("%s Agent:\n", agentType))
			for _, cap := range caps {
				output.WriteString(fmt.Sprintf("  • %s: %s\n", cap.Name, cap.Description))
			}
			output.WriteString("\n")
		}
	}

	return &CommandResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// handleStatus handles the status command
func (e *CommandExecutor) handleStatus(args map[string]interface{}) (*CommandResult, error) {
	var output strings.Builder
	
	registeredAgents := agents.ListAgents()
	output.WriteString(fmt.Sprintf("Active Agents: %d\n\n", len(registeredAgents)))

	for _, agentType := range registeredAgents {
		agent, _ := agents.Get(agentType)
		if agent != nil {
			// Get evaluation metrics if available
			eval, _ := agents.GetEvaluation(agentType)
			
			status := "✅ Active"
			output.WriteString(fmt.Sprintf("%s %s\n", agentType, status))
			
			if eval != nil {
				successRate := float64(0)
				if eval.TotalExecutions > 0 {
					successRate = float64(eval.SuccessfulExecutions) / float64(eval.TotalExecutions) * 100
				}
				output.WriteString(fmt.Sprintf("  Executions: %d | Success Rate: %.1f%% | Avg Time: %dms\n",
					eval.TotalExecutions, successRate, eval.AverageExecutionMS))
			}
			
			output.WriteString(fmt.Sprintf("  Description: %s\n\n", agent.GetDescription()))
		}
	}

	// Add session info
	output.WriteString(fmt.Sprintf("\nSession Info:\n"))
	output.WriteString(fmt.Sprintf("  Session ID: %s\n", e.sessionID))
	output.WriteString(fmt.Sprintf("  Workspace ID: %s\n", e.workspaceID))
	output.WriteString(fmt.Sprintf("  Commands Executed: %d\n", len(e.history)))

	return &CommandResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// handleWorkflow handles workflow commands
func (e *CommandExecutor) handleWorkflow(ctx context.Context, args map[string]interface{}) (*CommandResult, error) {
	template, ok := args["template"].(string)
	if !ok {
		return &CommandResult{
			Success: false,
			Output:  "Workflow template not specified",
		}, nil
	}

	// Define workflow templates
	workflows := map[string][]string{
		"feature-development": {
			"/analyze requirements",
			"/architect system",
			"/develop feature",
			"/quality test --coverage",
			"/deploy staging",
		},
		"bug-fix": {
			"/analyze code --depth deep",
			"/develop fix",
			"/quality test",
			"/quality review",
			"/deploy staging",
		},
		"deployment-pipeline": {
			"/quality test --coverage",
			"/quality lint",
			"/deploy staging",
			"/monitor performance",
			"/deploy production --rollback-on-failure",
		},
		"code-review": {
			"/analyze code",
			"/quality review --auto-fix",
			"/quality lint",
			"/communicate document",
		},
	}

	workflow, exists := workflows[template]
	if !exists {
		return &CommandResult{
			Success: false,
			Output:  fmt.Sprintf("Unknown workflow template: %s", template),
		}, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Executing workflow: %s\n\n", template))

	// Execute each command in the workflow
	for i, cmdStr := range workflow {
		output.WriteString(fmt.Sprintf("Step %d: %s\n", i+1, cmdStr))
		
		result, err := e.ExecuteCommand(ctx, cmdStr)
		if err != nil {
			output.WriteString(fmt.Sprintf("  ❌ Failed: %v\n", err))
			return &CommandResult{
				Success: false,
				Output:  output.String(),
			}, nil
		}

		if result.Success {
			output.WriteString("  ✅ Completed\n")
		} else {
			output.WriteString(fmt.Sprintf("  ⚠️ Partial: %s\n", result.Output))
		}
	}

	output.WriteString(fmt.Sprintf("\n✅ Workflow '%s' completed successfully", template))

	return &CommandResult{
		Success: true,
		Output:  output.String(),
		Data: map[string]interface{}{
			"workflow": template,
			"steps":    len(workflow),
		},
	}, nil
}

// recordHistory records command execution in history
func (e *CommandExecutor) recordHistory(cmdType CommandType, args map[string]interface{}, result *agents.Result, duration time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.history = append(e.history, CommandHistory{
		Command:   cmdType,
		Arguments: args,
		Result:    result,
		Timestamp: time.Now(),
		Duration:  duration,
	})

	// Keep only last 100 commands
	if len(e.history) > 100 {
		e.history = e.history[len(e.history)-100:]
	}
}

// GetHistory returns the command execution history
func (e *CommandExecutor) GetHistory() []CommandHistory {
	e.mu.RLock()
	defer e.mu.RUnlock()

	history := make([]CommandHistory, len(e.history))
	copy(history, e.history)
	return history
}

// ExecutePipeline executes multiple commands in sequence
func (e *CommandExecutor) ExecutePipeline(ctx context.Context, commands []string) ([]*CommandResult, error) {
	results := make([]*CommandResult, 0, len(commands))

	for _, cmd := range commands {
		result, err := e.ExecuteCommand(ctx, cmd)
		if err != nil {
			return results, fmt.Errorf("pipeline failed at command '%s': %w", cmd, err)
		}

		results = append(results, result)

		// Stop pipeline if command failed
		if !result.Success {
			return results, fmt.Errorf("pipeline stopped: command '%s' failed", cmd)
		}
	}

	return results, nil
}

// ExecuteParallel executes multiple commands in parallel
func (e *CommandExecutor) ExecuteParallel(ctx context.Context, commands []string) ([]*CommandResult, error) {
	results := make([]*CommandResult, len(commands))
	errChan := make(chan error, len(commands))
	var wg sync.WaitGroup

	for i, cmd := range commands {
		wg.Add(1)
		go func(index int, command string) {
			defer wg.Done()

			result, err := e.ExecuteCommand(ctx, command)
			if err != nil {
				errChan <- fmt.Errorf("command '%s' failed: %w", command, err)
				return
			}

			results[index] = result
		}(i, cmd)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []string
	for err := range errChan {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("parallel execution had errors: %s", strings.Join(errs, "; "))
	}

	return results, nil
}

// SuggestNextCommand suggests the next command based on history and current context
func (e *CommandExecutor) SuggestNextCommand() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.history) == 0 {
		return "/help"
	}

	lastCmd := e.history[len(e.history)-1]
	
	// Suggest based on last command and result
	suggestions := map[CommandType]string{
		CommandAnalyze:   "/architect system",
		CommandArchitect: "/develop feature",
		CommandDevelop:   "/quality test",
		CommandQuality:   "/deploy staging",
		CommandDeploy:    "/monitor performance",
	}

	if suggestion, ok := suggestions[lastCmd.Command]; ok {
		return suggestion
	}

	// Check if last result has a next agent suggestion
	if lastCmd.Result != nil && lastCmd.Result.NextAgent != "" {
		return fmt.Sprintf("/%s", strings.ToLower(string(lastCmd.Result.NextAgent)))
	}

	return "/status"
}