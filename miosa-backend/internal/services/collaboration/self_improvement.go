package collaboration

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
)

// SelfImprovementEngine analyzes agent collaboration patterns and improves them
type SelfImprovementEngine struct {
	redisClient       *redis.Client
	logger            *zap.Logger
	patterns          map[string]*CollaborationPattern
	learningRate      float64
	confidenceDecay   float64
	improvementBuffer []*ImprovementSuggestion
	mu                sync.RWMutex
}

// CollaborationPattern represents a learned pattern of agent collaboration
type CollaborationPattern struct {
	ID               uuid.UUID                      `json:"id"`
	Name             string                         `json:"name"`
	TaskType         string                         `json:"task_type"`
	AgentSequence    []agents.AgentType             `json:"agent_sequence"`
	SuccessRate      float64                        `json:"success_rate"`
	AverageTime      time.Duration                  `json:"average_time"`
	ConfidenceScore  float64                        `json:"confidence_score"`
	UsageCount       int64                          `json:"usage_count"`
	LastUpdated      time.Time                      `json:"last_updated"`
	ContextFeatures  map[string]interface{}         `json:"context_features"`
	Rewards          []float64                      `json:"rewards"`
	QValue           float64                        `json:"q_value"` // For reinforcement learning
}

// ImprovementSuggestion represents a suggested improvement to collaboration
type ImprovementSuggestion struct {
	ID              uuid.UUID              `json:"id"`
	PatternID       uuid.UUID              `json:"pattern_id"`
	Type            ImprovementType        `json:"type"`
	Description     string                 `json:"description"`
	ExpectedImpact  float64                `json:"expected_impact"`
	Confidence      float64                `json:"confidence"`
	Implementation  *ImplementationDetails `json:"implementation"`
	Status          SuggestionStatus       `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	AppliedAt       *time.Time             `json:"applied_at,omitempty"`
	Results         *ImprovementResults    `json:"results,omitempty"`
}

// ImprovementType categorizes different types of improvements
type ImprovementType string

const (
	ImprovementTypeAgentSwap      ImprovementType = "agent_swap"
	ImprovementTypeParallelization ImprovementType = "parallelization"
	ImprovementTypeContextEnrich   ImprovementType = "context_enrichment"
	ImprovementTypeSkipStep        ImprovementType = "skip_step"
	ImprovementTypeAddValidation   ImprovementType = "add_validation"
	ImprovementTypeCaching         ImprovementType = "caching"
)

// SuggestionStatus tracks the status of an improvement suggestion
type SuggestionStatus string

const (
	SuggestionStatusPending  SuggestionStatus = "pending"
	SuggestionStatusApproved SuggestionStatus = "approved"
	SuggestionStatusApplied  SuggestionStatus = "applied"
	SuggestionStatusRejected SuggestionStatus = "rejected"
)

// ImplementationDetails contains details for implementing an improvement
type ImplementationDetails struct {
	Code           string                 `json:"code"`
	Configuration  map[string]interface{} `json:"configuration"`
	Dependencies   []string               `json:"dependencies"`
	RollbackPlan   string                 `json:"rollback_plan"`
}

// ImprovementResults tracks the results of an applied improvement
type ImprovementResults struct {
	BeforeMetrics   PerformanceMetrics `json:"before_metrics"`
	AfterMetrics    PerformanceMetrics `json:"after_metrics"`
	ImprovementRate float64            `json:"improvement_rate"`
	Validated       bool               `json:"validated"`
}

// PerformanceMetrics captures performance data
type PerformanceMetrics struct {
	SuccessRate    float64       `json:"success_rate"`
	AverageTime    time.Duration `json:"average_time"`
	ConfidenceAvg  float64       `json:"confidence_avg"`
	ErrorRate      float64       `json:"error_rate"`
	ThroughputRate float64       `json:"throughput_rate"`
}

// NewSelfImprovementEngine creates a new self-improvement engine
func NewSelfImprovementEngine(redisClient *redis.Client, logger *zap.Logger) *SelfImprovementEngine {
	return &SelfImprovementEngine{
		redisClient:       redisClient,
		logger:            logger,
		patterns:          make(map[string]*CollaborationPattern),
		learningRate:      0.1,  // Alpha for Q-learning
		confidenceDecay:   0.95, // Decay factor for old patterns
		improvementBuffer: make([]*ImprovementSuggestion, 0),
	}
}

// AnalyzeCollaboration analyzes a completed collaboration for improvements
func (sie *SelfImprovementEngine) AnalyzeCollaboration(ctx context.Context, tasks []*CollaborativeTask) error {
	// Extract pattern from task sequence
	pattern := sie.extractPattern(tasks)
	
	// Update Q-value using reinforcement learning
	reward := sie.calculateReward(tasks)
	sie.updateQValue(pattern, reward)
	
	// Check if pattern needs improvement
	if pattern.ConfidenceScore < 7.0 || pattern.SuccessRate < 0.8 {
		suggestions := sie.generateImprovements(ctx, pattern, tasks)
		
		sie.mu.Lock()
		sie.improvementBuffer = append(sie.improvementBuffer, suggestions...)
		sie.mu.Unlock()
		
		// Auto-apply high-confidence improvements
		for _, suggestion := range suggestions {
			if suggestion.Confidence >= 9.0 && suggestion.ExpectedImpact >= 0.2 {
				sie.applyImprovement(ctx, suggestion)
			}
		}
	}
	
	// Store updated pattern
	sie.storePattern(ctx, pattern)
	
	return nil
}

// extractPattern extracts a collaboration pattern from task sequence
func (sie *SelfImprovementEngine) extractPattern(tasks []*CollaborativeTask) *CollaborationPattern {
	if len(tasks) == 0 {
		return nil
	}
	
	// Build agent sequence
	agentSequence := make([]agents.AgentType, 0)
	totalTime := time.Duration(0)
	successCount := 0
	totalConfidence := 0.0
	
	for _, task := range tasks {
		agentSequence = append(agentSequence, task.AssignedAgent)
		
		if task.Result != nil {
			totalTime += time.Duration(task.Result.ExecutionMS) * time.Millisecond
			if task.Status == TaskStatusCompleted {
				successCount++
			}
			totalConfidence += task.ConfidenceScore
		}
	}
	
	patternKey := sie.generatePatternKey(tasks[0].Type, agentSequence)
	
	sie.mu.RLock()
	existingPattern, exists := sie.patterns[patternKey]
	sie.mu.RUnlock()
	
	if exists {
		// Update existing pattern
		existingPattern.UsageCount++
		existingPattern.SuccessRate = (existingPattern.SuccessRate*float64(existingPattern.UsageCount-1) + 
			float64(successCount)/float64(len(tasks))) / float64(existingPattern.UsageCount)
		existingPattern.AverageTime = (existingPattern.AverageTime*time.Duration(existingPattern.UsageCount-1) + 
			totalTime) / time.Duration(existingPattern.UsageCount)
		existingPattern.ConfidenceScore = totalConfidence / float64(len(tasks))
		existingPattern.LastUpdated = time.Now()
		
		return existingPattern
	}
	
	// Create new pattern
	pattern := &CollaborationPattern{
		ID:              uuid.New(),
		Name:            fmt.Sprintf("Pattern_%s", patternKey),
		TaskType:        tasks[0].Type,
		AgentSequence:   agentSequence,
		SuccessRate:     float64(successCount) / float64(len(tasks)),
		AverageTime:     totalTime / time.Duration(len(tasks)),
		ConfidenceScore: totalConfidence / float64(len(tasks)),
		UsageCount:      1,
		LastUpdated:     time.Now(),
		ContextFeatures: sie.extractContextFeatures(tasks),
		Rewards:         []float64{},
		QValue:          0.5, // Initial Q-value
	}
	
	sie.mu.Lock()
	sie.patterns[patternKey] = pattern
	sie.mu.Unlock()
	
	return pattern
}

// calculateReward calculates the reward for a collaboration pattern
func (sie *SelfImprovementEngine) calculateReward(tasks []*CollaborativeTask) float64 {
	if len(tasks) == 0 {
		return 0
	}
	
	reward := 0.0
	
	for _, task := range tasks {
		// Success bonus
		if task.Status == TaskStatusCompleted {
			reward += 1.0
		} else if task.Status == TaskStatusFailed {
			reward -= 1.0
		}
		
		// Confidence bonus
		reward += (task.ConfidenceScore - 5.0) / 10.0
		
		// Time penalty (if task took too long)
		if task.Result != nil && task.Result.ExecutionMS > 10000 {
			reward -= 0.2
		}
		
		// Retry penalty
		reward -= float64(task.RetryCount) * 0.3
	}
	
	// Normalize by number of tasks
	return reward / float64(len(tasks))
}

// updateQValue updates the Q-value using Q-learning algorithm
func (sie *SelfImprovementEngine) updateQValue(pattern *CollaborationPattern, reward float64) {
	if pattern == nil {
		return
	}
	
	// Q-learning update: Q(s,a) = Q(s,a) + α[r + γ*max(Q(s',a')) - Q(s,a)]
	// Simplified version without next state (terminal state)
	oldQValue := pattern.QValue
	pattern.QValue = oldQValue + sie.learningRate*(reward-oldQValue)
	
	// Store reward history
	pattern.Rewards = append(pattern.Rewards, reward)
	if len(pattern.Rewards) > 100 {
		pattern.Rewards = pattern.Rewards[1:] // Keep last 100 rewards
	}
	
	// Update confidence based on Q-value stability
	if len(pattern.Rewards) >= 10 {
		variance := sie.calculateVariance(pattern.Rewards[len(pattern.Rewards)-10:])
		if variance < 0.1 {
			pattern.ConfidenceScore = math.Min(10.0, pattern.ConfidenceScore+0.5)
		}
	}
}

// generateImprovements generates improvement suggestions for a pattern
func (sie *SelfImprovementEngine) generateImprovements(ctx context.Context, pattern *CollaborationPattern, tasks []*CollaborativeTask) []*ImprovementSuggestion {
	suggestions := make([]*ImprovementSuggestion, 0)
	
	// Check for parallelization opportunities
	if len(pattern.AgentSequence) > 2 {
		parallelSuggestion := sie.checkParallelization(pattern, tasks)
		if parallelSuggestion != nil {
			suggestions = append(suggestions, parallelSuggestion)
		}
	}
	
	// Check for agent swapping opportunities
	if pattern.SuccessRate < 0.7 {
		swapSuggestion := sie.checkAgentSwap(pattern, tasks)
		if swapSuggestion != nil {
			suggestions = append(suggestions, swapSuggestion)
		}
	}
	
	// Check for context enrichment needs
	if pattern.ConfidenceScore < 7.0 {
		contextSuggestion := sie.checkContextEnrichment(pattern, tasks)
		if contextSuggestion != nil {
			suggestions = append(suggestions, contextSuggestion)
		}
	}
	
	// Check for caching opportunities
	if pattern.UsageCount > 10 && pattern.AverageTime > 5*time.Second {
		cacheSuggestion := sie.checkCaching(pattern, tasks)
		if cacheSuggestion != nil {
			suggestions = append(suggestions, cacheSuggestion)
		}
	}
	
	return suggestions
}

// checkParallelization checks if tasks can be parallelized
func (sie *SelfImprovementEngine) checkParallelization(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
	// Analyze task dependencies
	independentGroups := sie.findIndependentTaskGroups(tasks)
	
	if len(independentGroups) > 1 {
		return &ImprovementSuggestion{
			ID:             uuid.New(),
			PatternID:      pattern.ID,
			Type:           ImprovementTypeParallelization,
			Description:    fmt.Sprintf("Parallelize %d independent task groups", len(independentGroups)),
			ExpectedImpact: 0.3, // 30% improvement expected
			Confidence:     8.5,
			Implementation: &ImplementationDetails{
				Configuration: map[string]interface{}{
					"parallel_groups": independentGroups,
				},
			},
			Status:    SuggestionStatusPending,
			CreatedAt: time.Now(),
		}
	}
	
	return nil
}

// checkAgentSwap checks if a different agent would perform better
func (sie *SelfImprovementEngine) checkAgentSwap(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
	// Find the weakest performing agent
	weakestAgent, weakestScore := sie.findWeakestAgent(tasks)
	
	if weakestScore < 5.0 {
		// Find alternative agent
		alternativeAgent := sie.findAlternativeAgent(weakestAgent, tasks)
		
		if alternativeAgent != "" {
			return &ImprovementSuggestion{
				ID:             uuid.New(),
				PatternID:      pattern.ID,
				Type:           ImprovementTypeAgentSwap,
				Description:    fmt.Sprintf("Replace %s with %s for better performance", weakestAgent, alternativeAgent),
				ExpectedImpact: (7.0 - weakestScore) / 10.0,
				Confidence:     7.0,
				Implementation: &ImplementationDetails{
					Configuration: map[string]interface{}{
						"old_agent": weakestAgent,
						"new_agent": alternativeAgent,
					},
				},
				Status:    SuggestionStatusPending,
				CreatedAt: time.Now(),
			}
		}
	}
	
	return nil
}

// checkContextEnrichment checks if adding more context would help
func (sie *SelfImprovementEngine) checkContextEnrichment(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
	// Analyze feedback for context-related issues
	contextIssues := 0
	for _, task := range tasks {
		for _, feedback := range task.Feedback {
			if feedback.Type == FeedbackTypeImprovement {
				contextIssues++
			}
		}
	}
	
	if contextIssues > len(tasks)/3 {
		return &ImprovementSuggestion{
			ID:             uuid.New(),
			PatternID:      pattern.ID,
			Type:           ImprovementTypeContextEnrich,
			Description:    "Enrich task context with additional metadata and history",
			ExpectedImpact: 0.25,
			Confidence:     7.5,
			Implementation: &ImplementationDetails{
				Configuration: map[string]interface{}{
					"additional_context": []string{"full_history", "related_tasks", "user_preferences"},
				},
			},
			Status:    SuggestionStatusPending,
			CreatedAt: time.Now(),
		}
	}
	
	return nil
}

// checkCaching checks if results can be cached
func (sie *SelfImprovementEngine) checkCaching(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
	// Check for repeated similar inputs
	inputSimilarity := sie.calculateInputSimilarity(tasks)
	
	if inputSimilarity > 0.7 {
		return &ImprovementSuggestion{
			ID:             uuid.New(),
			PatternID:      pattern.ID,
			Type:           ImprovementTypeCaching,
			Description:    "Implement result caching for similar inputs",
			ExpectedImpact: 0.4,
			Confidence:     9.0,
			Implementation: &ImplementationDetails{
				Configuration: map[string]interface{}{
					"cache_ttl":      "1h",
					"cache_key_func": "hash(input + context)",
				},
			},
			Status:    SuggestionStatusPending,
			CreatedAt: time.Now(),
		}
	}
	
	return nil
}

// applyImprovement applies an improvement suggestion
func (sie *SelfImprovementEngine) applyImprovement(ctx context.Context, suggestion *ImprovementSuggestion) error {
	sie.logger.Info("Applying improvement",
		zap.String("suggestion_id", suggestion.ID.String()),
		zap.String("type", string(suggestion.Type)),
		zap.Float64("expected_impact", suggestion.ExpectedImpact))
	
	// Store improvement in Redis for tracking
	improvementKey := fmt.Sprintf("improvement:%s", suggestion.ID)
	improvementData, _ := json.Marshal(suggestion)
	sie.redisClient.Set(ctx, improvementKey, improvementData, 7*24*time.Hour)
	
	// Mark as applied
	now := time.Now()
	suggestion.AppliedAt = &now
	suggestion.Status = SuggestionStatusApplied
	
	// Trigger configuration update based on improvement type
	switch suggestion.Type {
	case ImprovementTypeParallelization:
		// Update orchestrator to use parallel execution
		sie.updateOrchestratorConfig(ctx, "parallel_execution", suggestion.Implementation.Configuration)
	case ImprovementTypeAgentSwap:
		// Update agent routing rules
		sie.updateAgentRouting(ctx, suggestion.Implementation.Configuration)
	case ImprovementTypeContextEnrich:
		// Update context builder
		sie.updateContextBuilder(ctx, suggestion.Implementation.Configuration)
	case ImprovementTypeCaching:
		// Enable caching for pattern
		sie.enablePatternCaching(ctx, suggestion.PatternID, suggestion.Implementation.Configuration)
	}
	
	return nil
}

// Helper methods

func (sie *SelfImprovementEngine) generatePatternKey(taskType string, sequence []agents.AgentType) string {
	key := taskType
	for _, agent := range sequence {
		key += "_" + string(agent)
	}
	return key
}

func (sie *SelfImprovementEngine) extractContextFeatures(tasks []*CollaborativeTask) map[string]interface{} {
	features := make(map[string]interface{})
	
	if len(tasks) > 0 {
		features["task_count"] = len(tasks)
		features["task_type"] = tasks[0].Type
		features["priority_avg"] = sie.calculateAveragePriority(tasks)
		features["has_deadlines"] = sie.hasDeadlines(tasks)
	}
	
	return features
}

func (sie *SelfImprovementEngine) calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	
	return variance / float64(len(values))
}

func (sie *SelfImprovementEngine) findIndependentTaskGroups(tasks []*CollaborativeTask) [][]uuid.UUID {
	// Simple dependency analysis - tasks with no shared dependencies can be parallelized
	groups := make([][]uuid.UUID, 0)
	processed := make(map[uuid.UUID]bool)
	
	for _, task := range tasks {
		if processed[task.ID] {
			continue
		}
		
		group := []uuid.UUID{task.ID}
		processed[task.ID] = true
		
		// Find tasks that don't depend on this one
		for _, other := range tasks {
			if processed[other.ID] {
				continue
			}
			
			hasDepedency := false
			for _, dep := range other.Dependencies {
				if dep == task.ID {
					hasDepedency = true
					break
				}
			}
			
			if !hasDepedency {
				group = append(group, other.ID)
				processed[other.ID] = true
			}
		}
		
		if len(group) > 1 {
			groups = append(groups, group)
		}
	}
	
	return groups
}

func (sie *SelfImprovementEngine) findWeakestAgent(tasks []*CollaborativeTask) (agents.AgentType, float64) {
	agentScores := make(map[agents.AgentType][]float64)
	
	for _, task := range tasks {
		agentScores[task.AssignedAgent] = append(agentScores[task.AssignedAgent], task.ConfidenceScore)
	}
	
	weakestAgent := agents.AgentType("")
	weakestScore := 10.0
	
	for agent, scores := range agentScores {
		avg := 0.0
		for _, score := range scores {
			avg += score
		}
		avg /= float64(len(scores))
		
		if avg < weakestScore {
			weakestScore = avg
			weakestAgent = agent
		}
	}
	
	return weakestAgent, weakestScore
}

func (sie *SelfImprovementEngine) findAlternativeAgent(current agents.AgentType, tasks []*CollaborativeTask) agents.AgentType {
	// This would query the agent registry for agents with similar capabilities
	// For now, return a simple mapping
	alternatives := map[agents.AgentType]agents.AgentType{
		agents.AnalysisAgent:    agents.StrategyAgent,
		agents.DevelopmentAgent: agents.ArchitectAgent,
		agents.QualityAgent:     agents.MonitoringAgent,
	}
	
	if alt, exists := alternatives[current]; exists {
		return alt
	}
	
	return ""
}

func (sie *SelfImprovementEngine) calculateInputSimilarity(tasks []*CollaborativeTask) float64 {
	if len(tasks) < 2 {
		return 0
	}
	
	// Simple similarity based on input length and common words
	// In production, use more sophisticated similarity metrics
	totalSimilarity := 0.0
	comparisons := 0
	
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			similarity := sie.stringSimilarity(tasks[i].Input, tasks[j].Input)
			totalSimilarity += similarity
			comparisons++
		}
	}
	
	if comparisons > 0 {
		return totalSimilarity / float64(comparisons)
	}
	
	return 0
}

func (sie *SelfImprovementEngine) stringSimilarity(s1, s2 string) float64 {
	// Simple Jaccard similarity
	if s1 == s2 {
		return 1.0
	}
	
	// This is a placeholder - use proper string similarity algorithm
	lenDiff := math.Abs(float64(len(s1) - len(s2)))
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))
	
	if maxLen == 0 {
		return 0
	}
	
	return 1.0 - (lenDiff / maxLen)
}

func (sie *SelfImprovementEngine) calculateAveragePriority(tasks []*CollaborativeTask) float64 {
	if len(tasks) == 0 {
		return 0
	}
	
	total := 0
	for _, task := range tasks {
		total += task.Priority
	}
	
	return float64(total) / float64(len(tasks))
}

func (sie *SelfImprovementEngine) hasDeadlines(tasks []*CollaborativeTask) bool {
	for _, task := range tasks {
		if task.Deadline != nil {
			return true
		}
	}
	return false
}

// RecordPattern records a new collaboration pattern for learning
func (sie *SelfImprovementEngine) RecordPattern(ctx context.Context, pattern *CollaborationPattern) error {
	sie.mu.Lock()
	defer sie.mu.Unlock()
	
	// Generate ID if not set
	if pattern.ID == uuid.Nil {
		pattern.ID = uuid.New()
	}
	
	// Store in memory
	key := sie.generatePatternKey(pattern.TaskType, pattern.AgentSequence)
	sie.patterns[key] = pattern
	
	// Store in Redis
	return sie.storePattern(ctx, pattern)
}

// GetBestPattern returns the best pattern for a given task type
func (sie *SelfImprovementEngine) GetBestPattern(ctx context.Context, taskType string) *CollaborationPattern {
	sie.mu.RLock()
	defer sie.mu.RUnlock()
	
	var bestPattern *CollaborationPattern
	highestQValue := 0.0
	
	// Search through patterns for the best Q-value
	for _, pattern := range sie.patterns {
		if pattern.TaskType == taskType && pattern.QValue > highestQValue {
			bestPattern = pattern
			highestQValue = pattern.QValue
		}
	}
	
	// If not found in memory, try Redis
	if bestPattern == nil {
		// Try to load from Redis
		keys, err := sie.redisClient.Keys(ctx, fmt.Sprintf("pattern:%s*", taskType)).Result()
		if err == nil && len(keys) > 0 {
			for _, redisKey := range keys {
				data, err := sie.redisClient.Get(ctx, redisKey).Result()
				if err == nil {
					var pattern CollaborationPattern
					if err := json.Unmarshal([]byte(data), &pattern); err == nil {
						if pattern.QValue > highestQValue {
							bestPattern = &pattern
							highestQValue = pattern.QValue
						}
					}
				}
			}
		}
	}
	
	return bestPattern
}

func (sie *SelfImprovementEngine) storePattern(ctx context.Context, pattern *CollaborationPattern) error {
	patternKey := fmt.Sprintf("pattern:%s", pattern.ID)
	patternData, _ := json.Marshal(pattern)
	return sie.redisClient.Set(ctx, patternKey, patternData, 0).Err()
}

func (sie *SelfImprovementEngine) updateOrchestratorConfig(ctx context.Context, configType string, config map[string]interface{}) {
	// Publish configuration update event
	event := map[string]interface{}{
		"type":   "config_update",
		"target": "orchestrator",
		"config": config,
	}
	eventData, _ := json.Marshal(event)
	sie.redisClient.Publish(ctx, "config_updates", eventData)
}

func (sie *SelfImprovementEngine) updateAgentRouting(ctx context.Context, config map[string]interface{}) {
	// Update agent routing rules in Redis
	routingKey := "agent_routing_rules"
	sie.redisClient.HSet(ctx, routingKey, config)
}

func (sie *SelfImprovementEngine) updateContextBuilder(ctx context.Context, config map[string]interface{}) {
	// Update context builder configuration
	contextKey := "context_builder_config"
	configData, _ := json.Marshal(config)
	sie.redisClient.Set(ctx, contextKey, configData, 0)
}

func (sie *SelfImprovementEngine) enablePatternCaching(ctx context.Context, patternID uuid.UUID, config map[string]interface{}) {
	// Enable caching for specific pattern
	cacheKey := fmt.Sprintf("cache_config:%s", patternID)
	configData, _ := json.Marshal(config)
	sie.redisClient.Set(ctx, cacheKey, configData, 0)
}