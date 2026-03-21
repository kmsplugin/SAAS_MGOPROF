package middleware

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/api/internal/model"
)

// Auth validates a Bearer JWT and sets tenant_id, user_id, role into context.
func Auth(parseToken func(string) (string, string, string, string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Status: "error", Message: "требуется авторизация",
			})
			return
		}
		userID, tenantID, role, email, err := parseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Status: "error", Message: "недействительный токен",
			})
			return
		}
		c.Set("user_id", userID)
		c.Set("tenant_id", tenantID)
		c.Set("role", role)
		c.Set("email", email)
		c.Next()
	}
}

// RequireRole aborts with 403 if JWT role is not in the allowed list.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := c.GetString("role")
		for _, r := range roles {
			if got == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, model.ErrorResponse{
			Status: "error", Message: "доступ запрещён",
		})
	}
}

// CORS sets permissive CORS headers.
func CORS() gin.HandlerFunc {
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*"
	}
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Logger logs every request.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		log.Info("http",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}

// ExtractIP returns the real client IP.
func ExtractIP(c *gin.Context) string {
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return strings.SplitN(ip, ",", 2)[0]
	}
	return c.ClientIP()
}
