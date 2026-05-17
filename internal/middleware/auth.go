package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	pkgjwt "github.com/ariesandjaya/omnichannel/pkg/jwt"
)

// AuthMiddleware validates the JWT from cookie or Authorization header.
// In mock mode (MOCK_MODE=true), injects demo claims without validation.
func AuthMiddleware(jwtSecret string, mockMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mockMode {
			c.Set("user_id", "00000000-0000-0000-0000-000000000001")
			c.Set("tenant_id", "00000000-0000-0000-0000-000000000001")
			c.Set("role", "owner")
			c.Next()
			return
		}

		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}

		claims, err := pkgjwt.Validate(tokenStr, jwtSecret)
		if err != nil {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	if cookie, err := c.Cookie("jwt"); err == nil && cookie != "" {
		return cookie
	}
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
