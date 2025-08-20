package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingMiddleware_Handle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		requestPath    string
		requestMethod  string
		requestBody    string
		responseStatus int
		responseBody   string
		checkLogs      func(*testing.T, []observer.LoggedEntry)
		simulateDelay  time.Duration
	}{
		{
			name:          "successful request with task context",
			requestPath:   "/api/agents/execute",
			requestMethod: "POST",
			requestBody:   `{"agent":"development","task":"Generate code"}`,
			setupContext: func(c *gin.Context) {
				taskCtx := &agents.TaskContext{
					ID:          uuid.New(),
					UserID:      uuid.New(),
					TenantID:    uuid.New(),
					WorkspaceID: uuid.New(),
					Phase:       agents.PhaseDevelopment,
					AgentType:   agents.DevelopmentAgent,
					Metadata:    map[string]string{"feature": "logging"},
				}
				c.Set("task_context", taskCtx)
			},
			responseStatus: http.StatusOK,
			responseBody:   `{"status":"success"}`,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				assert.GreaterOrEqual(t, len(logs), 1)
				// Check for request completed log
				found := false
				for _, log := range logs {
					if log.Message == "Request completed" {
						found = true
						assert.Equal(t, zapcore.InfoLevel, log.Level)
						// Check fields
						assert.Contains(t, log.ContextMap(), "method")
						assert.Contains(t, log.ContextMap(), "path")
						assert.Contains(t, log.ContextMap(), "status")
						assert.Contains(t, log.ContextMap(), "duration_ms")
						assert.Contains(t, log.ContextMap(), "task_id")
						assert.Contains(t, log.ContextMap(), "tenant_id")
						assert.Contains(t, log.ContextMap(), "agent_type")
						break
					}
				}
				assert.True(t, found, "Request completed log not found")
			},
		},
		{
			name:          "slow request triggers warning",
			requestPath:   "/api/slow",
			requestMethod: "GET",
			simulateDelay: 2 * time.Second,
			responseStatus: http.StatusOK,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				// Check for slow request warning
				found := false
				for _, log := range logs {
					if log.Message == "Slow request detected" {
						found = true
						assert.Equal(t, zapcore.WarnLevel, log.Level)
						break
					}
				}
				assert.True(t, found, "Slow request warning not found")
			},
		},
		{
			name:          "error response logged correctly",
			requestPath:   "/api/error",
			requestMethod: "POST",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error":"Internal server error"}`,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				found := false
				for _, log := range logs {
					if log.Message == "Request completed" {
						found = true
						assert.Equal(t, zapcore.ErrorLevel, log.Level)
						statusField := log.ContextMap()["status"]
						assert.Equal(t, int64(500), statusField)
						break
					}
				}
				assert.True(t, found, "Error log not found")
			},
		},
		{
			name:          "skip logging for health endpoint",
			requestPath:   "/health",
			requestMethod: "GET",
			responseStatus: http.StatusOK,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				// Should not log health checks
				for _, log := range logs {
					if log.Message == "Request completed" {
						path := log.ContextMap()["path"]
						assert.NotEqual(t, "/health", path)
					}
				}
			},
		},
		{
			name:          "sensitive data redacted from logs",
			requestPath:   "/api/login",
			requestMethod: "POST",
			requestBody:   `{"email":"user@example.com","password":"secret123","ssn":"123-45-6789"}`,
			responseStatus: http.StatusOK,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				// Check that sensitive data is redacted
				for _, log := range logs {
					logStr := log.Message
					for _, field := range log.Context {
						if str, ok := field.Interface.(string); ok {
							logStr += " " + str
						}
					}
					assert.NotContains(t, logStr, "secret123")
					assert.NotContains(t, logStr, "123-45-6789")
					// Email might be partially visible
					if log.ContextMap()["body"] != nil {
						body := log.ContextMap()["body"].(string)
						assert.Contains(t, body, "[REDACTED]")
					}
				}
			},
		},
		{
			name:          "agent execution tracking",
			requestPath:   "/api/agents/execute",
			requestMethod: "POST",
			setupContext: func(c *gin.Context) {
				taskCtx := &agents.TaskContext{
					ID:        uuid.New(),
					AgentType: agents.QualityAgent,
					Phase:     agents.PhaseQuality,
				}
				c.Set("task_context", taskCtx)
				c.Set("agent_confidence", 8.5)
				c.Set("agent_execution_time", 150*time.Millisecond)
			},
			responseStatus: http.StatusOK,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				found := false
				for _, log := range logs {
					if log.Message == "Agent execution completed" {
						found = true
						assert.Contains(t, log.ContextMap(), "agent_type")
						assert.Contains(t, log.ContextMap(), "confidence")
						assert.Contains(t, log.ContextMap(), "execution_time_ms")
						assert.Equal(t, "quality", log.ContextMap()["agent_type"])
						assert.Equal(t, 8.5, log.ContextMap()["confidence"])
						break
					}
				}
				assert.True(t, found, "Agent execution log not found")
			},
		},
		{
			name:          "request with tracing context",
			requestPath:   "/api/traced",
			requestMethod: "GET",
			responseStatus: http.StatusOK,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				found := false
				for _, log := range logs {
					if log.Message == "Request completed" {
						found = true
						// Should have trace and span IDs
						assert.Contains(t, log.ContextMap(), "trace_id")
						assert.Contains(t, log.ContextMap(), "span_id")
						break
					}
				}
				assert.True(t, found, "Traced request log not found")
			},
		},
		{
			name:          "panic recovery logging",
			requestPath:   "/api/panic",
			requestMethod: "GET",
			responseStatus: http.StatusInternalServerError,
			checkLogs: func(t *testing.T, logs []observer.LoggedEntry) {
				found := false
				for _, log := range logs {
					if log.Message == "Panic recovered" {
						found = true
						assert.Equal(t, zapcore.ErrorLevel, log.Level)
						assert.Contains(t, log.ContextMap(), "panic")
						assert.Contains(t, log.ContextMap(), "stack")
						break
					}
				}
				assert.True(t, found, "Panic recovery log not found")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create observer to capture logs
			core, observed := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)
			
			// Setup tracer
			tp := trace.NewTracerProvider(
				trace.WithSyncer(tracetest.NewInMemoryExporter()),
			)
			otel.SetTracerProvider(tp)
			
			// Create middleware
			config := &LoggingConfig{
				SkipPaths:       []string{"/health", "/metrics"},
				SlowRequestTime: 1 * time.Second,
				EnableTracing:   true,
			}
			middleware := NewLoggingMiddleware(logger, config)
			
			// Create test router
			router := gin.New()
			router.Use(middleware.Handle())
			
			// Add test handlers
			router.Any("/*path", func(c *gin.Context) {
				// Setup context if needed
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				
				// Simulate delay
				if tt.simulateDelay > 0 {
					time.Sleep(tt.simulateDelay)
				}
				
				// Simulate panic if testing panic recovery
				if c.Request.URL.Path == "/api/panic" {
					panic("test panic")
				}
				
				// Log agent execution if context present
				if taskCtx, exists := c.Get("task_context"); exists {
					if ctx, ok := taskCtx.(*agents.TaskContext); ok {
						middleware.logAgentExecution(c, ctx)
					}
				}
				
				// Return response
				if tt.responseBody != "" {
					c.Data(tt.responseStatus, "application/json", []byte(tt.responseBody))
				} else {
					c.Status(tt.responseStatus)
				}
			})
			
			// Create request
			var req *http.Request
			if tt.requestBody != "" {
				req = httptest.NewRequest(tt.requestMethod, tt.requestPath, bytes.NewBufferString(tt.requestBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.requestMethod, tt.requestPath, nil)
			}
			
			// Add tracing headers if testing tracing
			if tt.name == "request with tracing context" {
				req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
			}
			
			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Allow time for async logging
			time.Sleep(10 * time.Millisecond)
			
			// Check response status
			assert.Equal(t, tt.responseStatus, w.Code)
			
			// Check logs
			if tt.checkLogs != nil {
				logs := observed.All()
				tt.checkLogs(t, logs)
			}
		})
	}
}

func TestLoggingMiddleware_RedactSensitiveData(t *testing.T) {
	logger := zap.NewNop()
	config := &LoggingConfig{}
	middleware := NewLoggingMiddleware(logger, config)
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redact password",
			input:    `{"username":"john","password":"secret123"}`,
			expected: `{"username":"john","password":"[REDACTED]"}`,
		},
		{
			name:     "redact API key",
			input:    `{"api_key":"sk-1234567890abcdef","data":"test"}`,
			expected: `{"api_key":"[REDACTED]","data":"test"}`,
		},
		{
			name:     "redact token",
			input:    `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`,
			expected: `Authorization: Bearer [REDACTED]`,
		},
		{
			name:     "redact credit card",
			input:    `{"card":"4111111111111111","cvv":"123"}`,
			expected: `{"card":"[REDACTED]","cvv":"[REDACTED]"}`,
		},
		{
			name:     "redact SSN",
			input:    `{"ssn":"123-45-6789","name":"John"}`,
			expected: `{"ssn":"[REDACTED]","name":"John"}`,
		},
		{
			name:     "redact email partially",
			input:    `{"email":"user@example.com","id":123}`,
			expected: `{"email":"u***@example.com","id":123}`,
		},
		{
			name:     "multiple sensitive fields",
			input:    `{"password":"pass123","token":"abc123","api_key":"key123"}`,
			expected: `{"password":"[REDACTED]","token":"[REDACTED]","api_key":"[REDACTED]"}`,
		},
		{
			name:     "no sensitive data",
			input:    `{"name":"John","age":30,"city":"NYC"}`,
			expected: `{"name":"John","age":30,"city":"NYC"}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.redactSensitiveData(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoggingMiddleware_ExtractTraceContext(t *testing.T) {
	// Setup tracer
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	
	logger := zap.NewNop()
	config := &LoggingConfig{EnableTracing: true}
	middleware := NewLoggingMiddleware(logger, config)
	
	// Create request with W3C trace context
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	
	// Extract context
	ctx := middleware.extractTraceContext(req)
	
	// Verify context has trace information
	spanCtx := trace.SpanContextFromContext(ctx)
	assert.True(t, spanCtx.IsValid())
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spanCtx.TraceID().String())
	assert.Equal(t, "00f067aa0ba902b7", spanCtx.SpanID().String())
}

func TestLoggingMiddleware_StructuredLogging(t *testing.T) {
	// Create observer to capture structured logs
	core, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	
	config := &LoggingConfig{}
	middleware := NewLoggingMiddleware(logger, config)
	
	// Create test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Handle())
	
	router.POST("/api/test", func(c *gin.Context) {
		// Add some context
		taskCtx := &agents.TaskContext{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			AgentType: agents.AnalysisAgent,
			Metadata: map[string]string{
				"version": "1.0",
				"source":  "test",
			},
		}
		c.Set("task_context", taskCtx)
		c.JSON(http.StatusOK, gin.H{"result": "success"})
	})
	
	// Make request
	body := `{"input":"test data"}`
	req := httptest.NewRequest("POST", "/api/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestClient/1.0")
	req.Header.Set("X-Request-ID", "test-request-123")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Check structured fields in logs
	logs := observed.All()
	require.GreaterOrEqual(t, len(logs), 1)
	
	var requestLog observer.LoggedEntry
	for _, log := range logs {
		if log.Message == "Request completed" {
			requestLog = log
			break
		}
	}
	
	// Verify structured fields
	fields := requestLog.ContextMap()
	assert.Equal(t, "POST", fields["method"])
	assert.Equal(t, "/api/test", fields["path"])
	assert.Equal(t, int64(200), fields["status"])
	assert.Equal(t, "TestClient/1.0", fields["user_agent"])
	assert.Equal(t, "test-request-123", fields["request_id"])
	assert.NotNil(t, fields["task_id"])
	assert.NotNil(t, fields["tenant_id"])
	assert.Equal(t, "analysis", fields["agent_type"])
	assert.NotNil(t, fields["duration_ms"])
	assert.Greater(t, fields["duration_ms"], float64(0))
}

func TestLoggingMiddleware_PerformanceMetrics(t *testing.T) {
	// Create observer
	core, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	
	config := &LoggingConfig{
		SlowRequestTime: 100 * time.Millisecond,
	}
	middleware := NewLoggingMiddleware(logger, config)
	
	// Create router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Handle())
	
	router.GET("/fast", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(150 * time.Millisecond)
		c.Status(http.StatusOK)
	})
	
	// Test fast request
	req := httptest.NewRequest("GET", "/fast", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Test slow request
	req = httptest.NewRequest("GET", "/slow", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Check logs
	time.Sleep(10 * time.Millisecond)
	logs := observed.All()
	
	// Find slow request warning
	slowWarningFound := false
	for _, log := range logs {
		if log.Message == "Slow request detected" {
			slowWarningFound = true
			fields := log.ContextMap()
			assert.Equal(t, "/slow", fields["path"])
			assert.Greater(t, fields["duration_ms"], float64(100))
		}
	}
	assert.True(t, slowWarningFound, "Slow request warning not found")
}

func TestLoggingMiddleware_ErrorRecovery(t *testing.T) {
	// Create observer
	core, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	
	config := &LoggingConfig{}
	middleware := NewLoggingMiddleware(logger, config)
	
	// Create router with panic recovery
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery()) // Gin's recovery should come first
	router.Use(middleware.Handle())
	
	router.GET("/panic", func(c *gin.Context) {
		panic("intentional panic for testing")
	})
	
	// Make request
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	
	// Capture panic and continue
	defer func() {
		if r := recover(); r != nil {
			// Expected panic, continue test
		}
	}()
	
	router.ServeHTTP(w, req)
	
	// Check error was logged
	time.Sleep(10 * time.Millisecond)
	logs := observed.All()
	
	errorLogFound := false
	for _, log := range logs {
		if log.Level == zapcore.ErrorLevel {
			errorLogFound = true
			break
		}
	}
	assert.True(t, errorLogFound, "Error log for panic not found")
}

func TestLoggingMiddleware_RequestBodyCapture(t *testing.T) {
	// Create observer
	core, observed := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	
	config := &LoggingConfig{
		LogRequestBody: true,
		MaxBodySize:    1024,
	}
	middleware := NewLoggingMiddleware(logger, config)
	
	// Create router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Handle())
	
	router.POST("/api/data", func(c *gin.Context) {
		var data map[string]interface{}
		c.ShouldBindJSON(&data)
		c.JSON(http.StatusOK, data)
	})
	
	// Test with small body
	smallBody := `{"key":"value","number":42}`
	req := httptest.NewRequest("POST", "/api/data", bytes.NewBufferString(smallBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Test with large body (should be truncated)
	largeBody := `{"data":"` + string(make([]byte, 2000)) + `"}`
	req = httptest.NewRequest("POST", "/api/data", bytes.NewBufferString(largeBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Check logs
	time.Sleep(10 * time.Millisecond)
	logs := observed.All()
	
	// Verify body was captured and sensitive data redacted
	for _, log := range logs {
		if log.Message == "Request completed" {
			fields := log.ContextMap()
			if body, ok := fields["body"].(string); ok {
				// Should not exceed max size
				assert.LessOrEqual(t, len(body), 1024)
			}
		}
	}
}