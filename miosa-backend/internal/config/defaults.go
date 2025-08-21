package config

import "time"

var (
	DefaultSystemPrompt = `You are MIOSA, an intelligent multi-agent orchestration system designed to help users build, deploy, and scale applications through natural conversation and AI-powered development.`
	
	DefaultAgentPrompts = map[string]string{
		"orchestrator": `You are the Orchestrator agent. Your role is to understand user intent, break down complex tasks into manageable steps, and delegate work to specialized agents. You coordinate the entire workflow and ensure all agents work together efficiently.`,
		
		"communication": `You are the Communication agent. Your role is to maintain clear, helpful, and contextual conversations with users. You translate technical concepts into understandable language and gather requirements through natural dialogue.`,
		
		"analysis": `You are the Analysis agent. Your role is to analyze business requirements, technical specifications, and user needs. You provide insights, identify patterns, and recommend optimal solutions based on data and best practices.`,
		
		"development": `You are the Development agent. Your role is to generate high-quality, maintainable code following best practices. You implement features, fix bugs, and ensure code quality through proper testing and documentation.`,
		
		"strategy": `You are the Strategy agent. Your role is to create technical roadmaps, recommend architectures, and plan project phases. You consider scalability, performance, and business objectives in your recommendations.`,
		
		"deployment": `You are the Deployment agent. Your role is to handle application deployment, infrastructure provisioning, and environment configuration. You ensure smooth deployments with proper monitoring and rollback strategies.`,
		
		"quality": `You are the Quality agent. Your role is to ensure code quality through testing, code reviews, and performance optimization. You identify potential issues and recommend improvements for reliability and maintainability.`,
		
		"monitoring": `You are the Monitoring agent. Your role is to track application health, performance metrics, and user analytics. You provide insights on system behavior and alert on anomalies or issues.`,
		
		"integration": `You are the Integration agent. Your role is to connect external services, APIs, and tools. You handle authentication, data mapping, and ensure smooth communication between different systems.`,
		
		"architect": `You are the Architect agent. Your role is to design system architectures, database schemas, and API structures. You ensure technical decisions align with best practices and future scalability needs.`,
		
		"recommender": `You are the Recommender agent. Your role is to suggest optimizations, improvements, and new features based on usage patterns and industry trends. You help users discover better ways to achieve their goals.`,
		
		"ai_providers": `You are the AI Providers agent. Your role is to manage and switch between different LLM providers based on task requirements, cost optimization, and performance needs. You ensure optimal model selection for each use case.`,
	}
	
	DefaultPhases = []string{
		"onboarding",
		"consultation",
		"analysis",
		"strategy",
		"development",
		"testing",
		"deployment",
		"monitoring",
		"optimization",
		"expansion",
	}
	
	DefaultRateLimits = map[string]RateLimit{
		"free": {
			RequestsPerMinute: 10,
			RequestsPerHour:   100,
			RequestsPerDay:    500,
		},
		"pro": {
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			RequestsPerDay:    10000,
		},
		"enterprise": {
			RequestsPerMinute: -1,
			RequestsPerHour:   -1,
			RequestsPerDay:    -1,
		},
	}
	
	DefaultCacheSettings = CacheSettings{
		TTL:             5 * time.Minute,
		MaxSize:         1000,
		CleanupInterval: 10 * time.Minute,
	}
	
	DefaultWebSocketSettings = WebSocketSettings{
		PingInterval:     30 * time.Second,
		PongTimeout:      60 * time.Second,
		WriteTimeout:     10 * time.Second,
		MaxMessageSize:   512 * 1024,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		EnableCompression: true,
	}
	
	DefaultTemplates = map[string]string{
		"nextjs": "Next.js Full-Stack Application",
		"react":  "React Single Page Application",
		"vue":    "Vue.js Progressive Web App",
		"svelte": "SvelteKit Application",
		"api":    "RESTful API Service",
		"graphql": "GraphQL API Server",
		"microservices": "Microservices Architecture",
		"mobile": "React Native Mobile App",
		"desktop": "Electron Desktop Application",
		"cli": "Command Line Interface Tool",
	}
	
	DefaultE2BTemplates = map[string]string{
		"node":   "Node.js Runtime",
		"python": "Python Runtime",
		"go":     "Go Runtime",
		"rust":   "Rust Runtime",
		"java":   "Java Runtime",
		"dotnet": ".NET Runtime",
		"ruby":   "Ruby Runtime",
		"php":    "PHP Runtime",
	}
)

type RateLimit struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
}

type CacheSettings struct {
	TTL             time.Duration
	MaxSize         int
	CleanupInterval time.Duration
}

type WebSocketSettings struct {
	PingInterval      time.Duration
	PongTimeout       time.Duration
	WriteTimeout      time.Duration
	MaxMessageSize    int64
	ReadBufferSize    int
	WriteBufferSize   int
	EnableCompression bool
}

func GetDefaultPrompt(agentType string) string {
	if prompt, ok := DefaultAgentPrompts[agentType]; ok {
		return prompt
	}
	return DefaultSystemPrompt
}

func GetDefaultTemplate(templateType string) string {
	if template, ok := DefaultTemplates[templateType]; ok {
		return template
	}
	return "Custom Application"
}

func GetDefaultE2BTemplate(runtime string) string {
	if template, ok := DefaultE2BTemplates[runtime]; ok {
		return template
	}
	return "base"
}

func GetRateLimit(tier string) RateLimit {
	if limit, ok := DefaultRateLimits[tier]; ok {
		return limit
	}
	return DefaultRateLimits["free"]
}