package recommender

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
)

// RecommenderAgent uses iterative testing to improve tools and patterns
// It acts as a meta-agent that optimizes other agents and tools
type RecommenderAgent struct {
	id             uuid.UUID
	groqClient     *groq.Client
	redisClient    *redis.Client
	logger         *zap.Logger
	testResults    map[string]*ToolTestMetrics
	improvements   map[string]*Improvement
	semanticCache  *SemanticCache
	mu             sync.RWMutex
}

// ToolTestMetrics tracks tool performance over time
type ToolTestMetrics struct {
	ToolID          string
	TotalExecutions int
	SuccessRate     float64
	AvgLatency      time.Duration
	ErrorPatterns   map[string]int
	LastOptimized   time.Time
	Improvements    []string
}

// Improvement represents a refinement suggestion
type Improvement struct {
	ID               uuid.UUID
	TargetType       string // "tool", "agent", "pattern"
	TargetID         string
	OriginalSpec     map[string]interface{}
	ImprovedSpec     map[string]interface{}
	PerformanceGain  float64
	TestIterations   int
	AppliedAt        *time.Time
	ValidationStatus string
}

// SemanticCache for pattern transfer learning
type SemanticCache struct {
	embeddings map[string][]float64
	patterns   map[string]*agents.WorkflowPattern
	mu         sync.RWMutex
}

func New(groqClient *groq.Client) *RecommenderAgent {
	return &RecommenderAgent{
		id:           uuid.New(),
		groqClient:   groqClient,
		testResults:  make(map[string]*ToolTestMetrics),
		improvements: make(map[string]*Improvement),
		semanticCache: &SemanticCache{
			embeddings: make(map[string][]float64),
			patterns:   make(map[string]*agents.WorkflowPattern),
		},
	}
}

// SetRedis adds Redis client for caching
func (a *RecommenderAgent) SetRedis(client *redis.Client) {
	a.redisClient = client
}

// SetLogger adds logger
func (a *RecommenderAgent) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

func (a *RecommenderAgent) GetID() uuid.UUID {
	return a.id
}

func (a *RecommenderAgent) GetType() agents.AgentType {
	return agents.RecommenderAgent
}

func (a *RecommenderAgent) GetCapabilities() []agents.Capability {
	return []agents.Capability{
		{
			Name:        "tool_optimization",
			Description: "Iteratively test and refine tool definitions",
			Required:    false,
		},
		{
			Name:        "pattern_transfer",
			Description: "Use semantic search to adapt successful patterns",
			Required:    false,
		},
		{
			Name:        "agent_optimization",
			Description: "Optimize agent configurations and prompts",
			Required:    false,
		},
		{
			Name:        "performance_analysis",
			Description: "Analyze and improve system performance",
			Required:    false,
		},
	}
}

func (a *RecommenderAgent) GetDescription() string {
	return "Meta-agent that optimizes tools, patterns, and other agents through iterative testing and refinement"
}

func (a *RecommenderAgent) Execute(ctx context.Context, task agents.Task) (*agents.Result, error) {
	startTime := time.Now()
	
	// Determine recommendation type from task
	recommendationType := a.parseRecommendationType(task.Input)
	
	var output string
	var improvements []map[string]interface{}
	confidence := 7.0
	
	switch recommendationType {
	case "tool_optimization":
		// Run automated tool testing and refinement
		toolImprovements := a.optimizeTools(ctx, task)
		output = a.formatToolOptimizationReport(toolImprovements)
		improvements = a.convertToMaps(toolImprovements)
		confidence = a.calculateToolConfidence(toolImprovements)
		
	case "pattern_transfer":
		// Use semantic search to find and adapt patterns
		adaptedPatterns := a.transferPatterns(ctx, task)
		output = a.formatPatternTransferReport(adaptedPatterns)
		improvements = a.patternsToMaps(adaptedPatterns)
		confidence = 8.5
		
	case "agent_optimization":
		// Optimize agent configurations and prompts
		agentImprovements := a.optimizeAgents(ctx, task)
		output = a.formatAgentOptimizationReport(agentImprovements)
		improvements = a.convertToMaps(agentImprovements)
		confidence = 8.0
		
	default:
		// General recommendation using Kimi K2
		output = a.generateGeneralRecommendation(ctx, task)
		confidence = 7.5
	}
	
	// Cache the result with semantic key
	if a.redisClient != nil {
		a.cacheRecommendation(ctx, task, output, improvements)
	}
	
	return &agents.Result{
		Success:     true,
		Output:      output,
		Confidence:  confidence,
		ExecutionMS: time.Since(startTime).Milliseconds(),
		Data: map[string]interface{}{
			"recommendation_type": recommendationType,
			"improvements":        improvements,
			"cached":             a.redisClient != nil,
		},
	}, nil
}

// optimizeTools runs iterative testing to improve tool definitions
func (a *RecommenderAgent) optimizeTools(ctx context.Context, task agents.Task) []*Improvement {
	improvements := []*Improvement{}
	
	// Get tools to optimize from task context
	toolsToTest := a.extractToolsFromTask(task)
	
	for _, toolID := range toolsToTest {
		if a.logger != nil {
			a.logger.Info("Optimizing tool", zap.String("tool", toolID))
		}
		
		// Run iterative refinement
		improvement := a.runToolRefinement(ctx, toolID)
		if improvement != nil && improvement.PerformanceGain > 10 {
			improvements = append(improvements, improvement)
			
			// Apply improvement if gain is significant
			if improvement.PerformanceGain > 40 {
				a.applyImprovement(ctx, improvement)
			}
		}
	}
	
	return improvements
}

// runToolRefinement performs iterative testing and refinement
func (a *RecommenderAgent) runToolRefinement(ctx context.Context, toolID string) *Improvement {
	const maxIterations = 10
	improvement := &Improvement{
		ID:         uuid.New(),
		TargetType: "tool",
		TargetID:   toolID,
		OriginalSpec: map[string]interface{}{
			"tool_id": toolID,
		},
	}
	
	baselinePerf := a.getBaselinePerformance(toolID)
	currentPerf := baselinePerf
	
	for i := 0; i < maxIterations; i++ {
		// Generate test scenarios using code execution
		scenarios := a.generateTestScenarios(ctx, toolID, i)
		
		// Execute tests
		results := a.executeToolTests(ctx, toolID, scenarios)
		
		// Analyze results
		analysis := a.analyzeTestResults(results)
		
		// Calculate new performance
		newPerf := a.calculatePerformance(results)
		
		// Check if we've improved enough
		gain := ((newPerf - baselinePerf) / baselinePerf) * 100
		if gain > 40 {
			improvement.PerformanceGain = gain
			improvement.TestIterations = i + 1
			break
		}
		
		// Generate refinements using Kimi K2
		refinements := a.generateRefinements(ctx, toolID, analysis)
		improvement.ImprovedSpec = refinements
		
		currentPerf = newPerf
		improvement.TestIterations = i + 1
	}
	
	improvement.PerformanceGain = ((currentPerf - baselinePerf) / baselinePerf) * 100
	return improvement
}

// generateTestScenarios creates diverse test cases
func (a *RecommenderAgent) generateTestScenarios(ctx context.Context, toolID string, iteration int) []map[string]interface{} {
	// Use Kimi K2 with code execution to generate test scenarios
	prompt := fmt.Sprintf(`Generate %d test scenarios for tool '%s'.
Iteration %d - be creative with edge cases.
Return as JSON array with: input, expected_output, constraints`, 
		5+iteration, toolID, iteration)
	
	response, _ := a.callKimiK2WithTools(ctx, prompt)
	
	var scenarios []map[string]interface{}
	json.Unmarshal([]byte(response), &scenarios)
	return scenarios
}

// executeToolTests runs tests using code execution
func (a *RecommenderAgent) executeToolTests(ctx context.Context, toolID string, scenarios []map[string]interface{}) []map[string]interface{} {
	results := []map[string]interface{}{}
	
	for _, scenario := range scenarios {
		// Build Python code for testing
		code := fmt.Sprintf(`
import json
import time

def test_tool_%s(input_data):
    start = time.time()
    # Simulate tool execution
    result = {"success": True, "output": "test_output"}
    latency = time.time() - start
    return {"result": result, "latency": latency}

input_data = %s
output = test_tool_%s(input_data)
print(json.dumps(output))
`, toolID, a.toJSON(scenario["input"]), toolID)
		
		// Execute using compound-beta-mini for code execution
		execResult := a.executeCode(ctx, code)
		results = append(results, map[string]interface{}{
			"scenario": scenario,
			"result":   execResult,
		})
	}
	
	return results
}

// transferPatterns uses semantic search to find and adapt patterns
func (a *RecommenderAgent) transferPatterns(ctx context.Context, task agents.Task) []*agents.WorkflowPattern {
	// Extract task embedding
	taskEmbedding := a.generateEmbedding(ctx, task.Input)
	
	// Find similar patterns using semantic search
	similarPatterns := a.semanticSearch(taskEmbedding, 5)
	
	// Adapt patterns to current task using strategy agent logic
	adaptedPatterns := []*agents.WorkflowPattern{}
	for _, pattern := range similarPatterns {
		adapted := a.adaptPattern(ctx, pattern, task)
		adaptedPatterns = append(adaptedPatterns, adapted)
	}
	
	return adaptedPatterns
}

// semanticSearch finds similar patterns using vector similarity
func (a *RecommenderAgent) semanticSearch(embedding []float64, k int) []*agents.WorkflowPattern {
	a.semanticCache.mu.RLock()
	defer a.semanticCache.mu.RUnlock()
	
	type scoredPattern struct {
		pattern    *agents.WorkflowPattern
		similarity float64
	}
	
	scored := []scoredPattern{}
	for id, patternEmb := range a.semanticCache.embeddings {
		similarity := a.cosineSimilarity(embedding, patternEmb)
		if pattern, exists := a.semanticCache.patterns[id]; exists {
			scored = append(scored, scoredPattern{
				pattern:    pattern,
				similarity: similarity,
			})
		}
	}
	
	// Sort by similarity and return top k
	// Simplified: just return first k patterns
	results := []*agents.WorkflowPattern{}
	for i := 0; i < k && i < len(scored); i++ {
		results = append(results, scored[i].pattern)
	}
	
	return results
}

// cacheRecommendation stores results in Redis with semantic keys
func (a *RecommenderAgent) cacheRecommendation(ctx context.Context, task agents.Task, output string, improvements []map[string]interface{}) {
	// Generate cache key using task embedding
	embedding := a.generateEmbedding(ctx, task.Input)
	cacheKey := fmt.Sprintf("cache:recommendation:%s", a.embeddingToKey(embedding))
	
	cacheData := map[string]interface{}{
		"output":       output,
		"improvements": improvements,
		"timestamp":    time.Now().Unix(),
		"task_type":    task.Type,
	}
	
	data, _ := json.Marshal(cacheData)
	a.redisClient.Set(ctx, cacheKey, data, 1*time.Hour)
	
	// Also store in semantic cache for pattern learning
	a.semanticCache.mu.Lock()
	a.semanticCache.embeddings[cacheKey] = embedding
	a.semanticCache.mu.Unlock()
}

// Helper methods

func (a *RecommenderAgent) callKimiK2WithTools(ctx context.Context, prompt string) (string, error) {
	// Note: groq-go library may not have direct tool support yet
	// For now, we'll use standard chat completion
	
	response, err := a.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: "moonshotai/kimi-k2-instruct",
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are an expert at optimizing tools and generating test scenarios.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
	})
	
	if err != nil {
		return "", err
	}
	
	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}
	
	return "", fmt.Errorf("no response from Kimi K2")
}

func (a *RecommenderAgent) executeCode(ctx context.Context, code string) map[string]interface{} {
	// Use compound-beta-mini for code execution
	response, err := a.groqClient.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model: "compound-beta-mini",
		Messages: []groq.ChatCompletionMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("Execute this Python code:\n```python\n%s\n```", code),
			},
		},
	})
	
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	
	if len(response.Choices) > 0 {
		var result map[string]interface{}
		json.Unmarshal([]byte(response.Choices[0].Message.Content), &result)
		return result
	}
	
	return map[string]interface{}{"error": "no response"}
}

func (a *RecommenderAgent) generateEmbedding(ctx context.Context, text string) []float64 {
	// Simplified: generate mock embedding
	// In production, use a real embedding model
	embedding := make([]float64, 768)
	for i := range embedding {
		embedding[i] = float64(len(text)%100) / 100.0
	}
	return embedding
}

func (a *RecommenderAgent) cosineSimilarity(a1, a2 []float64) float64 {
	var dotProduct, normA, normB float64
	for i := range a1 {
		dotProduct += a1[i] * a2[i]
		normA += a1[i] * a1[i]
		normB += a2[i] * a2[i]
	}
	return dotProduct / (normA * normB)
}

func (a *RecommenderAgent) parseRecommendationType(input string) string {
	lower := strings.ToLower(input)
	switch {
	case strings.Contains(lower, "tool"):
		return "tool_optimization"
	case strings.Contains(lower, "pattern"):
		return "pattern_transfer"
	case strings.Contains(lower, "agent"):
		return "agent_optimization"
	default:
		return "general"
	}
}

func (a *RecommenderAgent) toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Additional helper methods would be implemented here...
func (a *RecommenderAgent) extractToolsFromTask(task agents.Task) []string {
	// Extract tool IDs from task
	return []string{"example_tool"}
}

func (a *RecommenderAgent) getBaselinePerformance(toolID string) float64 {
	if metrics, exists := a.testResults[toolID]; exists {
		return metrics.SuccessRate
	}
	return 0.5
}

func (a *RecommenderAgent) analyzeTestResults(results []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"total_tests":  len(results),
		"success_rate": 0.8,
	}
}

func (a *RecommenderAgent) calculatePerformance(results []map[string]interface{}) float64 {
	return 0.75
}

func (a *RecommenderAgent) generateRefinements(ctx context.Context, toolID string, analysis map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"improved_description": "Optimized tool description",
		"schema_changes":       map[string]interface{}{},
	}
}

func (a *RecommenderAgent) applyImprovement(ctx context.Context, improvement *Improvement) {
	now := time.Now()
	improvement.AppliedAt = &now
	improvement.ValidationStatus = "applied"
}

func (a *RecommenderAgent) optimizeAgents(ctx context.Context, task agents.Task) []*Improvement {
	return []*Improvement{}
}

func (a *RecommenderAgent) adaptPattern(ctx context.Context, pattern *agents.WorkflowPattern, task agents.Task) *agents.WorkflowPattern {
	return pattern
}

func (a *RecommenderAgent) formatToolOptimizationReport(improvements []*Improvement) string {
	return fmt.Sprintf("Tool Optimization Report: %d improvements found", len(improvements))
}

func (a *RecommenderAgent) formatPatternTransferReport(patterns []*agents.WorkflowPattern) string {
	return fmt.Sprintf("Pattern Transfer Report: %d patterns adapted", len(patterns))
}

func (a *RecommenderAgent) formatAgentOptimizationReport(improvements []*Improvement) string {
	return fmt.Sprintf("Agent Optimization Report: %d improvements found", len(improvements))
}

func (a *RecommenderAgent) generateGeneralRecommendation(ctx context.Context, task agents.Task) string {
	return "General recommendation based on task analysis"
}

func (a *RecommenderAgent) convertToMaps(improvements []*Improvement) []map[string]interface{} {
	maps := []map[string]interface{}{}
	for _, imp := range improvements {
		maps = append(maps, map[string]interface{}{
			"id":               imp.ID,
			"target":           imp.TargetID,
			"performance_gain": imp.PerformanceGain,
		})
	}
	return maps
}

func (a *RecommenderAgent) patternsToMaps(patterns []*agents.WorkflowPattern) []map[string]interface{} {
	maps := []map[string]interface{}{}
	for _, p := range patterns {
		maps = append(maps, map[string]interface{}{
			"id":           p.ID,
			"success_rate": p.SuccessRate,
		})
	}
	return maps
}

func (a *RecommenderAgent) calculateToolConfidence(improvements []*Improvement) float64 {
	if len(improvements) == 0 {
		return 7.0
	}
	
	avgGain := 0.0
	for _, imp := range improvements {
		avgGain += imp.PerformanceGain
	}
	avgGain /= float64(len(improvements))
	
	confidence := 7.0
	if avgGain > 20 {
		confidence += 1.0
	}
	if avgGain > 40 {
		confidence += 1.0
	}
	
	return confidence
}

func (a *RecommenderAgent) embeddingToKey(embedding []float64) string {
	// Convert embedding to a cache key
	hash := 0.0
	for _, v := range embedding {
		hash += v
	}
	return fmt.Sprintf("%x", hash)
}