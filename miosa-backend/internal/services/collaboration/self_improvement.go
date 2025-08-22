package collaboration

import (
    "context"
    "encoding/json"
    "fmt"
    "math"
    "sort"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "github.com/sormind/OSA/miosa-backend/internal/agents"
    "go.uber.org/zap"
)

// SelfImprovementEngine analyzes agent collaboration patterns and improves them
type SelfImprovementEngine struct {
    redisClient *redis.Client
    logger      *zap.Logger

    patterns        map[string]*CollaborationPattern
    improvementBuffer []*ImprovementSuggestion

    // RL params
    learningRate float64 // alpha
    discount     float64 // gamma

    // Confidence decay of old patterns
    confidenceDecay float64

    // Reward weights (can be hot-reloaded from Redis)
    weights           RewardWeights
    weightsTTL        time.Duration
    weightsLastLoaded time.Time

    mu sync.RWMutex
}

// RewardWeights controls contribution of each factor to the reward
type RewardWeights struct {
    SuccessBonus        float64 `json:"success_bonus"`          // per completed task
    FailurePenalty      float64 `json:"failure_penalty"`        // per failed task
    ConfidenceWeight    float64 `json:"confidence_weight"`      // scales (score-5)/10
    TimePenaltyPerSec   float64 `json:"time_penalty_per_sec"`   // per second above threshold
    TimeThresholdMS     int64   `json:"time_threshold_ms"`      // threshold to start penalizing
    RetryPenalty        float64 `json:"retry_penalty"`          // per retry
    ThroughputWeight    float64 `json:"throughput_weight"`      // optional: not used if 0
    CompositeBoost      float64 `json:"composite_boost"`        // bonus for composite suggestion expected impact
    HighImpactThreshold float64 `json:"high_impact_threshold"`  // auto-apply if ExpectedImpact >= this
    HighConfidenceMin   float64 `json:"high_confidence_min"`    // auto-apply if Confidence >= this
}

// CollaborationPattern represents a learned pattern of agent collaboration
type CollaborationPattern struct {
    ID              uuid.UUID          `json:"id"`
    Name            string             `json:"name"`
    TaskType        string             `json:"task_type"`
    AgentSequence   []agents.AgentType `json:"agent_sequence"`
    SuccessRate     float64            `json:"success_rate"`
    AverageTime     time.Duration      `json:"average_time"`
    ConfidenceScore float64            `json:"confidence_score"`
    UsageCount      int64              `json:"usage_count"`
    LastUpdated     time.Time          `json:"last_updated"`
    ContextFeatures map[string]interface{} `json:"context_features"`

    // Reinforcement learning
    Rewards []float64 `json:"rewards"`
    QValue  float64   `json:"q_value"`
}

// ImprovementSuggestion represents a suggested improvement to collaboration
type ImprovementSuggestion struct {
    ID             uuid.UUID           `json:"id"`
    PatternID      uuid.UUID           `json:"pattern_id"`
    Type           ImprovementType     `json:"type"`
    Description    string              `json:"description"`
    ExpectedImpact float64             `json:"expected_impact"`
    Confidence     float64             `json:"confidence"`
    Implementation *ImplementationDetails `json:"implementation"`
    Status         SuggestionStatus    `json:"status"`
    CreatedAt      time.Time           `json:"created_at"`
    AppliedAt      *time.Time          `json:"applied_at,omitempty"`
    Results        *ImprovementResults `json:"results,omitempty"`
}

// ImprovementType categorizes different types of improvements
type ImprovementType string

const (
    ImprovementTypeAgentSwap       ImprovementType = "agent_swap"
    ImprovementTypeParallelization ImprovementType = "parallelization"
    ImprovementTypeContextEnrich   ImprovementType = "context_enrichment"
    ImprovementTypeSkipStep        ImprovementType = "skip_step"
    ImprovementTypeAddValidation   ImprovementType = "add_validation"
    ImprovementTypeCaching         ImprovementType = "caching"
    ImprovementTypeComposite       ImprovementType = "composite"
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
    Code         string                 `json:"code"`
    Configuration map[string]interface{} `json:"configuration"`
    Dependencies []string               `json:"dependencies"`
    RollbackPlan string                 `json:"rollback_plan"`
}

// ImprovementResults tracks the results of an applied improvement
type ImprovementResults struct {
    BeforeMetrics PerformanceMetrics `json:"before_metrics"`
    AfterMetrics  PerformanceMetrics `json:"after_metrics"`
    ImprovementRate float64          `json:"improvement_rate"`
    Validated     bool               `json:"validated"`
    ValidatedAt   *time.Time         `json:"validated_at,omitempty"`
}

// PerformanceMetrics captures performance data
type PerformanceMetrics struct {
    SuccessRate   float64       `json:"success_rate"`
    AverageTime   time.Duration `json:"average_time"`
    ConfidenceAvg float64       `json:"confidence_avg"`
    ErrorRate     float64       `json:"error_rate"`
    ThroughputRate float64      `json:"throughput_rate"`
}

// NewSelfImprovementEngine creates a new self-improvement engine
func NewSelfImprovementEngine(redisClient *redis.Client, logger *zap.Logger) *SelfImprovementEngine {
    return &SelfImprovementEngine{
        redisClient: redisClient,
        logger:      logger,
        patterns:    make(map[string]*CollaborationPattern),

        learningRate:    0.15,
        discount:        0.85,
        confidenceDecay: 0.95,

        weights: RewardWeights{
            SuccessBonus:        1.0,
            FailurePenalty:      -1.0,
            ConfidenceWeight:    1.0,
            TimePenaltyPerSec:   -0.02,
            TimeThresholdMS:     10_000,
            RetryPenalty:        -0.3,
            ThroughputWeight:    0.0,
            CompositeBoost:      0.1,
            HighImpactThreshold: 0.25,
            HighConfidenceMin:   9.0,
        },
        weightsTTL:        60 * time.Second,
        weightsLastLoaded: time.Time{},
        improvementBuffer: make([]*ImprovementSuggestion, 0),
    }
}

// AnalyzeCollaboration analyzes a completed collaboration for improvements
func (sie *SelfImprovementEngine) AnalyzeCollaboration(ctx context.Context, tasks []*CollaborativeTask) error {
    if len(tasks) == 0 {
        return nil
    }

    // Try hot-reload reward weights
    _ = sie.loadWeights(ctx)

    // Extract current pattern
    pattern := sie.extractPattern(tasks)
    if pattern == nil {
        return nil
    }

    // Compute reward with current weights
    reward := sie.calculateReward(tasks)

    // Estimate future value (next state) using neighbors / same TaskType best Q
    nextMax := sie.getNextMaxQ(pattern)

    // Q-learning update with next state
    sie.updateQValue(pattern, reward, nextMax)

    // Generate improvements when needed
    if pattern.ConfidenceScore < 7.0 || pattern.SuccessRate < 0.8 {
        suggestions := sie.generateImprovements(ctx, pattern, tasks)
        // Build composite suggestions if helpful
        if comp := sie.buildCompositeSuggestion(suggestions, pattern); comp != nil {
            suggestions = append(suggestions, comp)
        }

        sie.mu.Lock()
        sie.improvementBuffer = append(sie.improvementBuffer, suggestions...)
        sie.mu.Unlock()

        // Auto-apply high-confidence & high-impact improvements
        for _, suggestion := range suggestions {
            if suggestion.Confidence >= sie.weights.HighConfidenceMin &&
                suggestion.ExpectedImpact >= sie.weights.HighImpactThreshold {
                if err := sie.applyImprovement(ctx, suggestion); err != nil {
                    sie.logger.Warn("Auto-apply improvement failed",
                        zap.String("suggestion_id", suggestion.ID.String()),
                        zap.Error(err))
                }
            }
        }
    }

    // Store updated pattern
    if err := sie.storePattern(ctx, pattern); err != nil {
        sie.logger.Warn("Failed to persist pattern", zap.String("pattern_id", pattern.ID.String()), zap.Error(err))
    }

    return nil
}

// extractPattern extracts a collaboration pattern from task sequence
func (sie *SelfImprovementEngine) extractPattern(tasks []*CollaborativeTask) *CollaborationPattern {
    if len(tasks) == 0 {
        return nil
    }

    agentSequence := make([]agents.AgentType, 0, len(tasks))
    totalTime := time.Duration(0)
    successCount := 0
    totalConfidence := 0.0

    for _, task := range tasks {
        agentSequence = append(agentSequence, task.AssignedAgent)
        if task.Result != nil {
            totalTime += time.Duration(task.Result.ExecutionMS) * time.Millisecond
        }
        if task.Status == TaskStatusCompleted {
            successCount++
        }
        totalConfidence += task.ConfidenceScore
    }

    taskType := tasks[0].Type
    patternKey := sie.generatePatternKey(taskType, agentSequence)

    sie.mu.RLock()
    existingPattern, exists := sie.patterns[patternKey]
    sie.mu.RUnlock()

    if exists && existingPattern != nil {
        // Update existing pattern with decays
        existingPattern.UsageCount++
        existingPattern.SuccessRate = (existingPattern.SuccessRate*float64(existingPattern.UsageCount-1) + float64(successCount)/float64(len(tasks))) / float64(existingPattern.UsageCount)
        existingPattern.AverageTime = (existingPattern.AverageTime*time.Duration(existingPattern.UsageCount-1) + totalTime) / time.Duration(existingPattern.UsageCount)
        // Confidence updated by recent evidence (blend with decay to keep stability)
        recent := totalConfidence / float64(len(tasks))
        existingPattern.ConfidenceScore = math.Min(10.0, sie.confidenceDecay*existingPattern.ConfidenceScore+(1.0-sie.confidenceDecay)*recent)
        existingPattern.LastUpdated = time.Now()
        return existingPattern
    }

    // Create new pattern
    p := &CollaborationPattern{
        ID:              uuid.New(),
        Name:            fmt.Sprintf("Pattern_%s", patternKey),
        TaskType:        taskType,
        AgentSequence:   agentSequence,
        SuccessRate:     float64(successCount) / float64(len(tasks)),
        AverageTime:     totalTime / time.Duration(len(tasks)),
        ConfidenceScore: totalConfidence / float64(len(tasks)),
        UsageCount:      1,
        LastUpdated:     time.Now(),
        ContextFeatures: sie.extractContextFeatures(tasks),
        Rewards:         []float64{},
        QValue:          0.5, // initial prior
    }

    sie.mu.Lock()
    sie.patterns[patternKey] = p
    sie.mu.Unlock()

    return p
}

// calculateReward calculates the reward for a collaboration pattern
func (sie *SelfImprovementEngine) calculateReward(tasks []*CollaborativeTask) float64 {
    if len(tasks) == 0 {
        return 0
    }
    w := sie.weights

    reward := 0.0
    totalMS := int64(0)
    totalConfidence := 0.0
    completed := 0
    failed := 0
    totalRetries := 0

    for _, task := range tasks {
        if task.Result != nil {
            totalMS += task.Result.ExecutionMS
        }
        if task.Status == TaskStatusCompleted {
            completed++
            reward += w.SuccessBonus
        } else if task.Status == TaskStatusFailed {
            failed++
            reward += w.FailurePenalty
        }
        totalConfidence += (task.ConfidenceScore - 5.0) / 10.0 * w.ConfidenceWeight
        totalRetries += task.RetryCount
    }

    // Time penalty above threshold
    if totalMS > w.TimeThresholdMS {
        over := (totalMS - w.TimeThresholdMS) / 1000 // seconds over threshold
        reward += float64(over) * w.TimePenaltyPerSec
    }

    // Retry penalty
    if totalRetries > 0 {
        reward += float64(totalRetries) * w.RetryPenalty
    }

    // Optional throughput term: tasks per second
    if w.ThroughputWeight != 0 && totalMS > 0 {
        throughput := float64(len(tasks)) / (float64(totalMS) / 1000.0)
        reward += throughput * w.ThroughputWeight
    }

    // Confidence aggregate
    reward += totalConfidence

    // Normalize
    return reward / float64(len(tasks))
}

// updateQValue updates the Q-value using Q-learning with next-state value
func (sie *SelfImprovementEngine) updateQValue(pattern *CollaborationPattern, reward float64, nextMax float64) {
    if pattern == nil {
        return
    }
    old := pattern.QValue
    pattern.QValue = old + sie.learningRate*(reward+sie.discount*nextMax-old)

    // Store reward history (rolling window 100)
    pattern.Rewards = append(pattern.Rewards, reward)
    if len(pattern.Rewards) > 100 {
        pattern.Rewards = pattern.Rewards[1:]
    }

    // Increase confidence if reward variance is low recently
    if len(pattern.Rewards) >= 10 {
        variance := sie.calculateVariance(pattern.Rewards[len(pattern.Rewards)-10:])
        if variance < 0.1 {
            pattern.ConfidenceScore = math.Min(10.0, pattern.ConfidenceScore+0.5)
        }
    }
}

// generateImprovements creates improvement suggestions for a pattern
func (sie *SelfImprovementEngine) generateImprovements(ctx context.Context, pattern *CollaborationPattern, tasks []*CollaborativeTask) []*ImprovementSuggestion {
    suggestions := make([]*ImprovementSuggestion, 0, 4)

    // Parallelization
    if len(pattern.AgentSequence) > 2 {
        if s := sie.checkParallelization(pattern, tasks); s != nil {
            suggestions = append(suggestions, s)
        }
    }

    // Agent swap
    if pattern.SuccessRate < 0.7 {
        if s := sie.checkAgentSwap(pattern, tasks); s != nil {
            suggestions = append(suggestions, s)
        }
    }

    // Context enrichment
    if pattern.ConfidenceScore < 7.0 {
        if s := sie.checkContextEnrichment(pattern, tasks); s != nil {
            suggestions = append(suggestions, s)
        }
    }

    // Caching
    if pattern.UsageCount > 10 && pattern.AverageTime > 5*time.Second {
        if s := sie.checkCaching(pattern, tasks); s != nil {
            suggestions = append(suggestions, s)
        }
    }

    // Skip step (detect clearly redundant step with low confidence + low impact)
    if s := sie.checkSkipStep(pattern, tasks); s != nil {
        suggestions = append(suggestions, s)
    }

    // Add validation (if many failures linked to input quality)
    if s := sie.checkAddValidation(pattern, tasks); s != nil {
        suggestions = append(suggestions, s)
    }

    // Rank by ExpectedImpact descending
    sort.SliceStable(suggestions, func(i, j int) bool {
        return suggestions[i].ExpectedImpact > suggestions[j].ExpectedImpact
    })

    return suggestions
}

// buildCompositeSuggestion combines two compatible improvements into a composite with boosted impact
func (sie *SelfImprovementEngine) buildCompositeSuggestion(base []*ImprovementSuggestion, pattern *CollaborationPattern) *ImprovementSuggestion {
    if len(base) < 2 {
        return nil
    }

    // pick top two with different types and complementary effects
    for i := 0; i < len(base)-1; i++ {
        for j := i + 1; j < len(base); j++ {
            a := base[i]
            b := base[j]
            if a.Type == b.Type {
                continue
            }
            // Favor pairs: parallelization + context, swap + validation, caching + parallelization
            if isComplementary(a.Type, b.Type) {
                impact := a.ExpectedImpact + b.ExpectedImpact + sie.weights.CompositeBoost
                conf := math.Min(10.0, (a.Confidence+b.Confidence)/2.0+0.3)
                return &ImprovementSuggestion{
                    ID:             uuid.New(),
                    PatternID:      pattern.ID,
                    Type:           ImprovementTypeComposite,
                    Description:    fmt.Sprintf("Composite: %s + %s", a.Type, b.Type),
                    ExpectedImpact: impact,
                    Confidence:     conf,
                    Implementation: &ImplementationDetails{
                        Configuration: map[string]interface{}{
                            "actions": []map[string]interface{}{
                                {"type": a.Type, "config": a.Implementation.Configuration},
                                {"type": b.Type, "config": b.Implementation.Configuration},
                            },
                        },
                        RollbackPlan: "Rollback both actions in reverse order if metrics degrade.",
                    },
                    Status:    SuggestionStatusPending,
                    CreatedAt: time.Now(),
                }
            }
        }
    }
    return nil
}

func isComplementary(a, b ImprovementType) bool {
    switch a {
    case ImprovementTypeParallelization:
        return b == ImprovementTypeContextEnrich || b == ImprovementTypeCaching
    case ImprovementTypeAgentSwap:
        return b == ImprovementTypeAddValidation || b == ImprovementTypeContextEnrich
    case ImprovementTypeCaching:
        return b == ImprovementTypeParallelization || b == ImprovementTypeContextEnrich
    case ImprovementTypeContextEnrich:
        return b == ImprovementTypeParallelization || b == ImprovementTypeAgentSwap || b == ImprovementTypeCaching
    case ImprovementTypeAddValidation:
        return b == ImprovementTypeAgentSwap
    default:
        return false
    }
}

// checkParallelization checks if tasks can be parallelized
func (sie *SelfImprovementEngine) checkParallelization(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
    independentGroups := sie.findIndependentTaskGroups(tasks)
    if len(independentGroups) > 1 {
        return &ImprovementSuggestion{
            ID:             uuid.New(),
            PatternID:      pattern.ID,
            Type:           ImprovementTypeParallelization,
            Description:    fmt.Sprintf("Parallelize %d independent task groups", len(independentGroups)),
            ExpectedImpact: 0.30,
            Confidence:     8.5,
            Implementation: &ImplementationDetails{
                Configuration: map[string]interface{}{
                    "parallel_groups": independentGroups,
                    "max_concurrency": min(len(independentGroups), 4),
                },
                RollbackPlan: "Restore sequential execution if error rate increases >2%.",
            },
            Status:    SuggestionStatusPending,
            CreatedAt: time.Now(),
        }
    }
    return nil
}

// checkAgentSwap checks if a different agent would perform better
func (sie *SelfImprovementEngine) checkAgentSwap(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
    weakestAgent, weakestScore := sie.findWeakestAgent(tasks)
    if weakestAgent == "" {
        return nil
    }
    if weakestScore < 5.0 {
        if alternative := sie.findAlternativeAgent(weakestAgent, tasks); alternative != "" {
            return &ImprovementSuggestion{
                ID:             uuid.New(),
                PatternID:      pattern.ID,
                Type:           ImprovementTypeAgentSwap,
                Description:    fmt.Sprintf("Replace %s with %s for better performance", weakestAgent, alternative),
                ExpectedImpact: (7.0 - weakestScore) / 10.0,
                Confidence:     7.2,
                Implementation: &ImplementationDetails{
                    Configuration: map[string]interface{}{
                        "old_agent": weakestAgent,
                        "new_agent": alternative,
                    },
                    RollbackPlan: "Revert routing to previous agent if success rate drops.",
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
    contextIssues := 0
    for _, task := range tasks {
        for _, fb := range task.Feedback {
            if fb.Type == FeedbackTypeImprovement {
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
            Confidence:     7.8,
            Implementation: &ImplementationDetails{
                Configuration: map[string]interface{}{
                    "additional_context": []string{"full_history", "related_tasks", "user_preferences"},
                },
                RollbackPlan: "Remove added context fields if latency increases >15%.",
            },
            Status:    SuggestionStatusPending,
            CreatedAt: time.Now(),
        }
    }
    return nil
}

// checkCaching checks if results can be cached
func (sie *SelfImprovementEngine) checkCaching(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
    if sim := sie.calculateInputSimilarity(tasks); sim > 0.7 {
        return &ImprovementSuggestion{
            ID:             uuid.New(),
            PatternID:      pattern.ID,
            Type:           ImprovementTypeCaching,
            Description:    "Implement result caching for similar inputs",
            ExpectedImpact: 0.40,
            Confidence:     9.0,
            Implementation: &ImplementationDetails{
                Configuration: map[string]interface{}{
                    "cache_ttl":      "1h",
                    "cache_key_func": "hash(input + context)",
                    "max_entries":    5000,
                },
                RollbackPlan: "Disable cache if hit ratio <20% for 24h.",
            },
            Status:    SuggestionStatusPending,
            CreatedAt: time.Now(),
        }
    }
    return nil
}

// checkSkipStep suggests skipping a redundant low-confidence step
func (sie *SelfImprovementEngine) checkSkipStep(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
    // Heuristic: find a step with low confidence and low dependency fan-in/out
    minScore := 10.0
    var idx int = -1
    for i, t := range tasks {
        if t.ConfidenceScore < minScore {
            minScore = t.ConfidenceScore
            idx = i
        }
    }
    if idx == -1 || minScore >= 5.0 {
        return nil
    }
    // Only suggest if skipping doesn't break explicit dependency chain
    if sie.stepCritical(tasks, idx) {
        return nil
    }
    return &ImprovementSuggestion{
        ID:             uuid.New(),
        PatternID:      pattern.ID,
        Type:           ImprovementTypeSkipStep,
        Description:    fmt.Sprintf("Skip step %d (%s) to reduce latency", idx, tasks[idx].AssignedAgent),
        ExpectedImpact: 0.15,
        Confidence:     6.8,
        Implementation: &ImplementationDetails{
            Configuration: map[string]interface{}{
                "skip_index": idx,
            },
            RollbackPlan: "Reinsert step if error rate increases.",
        },
        Status:    SuggestionStatusPending,
        CreatedAt: time.Now(),
    }
}

// checkAddValidation adds an early validation if many failures happen later
func (sie *SelfImprovementEngine) checkAddValidation(pattern *CollaborationPattern, tasks []*CollaborativeTask) *ImprovementSuggestion {
    failures := 0
    for _, t := range tasks {
        if t.Status == TaskStatusFailed {
            failures++
        }
    }
    if failures >= 2 {
        return &ImprovementSuggestion{
            ID:             uuid.New(),
            PatternID:      pattern.ID,
            Type:           ImprovementTypeAddValidation,
            Description:    "Add early validation to catch issues before expensive steps",
            ExpectedImpact: 0.18,
            Confidence:     7.0,
            Implementation: &ImplementationDetails{
                Configuration: map[string]interface{}{
                    "validation_rules": []string{"schema_check", "guardrails", "rate_limit"},
                },
            },
            Status:    SuggestionStatusPending,
            CreatedAt: time.Now(),
        }
    }
    return nil
}

// applyImprovement applies an improvement suggestion and schedules evaluation
func (sie *SelfImprovementEngine) applyImprovement(ctx context.Context, suggestion *ImprovementSuggestion) error {
    sie.logger.Info("Applying improvement",
        zap.String("suggestion_id", suggestion.ID.String()),
        zap.String("type", string(suggestion.Type)),
        zap.Float64("expected_impact", suggestion.ExpectedImpact))

    // Persist suggestion
    improvementKey := fmt.Sprintf("improvement:%s", suggestion.ID)
    improvementData, _ := json.Marshal(suggestion)
    if err := sie.redisClient.Set(ctx, improvementKey, improvementData, 7*24*time.Hour).Err(); err != nil {
        return err
    }

    // Mark as applied
    now := time.Now()
    suggestion.AppliedAt = &now
    suggestion.Status = SuggestionStatusApplied

    // Capture "before" metrics (from monitoring or fallback to pattern)
    before := sie.getCurrentMetrics(ctx, suggestion.PatternID)

    // Trigger configuration update based on improvement type
    switch suggestion.Type {
    case ImprovementTypeParallelization:
        sie.updateOrchestratorConfig(ctx, "parallel_execution", suggestion.Implementation.Configuration)
    case ImprovementTypeAgentSwap:
        sie.updateAgentRouting(ctx, suggestion.Implementation.Configuration)
    case ImprovementTypeContextEnrich:
        sie.updateContextBuilder(ctx, suggestion.Implementation.Configuration)
    case ImprovementTypeCaching:
        sie.enablePatternCaching(ctx, suggestion.PatternID, suggestion.Implementation.Configuration)
    case ImprovementTypeSkipStep, ImprovementTypeAddValidation:
        sie.updateOrchestratorConfig(ctx, string(suggestion.Type), suggestion.Implementation.Configuration)
    case ImprovementTypeComposite:
        // Expand composite actions
        if cfg, ok := suggestion.Implementation.Configuration["actions"].([]map[string]interface{}); ok {
            for _, act := range cfg {
                if t, ok := act["type"].(string); ok {
                    if c, ok := act["config"].(map[string]interface{}); ok {
                        sie.updateOrchestratorConfig(ctx, t, c)
                    }
                }
            }
        }
    }

    // Schedule post-application evaluation with monitoring-service
    sie.requestEvaluation(ctx, suggestion.PatternID, suggestion.ID)

    // Stash preliminary results with BeforeMetrics; AfterMetrics to be filled later by evaluator
    suggestion.Results = &ImprovementResults{
        BeforeMetrics:  before,
        AfterMetrics:   PerformanceMetrics{},
        ImprovementRate: 0.0,
        Validated:      false,
    }

    // Update stored record with "applied" status and before-metrics
    improvementData, _ = json.Marshal(suggestion)
    _ = sie.redisClient.Set(ctx, improvementKey, improvementData, 7*24*time.Hour).Err()

    return nil
}

// requestEvaluation publishes a request for monitoring-service to evaluate impact
func (sie *SelfImprovementEngine) requestEvaluation(ctx context.Context, patternID uuid.UUID, suggestionID uuid.UUID) {
    payload := map[string]interface{}{
        "type":          "evaluate_improvement",
        "pattern_id":    patternID.String(),
        "suggestion_id": suggestionID.String(),
        // e.g., evaluate after 30 minutes window
        "window": "30m",
    }
    data, _ := json.Marshal(payload)
    if err := sie.redisClient.Publish(ctx, "monitoring:requests", data).Err(); err != nil {
        sie.logger.Warn("Failed to publish evaluation request", zap.Error(err))
    }
}

// getCurrentMetrics tries monitoring first, falls back to pattern snapshot
func (sie *SelfImprovementEngine) getCurrentMetrics(ctx context.Context, patternID uuid.UUID) PerformanceMetrics {
    key := fmt.Sprintf("metrics:pattern:%s:current", patternID.String())
    if raw, err := sie.redisClient.Get(ctx, key).Bytes(); err == nil {
        var m PerformanceMetrics
        if json.Unmarshal(raw, &m) == nil {
            return m
        }
    }
    // Fallback: derive from in-memory pattern if available
    sie.mu.RLock()
    defer sie.mu.RUnlock()
    for _, p := range sie.patterns {
        if p.ID == patternID {
            return PerformanceMetrics{
                SuccessRate:   p.SuccessRate,
                AverageTime:   p.AverageTime,
                ConfidenceAvg: p.ConfidenceScore,
                ErrorRate:     math.Max(0, 1.0-p.SuccessRate),
                ThroughputRate: 0.0,
            }
        }
    }
    return PerformanceMetrics{}
}

// RecordPattern records a new collaboration pattern for learning
func (sie *SelfImprovementEngine) RecordPattern(ctx context.Context, pattern *CollaborationPattern) error {
    sie.mu.Lock()
    defer sie.mu.Unlock()

    if pattern.ID == uuid.Nil {
        pattern.ID = uuid.New()
    }
    key := sie.generatePatternKey(pattern.TaskType, pattern.AgentSequence)
    sie.patterns[key] = pattern

    return sie.storePattern(ctx, pattern)
}

// GetBestPattern returns the best pattern for a given task type
func (sie *SelfImprovementEngine) GetBestPattern(ctx context.Context, taskType string) *CollaborationPattern {
    sie.mu.RLock()
    var best *CollaborationPattern
    highest := -1.0
    for _, p := range sie.patterns {
        if p.TaskType == taskType && p.QValue > highest {
            best = p
            highest = p.QValue
        }
    }
    sie.mu.RUnlock()

    if best != nil {
        return best
    }

    // Try Redis fallback
    keys, err := sie.redisClient.Keys(ctx, fmt.Sprintf("pattern:%s*", taskType)).Result()
    if err == nil {
        for _, k := range keys {
            if data, err := sie.redisClient.Get(ctx, k).Result(); err == nil {
                var p CollaborationPattern
                if json.Unmarshal([]byte(data), &p) == nil {
                    if p.QValue > highest {
                        highest = p.QValue
                        cp := p
                        best = &cp
                    }
                }
            }
        }
    }
    return best
}

func (sie *SelfImprovementEngine) storePattern(ctx context.Context, pattern *CollaborationPattern) error {
    patternKey := fmt.Sprintf("pattern:%s", pattern.ID.String())
    b, _ := json.Marshal(pattern)
    return sie.redisClient.Set(ctx, patternKey, b, 0).Err()
}

func (sie *SelfImprovementEngine) updateOrchestratorConfig(ctx context.Context, configType string, config map[string]interface{}) {
    event := map[string]interface{}{
        "type":   "config_update",
        "target": "orchestrator",
        "update": map[string]interface{}{
            "kind":   configType,
            "config": config,
        },
    }
    data, _ := json.Marshal(event)
    if err := sie.redisClient.Publish(ctx, "config_updates", data).Err(); err != nil {
        sie.logger.Warn("Failed to publish orchestrator config update", zap.Error(err))
    }
}

func (sie *SelfImprovementEngine) updateAgentRouting(ctx context.Context, config map[string]interface{}) {
    routingKey := "agent_routing_rules"
    if err := sie.redisClient.HSet(ctx, routingKey, config).Err(); err != nil {
        sie.logger.Warn("Failed to update agent routing", zap.Error(err))
    }
}

func (sie *SelfImprovementEngine) updateContextBuilder(ctx context.Context, config map[string]interface{}) {
    key := "context_builder_config"
    b, _ := json.Marshal(config)
    if err := sie.redisClient.Set(ctx, key, b, 0).Err(); err != nil {
        sie.logger.Warn("Failed to update context builder", zap.Error(err))
    }
}

func (sie *SelfImprovementEngine) enablePatternCaching(ctx context.Context, patternID uuid.UUID, config map[string]interface{}) {
    key := fmt.Sprintf("cache_config:%s", patternID.String())
    b, _ := json.Marshal(config)
    if err := sie.redisClient.Set(ctx, key, b, 0).Err(); err != nil {
        sie.logger.Warn("Failed to enable pattern caching", zap.Error(err))
    }
}

// Helper methods

func (sie *SelfImprovementEngine) generatePatternKey(taskType string, sequence []agents.AgentType) string {
    key := taskType
    for _, a := range sequence {
        key += "_" + string(a)
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
    sum := 0.0
    for _, v := range values {
        sum += v
    }
    mean := sum / float64(len(values))
    var varSum float64
    for _, v := range values {
        d := v - mean
        varSum += d * d
    }
    return varSum / float64(len(values))
}

func (sie *SelfImprovementEngine) findIndependentTaskGroups(tasks []*CollaborativeTask) [][]uuid.UUID {
    groups := make([][]uuid.UUID, 0)
    processed := make(map[uuid.UUID]bool)

    for _, task := range tasks {
        if processed[task.ID] {
            continue
        }
        group := []uuid.UUID{task.ID}
        processed[task.ID] = true

        for _, other := range tasks {
            if processed[other.ID] {
                continue
            }
            dep := false
            for _, d := range other.Dependencies {
                if d == task.ID {
                    dep = true
                    break
                }
            }
            if !dep {
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

func (sie *SelfImprovementEngine) stepCritical(tasks []*CollaborativeTask, idx int) bool {
    cur := tasks[idx].ID
    for _, t := range tasks {
        for _, d := range t.Dependencies {
            if d == cur {
                return true
            }
        }
    }
    return false
}

func (sie *SelfImprovementEngine) findWeakestAgent(tasks []*CollaborativeTask) (agents.AgentType, float64) {
    agentScores := make(map[agents.AgentType][]float64)
    for _, t := range tasks {
        agentScores[t.AssignedAgent] = append(agentScores[t.AssignedAgent], t.ConfidenceScore)
    }
    weakest := agents.AgentType("")
    minAvg := 10.0
    for ag, scores := range agentScores {
        sum := 0.0
        for _, s := range scores {
            sum += s
        }
        avg := sum / float64(len(scores))
        if avg < minAvg {
            minAvg = avg
            weakest = ag
        }
    }
    return weakest, minAvg
}

func (sie *SelfImprovementEngine) findAlternativeAgent(current agents.AgentType, tasks []*CollaborativeTask) agents.AgentType {
    // TODO: query agent registry for capabilities; simple static fallback for now
    alts := map[agents.AgentType]agents.AgentType{
        agents.AnalysisAgent:   agents.StrategyAgent,
        agents.DevelopmentAgent: agents.ArchitectAgent,
        agents.QualityAgent:     agents.MonitoringAgent,
    }
    if a, ok := alts[current]; ok {
        return a
    }
    return ""
}

func (sie *SelfImprovementEngine) calculateInputSimilarity(tasks []*CollaborativeTask) float64 {
    if len(tasks) < 2 {
        return 0
    }
    total := 0.0
    comp := 0
    for i := 0; i < len(tasks)-1; i++ {
        for j := i + 1; j < len(tasks); j++ {
            total += sie.stringSimilarity(tasks[i].Input, tasks[j].Input)
            comp++
        }
    }
    if comp == 0 {
        return 0
    }
    return total / float64(comp)
}

func (sie *SelfImprovementEngine) stringSimilarity(a, b string) float64 {
    if a == b {
        return 1.0
    }
    lenDiff := math.Abs(float64(len(a) - len(b)))
    maxLen := math.Max(float64(len(a)), float64(len(b)))
    if maxLen == 0 {
        return 0
    }
    return 1.0 - (lenDiff / maxLen)
}

func (sie *SelfImprovementEngine) calculateAveragePriority(tasks []*CollaborativeTask) float64 {
    if len(tasks) == 0 {
        return 0
    }
    sum := 0
    for _, t := range tasks {
        sum += t.Priority
    }
    return float64(sum) / float64(len(tasks))
}

func (sie *SelfImprovementEngine) hasDeadlines(tasks []*CollaborativeTask) bool {
    for _, t := range tasks {
        if t.Deadline != nil {
            return true
        }
    }
    return false
}

// loadWeights hot-reloads reward weights from Redis key "self_improvement:weights"
func (sie *SelfImprovementEngine) loadWeights(ctx context.Context) error {
    if time.Since(sie.weightsLastLoaded) < sie.weightsTTL {
        return nil
    }
    raw, err := sie.redisClient.Get(ctx, "self_improvement:weights").Bytes()
    if err != nil {
        // no override; keep defaults
        sie.weightsLastLoaded = time.Now()
        return nil
    }
    var w RewardWeights
    if json.Unmarshal(raw, &w) == nil {
        sie.weights = w
        sie.logger.Info("Self-improvement weights reloaded")
    }
    sie.weightsLastLoaded = time.Now()
    return nil
}

// getNextMaxQ estimates the best possible future Q for the same task type
func (sie *SelfImprovementEngine) getNextMaxQ(pattern *CollaborationPattern) float64 {
    // Use best known pattern for same task type (neighboring/alternative sequences)
    sie.mu.RLock()
    defer sie.mu.RUnlock()

    best := 0.0
    for _, p := range sie.patterns {
        if p.TaskType == pattern.TaskType {
            // Prefer similar-length sequences; tiny bias
            bias := 0.0
            if len(p.AgentSequence) == len(pattern.AgentSequence) {
                bias = 0.02
            }
            if p.QValue+bias > best {
                best = p.QValue + bias
            }
        }
    }
    return best
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
