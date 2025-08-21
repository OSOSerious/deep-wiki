package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggingMiddleware provides structured logging with trace IDs and GDPR compliance
type LoggingMiddleware struct {
	logger           *zap.Logger
	tracer           trace.Tracer
	propagator       propagation.TextMapPropagator
	skipPaths        []string
	slowRequestTime  time.Duration
	logRequestBody   bool
	logResponseBody  bool
	maxBodySize      int64
	sensitiveFields  []string
	redactPatterns   []*regexp.Regexp
	loggerPool       *sync.Pool
}

// responseWriter wraps gin.ResponseWriter to capture response
type responseWriter struct {
	gin.ResponseWriter
	body        *bytes.Buffer
	status      int
	written     bool
	captureBody bool
}

// LoggingConfig holds configuration for the logging middleware
type LoggingConfig struct {
	Level            string
	Environment      string
	SkipPaths        []string
	SlowRequestTime  time.Duration
	LogRequestBody   bool
	LogResponseBody  bool
	MaxBodySize      int64
	SensitiveFields  []string
	EnableOTel       bool
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:           "info",
		Environment:     "development",
		SkipPaths:       []string{"/health", "/metrics", "/favicon.ico"},
		SlowRequestTime: 5 * time.Second,
		LogRequestBody:  true,
		LogResponseBody: false,
		MaxBodySize:     1024 * 1024, // 1MB
		SensitiveFields: []string{
			"password", "secret", "token", "api_key", "apikey",
			"authorization", "credit_card", "ssn", "tax_id",
		},
		EnableOTel: true,
	}
}

// NewLoggingMiddleware creates a new logging middleware with OpenTelemetry support
func NewLoggingMiddleware(config *LoggingConfig) (*LoggingMiddleware, error) {
	logger, err := CreateLogger(config.Level, config.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	m := &LoggingMiddleware{
		logger:          logger,
		skipPaths:       config.SkipPaths,
		slowRequestTime: config.SlowRequestTime,
		logRequestBody:  config.LogRequestBody,
		logResponseBody: config.LogResponseBody,
		maxBodySize:     config.MaxBodySize,
		sensitiveFields: config.SensitiveFields,
		loggerPool: &sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}

	// Initialize OpenTelemetry if enabled
	if config.EnableOTel {
		m.tracer = otel.Tracer("miosa-backend")
		m.propagator = otel.GetTextMapPropagator()
	}

	// Compile redaction patterns for GDPR compliance
	m.compileRedactPatterns()

	return m, nil
}

// Handle processes logging for requests and responses with distributed tracing
func (m *LoggingMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		if m.shouldSkipLogging(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Extract or generate trace context
		ctx := m.extractTraceContext(c)
		traceID := m.getOrGenerateTraceID(ctx)
		spanID := uuid.New().String()[:16] // Use first 16 chars of UUID for span ID

		// Start OpenTelemetry span if enabled
		var span trace.Span
		if m.tracer != nil {
			ctx, span = m.tracer.Start(ctx, fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
				trace.WithAttributes(
					attribute.String("http.method", c.Request.Method),
					attribute.String("http.url", c.Request.URL.String()),
					attribute.String("http.user_agent", c.Request.UserAgent()),
					attribute.String("http.remote_addr", c.ClientIP()),
				),
			)
			defer span.End()
		}

		// Store trace info in context
		c.Set("trace_id", traceID)
		c.Set("span_id", spanID)
		c.Set("trace_context", ctx)
		c.Header("X-Trace-ID", traceID)

		// Get tenant and user context from auth middleware
		var tenantID, userID, workspaceID string
		if taskCtx, exists := c.Get("task_context"); exists {
			if tCtx, ok := taskCtx.(*agents.TaskContext); ok {
				tenantID = tCtx.TenantID.String()
				userID = tCtx.UserID.String()
				workspaceID = tCtx.WorkspaceID.String()
				// Add trace ID to task context metadata
				tCtx.Metadata["trace_id"] = traceID
				tCtx.Metadata["span_id"] = spanID

				// Add to OpenTelemetry span
				if span != nil {
					span.SetAttributes(
						attribute.String("tenant.id", tenantID),
						attribute.String("user.id", userID),
						attribute.String("workspace.id", workspaceID),
					)
				}
			}
		}

		// Create request-scoped logger with context
		reqLogger := m.logger.With(
			zap.String("trace_id", traceID),
			zap.String("span_id", spanID),
			zap.String("tenant_id", tenantID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", m.redactSensitive(c.Request.URL.RawQuery)),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		// Store logger in context for use in handlers
		c.Set("logger", reqLogger)

		// Log request body if enabled (with redaction)
		var requestBody []byte
		if m.logRequestBody && c.Request.Body != nil {
			requestBody, _ = m.readBody(c.Request)
			if len(requestBody) > 0 {
				redactedBody := m.redactJSON(requestBody)
				reqLogger.Debug("Request body",
					zap.ByteString("body", redactedBody))
			}
		}

		// Wrap response writer to capture response
		blw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
			captureBody:    m.logResponseBody,
		}
		c.Writer = blw

		// Process request with panic recovery
		func() {
			defer func() {
				if err := recover(); err != nil {
					reqLogger.Error("Panic recovered",
						zap.Any("error", err),
						zap.Stack("stack"))
					if span != nil {
						span.RecordError(fmt.Errorf("panic: %v", err))
					}
					c.AbortWithStatus(500)
				}
			}()
			c.Next()
		}()

		// Calculate latency
		latency := time.Since(start)

		// Get response status
		status := blw.status
		if status == 0 {
			status = 200
		}

		// Update OpenTelemetry span
		if span != nil {
			span.SetAttributes(
				attribute.Int("http.status_code", status),
				attribute.Int64("http.response_size", int64(blw.body.Len())),
				attribute.Int64("http.duration_ms", latency.Milliseconds()),
			)
			if status >= 400 {
				span.RecordError(fmt.Errorf("HTTP %d", status))
			}
		}

		// Build log fields
		fields := []zapcore.Field{
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int64("latency_ms", latency.Milliseconds()),
			zap.Int("response_size", blw.body.Len()),
		}

		// Add agent execution metrics if available
		if agentResult, exists := c.Get("agent_result"); exists {
			if result, ok := agentResult.(*agents.Result); ok {
				fields = append(fields,
					zap.Bool("agent_success", result.Success),
					zap.Float64("agent_confidence", result.Confidence),
					zap.Int64("agent_execution_ms", result.ExecutionMS),
				)
				if span != nil {
					span.SetAttributes(
						attribute.Bool("agent.success", result.Success),
						attribute.Float64("agent.confidence", result.Confidence),
						attribute.Int64("agent.execution_ms", result.ExecutionMS),
					)
				}
			}
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields = append(fields,
				zap.String("error", c.Errors.String()))
			if span != nil {
				for _, err := range c.Errors {
					span.RecordError(err)
				}
			}
		}

		// Choose log level based on status code and latency
		switch {
		case status >= 500:
			reqLogger.Error("Request completed", fields...)
		case status >= 400:
			reqLogger.Warn("Request completed", fields...)
		case latency > m.slowRequestTime:
			reqLogger.Warn("Slow request detected", fields...)
			m.logSlowRequest(reqLogger, c, latency, requestBody)
		default:
			reqLogger.Info("Request completed", fields...)
		}
	}
}

// extractTraceContext extracts OpenTelemetry trace context from headers
func (m *LoggingMiddleware) extractTraceContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	if m.propagator != nil {
		ctx = m.propagator.Extract(ctx, propagation.HeaderCarrier(c.Request.Header))
	}
	return ctx
}

// getOrGenerateTraceID gets trace ID from context or generates new one
func (m *LoggingMiddleware) getOrGenerateTraceID(ctx context.Context) string {
	if span := trace.SpanFromContext(ctx); span != nil {
		if span.SpanContext().HasTraceID() {
			return span.SpanContext().TraceID().String()
		}
	}
	return uuid.New().String()
}

// Write captures the response
func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(200)
	}
	if w.captureBody && w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// WriteHeader captures the response status
func (w *responseWriter) WriteHeader(code int) {
	if !w.written {
		w.status = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

// readBody reads request body and restores it
func (m *LoggingMiddleware) readBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, m.maxBodySize))
	if err != nil {
		return nil, err
	}

	// Restore body for handlers
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return body, nil
}

// compileRedactPatterns compiles regex patterns for sensitive data redaction
func (m *LoggingMiddleware) compileRedactPatterns() {
	patterns := []string{
		`"password"\s*:\s*"[^"]*"`,                    // JSON password fields
		`"token"\s*:\s*"[^"]*"`,                       // JSON token fields
		`"api_key"\s*:\s*"[^"]*"`,                     // JSON API key fields
		`Bearer\s+[A-Za-z0-9\-._~+/]+=*`,                // Bearer tokens
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, // Email addresses
		`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,      // Credit card numbers
		`\b\d{3}-\d{2}-\d{4}\b`,                          // SSN format
	}

	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			m.redactPatterns = append(m.redactPatterns, re)
		}
	}
}

// redactSensitive redacts sensitive information from string
func (m *LoggingMiddleware) redactSensitive(input string) string {
	for _, pattern := range m.redactPatterns {
		input = pattern.ReplaceAllString(input, "[REDACTED]")
	}
	return input
}

// redactJSON redacts sensitive fields from JSON data
func (m *LoggingMiddleware) redactJSON(data []byte) []byte {
	if !json.Valid(data) {
		return data
	}

	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}

	m.redactMap(obj)

	redacted, err := json.Marshal(obj)
	if err != nil {
		return data
	}

	return redacted
}

// redactMap recursively redacts sensitive fields in maps and slices
func (m *LoggingMiddleware) redactMap(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		for key, value := range val {
			lowerKey := strings.ToLower(key)
			for _, field := range m.sensitiveFields {
				if strings.Contains(lowerKey, strings.ToLower(field)) {
					val[key] = "[REDACTED]"
					break
				}
			}
			if _, ok := val[key].(string); !ok {
				m.redactMap(value)
			}
		}
	case []interface{}:
		for _, item := range val {
			m.redactMap(item)
		}
	}
}

// shouldSkipLogging checks if path should skip logging
func (m *LoggingMiddleware) shouldSkipLogging(path string) bool {
	for _, skipPath := range m.skipPaths {
		if path == skipPath || strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// logSlowRequest logs detailed information about slow requests
func (m *LoggingMiddleware) logSlowRequest(logger *zap.Logger, c *gin.Context, latency time.Duration, body []byte) {
	// Get buffer from pool
	buf := m.loggerPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		m.loggerPool.Put(buf)
	}()

	// Build detailed slow request info
	fields := []zapcore.Field{
		zap.Duration("latency", latency),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", m.redactSensitive(c.Request.URL.RawQuery)),
		zap.String("alert", "SLOW_REQUEST"),
	}

	// Add request body if available
	if len(body) > 0 {
		redactedBody := m.redactJSON(body)
		fields = append(fields, zap.ByteString("request_body", redactedBody))
	}

	// Add headers (redacted)
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "authorization") || strings.Contains(lowerKey, "token") {
			headers[key] = "[REDACTED]"
		} else {
			headers[key] = strings.Join(values, ", ")
		}
	}
	fields = append(fields, zap.Any("headers", headers))

	logger.Warn("Slow request detected", fields...)
}

// GetLogger retrieves the request-scoped logger from context
func GetLogger(c *gin.Context) *zap.Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*zap.Logger); ok {
			return l
		}
	}
	return zap.NewNop()
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("trace_id"); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// GetSpanID retrieves the span ID from context
func GetSpanID(c *gin.Context) string {
	if spanID, exists := c.Get("span_id"); exists {
		if id, ok := spanID.(string); ok {
			return id
		}
	}
	return ""
}

// LogAgentExecution logs agent execution details with tracing
func LogAgentExecution(c *gin.Context, agentType agents.AgentType, result *agents.Result) {
	logger := GetLogger(c)
	traceID := GetTraceID(c)
	spanID := GetSpanID(c)

	// Get OpenTelemetry span if available
	if ctx, exists := c.Get("trace_context"); exists {
		if traceCtx, ok := ctx.(context.Context); ok {
			if span := trace.SpanFromContext(traceCtx); span != nil {
				span.AddEvent("agent_execution",
					trace.WithAttributes(
						attribute.String("agent.type", string(agentType)),
						attribute.Bool("agent.success", result.Success),
						attribute.Float64("agent.confidence", result.Confidence),
						attribute.Int64("agent.execution_ms", result.ExecutionMS),
					),
				)
			}
		}
	}

	fields := []zapcore.Field{
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("agent_type", string(agentType)),
		zap.Float64("confidence", result.Confidence),
		zap.Int64("execution_ms", result.ExecutionMS),
	}

	if result.Success {
		if result.NextAgent != "" {
			fields = append(fields, zap.String("next_agent", string(result.NextAgent)))
		}
		logger.Info("Agent execution completed", fields...)
	} else {
		if result.Error != nil {
			fields = append(fields, zap.Error(result.Error))
		}
		logger.Error("Agent execution failed", fields...)
	}

	// Store result for request logging
	c.Set("agent_result", result)
}

// CreateLogger creates a configured zap logger optimized for production
func CreateLogger(level string, environment string) (*zap.Logger, error) {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
		// Enable sampling to reduce log volume in production
		config.Sampling = &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
		// Use JSON encoder for structured logging
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeDuration = zapcore.MillisDurationEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		// Use console encoder for human-readable output
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
	}

	// Parse log level
	var zapLevel zapcore.Level
	if err := zapLevel.Set(level); err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", level, err)
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Add service info
	config.InitialFields = map[string]interface{}{
		"service": "miosa-backend",
		"version": "1.0.0",
	}

	// Build logger with additional options
	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1), // Skip wrapper functions
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	// Replace global logger for libraries that use it
	zap.ReplaceGlobals(logger)

	return logger, nil
}

// Sync flushes any buffered log entries
func (m *LoggingMiddleware) Sync() error {
	return m.logger.Sync()
}