package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sormind/OSA/miosa-backend/internal/agents"
	"github.com/sormind/OSA/miosa-backend/internal/config"
	"go.uber.org/zap"
)

// Claims represents JWT claims with tenant context
type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Scopes      []string  `json:"scopes"`
	jwt.RegisteredClaims
}

// AuthMiddleware handles JWT validation and tenant context
type AuthMiddleware struct {
	jwtSecret      []byte
	db             *sql.DB
	redis          redis.UniversalClient
	logger         *zap.Logger
	skipPaths      []string
	requiredScopes map[string][]string // path -> required scopes
}

// NewAuthMiddleware creates a new auth middleware instance
func NewAuthMiddleware(config *config.AuthConfig, db *sql.DB, redis redis.UniversalClient, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(config.JWTSecret),
		db:        db,
		redis:     redis,
		logger:    logger,
		skipPaths: []string{
			"/health",
			"/metrics",
			"/api/auth/login",
			"/api/auth/register",
		},
		requiredScopes: map[string][]string{
			"/api/agents":         {"agents:read"},
			"/api/agents/execute": {"agents:execute"},
			"/api/workspaces":     {"workspaces:read"},
			"/api/admin":          {"admin:all"},
		},
	}
}

// Handle processes authentication for incoming requests
func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if m.shouldSkipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract JWT from Authorization header
		token, err := m.extractToken(c)
		if err != nil {
			m.logger.Debug("Token extraction failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid authorization header",
			})
			c.Abort()
			return
		}

		// Parse and validate JWT
		claims, err := m.validateToken(token)
		if err != nil {
			m.logger.Warn("Token validation failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Check if token is blacklisted (e.g., after logout)
		if m.isTokenBlacklisted(c.Request.Context(), token) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token has been revoked",
			})
			c.Abort()
			return
		}

		// Verify tenant access
		if !m.hastenantAccess(c.Request.Context(), claims.UserID, claims.TenantID) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied for this tenant",
			})
			c.Abort()
			return
		}

		// Check required scopes for the path
		if !m.hasRequiredScopes(c.Request.URL.Path, claims.Scopes) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		// Create TaskContext for agent system
		taskContext := &agents.TaskContext{
			UserID:      claims.UserID,
			TenantID:    claims.TenantID,
			WorkspaceID: claims.WorkspaceID,
			Phase:       m.determinePhase(c.Request.URL.Path),
			Metadata: map[string]string{
				"email":      claims.Email,
				"role":       claims.Role,
				"request_id": c.GetHeader("X-Request-ID"),
			},
		}

		// Set context for downstream handlers
		c.Set("user_id", claims.UserID.String())
		c.Set("tenant_id", claims.TenantID.String())
		c.Set("workspace_id", claims.WorkspaceID.String())
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("user_scopes", claims.Scopes)
		c.Set("task_context", taskContext)
		c.Set("claims", claims)

		// Log successful authentication
		m.logger.Debug("Request authenticated",
			zap.String("user_id", claims.UserID.String()),
			zap.String("tenant_id", claims.TenantID.String()),
			zap.String("path", c.Request.URL.Path))

		c.Next()
	}
}

// extractToken extracts JWT from Authorization header
func (m *AuthMiddleware) extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	// Expected format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization format")
	}

	return parts[1], nil
}

// validateToken validates JWT and returns claims
func (m *AuthMiddleware) validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// isTokenBlacklisted checks if token has been revoked
func (m *AuthMiddleware) isTokenBlacklisted(ctx context.Context, token string) bool {
	key := fmt.Sprintf("blacklist:token:%s", token)
	
	exists, err := m.redis.Exists(ctx, key).Result()
	if err != nil {
		m.logger.Error("Failed to check token blacklist",
			zap.Error(err))
		return false // Allow on error to prevent lockout
	}

	return exists > 0
}

// hastenantAccess verifies user has access to tenant
func (m *AuthMiddleware) hastenantAccess(ctx context.Context, userID, tenantID uuid.UUID) bool {
	// Check if user belongs to tenant
	var count int
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM user_tenants 
		WHERE user_id = $1 AND tenant_id = $2 AND active = true
	`, userID, tenantID).Scan(&count)

	if err != nil {
		m.logger.Error("Failed to verify tenant access",
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.String("tenant_id", tenantID.String()))
		return false
	}

	return count > 0
}

// hasRequiredScopes checks if user has required scopes for path
func (m *AuthMiddleware) hasRequiredScopes(path string, userScopes []string) bool {
	// Check for admin override
	for _, scope := range userScopes {
		if scope == "admin:all" {
			return true
		}
	}

	// Find required scopes for path
	requiredScopes, exists := m.requiredScopes[path]
	if !exists {
		// No specific scopes required
		return true
	}

	// Check if user has at least one required scope
	scopeMap := make(map[string]bool)
	for _, scope := range userScopes {
		scopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if scopeMap[required] {
			return true
		}
	}

	return false
}

// shouldSkipAuth checks if path should skip authentication
func (m *AuthMiddleware) shouldSkipAuth(path string) bool {
	for _, skipPath := range m.skipPaths {
		if path == skipPath || strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// determinePhase determines agent phase from request path
func (m *AuthMiddleware) determinePhase(path string) string {
	switch {
	case strings.Contains(path, "/consultation"):
		return string(agents.PhaseConsultation)
	case strings.Contains(path, "/analyze"):
		return string(agents.PhaseAnalysis)
	case strings.Contains(path, "/develop"):
		return string(agents.PhaseDevelopment)
	case strings.Contains(path, "/deploy"):
		return string(agents.PhaseDeployment)
	case strings.Contains(path, "/monitor"):
		return string(agents.PhaseMonitoring)
	default:
		return string(agents.PhaseStrategy)
	}
}

// GenerateToken generates a new JWT token
func (m *AuthMiddleware) GenerateToken(claims *Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.jwtSecret)
}

// RefreshToken refreshes an existing token
func (m *AuthMiddleware) RefreshToken(tokenString string) (string, error) {
	// Parse existing token
	claims, err := m.validateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Create new token with extended expiry
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	return m.GenerateToken(claims)
}

// RevokeToken adds token to blacklist
func (m *AuthMiddleware) RevokeToken(ctx context.Context, token string, expiry time.Duration) error {
	key := fmt.Sprintf("blacklist:token:%s", token)
	return m.redis.Set(ctx, key, true, expiry).Err()
}

// ValidateAPIKey validates API key for service-to-service auth
func (m *AuthMiddleware) ValidateAPIKey(apiKey string) (*Claims, error) {
	// Query database for API key
	var claims Claims
	err := m.db.QueryRow(`
		SELECT user_id, tenant_id, workspace_id, email, role, scopes
		FROM api_keys
		WHERE key_hash = crypt($1, key_hash) AND active = true
	`, apiKey).Scan(
		&claims.UserID,
		&claims.TenantID,
		&claims.WorkspaceID,
		&claims.Email,
		&claims.Role,
		&claims.Scopes,
	)

	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	return &claims, nil
}