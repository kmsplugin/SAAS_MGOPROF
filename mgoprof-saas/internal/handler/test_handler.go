package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// TestHandler exposes test-only endpoints.
// These routes MUST only be registered when TEST_MODE=true.
// They are never compiled out — the guard is the route registration in main.go.
type TestHandler struct {
	regRepo *repository.RegistrationRepository
	logger  *zap.Logger
}

func NewTestHandler(regRepo *repository.RegistrationRepository, logger *zap.Logger) *TestHandler {
	return &TestHandler{regRepo: regRepo, logger: logger}
}

// RegisterRoutes wires test endpoints under /api/test.
// Call this only when cfg.TestMode == true.
func (h *TestHandler) RegisterRoutes(r *gin.RouterGroup) {
	t := r.Group("/test")
	t.GET("/last-otp", h.LastOTP)
}

// LastOTP returns the current (unexpired) OTP code for the given email.
//
// GET /api/test/last-otp?email=smoke@example.com
//
// Response 200: {"email":"...","otp":"123456"}
// Response 404: no pending OTP found
//
// Security contract:
//   - Only reachable when TEST_MODE=true (route not registered otherwise)
//   - Returns 404 (not an error) when no valid OTP exists — safe to probe
//   - Never logs the OTP value to prevent leaking into CI logs
func (h *TestHandler) LastOTP(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "параметр email обязателен",
		})
		return
	}

	code, err := h.regRepo.GetLastOTP(c.Request.Context(), email)
	if err != nil {
		h.logger.Error("test/last-otp: db error", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "ошибка базы данных",
		})
		return
	}
	if code == "" {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Status:  "not_found",
			Message: "нет активного OTP для указанного email",
		})
		return
	}

	// Intentionally not logging the OTP itself.
	h.logger.Info("test/last-otp: served", zap.String("email", email))
	c.JSON(http.StatusOK, gin.H{"email": email, "otp": code})
}
