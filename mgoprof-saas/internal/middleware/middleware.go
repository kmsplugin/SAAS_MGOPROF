// Package middleware provides Gin middleware for MGOPROF.
package middleware

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// Auth validates a Bearer JWT from the Authorization header OR ?token= query param.
// The query-param fallback enables browser-accessible admin HTML pages.
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""

		// 1. Authorization: Bearer <token>
		if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		// 2. ?token=<jwt>  (browser fallback for HTML report pages)
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Status:  "error",
				Message: "требуется авторизация",
			})
			return
		}

		userID, role, email, err := authSvc.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorResponse{
				Status:  "error",
				Message: "недействительный токен",
			})
			return
		}

		c.Set("user_id", userID)
		c.Set("role", role)
		c.Set("email", email)
		c.Next()
	}
}

// RequireRole aborts with 403 if the JWT role is not in the allowed list.
// Accepts one or more roles: RequireRole("admin") or RequireRole("admin","operator").
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
			Status:  "error",
			Message: "доступ запрещён",
		})
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

// Logger logs every request with status, latency and IP.
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
			zap.String("ip", ExtractIP(c)),
		)
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

// ── Rate limiter ──────────────────────────────────────────────────────────────

type ipEntry struct {
	count     int
	windowEnd time.Time
}

// RateLimiter is a simple in-memory, per-IP sliding-window limiter.
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*ipEntry
	limit    int
	window   time.Duration
	lastGC   time.Time
}

// NewRateLimiter creates a limiter allowing `limit` requests per `window` per IP.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*ipEntry),
		limit:   limit,
		window:  window,
		lastGC:  time.Now(),
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Periodic GC to prevent unbounded map growth.
	if now.Sub(rl.lastGC) > 5*time.Minute {
		for k, e := range rl.entries {
			if now.After(e.windowEnd) {
				delete(rl.entries, k)
			}
		}
		rl.lastGC = now
	}

	e, ok := rl.entries[ip]
	if !ok || now.After(e.windowEnd) {
		rl.entries[ip] = &ipEntry{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	e.count++
	return e.count <= rl.limit
}

// Limit returns a Gin middleware that rate-limits by IP.
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(ExtractIP(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.ErrorResponse{
				Status:  "error",
				Message: "Слишком много запросов. Подождите немного.",
			})
			return
		}
		c.Next()
	}
}
