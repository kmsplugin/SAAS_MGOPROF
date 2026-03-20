package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// Auth validates a Bearer JWT and stores userID / role in the context.
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Status:  "error",
				Message: "требуется авторизация",
			})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")

		userID, role, err := authSvc.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Status:  "error",
				Message: "недействительный токен",
			})
			return
		}

		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	}
}

// RequireRole aborts if the JWT role doesn't match.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("role") != role {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ErrorResponse{
				Status:  "error",
				Message: "доступ запрещён",
			})
			return
		}
		c.Next()
	}
}

// CORS sets CORS headers according to ALLOWED_ORIGIN env variable.
func CORS() gin.HandlerFunc {
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*"
	}
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ExtractIP returns the real client IP, preferring Cloudflare and X-Forwarded-For headers.
func ExtractIP(c *gin.Context) string {
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return strings.SplitN(ip, ",", 2)[0]
	}
	return c.ClientIP()
}
