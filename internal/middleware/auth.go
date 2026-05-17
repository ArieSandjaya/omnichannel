package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ariesandjaya/omnichannel/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	jwt *jwt.Provider
}

func NewAuthMiddleware(jwtProvider *jwt.Provider) *AuthMiddleware {
	return &AuthMiddleware{jwt: jwtProvider}
}

// JWT validates the token from HttpOnly cookie (web) or Authorization header (API clients).
func (m *AuthMiddleware) JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		if cookie, err := c.Cookie("jwt"); err == nil {
			tokenStr = cookie
		} else {
			raw := c.GetHeader("Authorization")
			if !strings.HasPrefix(raw, "Bearer ") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth token"})
				return
			}
			tokenStr = strings.TrimPrefix(raw, "Bearer ")
		}

		claims, err := m.jwt.Parse(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	TenantID uuid.UUID `json:"tenant_id"`
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

// Auth validates the Bearer JWT and injects tenant_id, user_id, role into the context.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims := &Claims{}

		_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("tenant_id", claims.TenantID)
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// ResolveTenant ensures tenant_id is present in context (set by JWT middleware).
func (m *AuthMiddleware) ResolveTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("tenant_id") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "tenant not resolved"})
			return
		}
		c.Next()
	}
}

// RequireRole aborts with 403 if the user's role is not in the allowed list.
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role := c.GetString("role")
		if !allowed[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}

func (m *AuthMiddleware) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"ip", c.ClientIP(),
		)
	}
}

func (m *AuthMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
// TenantID extracts the tenant UUID from the Gin context.
// Panics if Auth middleware was not applied — intentional, this is a programming error.
func TenantID(c *gin.Context) uuid.UUID {
	return c.MustGet("tenant_id").(uuid.UUID)
}

// UserID extracts the user UUID from the Gin context.
func UserID(c *gin.Context) uuid.UUID {
	return c.MustGet("user_id").(uuid.UUID)
}
