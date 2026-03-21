package middleware

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/api/internal/model"
)

// RateLimiter is a simple sliding-window IP-based rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	limit   int
	window  time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
	// Periodic cleanup every 5 minutes to avoid unbounded memory growth.
	go func() {
		for range time.Tick(5 * time.Minute) {
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.window)
			for ip, ts := range rl.buckets {
				start := 0
				for start < len(ts) && ts[start].Before(cutoff) {
					start++
				}
				if start == len(ts) {
					delete(rl.buckets, ip)
				} else {
					rl.buckets[ip] = ts[start:]
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := ExtractIP(c)
		now := time.Now()
		cutoff := now.Add(-rl.window)

		rl.mu.Lock()
		ts := rl.buckets[ip]
		// Drop timestamps outside the window.
		start := 0
		for start < len(ts) && ts[start].Before(cutoff) {
			start++
		}
		ts = ts[start:]
		if len(ts) >= rl.limit {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.ErrorResponse{
				Status:  "error",
				Code:    "RATE_LIMIT_EXCEEDED",
				Message: "слишком много запросов, попробуйте позже",
			})
			return
		}
		rl.buckets[ip] = append(ts, now)
		rl.mu.Unlock()
		c.Next()
	}
}

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
