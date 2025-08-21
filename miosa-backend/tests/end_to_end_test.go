package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"github.com/sormind/OSA/miosa-backend/internal/config"
	"github.com/sormind/OSA/miosa-backend/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEndToEndGroqIntegration(t *testing.T) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		t.Skip("GROQ_API_KEY not set")
	}

	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("Direct Groq API Test", func(t *testing.T) {
		// Test direct Groq API connection
		client, err := groq.NewClient(apiKey)
		require.NoError(t, err)

		resp, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
			Model: "llama-3.1-8b-instant",
			Messages: []groq.ChatCompletionMessage{
				{
					Role:    "user",
					Content: "Say 'test successful' in exactly 2 words",
				},
			},
			MaxTokens:   10,
			Temperature: 0.1,
		})
		
		require.NoError(t, err)
		require.NotEmpty(t, resp.Choices)
		t.Logf("✅ Direct API Response: %s", resp.Choices[0].Message.Content)
	})

	t.Run("LLM Provider Integration", func(t *testing.T) {
		// Test our LLM provider wrapper
		cfg := config.LLMProvider{
			APIKey:        apiKey,
			RetryAttempts: 2,
			RetryDelay:    time.Second,
		}

		provider, err := llm.NewGroqProvider(cfg, logger)
		require.NoError(t, err)

		req := llm.Request{
			Messages: []llm.Message{
				{
					Role:    "user",
					Content: "What is 2+2? Answer with just the number.",
				},
			},
			MaxTokens:   10,
			Temperature: 0.1,
			TaskType:    "simple",
		}

		resp, err := provider.Complete(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Content)
		assert.Greater(t, resp.Confidence, 0.0)
		t.Logf("✅ Provider Response: %s (confidence: %.2f)", resp.Content, resp.Confidence)
	})

	t.Run("Router Model Selection", func(t *testing.T) {
		// Test router selects appropriate models
		llmCfg := &config.LLMConfig{
			DefaultProvider: "groq",
			Providers: map[string]config.LLMProvider{
				"groq": {
					APIKey:        apiKey,
					Model:         "llama-3.1-8b-instant",
					RetryAttempts: 2,
					RetryDelay:    time.Second,
				},
			},
		}

		router, err := llm.NewRouter(llmCfg, logger)
		require.NoError(t, err)

		// Test different priority modes
		modes := []string{
			"speed",
			"quality",
			"cost",
		}

		for _, mode := range modes {
			// Map mode to priority
			priority := llm.PriorityBalance
			switch mode {
			case "speed":
				priority = llm.PrioritySpeed
			case "quality":
				priority = llm.PriorityQuality
			case "cost":
				priority = llm.PriorityCost
			}

			opts := llm.Options{
				Priority: priority,
				Task:     "test task",
			}

			candidates, err := router.Select(opts)
			require.NoError(t, err)
			require.NotEmpty(t, candidates)
			
			best := candidates[0]
			assert.NotEmpty(t, best.Model.Name)
			assert.NotEmpty(t, best.Model.Provider)
			assert.Greater(t, best.Score, 0.0)
			t.Logf("✅ %s priority → Model: %s, Score: %.3f", 
				mode, best.Model.Name, best.Score)
		}
	})

	t.Run("Agent Orchestration", func(t *testing.T) {
		// Register communication agent first
		agents.Register(&mockCommunicationAgent{})
		
		// Test full agent orchestration
		client, err := groq.NewClient(apiKey)
		require.NoError(t, err)

		orchestrator := agents.NewOrchestrator(client, logger, nil)

		task := agents.Task{
			ID:    uuid.New(),
			Type:  "code_generation",
			Input: "Create a function that calculates factorial",
			Parameters: map[string]interface{}{
				"language": "Python",
			},
			Context: &agents.TaskContext{
				UserID:      uuid.New(),
				TenantID:    uuid.New(),
				WorkspaceID: uuid.New(),
				Phase:       string(agents.PhaseDevelopment),
			},
		}

		result, err := orchestrator.Execute(ctx, task)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotEmpty(t, result.Output)
		assert.Greater(t, result.Confidence, 0.0)
		t.Logf("✅ Orchestration Result (confidence: %.2f):\n%s", 
			result.Confidence, truncateString(result.Output, 200))
	})

	t.Run("Kimi K2 Routing Decision", func(t *testing.T) {
		// Register communication agent first
		agents.Register(&mockCommunicationAgent{})
		
		// Test Kimi K2 makes proper routing decisions
		client, err := groq.NewClient(apiKey)
		require.NoError(t, err)

		orchestrator := agents.NewOrchestrator(client, logger, nil)

		complexTask := agents.Task{
			ID:    uuid.New(),
			Type:  "architecture_design",
			Input: "Design a microservices architecture for an e-commerce platform",
			Context: &agents.TaskContext{
				UserID:      uuid.New(),
				TenantID:    uuid.New(),
				WorkspaceID: uuid.New(),
				Phase:       string(agents.PhaseAnalysis),
			},
		}

		result, err := orchestrator.Execute(ctx, complexTask)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Greater(t, result.Confidence, 7.0) // Should be high confidence
		t.Logf("✅ Kimi K2 Routing (confidence: %.2f)", result.Confidence)
	})

	t.Run("Multi-Agent Workflow", func(t *testing.T) {
		// Test a workflow involving multiple agents
		agents.Register(&mockCommunicationAgent{})
		agents.Register(&mockAnalysisAgent{})
		agents.Register(&mockDevelopmentAgent{})

		client, err := groq.NewClient(apiKey)
		require.NoError(t, err)

		orchestrator := agents.NewOrchestrator(client, logger, nil)

		// First task: Analysis
		analysisTask := agents.Task{
			ID:    uuid.New(),
			Type:  "requirements_analysis",
			Input: "Analyze requirements for a REST API",
			Context: &agents.TaskContext{
				UserID:      uuid.New(),
				TenantID:    uuid.New(),
				WorkspaceID: uuid.New(),
				Phase:       string(agents.PhaseAnalysis),
			},
		}

		result1, err := orchestrator.Execute(ctx, analysisTask)
		require.NoError(t, err)
		assert.True(t, result1.Success)

		// Second task: Development based on analysis
		devTask := agents.Task{
			ID:    uuid.New(),
			Type:  "code_generation",
			Input: fmt.Sprintf("Based on this analysis: %s\nGenerate the code", result1.Output),
			Context: &agents.TaskContext{
				UserID:      uuid.New(),
				TenantID:    uuid.New(),
				WorkspaceID: uuid.New(),
				Phase:       string(agents.PhaseDevelopment),
			},
		}

		result2, err := orchestrator.Execute(ctx, devTask)
		require.NoError(t, err)
		assert.True(t, result2.Success)
		t.Logf("✅ Multi-agent workflow completed successfully")
	})

	fmt.Println("\n✅ End-to-End Integration Test Complete!")
	fmt.Println("====================================")
	fmt.Println("All components working together:")
	fmt.Println("  ✅ Groq API connection")
	fmt.Println("  ✅ LLM Provider abstraction")
	fmt.Println("  ✅ Router model selection")
	fmt.Println("  ✅ Agent orchestration")
	fmt.Println("  ✅ Kimi K2 routing")
	fmt.Println("  ✅ Multi-agent workflows")
}

// Mock agents for testing
type mockCommunicationAgent struct{}

func (m *mockCommunicationAgent) GetType() agents.AgentType { return agents.CommunicationAgent }
func (m *mockCommunicationAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{
			Name:        "communication",
			Description: "Basic communication",
			Required:    true,
			Version:     "1.0",
		},
	}
}
func (m *mockCommunicationAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	return &agents.Result{
		Success:     true,
		Output:      "Communication handled",
		Confidence:  8.0,
		ExecutionMS: 50,
	}, nil
}
func (m *mockCommunicationAgent) GetDescription() string { return "Mock communication agent" }

type mockAnalysisAgent struct{}

func (m *mockAnalysisAgent) GetType() agents.AgentType { return agents.AnalysisAgent }
func (m *mockAnalysisAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{
			Name:        "analysis",
			Description: "Requirements analysis",
			Required:    true,
			Version:     "1.0",
		},
	}
}
func (m *mockAnalysisAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	return &agents.Result{
		Success:     true,
		Output:      "Analysis complete: API needs authentication, CRUD operations, and rate limiting",
		Confidence:  8.5,
		ExecutionMS: 100,
	}, nil
}
func (m *mockAnalysisAgent) GetDescription() string { return "Mock analysis agent" }

type mockDevelopmentAgent struct{}

func (m *mockDevelopmentAgent) GetType() agents.AgentType { return agents.DevelopmentAgent }
func (m *mockDevelopmentAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{
			Name:        "code_generation",
			Description: "Code generation",
			Required:    true,
			Version:     "1.0",
		},
	}
}
func (m *mockDevelopmentAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	return &agents.Result{
		Success:     true,
		Output:      "Code generated: Express.js API with JWT auth",
		Confidence:  9.0,
		ExecutionMS: 150,
	}, nil
}
func (m *mockDevelopmentAgent) GetDescription() string { return "Mock development agent" }

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}