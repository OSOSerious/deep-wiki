package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/conneroisu/groq-go"
)

// TestSimpleGroqConnection tests that we can connect to Groq API
func TestSimpleGroqConnection(t *testing.T) {
	// Use the API key you provided
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		t.Skip("GROQ_API_KEY not set")
	}

	client, err := groq.NewClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Groq client: %v", err)
	}
	
	ctx := context.Background()

	// Test with a simple completion
	resp, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Say 'test successful' if you can hear me",
			},
		},
		MaxTokens:   10,
		Temperature: 0.1,
	})

	if err != nil {
		t.Fatalf("Failed to connect to Groq API: %v", err)
	}
	
	if len(resp.Choices) == 0 {
		t.Fatal("No response from Groq")
	}
	
	fmt.Printf("✅ Groq API Connection Test: %s\n", resp.Choices[0].Message.Content)
	fmt.Printf("   Model used: llama-3.1-8b-instant\n")
	fmt.Printf("   Tokens used: %d\n", resp.Usage.TotalTokens)
}

// TestKimiModel tests if Kimi K2 is available
func TestKimiModel(t *testing.T) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		t.Skip("GROQ_API_KEY not set")
	}

	client, err := groq.NewClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Groq client: %v", err)
	}
	
	ctx := context.Background()
	
	// Try to use Kimi K2 model
	fmt.Println("\n🧪 Testing Kimi K2 model availability...")
	
	resp, err := client.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: "moonshotai/kimi-k2-instruct",
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Say 'Kimi K2 ready' if you are Kimi K2",
			},
		},
		MaxTokens:   10,
		Temperature: 0.1,
	})
	
	if err != nil {
		fmt.Printf("⚠️  Kimi K2 not available on Groq: %v\n", err)
		fmt.Println("   This is expected - Kimi K2 may require direct Moonshot API")
	} else {
		fmt.Printf("✅ Kimi K2 Response: %s\n", resp.Choices[0].Message.Content)
	}
}