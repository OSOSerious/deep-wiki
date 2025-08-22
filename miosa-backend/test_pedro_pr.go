package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"github.com/sormind/OSA/miosa-backend/internal/agents/quality"
	"github.com/sormind/OSA/miosa-backend/internal/config"
	"github.com/sormind/OSA/miosa-backend/internal/llm"
)

func main() {
	fmt.Println("=== Testing Pedro's PR Changes ===\n")

	// Load environment
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Initialize Groq client
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	if groqAPIKey == "" {
		log.Fatal("GROQ_API_KEY not set")
	}

	groqClient, err := groq.NewClient(groqAPIKey)
	if err != nil {
		log.Fatalf("Failed to create Groq client: %v", err)
	}

	// Load config
	cfg := config.LoadConfig()

	// Initialize LLM router
	router := llm.NewRouter(cfg)

	// Initialize agents
	registry := agents.GetRegistry()
	
	// Test 1: Strategic Reasoning Middleware
	fmt.Println("TEST 1: Strategic Reasoning Middleware")
	fmt.Println("=====================================")
	testStrategicReasoning(groqClient, router, registry)
	
	// Test 2: Quality Agent with new capabilities
	fmt.Println("\nTEST 2: Quality Agent")
	fmt.Println("=====================")
	testQualityAgent(groqClient, cfg, registry)
	
	// Test 3: Tool Registration and Association
	fmt.Println("\nTEST 3: Tool Registry")
	fmt.Println("=====================")
	testToolRegistry(registry)
	
	// Test 4: End-to-end with Strategic Wrapper
	fmt.Println("\nTEST 4: End-to-End Strategic Execution")
	fmt.Println("=======================================")
	testEndToEndStrategic(groqClient, router, registry)

	fmt.Println("\n=== All Tests Completed ===")
}

func testStrategicReasoning(groqClient *groq.Client, router *llm.Router, registry *agents.Registry) {
	ctx := context.Background()

	// Create a simple test agent
	testAgent := &agents.BaseAgent{
		Type_:        agents.AgentTypeAnalysis,
		Name_:        "TestAnalysisAgent",
		Description_: "Test agent for strategic reasoning",
		Capabilities_: []agents.Capability{
			agents.CapabilityCodeAnalysis,
			agents.CapabilityPatternRecognition,
		},
		ExecuteFunc: func(ctx context.Context, task agents.Task) (*agents.Result, error) {
			// Check if strategy was injected
			strategy := agents.ChosenStrategyFromCtx(ctx)
			
			output := fmt.Sprintf("Executed task: %s", task.Input)
			if strategy != nil {
				output += fmt.Sprintf("\nUsing strategy: %s", strategy.Approach)
			}
			
			return &agents.Result{
				Success:    true,
				Output:     output,
				Confidence: 8.5,
			}, nil
		},
	}

	// Register with strategic reasoning
	registry.RegisterWithStrategic(testAgent, 3) // Request 3 candidate strategies
	
	// Execute task
	task := agents.Task{
		ID:    uuid.New(),
		Type:  "analysis",
		Input: "Analyze the architecture of a microservices system",
	}

	// Get the wrapped agent
	agent, err := registry.GetAgent(agents.AgentTypeAnalysis)
	if err != nil {
		log.Printf("Failed to get agent: %v", err)
		return
	}

	result, err := agent.Execute(ctx, task)
	if err != nil {
		log.Printf("Execution failed: %v", err)
		return
	}

	fmt.Printf("✅ Strategic execution successful!\n")
	fmt.Printf("   Output: %s\n", result.Output)
	fmt.Printf("   Confidence: %.1f\n", result.Confidence)
}

func testQualityAgent(groqClient *groq.Client, cfg *config.Config, registry *agents.Registry) {
	ctx := context.Background()

	// Create Quality Agent with config
	qualityConfig := quality.Config{
		Model:       cfg.FastModel,
		MaxTokens:   2000,
		Temperature: 0.3,
		TopP:        0.9,
	}

	qualityAgent := quality.NewQualityAgent(groqClient, qualityConfig)
	
	// Register the agent
	registry.Register(qualityAgent)

	// Register quality tools
	quality.RegisterQualityTools(qualityAgent)

	// Test the quality agent
	task := agents.Task{
		ID:    uuid.New(),
		Type:  "quality",
		Input: `
package main

import "fmt"

func calculateSum(numbers []int) int {
    sum := 0
    for _, num := range numbers {
        sum += num
    }
    return sum
}

func main() {
    nums := []int{1, 2, 3, 4, 5}
    result := calculateSum(nums)
    fmt.Printf("Sum: %d\n", result)
}
`,
		Parameters: map[string]interface{}{
			"language": "go",
			"purpose":  "testing",
		},
	}

	result, err := qualityAgent.Execute(ctx, task)
	if err != nil {
		log.Printf("Quality agent execution failed: %v", err)
		return
	}

	fmt.Printf("✅ Quality Agent execution successful!\n")
	fmt.Printf("   Confidence: %.1f\n", result.Confidence)
	fmt.Printf("   Success: %v\n", result.Success)
	
	// Parse and display metrics if available
	if result.Metadata != nil {
		if metricsJSON, ok := result.Metadata["metrics"].(string); ok {
			var metrics quality.Metrics
			if err := json.Unmarshal([]byte(metricsJSON), &metrics); err == nil {
				fmt.Printf("   Metrics:\n")
				fmt.Printf("     - Issues Found: %d\n", metrics.IssuesFound)
				fmt.Printf("     - Tests Generated: %d\n", metrics.TestsGenerated)
				fmt.Printf("     - Coverage: %.1f%%\n", metrics.CoveragePercent)
			}
		}
	}
}

func testToolRegistry(registry *agents.Registry) {
	// Test tool registration and retrieval
	tools := registry.GetToolsForAgent(agents.AgentTypeQuality)
	
	fmt.Printf("Tools registered for QualityAgent: %d\n", len(tools))
	for _, toolName := range tools {
		tool, err := registry.GetTool(toolName)
		if err != nil {
			log.Printf("Failed to get tool %s: %v", toolName, err)
			continue
		}
		fmt.Printf("  - %s: %s\n", tool.GetName(), tool.GetDescription())
	}

	// Test executing a tool
	ctx := context.Background()
	tool, err := registry.GetTool("quality_confidence_estimator")
	if err != nil {
		log.Printf("Failed to get confidence estimator tool: %v", err)
		return
	}

	params := map[string]interface{}{
		"total_files":     "10",
		"total_lines":     "1000",
		"issues_found":    "2",
		"tests_generated": "5",
		"tests_passed":    "5",
		"coverage":        "85.5",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		log.Printf("Tool execution failed: %v", err)
		return
	}

	fmt.Printf("✅ Tool execution successful: %v\n", result)
}

func testEndToEndStrategic(groqClient *groq.Client, router *llm.Router, registry *agents.Registry) {
	ctx := context.Background()

	// Initialize orchestrator with strategic reasoning
	orchestrator := agents.NewOrchestrator(groqClient, router, registry, nil)

	// Create a complex task that will benefit from strategic reasoning
	task := agents.Task{
		ID:    uuid.New(),
		Type:  "development",
		Input: "Create a fault-tolerant message queue system with retry logic",
		Parameters: map[string]interface{}{
			"language":                "go",
			"enable_strategic_reasoning": true,
			"min_candidates":          3,
		},
		Context: &agents.TaskContext{
			UserID:      uuid.New(),
			TenantID:    uuid.New(),
			WorkspaceID: uuid.New(),
			Phase:       "build",
		},
	}

	// Execute through orchestrator
	result, err := orchestrator.Execute(ctx, task)
	if err != nil {
		log.Printf("Orchestrator execution failed: %v", err)
		return
	}

	fmt.Printf("✅ End-to-end strategic execution successful!\n")
	fmt.Printf("   Success: %v\n", result.Success)
	fmt.Printf("   Confidence: %.1f\n", result.Confidence)
	fmt.Printf("   Execution Time: %dms\n", result.ExecutionMS)
	
	if result.NextAgent != "" {
		fmt.Printf("   Suggested Next Agent: %s\n", result.NextAgent)
	}
}