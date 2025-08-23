package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/ososerious/miosa-backend/internal/agents"
)

// CommandType represents a Claude Code slash command
type CommandType string

const (
	CommandOrchestrate  CommandType = "/orchestrate"
	CommandAnalyze      CommandType = "/analyze"
	CommandArchitect    CommandType = "/architect"
	CommandCommunicate  CommandType = "/communicate"
	CommandDeploy       CommandType = "/deploy"
	CommandDevelop      CommandType = "/develop"
	CommandIntegrate    CommandType = "/integrate"
	CommandMonitor      CommandType = "/monitor"
	CommandQuality      CommandType = "/quality"
	CommandStrategy     CommandType = "/strategy"
	CommandRecommend    CommandType = "/recommend"
	CommandAIProvider   CommandType = "/ai-provider"
	CommandHelp         CommandType = "/help"
	CommandCapabilities CommandType = "/capabilities"
	CommandStatus       CommandType = "/status"
	CommandWorkflow     CommandType = "/workflow"
	CommandConfig       CommandType = "/config"
)

// Command represents a Claude Code command
type Command struct {
	Type        CommandType
	Name        string
	Description string
	Agent       agents.AgentType
	Actions     []string
	Parameters  map[string]ParameterDef
	Examples    []string
	Icon        string
}

// ParameterDef defines a command parameter
type ParameterDef struct {
	Name        string
	Type        string // string, int, bool, enum
	Required    bool
	Default     interface{}
	Description string
	Values      []string // for enum types
}

// CommandRegistry manages all available commands
type CommandRegistry struct {
	commands map[CommandType]*Command
}

// NewCommandRegistry creates and initializes the command registry
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[CommandType]*Command),
	}
	registry.initializeCommands()
	return registry
}

// initializeCommands sets up all available commands
func (r *CommandRegistry) initializeCommands() {
	// Orchestrate Command
	r.Register(&Command{
		Type:        CommandOrchestrate,
		Name:        "orchestrate",
		Description: "Routes and coordinates tasks between multiple specialized agents",
		Agent:       agents.OrchestratorAgent,
		Icon:        "🎭",
		Actions:     []string{"pipeline", "coordinate", "route"},
		Parameters: map[string]ParameterDef{
			"task": {
				Name:        "task",
				Type:        "string",
				Required:    true,
				Description: "The complex task to orchestrate",
			},
			"parallel": {
				Name:        "parallel",
				Type:        "bool",
				Required:    false,
				Default:     false,
				Description: "Execute agents in parallel when possible",
			},
		},
		Examples: []string{
			`/orchestrate "Build a REST API with authentication"`,
			`/orchestrate pipeline --parallel`,
		},
	})

	// Analyze Command
	r.Register(&Command{
		Type:        CommandAnalyze,
		Name:        "analyze",
		Description: "Performs deep analysis of code structure and requirements",
		Agent:       agents.AnalysisAgent,
		Icon:        "📊",
		Actions:     []string{"code", "architecture", "requirements", "metrics", "dependencies"},
		Parameters: map[string]ParameterDef{
			"target": {
				Name:        "target",
				Type:        "enum",
				Required:    true,
				Values:      []string{"code", "architecture", "requirements"},
				Description: "What to analyze",
			},
			"depth": {
				Name:        "depth",
				Type:        "enum",
				Required:    false,
				Default:     "normal",
				Values:      []string{"shallow", "normal", "deep"},
				Description: "Analysis depth level",
			},
			"file": {
				Name:        "file",
				Type:        "string",
				Required:    false,
				Description: "Specific file to analyze",
			},
		},
		Examples: []string{
			`/analyze code --file "main.go" --depth deep`,
			`/analyze architecture`,
			`/analyze requirements --depth shallow`,
		},
	})

	// Architect Command
	r.Register(&Command{
		Type:        CommandArchitect,
		Name:        "architect",
		Description: "Designs system architecture and database schemas",
		Agent:       agents.ArchitectAgent,
		Icon:        "🏗️",
		Actions:     []string{"system", "database", "api", "microservices", "infrastructure"},
		Parameters: map[string]ParameterDef{
			"design": {
				Name:        "design",
				Type:        "enum",
				Required:    true,
				Values:      []string{"system", "database", "api", "microservices"},
				Description: "Type of architecture to design",
			},
			"pattern": {
				Name:        "pattern",
				Type:        "string",
				Required:    false,
				Description: "Specific architectural pattern to use",
			},
		},
		Examples: []string{
			`/architect system --pattern "event-driven"`,
			`/architect database`,
			`/architect api --pattern "REST"`,
		},
	})

	// Development Command
	r.Register(&Command{
		Type:        CommandDevelop,
		Name:        "develop",
		Description: "Writes code, implements features, and fixes bugs",
		Agent:       agents.DevelopmentAgent,
		Icon:        "💻",
		Actions:     []string{"feature", "fix", "refactor", "implement", "optimize"},
		Parameters: map[string]ParameterDef{
			"action": {
				Name:        "action",
				Type:        "enum",
				Required:    true,
				Values:      []string{"feature", "fix", "refactor"},
				Description: "Development action to perform",
			},
			"name": {
				Name:        "name",
				Type:        "string",
				Required:    false,
				Description: "Name or description of the task",
			},
			"language": {
				Name:        "language",
				Type:        "string",
				Required:    false,
				Default:     "go",
				Description: "Programming language to use",
			},
		},
		Examples: []string{
			`/develop feature --name "user-auth" --language go`,
			`/develop fix "memory leak in handler"`,
			`/develop refactor --name "improve error handling"`,
		},
	})

	// Quality Command
	r.Register(&Command{
		Type:        CommandQuality,
		Name:        "quality",
		Description: "Runs tests and ensures code quality",
		Agent:       agents.QualityAgent,
		Icon:        "✅",
		Actions:     []string{"test", "review", "lint", "coverage", "security"},
		Parameters: map[string]ParameterDef{
			"action": {
				Name:        "action",
				Type:        "enum",
				Required:    true,
				Values:      []string{"test", "review", "lint"},
				Description: "Quality check to perform",
			},
			"coverage": {
				Name:        "coverage",
				Type:        "bool",
				Required:    false,
				Default:     false,
				Description: "Include coverage report",
			},
			"auto-fix": {
				Name:        "auto-fix",
				Type:        "bool",
				Required:    false,
				Default:     false,
				Description: "Automatically fix issues when possible",
			},
		},
		Examples: []string{
			`/quality test --coverage`,
			`/quality review --auto-fix`,
			`/quality lint`,
		},
	})

	// Deployment Command
	r.Register(&Command{
		Type:        CommandDeploy,
		Name:        "deploy",
		Description: "Manages deployment pipelines and environments",
		Agent:       agents.DeploymentAgent,
		Icon:        "🚀",
		Actions:     []string{"staging", "production", "rollback", "preview", "scale"},
		Parameters: map[string]ParameterDef{
			"environment": {
				Name:        "environment",
				Type:        "enum",
				Required:    true,
				Values:      []string{"staging", "production", "preview"},
				Description: "Target deployment environment",
			},
			"version": {
				Name:        "version",
				Type:        "string",
				Required:    false,
				Description: "Version tag for deployment",
			},
			"rollback-on-failure": {
				Name:        "rollback-on-failure",
				Type:        "bool",
				Required:    false,
				Default:     true,
				Description: "Automatically rollback on deployment failure",
			},
		},
		Examples: []string{
			`/deploy staging --branch dev`,
			`/deploy production --version v1.2.0 --rollback-on-failure`,
			`/deploy rollback --environment production`,
		},
	})

	// Communication Command
	r.Register(&Command{
		Type:        CommandCommunicate,
		Name:        "communicate",
		Description: "Handles documentation and explanations",
		Agent:       agents.CommunicationAgent,
		Icon:        "💬",
		Actions:     []string{"summarize", "explain", "document", "translate", "report"},
		Parameters: map[string]ParameterDef{
			"action": {
				Name:        "action",
				Type:        "enum",
				Required:    true,
				Values:      []string{"summarize", "explain", "document"},
				Description: "Communication action to perform",
			},
			"format": {
				Name:        "format",
				Type:        "enum",
				Required:    false,
				Default:     "markdown",
				Values:      []string{"markdown", "html", "pdf", "plain"},
				Description: "Output format",
			},
		},
		Examples: []string{
			`/communicate document --format markdown`,
			`/communicate explain "authentication flow"`,
			`/communicate summarize --format plain`,
		},
	})

	// Strategy Command
	r.Register(&Command{
		Type:        CommandStrategy,
		Name:        "strategy",
		Description: "Plans technical roadmaps and priorities",
		Agent:       agents.StrategyAgent,
		Icon:        "🎯",
		Actions:     []string{"roadmap", "priorities", "optimization", "planning", "assessment"},
		Parameters: map[string]ParameterDef{
			"focus": {
				Name:        "focus",
				Type:        "enum",
				Required:    true,
				Values:      []string{"roadmap", "priorities", "optimization"},
				Description: "Strategic focus area",
			},
			"timeframe": {
				Name:        "timeframe",
				Type:        "enum",
				Required:    false,
				Default:     "quarter",
				Values:      []string{"sprint", "month", "quarter", "year"},
				Description: "Planning timeframe",
			},
		},
		Examples: []string{
			`/strategy roadmap --timeframe quarter`,
			`/strategy priorities --timeframe sprint`,
			`/strategy optimization`,
		},
	})

	// Integration Command
	r.Register(&Command{
		Type:        CommandIntegrate,
		Name:        "integrate",
		Description: "Handles external service integrations",
		Agent:       agents.IntegrationAgent,
		Icon:        "🔌",
		Actions:     []string{"api", "service", "webhook", "database", "auth"},
		Parameters: map[string]ParameterDef{
			"type": {
				Name:        "type",
				Type:        "enum",
				Required:    true,
				Values:      []string{"api", "service", "webhook"},
				Description: "Integration type",
			},
			"provider": {
				Name:        "provider",
				Type:        "string",
				Required:    false,
				Description: "Service provider name",
			},
		},
		Examples: []string{
			`/integrate api --provider stripe`,
			`/integrate webhook --provider github`,
			`/integrate service --provider aws`,
		},
	})

	// Monitoring Command
	r.Register(&Command{
		Type:        CommandMonitor,
		Name:        "monitor",
		Description: "Sets up monitoring and alerting",
		Agent:       agents.MonitoringAgent,
		Icon:        "📈",
		Actions:     []string{"performance", "errors", "metrics", "alerts", "logs"},
		Parameters: map[string]ParameterDef{
			"target": {
				Name:        "target",
				Type:        "enum",
				Required:    true,
				Values:      []string{"performance", "errors", "metrics"},
				Description: "What to monitor",
			},
			"interval": {
				Name:        "interval",
				Type:        "string",
				Required:    false,
				Default:     "1m",
				Description: "Monitoring interval",
			},
		},
		Examples: []string{
			`/monitor performance --interval 30s`,
			`/monitor errors`,
			`/monitor metrics --interval 5m`,
		},
	})

	// Recommender Command
	r.Register(&Command{
		Type:        CommandRecommend,
		Name:        "recommend",
		Description: "Suggests tools, libraries, and patterns",
		Agent:       agents.RecommenderAgent,
		Icon:        "💡",
		Actions:     []string{"tools", "libraries", "patterns", "practices", "frameworks"},
		Parameters: map[string]ParameterDef{
			"category": {
				Name:        "category",
				Type:        "enum",
				Required:    true,
				Values:      []string{"tools", "libraries", "patterns"},
				Description: "Recommendation category",
			},
			"context": {
				Name:        "context",
				Type:        "string",
				Required:    false,
				Description: "Specific context for recommendations",
			},
		},
		Examples: []string{
			`/recommend tools --context "testing"`,
			`/recommend libraries --context "authentication"`,
			`/recommend patterns --context "microservices"`,
		},
	})

	// AI Provider Command
	r.Register(&Command{
		Type:        CommandAIProvider,
		Name:        "ai-provider",
		Description: "Manages AI provider integrations",
		Agent:       agents.AIProvidersAgent,
		Icon:        "🤖",
		Actions:     []string{"configure", "switch", "test", "compare", "optimize"},
		Parameters: map[string]ParameterDef{
			"provider": {
				Name:        "provider",
				Type:        "enum",
				Required:    true,
				Values:      []string{"groq", "kimi", "openai", "anthropic"},
				Description: "AI provider to configure",
			},
			"model": {
				Name:        "model",
				Type:        "string",
				Required:    false,
				Description: "Specific model to use",
			},
		},
		Examples: []string{
			`/ai-provider groq --model mixtral-8x7b`,
			`/ai-provider switch --provider openai`,
			`/ai-provider test --provider kimi`,
		},
	})

	// Meta Commands
	r.Register(&Command{
		Type:        CommandHelp,
		Name:        "help",
		Description: "Shows help information",
		Icon:        "❓",
		Actions:     []string{"all", "agent", "command"},
		Parameters: map[string]ParameterDef{
			"topic": {
				Name:        "topic",
				Type:        "string",
				Required:    false,
				Description: "Specific help topic",
			},
		},
		Examples: []string{
			`/help`,
			`/help develop`,
			`/help --topic "workflows"`,
		},
	})

	r.Register(&Command{
		Type:        CommandCapabilities,
		Name:        "capabilities",
		Description: "Lists all agent capabilities",
		Icon:        "🔧",
		Actions:     []string{"list", "detail"},
		Examples: []string{
			`/capabilities`,
			`/capabilities --agent development`,
		},
	})

	r.Register(&Command{
		Type:        CommandStatus,
		Name:        "status",
		Description: "Shows active agents status",
		Icon:        "📊",
		Actions:     []string{"all", "agent", "health"},
		Examples: []string{
			`/status`,
			`/status --agent orchestrator`,
		},
	})

	r.Register(&Command{
		Type:        CommandWorkflow,
		Name:        "workflow",
		Description: "Uses predefined workflow templates",
		Icon:        "🔄",
		Actions:     []string{"feature-development", "bug-fix", "deployment-pipeline", "code-review"},
		Parameters: map[string]ParameterDef{
			"template": {
				Name:        "template",
				Type:        "enum",
				Required:    true,
				Values:      []string{"feature-development", "bug-fix", "deployment-pipeline", "code-review"},
				Description: "Workflow template to use",
			},
		},
		Examples: []string{
			`/workflow feature-development`,
			`/workflow bug-fix`,
			`/workflow deployment-pipeline`,
		},
	})
}

// Register adds a command to the registry
func (r *CommandRegistry) Register(cmd *Command) {
	r.commands[cmd.Type] = cmd
}

// Get retrieves a command by type
func (r *CommandRegistry) Get(cmdType CommandType) (*Command, bool) {
	cmd, exists := r.commands[cmdType]
	return cmd, exists
}

// GetByName retrieves a command by name string
func (r *CommandRegistry) GetByName(name string) (*Command, bool) {
	// Ensure the name starts with /
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	
	cmdType := CommandType(name)
	return r.Get(cmdType)
}

// List returns all registered commands
func (r *CommandRegistry) List() []*Command {
	commands := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		commands = append(commands, cmd)
	}
	return commands
}

// Execute runs a command with the given context and arguments
func (r *CommandRegistry) Execute(ctx context.Context, cmdName string, args map[string]interface{}) (*agents.Result, error) {
	cmd, exists := r.GetByName(cmdName)
	if !exists {
		return nil, fmt.Errorf("command %s not found", cmdName)
	}

	// Validate parameters
	for paramName, paramDef := range cmd.Parameters {
		if paramDef.Required {
			if _, exists := args[paramName]; !exists {
				return nil, fmt.Errorf("required parameter %s missing", paramName)
			}
		}
	}

	// Get the corresponding agent
	agent, err := agents.Get(cmd.Agent)
	if err != nil {
		return nil, fmt.Errorf("agent %s not available: %w", cmd.Agent, err)
	}

	// Create task from command arguments
	task := agents.Task{
		Type:       string(cmd.Type),
		Input:      fmt.Sprintf("%v", args["input"]),
		Parameters: args,
	}

	// Execute through agent
	return agent.Execute(ctx, task)
}

// ParseCommand parses a command string into command type and arguments
func (r *CommandRegistry) ParseCommand(input string) (CommandType, map[string]interface{}, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	cmdName := parts[0]
	if !strings.HasPrefix(cmdName, "/") {
		return "", nil, fmt.Errorf("commands must start with /")
	}

	cmd, exists := r.GetByName(cmdName)
	if !exists {
		return "", nil, fmt.Errorf("unknown command: %s", cmdName)
	}

	// Parse arguments
	args := make(map[string]interface{})
	for i := 1; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "--") {
			key := strings.TrimPrefix(parts[i], "--")
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "--") {
				args[key] = parts[i+1]
				i++
			} else {
				args[key] = true // Boolean flag
			}
		} else if i == 1 {
			// First non-flag argument is the main input
			args["input"] = strings.Join(parts[i:], " ")
			break
		}
	}

	return cmd.Type, args, nil
}

// GetHelp generates help text for a command
func (r *CommandRegistry) GetHelp(cmdType CommandType) string {
	cmd, exists := r.Get(cmdType)
	if !exists {
		return "Command not found"
	}

	var help strings.Builder
	help.WriteString(fmt.Sprintf("%s %s - %s\n\n", cmd.Icon, cmd.Name, cmd.Description))
	
	if len(cmd.Actions) > 0 {
		help.WriteString("Actions: " + strings.Join(cmd.Actions, ", ") + "\n\n")
	}

	if len(cmd.Parameters) > 0 {
		help.WriteString("Parameters:\n")
		for _, param := range cmd.Parameters {
			required := ""
			if param.Required {
				required = " (required)"
			}
			help.WriteString(fmt.Sprintf("  --%s%s: %s\n", param.Name, required, param.Description))
			if param.Type == "enum" && len(param.Values) > 0 {
				help.WriteString(fmt.Sprintf("    Values: %s\n", strings.Join(param.Values, ", ")))
			}
			if param.Default != nil {
				help.WriteString(fmt.Sprintf("    Default: %v\n", param.Default))
			}
		}
		help.WriteString("\n")
	}

	if len(cmd.Examples) > 0 {
		help.WriteString("Examples:\n")
		for _, example := range cmd.Examples {
			help.WriteString("  " + example + "\n")
		}
	}

	return help.String()
}

// GetAllHelp generates help text for all commands
func (r *CommandRegistry) GetAllHelp() string {
	var help strings.Builder
	help.WriteString("Available Claude Code Commands:\n\n")

	// Group commands by category
	categories := map[string][]*Command{
		"Core":         {},
		"Development":  {},
		"Operations":   {},
		"Analysis":     {},
		"Integration":  {},
		"Meta":         {},
	}

	for _, cmd := range r.commands {
		switch cmd.Type {
		case CommandOrchestrate:
			categories["Core"] = append(categories["Core"], cmd)
		case CommandDevelop, CommandArchitect, CommandQuality:
			categories["Development"] = append(categories["Development"], cmd)
		case CommandDeploy, CommandMonitor:
			categories["Operations"] = append(categories["Operations"], cmd)
		case CommandAnalyze, CommandStrategy, CommandRecommend:
			categories["Analysis"] = append(categories["Analysis"], cmd)
		case CommandIntegrate, CommandCommunicate, CommandAIProvider:
			categories["Integration"] = append(categories["Integration"], cmd)
		case CommandHelp, CommandCapabilities, CommandStatus, CommandWorkflow, CommandConfig:
			categories["Meta"] = append(categories["Meta"], cmd)
		}
	}

	for category, cmds := range categories {
		if len(cmds) > 0 {
			help.WriteString(fmt.Sprintf("\n%s Commands:\n", category))
			for _, cmd := range cmds {
				help.WriteString(fmt.Sprintf("  %s %-15s %s\n", cmd.Icon, string(cmd.Type), cmd.Description))
			}
		}
	}

	help.WriteString("\nUse /help [command] for detailed information about a specific command\n")
	return help.String()
}