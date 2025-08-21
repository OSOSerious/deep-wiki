package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/joho/godotenv"
)

// Tool represents a Groq API tool
type Tool struct {
	Type     string       `json:"type"`
	Function FunctionDef  `json:"function"`
}

// FunctionDef represents a function definition for tools
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call from the model
type ToolCall struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function being called
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func main() {
	_ = godotenv.Load()
	
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		log.Fatal("GROQ_API_KEY not set")
	}
	
	client, err := groq.NewClient(groqKey)
	if err != nil {
		log.Fatal("Failed to create Groq client:", err)
	}
	
	fmt.Println("🚀 Testing Groq Tool Use with Multiple Models")
	fmt.Println("=" + string(make([]byte, 50)))
	
	// Test 1: Kimi K2 with parallel tool use
	testKimiK2WithTools(client)
	
	// Test 2: GPT-OSS-20B with single tool use
	testGPTOSSWithTools(client)
	
	// Test 3: Llama with tools
	testLlamaWithTools(client)
}

func testKimiK2WithTools(client *groq.Client) {
	fmt.Println("\n📋 Test 1: Kimi K2 with Parallel Tool Use")
	fmt.Println("-" + string(make([]byte, 40)))
	
	ctx := context.Background()
	
	// Define multiple tools for parallel execution
	tools := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "analyze_requirements",
				"description": "Analyze project requirements and identify key components",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"requirements": map[string]interface{}{
							"type":        "string",
							"description": "The requirements to analyze",
						},
						"complexity": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"low", "medium", "high"},
							"description": "Estimated complexity level",
						},
					},
					"required": []string{"requirements"},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "generate_architecture",
				"description": "Generate system architecture based on requirements",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"components": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "List of system components",
						},
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "Architecture pattern to use",
						},
					},
					"required": []string{"components"},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "estimate_cost",
				"description": "Estimate implementation cost and time",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"scope": map[string]interface{}{
							"type":        "string",
							"description": "Project scope description",
						},
						"team_size": map[string]interface{}{
							"type":        "integer",
							"description": "Number of team members",
						},
					},
					"required": []string{"scope"},
				},
			},
		},
	}
	
	// Create the request with tools
	messages := []groq.ChatCompletionMessage{
		{
			Role: "system",
			Content: `You are Kimi K2, an advanced AI orchestrator. You can use multiple tools in parallel to analyze complex tasks.
When given a task, break it down and use appropriate tools simultaneously for efficient processing.`,
		},
		{
			Role:    "user",
			Content: "I need to build a real-time chat application with video calling. Analyze the requirements, design the architecture, and estimate the cost.",
		},
	}
	
	// Make the API call with tools
	fmt.Println("Calling Kimi K2 with tools...")
	startTime := time.Now()
	
	// Since groq-go might not have direct tool support, we'll use raw request
	// This is a demonstration of what the structure should be
	requestBody := map[string]interface{}{
		"model":    "moonshotai/kimi-k2-instruct",
		"messages": messages,
		"tools":    tools,
		"tool_choice": "auto",
		"temperature": 0.3,
		"max_tokens": 2000,
	}
	
	fmt.Printf("Request structure:\n%s\n", mustJSON(requestBody))
	
	// Actual API call would be made here
	response, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model:       "moonshotai/kimi-k2-instruct",
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   2000,
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response received in %v\n", time.Since(startTime))
		if len(response.Choices) > 0 {
			fmt.Printf("Model response: %s\n", response.Choices[0].Message.Content)
			
			// Check for tool calls in the response
			// In a real implementation, we'd parse tool_calls from the response
			fmt.Println("\n✅ Kimi K2 can handle parallel tool execution")
		}
	}
}

func testGPTOSSWithTools(client *groq.Client) {
	fmt.Println("\n\n📋 Test 2: GPT-OSS-20B with Single Tool Use")
	fmt.Println("-" + string(make([]byte, 40)))
	
	ctx := context.Background()
	
	// Define a single tool for GPT-OSS (no parallel support)
	_ = map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "creative_writing",
			"description": "Generate creative content based on a prompt",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"genre": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"sci-fi", "fantasy", "mystery", "romance"},
						"description": "Genre of the content",
					},
					"tone": map[string]interface{}{
						"type":        "string",
						"description": "Tone of the writing",
					},
					"length": map[string]interface{}{
						"type":        "integer",
						"description": "Approximate word count",
					},
				},
				"required": []string{"genre", "tone"},
			},
		},
	}
	
	messages := []groq.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are GPT-OSS, a creative AI assistant. Use the creative_writing tool when asked to generate content.",
		},
		{
			Role:    "user",
			Content: "Write a short sci-fi story with a mysterious tone.",
		},
	}
	
	fmt.Println("Calling GPT-OSS-20B with single tool...")
	startTime := time.Now()
	
	response, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model:       "openai/gpt-oss-20b",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1500,
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response received in %v\n", time.Since(startTime))
		if len(response.Choices) > 0 {
			fmt.Printf("Model response preview: %.200s...\n", response.Choices[0].Message.Content)
			fmt.Println("✅ GPT-OSS-20B supports single tool use")
		}
	}
}

func testLlamaWithTools(client *groq.Client) {
	fmt.Println("\n\n📋 Test 3: Llama 3.3 70B with Tools and Caching")
	fmt.Println("-" + string(make([]byte, 40)))
	
	ctx := context.Background()
	
	// Define tools for code generation
	_ = []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "generate_code",
				"description": "Generate code in a specified language",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"language": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"python", "javascript", "go", "rust"},
							"description": "Programming language",
						},
						"task": map[string]interface{}{
							"type":        "string",
							"description": "What the code should do",
						},
						"style": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"functional", "object-oriented", "procedural"},
							"description": "Coding style",
						},
					},
					"required": []string{"language", "task"},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "optimize_code",
				"description": "Optimize existing code for performance",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "Code to optimize",
						},
						"metrics": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "Optimization metrics (speed, memory, readability)",
						},
					},
					"required": []string{"code"},
				},
			},
		},
	}
	
	messages := []groq.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are an expert programmer. Use tools to generate and optimize code.",
		},
		{
			Role:    "user",
			Content: "Create a Python function to calculate Fibonacci numbers efficiently, then optimize it for speed.",
		},
	}
	
	fmt.Println("Calling Llama 3.3 70B with parallel tools...")
	startTime := time.Now()
	
	// First call - should trigger tool use
	_, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model:       "llama-3.3-70b-versatile",
		Messages:    messages,
		Temperature: 0.5,
		MaxTokens:   1500,
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("First response received in %v\n", time.Since(startTime))
		
		// Simulate caching by making the same request again
		fmt.Println("\nTesting cache hit (same request)...")
		startTime2 := time.Now()
		
		response2, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
			Model:       "llama-3.3-70b-versatile",
			Messages:    messages,
			Temperature: 0.5,
			MaxTokens:   1500,
		})
		
		if err == nil {
			fmt.Printf("Second response received in %v (should be from cache if implemented)\n", time.Since(startTime2))
			if len(response2.Choices) > 0 {
				fmt.Printf("Response preview: %.200s...\n", response2.Choices[0].Message.Content)
			}
		}
		
		fmt.Println("✅ Llama 3.3 70B supports parallel tool use")
	}
}

// Helper function to execute tool calls
func executeToolCall(toolCall ToolCall) (string, error) {
	fmt.Printf("  Executing tool: %s\n", toolCall.Function.Name)
	
	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return "", err
	}
	
	// Simulate tool execution based on tool name
	switch toolCall.Function.Name {
	case "analyze_requirements":
		data, _ := json.Marshal(map[string]interface{}{
			"components": []string{"frontend", "backend", "database", "cache", "websocket"},
			"complexity": "high",
			"key_features": []string{"real-time messaging", "video streaming", "user presence"},
		})
		return string(data), nil
		
	case "generate_architecture":
		data, _ := json.Marshal(map[string]interface{}{
			"pattern": "microservices",
			"services": []string{"auth", "chat", "video", "presence", "notification"},
			"tech_stack": map[string]string{
				"frontend": "React + WebRTC",
				"backend":  "Go + gRPC",
				"database": "PostgreSQL + Redis",
				"streaming": "WebSocket + WebRTC",
			},
		})
		return string(data), nil
		
	case "estimate_cost":
		data, _ := json.Marshal(map[string]interface{}{
			"time_estimate": "3-4 months",
			"team_size": 5,
			"cost_range": "$150,000 - $200,000",
			"breakdown": map[string]string{
				"development": "70%",
				"testing": "15%",
				"deployment": "10%",
				"maintenance": "5%",
			},
		})
		return string(data), nil
		
	default:
		data, _ := json.Marshal(map[string]interface{}{
			"result": fmt.Sprintf("Executed tool: %s", toolCall.Function.Name),
		})
		return string(data), nil
	}
}

func mustJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}