package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware provides tenant-aware rate limiting
type RateLimitMiddleware struct {
	redis          redis.UniversalClient
	logger         *zap.Logger
	defaultLimit   int
	windowDuration time.Duration
	planLimits     map[string]PlanLimit
	skipPaths      []string
	localLimiters  map[string]*rate.Limiter
	mu             sync.RWMutex
	circuitBreaker *CircuitBreaker
}

// PlanLimit defines rate limits for different subscription plans
type PlanLimit struct {
	RequestsPerHour int
	BurstSize       int
	AgentLimits     map[agents.AgentType]int // Per-agent limits
}

// CircuitBreaker protects the system from overload
type CircuitBreaker struct {
	maxFailures    int
	resetTimeout   time.Duration
	failures       int
	lastFailTime   time.Time
	state          string // "closed", "open", "half-open"
	mu             sync.RWMutex
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	DefaultLimit   int
	WindowDuration time.Duration
	SkipPaths      []string
	EnableCircuitBreaker bool
}

// DefaultRateLimitConfig returns default configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		DefaultLimit:   100, // requests per hour
		WindowDuration: time.Hour,
		SkipPaths:      []string{"/health", "/metrics"},
		EnableCircuitBreaker: true,
	}
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(redis redis.UniversalClient, logger *zap.Logger, config *RateLimitConfig) *RateLimitMiddleware {
	m := &RateLimitMiddleware{
		redis:          redis,
		logger:         logger,
		defaultLimit:   config.DefaultLimit,
		windowDuration: config.WindowDuration,
		skipPaths:      config.SkipPaths,
		localLimiters:  make(map[string]*rate.Limiter),
		planLimits: map[string]PlanLimit{
			"free": {
				RequestsPerHour: 100,
				BurstSize:       10,
				AgentLimits: map[agents.AgentType]int{
					agents.DevelopmentAgent: 10,
					agents.DeploymentAgent:  5,
				},
			},
			"pro": {
				RequestsPerHour: 10000,
				BurstSize:       100,
				AgentLimits: map[agents.AgentType]int{
					agents.DevelopmentAgent: 1000,
					agents.DeploymentAgent:  100,
				},
			},
			"enterprise": {
				RequestsPerHour: -1, // Unlimited
				BurstSize:       1000,
				AgentLimits:     map[agents.AgentType]int{}, // No limits
			},
		},
	}

	if config.EnableCircuitBreaker {
		m.circuitBreaker = &CircuitBreaker{
			maxFailures:  5,
			resetTimeout: 30 * time.Second,
			state:        "closed",
		}
	}

	return m
}

// Handle processes rate limiting for incoming requests
func (m *RateLimitMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for certain paths
		if m.shouldSkipRateLimit(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Check circuit breaker state
		if m.circuitBreaker != nil && !m.circuitBreaker.Allow() {
			c.Header("Retry-After", "30")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable",
			})
			c.Abort()
			return
		}

		// Get tenant context from auth middleware
		var tenantID, userID string
		var plan string = "free" // Default plan
		
		if taskCtx, exists := c.Get("task_context"); exists {
			if ctx, ok := taskCtx.(*agents.TaskContext); ok {
				tenantID = ctx.TenantID.String()
				userID = ctx.UserID.String()
				// Get plan from metadata if available
				if p, ok := ctx.Metadata["plan"]; ok {
					plan = p
				}
			}
		}

		// Use IP-based rate limiting if no tenant context
		if tenantID == "" {
			tenantID = c.ClientIP()
		}

		// Get rate limit for the plan
		limit := m.getPlanLimit(plan)
		
		// Check agent-specific limits if this is an agent execution request
		if agentType := m.extractAgentType(c); agentType != "" {
			if !m.checkAgentLimit(c.Request.Context(), tenantID, agentType, plan) {
				c.Header("X-RateLimit-Limit-Agent", strconv.Itoa(m.planLimits[plan].AgentLimits[agentType]))
				c.Header("X-RateLimit-Remaining-Agent", "0")
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": fmt.Sprintf("Agent rate limit exceeded for %s", agentType),
				})
				c.Abort()
				return
			}
		}

		// Check global rate limit using Redis sliding window
		allowed, remaining, resetTime, err := m.checkRateLimit(c.Request.Context(), tenantID, limit)
		if err != nil {
			m.logger.Error("Rate limit check failed",
				zap.Error(err),
				zap.String("tenant_id", tenantID))
			// Fall back to local rate limiting
			if !m.checkLocalRateLimit(tenantID, limit) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Rate limit exceeded",
				})
				c.Abort()
				return
			}
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.RequestsPerHour))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			// Record circuit breaker failure if too many rate limit violations
			if m.circuitBreaker != nil {
				m.circuitBreaker.RecordFailure()
			}

			retryAfter := int(time.Until(resetTime).Seconds())
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			
			m.logger.Warn("Rate limit exceeded",
				zap.String("tenant_id", tenantID),
				zap.String("user_id", userID),
				zap.String("plan", plan),
				zap.String("ip", c.ClientIP()))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		// Record success for circuit breaker
		if m.circuitBreaker != nil {
			m.circuitBreaker.RecordSuccess()
		}

		c.Next()
	}
}

// checkRateLimit checks rate limit using Redis sliding window algorithm
func (m *RateLimitMiddleware) checkRateLimit(ctx context.Context, key string, limit PlanLimit) (bool, int, time.Time, error) {
	if limit.RequestsPerHour == -1 {
		// Unlimited plan
		return true, -1, time.Now().Add(m.windowDuration), nil
	}

	now := time.Now()
	windowStart := now.Add(-m.windowDuration)
	
	// Redis key for this tenant/IP
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	
	// Use Redis pipeline for atomic operations
	pipe := m.redis.Pipeline()
	
	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart.UnixNano(), 10))
	
	// Count requests in current window
	countCmd := pipe.ZCard(ctx, redisKey)
	
	// Add current request
	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d:%s", now.UnixNano(), now.String()),
	})
	
	// Set expiry on the key
	pipe.Expire(ctx, redisKey, m.windowDuration+time.Minute)
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, time.Now(), err
	}
	
	count := int(countCmd.Val())
	remaining := limit.RequestsPerHour - count
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime := now.Add(m.windowDuration)
	allowed := count <= limit.RequestsPerHour
	
	return allowed, remaining, resetTime, nil
}

// checkLocalRateLimit uses in-memory rate limiting as fallback
func (m *RateLimitMiddleware) checkLocalRateLimit(key string, limit PlanLimit) bool {
	if limit.RequestsPerHour == -1 {
		return true // Unlimited
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	limiter, exists := m.localLimiters[key]
	if !exists {
		// Create new limiter
		ratePerSecond := float64(limit.RequestsPerHour) / 3600.0
		limiter = rate.NewLimiter(rate.Limit(ratePerSecond), limit.BurstSize)
		m.localLimiters[key] = limiter
		
		// Clean up old limiters periodically
		if len(m.localLimiters) > 10000 {
			// Simple cleanup: remove half of the limiters
			for k := range m.localLimiters {
				delete(m.localLimiters, k)
				if len(m.localLimiters) <= 5000 {
					break
				}
			}
		}
	}

	return limiter.Allow()
}

// checkAgentLimit checks agent-specific rate limits
func (m *RateLimitMiddleware) checkAgentLimit(ctx context.Context, tenantID string, agentType agents.AgentType, plan string) bool {
	planLimit, exists := m.planLimits[plan]
	if !exists {
		return true
	}

	agentLimit, hasLimit := planLimit.AgentLimits[agentType]
	if !hasLimit || agentLimit == -1 {
		return true // No limit for this agent
	}

	key := fmt.Sprintf("ratelimit:agent:%s:%s", tenantID, agentType)
	
	// Check using Redis counter with 1-hour window
	count, err := m.redis.Incr(ctx, key).Result()
	if err != nil {
		m.logger.Error("Failed to check agent rate limit",
			zap.Error(err),
			zap.String("tenant_id", tenantID),
			zap.String("agent_type", string(agentType)))
		return true // Allow on error
	}

	// Set expiry on first increment
	if count == 1 {
		m.redis.Expire(ctx, key, time.Hour)
	}

	return int(count) <= agentLimit
}

// getPlanLimit returns the rate limit for a plan
func (m *RateLimitMiddleware) getPlanLimit(plan string) PlanLimit {
	if limit, exists := m.planLimits[plan]; exists {
		return limit
	}
	// Return default free plan limit
	return m.planLimits["free"]
}

// extractAgentType extracts agent type from request path or parameters
func (m *RateLimitMiddleware) extractAgentType(c *gin.Context) agents.AgentType {
	// Check path for agent type
	path := c.Request.URL.Path
	if path == "/api/agents/execute" {
		// Check request body or query params
		if agentType := c.Query("agent"); agentType != "" {
			return agents.AgentType(agentType)
		}
		// Could also check request body here if needed
	}
	return ""
}

// shouldSkipRateLimit checks if path should skip rate limiting
func (m *RateLimitMiddleware) shouldSkipRateLimit(path string) bool {
	for _, skipPath := range m.skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// UpdatePlanLimit updates rate limit for a specific plan
func (m *RateLimitMiddleware) UpdatePlanLimit(plan string, limit PlanLimit) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.planLimits[plan] = limit
}

// GetUsage returns current usage for a tenant
func (m *RateLimitMiddleware) GetUsage(ctx context.Context, tenantID string) (int, error) {
	key := fmt.Sprintf("ratelimit:%s", tenantID)
	
	now := time.Now()
	windowStart := now.Add(-m.windowDuration)
	
	count, err := m.redis.ZCount(ctx, key, 
		strconv.FormatInt(windowStart.UnixNano(), 10),
		strconv.FormatInt(now.UnixNano(), 10)).Result()
	
	return int(count), err
}

// CircuitBreaker methods

// Allow checks if requests should be allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case "open":
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = "half-open"
			cb.failures = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case "half-open":
		// Allow limited requests to test if service recovered
		return true
	default: // closed
		return true
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failures = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = "open"
	}
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = "closed"
	cb.failures = 0
}