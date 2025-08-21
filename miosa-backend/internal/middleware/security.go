package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SecurityMiddleware provides security headers and CORS handling
type SecurityMiddleware struct {
	logger         *zap.Logger
	corsOrigins    []string
	enableHSTS     bool
	hstsMaxAge     int
	enableCSP      bool
	cspDirectives  string
}

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	CORSOrigins    []string
	EnableHSTS     bool
	HSTSMaxAge     int
	EnableCSP      bool
	CSPDirectives  string
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		CORSOrigins: []string{"*"},
		EnableHSTS:  true,
		HSTSMaxAge:  31536000, // 1 year
		EnableCSP:   true,
		CSPDirectives: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
	}
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(logger *zap.Logger, config *SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		logger:        logger,
		corsOrigins:   config.CORSOrigins,
		enableHSTS:    config.EnableHSTS,
		hstsMaxAge:    config.HSTSMaxAge,
		enableCSP:     config.EnableCSP,
		cspDirectives: config.CSPDirectives,
	}
}

// Handle applies security headers and CORS policies
func (m *SecurityMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers
		m.addSecurityHeaders(c)
		
		// Handle CORS
		m.handleCORS(c)
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// addSecurityHeaders adds security-related headers
func (m *SecurityMiddleware) addSecurityHeaders(c *gin.Context) {
	// Prevent clickjacking
	c.Header("X-Frame-Options", "DENY")
	
	// Prevent MIME type sniffing
	c.Header("X-Content-Type-Options", "nosniff")
	
	// Enable XSS protection
	c.Header("X-XSS-Protection", "1; mode=block")
	
	// Referrer policy
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	
	// HSTS (HTTP Strict Transport Security)
	if m.enableHSTS {
		c.Header("Strict-Transport-Security", 
			"max-age="+string(m.hstsMaxAge)+"; includeSubDomains; preload")
	}
	
	// Content Security Policy
	if m.enableCSP {
		c.Header("Content-Security-Policy", m.cspDirectives)
	}
	
	// Permission policy (formerly Feature Policy)
	c.Header("Permissions-Policy", 
		"camera=(), microphone=(), geolocation=(), payment=()")
}

// handleCORS handles Cross-Origin Resource Sharing
func (m *SecurityMiddleware) handleCORS(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	
	// Check if origin is allowed
	if m.isOriginAllowed(origin) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", 
			"GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", 
			"Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Tenant-ID")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours
	}
}

// isOriginAllowed checks if origin is in allowed list
func (m *SecurityMiddleware) isOriginAllowed(origin string) bool {
	if len(m.corsOrigins) == 0 {
		return false
	}
	
	for _, allowed := range m.corsOrigins {
		if allowed == "*" {
			return true
		}
		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		}
		if origin == allowed {
			return true
		}
	}
	
	return false
}