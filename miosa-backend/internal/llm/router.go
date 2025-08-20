package llm

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sormind/OSA/miosa-backend/internal/config"
	"go.uber.org/zap"
)

// Task defines the types of tasks for routing
type Task string

const (
	TaskChat          Task = "chat"
	TaskCode          Task = "code"
	TaskSummarize     Task = "summarize"
	TaskExtract       Task = "extract"
	TaskReason        Task = "reason"
	TaskEmbedding     Task = "embedding"
	TaskOrchestration Task = "orchestration"
)

// Priority defines routing priorities
type Priority string

const (
	PrioritySpeed   Priority = "speed"
	PriorityQuality Priority = "quality"
	PriorityCost    Priority = "cost"
	PriorityBalance Priority = "balance"
)

// Model represents a model in the catalog
type Model struct {
	Name               string
	Provider           string
	MaxInputTokens     int
	SupportsFunctions  bool
	SupportsEmbedding  bool
	Quality            int
	Speed              int
	Cost               int
	Fit                map[Task]int
}

// Catalog lists available models
var Catalog = []Model{
	{
		Name:              "llama-3.1-8b-instant",
		Provider:          "groq",
		MaxInputTokens:    128000,
		SupportsFunctions: true,
		Quality:           7,
		Speed:             9,
		Cost:              9,
		Fit: map[Task]int{
			TaskChat: 8, TaskCode: 7, TaskSummarize: 8, TaskExtract: 8, TaskReason: 6,
		},
	},
	{
		Name:              "llama-3.3-70b-versatile",
		Provider:          "groq",
		MaxInputTokens:    128000,
		SupportsFunctions: true,
		Quality:           9,
		Speed:             6,
		Cost:              6,
		Fit: map[Task]int{
			TaskChat: 9, TaskCode: 8, TaskSummarize: 9, TaskExtract: 8, TaskReason: 8,
		},
	},
	{
		Name:              "moonshotai/kimi-k2-instruct",
		Provider:          "kimi",
		MaxInputTokens:    200000,
		SupportsFunctions: true,
		Quality:           10,
		Speed:             7,
		Cost:              5,
		Fit: map[Task]int{
			TaskOrchestration: 10, TaskReason: 9, TaskCode: 9, TaskChat: 8, TaskSummarize: 8,
		},
	},
}

// Node represents a scoring node
type Node struct {
	Name   string
	Weight float64
	Score  float64
	Reason string
}

// Candidate represents a model candidate
type Candidate struct {
	Model Model
	Score float64
	Nodes []Node
	Why   string
}

// Options for model selection
type Options struct {
	Task              Task
	InputTokens       int
	NeedFunctionCalls bool
	Priority          Priority
}

// Router manages model selection with node-based scoring
type Router struct {
	logger *zap.Logger
	stats  map[string]*Stats
	mu     sync.RWMutex
}

// Stats tracks model performance for self-improvement
type Stats struct {
	TotalRequests  int64
	SuccessRate    float64
	AvgLatency     time.Duration
	AvgConfidence  float64
	LastImproved   time.Time
}

// NewRouter creates a new router
func NewRouter(cfg *config.LLMConfig, logger *zap.Logger) (*Router, error) {
	return &Router{
		logger: logger,
		stats:  make(map[string]*Stats),
	}, nil
}

// Select chooses models using node-based scoring
func (r *Router) Select(opts Options) ([]Candidate, error) {
	if opts.Task == "" {
		return nil, errors.New("task cannot be empty")
	}

	// Get weights based on priority
	wq, ws, wc := r.getWeights(opts.Priority)
	
	var candidates []Candidate
	
	for _, model := range Catalog {
		// Apply constraints
		if opts.InputTokens > 0 && opts.InputTokens > model.MaxInputTokens {
			continue
		}
		if opts.NeedFunctionCalls && !model.SupportsFunctions {
			continue
		}
		
		// Build nodes
		nodes := r.buildNodes(model, opts, wq, ws, wc)
		
		// Calculate total score
		totalScore := 0.0
		for _, node := range nodes {
			totalScore += node.Weight * node.Score
		}
		
		// Apply self-improvement bonus if we have good stats
		if stats := r.getStats(model.Name); stats != nil && stats.SuccessRate > 0.8 {
			totalScore *= 1.1 // 10% bonus for proven performers
		}
		
		candidates = append(candidates, Candidate{
			Model: model,
			Score: totalScore,
			Nodes: nodes,
			Why:   fmt.Sprintf("fit=%d q=%d s=%d c=%d", model.Fit[opts.Task], model.Quality, model.Speed, model.Cost),
		})
	}
	
	if len(candidates) == 0 {
		return nil, errors.New("no compatible model found")
	}
	
	// Sort by score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	
	// Return top 3
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	
	return candidates, nil
}

// buildNodes creates scoring nodes for a model
func (r *Router) buildNodes(model Model, opts Options, wq, ws, wc float64) []Node {
	nodes := []Node{}
	
	// Task fit node
	fit := 6.0
	if f, ok := model.Fit[opts.Task]; ok {
		fit = float64(f)
	}
	nodes = append(nodes, Node{
		Name:   "TaskFit",
		Weight: 0.3,
		Score:  fit / 10.0,
		Reason: fmt.Sprintf("Task %s fit: %.0f/10", opts.Task, fit),
	})
	
	// Quality node
	quality := float64(model.Quality) / 10.0
	nodes = append(nodes, Node{
		Name:   "Quality",
		Weight: wq,
		Score:  quality,
		Reason: fmt.Sprintf("Quality: %d/10", model.Quality),
	})
	
	// Speed node
	nodes = append(nodes, Node{
		Name:   "Speed",
		Weight: ws,
		Score:  float64(model.Speed) / 10.0,
		Reason: fmt.Sprintf("Speed: %d/10", model.Speed),
	})
	
	// Cost node
	nodes = append(nodes, Node{
		Name:   "Cost",
		Weight: wc,
		Score:  float64(model.Cost) / 10.0,
		Reason: fmt.Sprintf("Cost efficiency: %d/10", model.Cost),
	})
	
	return nodes
}

// getWeights returns weights based on priority
func (r *Router) getWeights(priority Priority) (float64, float64, float64) {
	switch priority {
	case PrioritySpeed:
		return 0.3, 0.5, 0.2
	case PriorityQuality:
		return 0.55, 0.25, 0.2
	case PriorityCost:
		return 0.25, 0.25, 0.5
	default: // Balance
		return 0.4, 0.35, 0.25
	}
}

// UpdateStats updates model performance stats for self-improvement
func (r *Router) UpdateStats(modelName string, success bool, latency time.Duration, confidence float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	stats, ok := r.stats[modelName]
	if !ok {
		stats = &Stats{}
		r.stats[modelName] = stats
	}
	
	stats.TotalRequests++
	
	// Update success rate (moving average)
	if success {
		stats.SuccessRate = (stats.SuccessRate*float64(stats.TotalRequests-1) + 1.0) / float64(stats.TotalRequests)
	} else {
		stats.SuccessRate = (stats.SuccessRate * float64(stats.TotalRequests-1)) / float64(stats.TotalRequests)
	}
	
	// Update average latency
	stats.AvgLatency = (stats.AvgLatency*time.Duration(stats.TotalRequests-1) + latency) / time.Duration(stats.TotalRequests)
	
	// Update average confidence
	stats.AvgConfidence = (stats.AvgConfidence*float64(stats.TotalRequests-1) + confidence) / float64(stats.TotalRequests)
	
	// Mark improvement if performance is getting better
	if stats.TotalRequests%100 == 0 && stats.SuccessRate > 0.85 {
		stats.LastImproved = time.Now()
		r.logger.Info("Model performance improved",
			zap.String("model", modelName),
			zap.Float64("success_rate", stats.SuccessRate),
			zap.Float64("avg_confidence", stats.AvgConfidence))
	}
}

// getStats returns stats for a model
func (r *Router) getStats(modelName string) *Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats[modelName]
}

// GetBestModel returns the top candidate
func (r *Router) GetBestModel(opts Options) (*Model, error) {
	candidates, err := r.Select(opts)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, errors.New("no models available")
	}
	return &candidates[0].Model, nil
}

// Provider interface for LLM providers
type Provider interface {
	Complete(ctx context.Context, req Request) (*Response, error)
	Stream(ctx context.Context, req Request, callback StreamCallback) error
	GetName() string
	HealthCheck(ctx context.Context) error
}

// Request represents an LLM request
type Request struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	TopP        float64
	TaskType    string
	Metadata    map[string]interface{}
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// Response represents an LLM response
type Response struct {
	Content    string
	TokensUsed int
	Latency    time.Duration
	Provider   string
	Confidence float64
}

// StreamCallback is called for streaming responses
type StreamCallback func(chunk string) error