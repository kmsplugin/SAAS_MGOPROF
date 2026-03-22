package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// RegistrationHandler exposes registration endpoints.
type RegistrationHandler struct {
	svc    *service.RegistrationService
	logger *zap.Logger
}

func NewRegistrationHandler(svc *service.RegistrationService, logger *zap.Logger) *RegistrationHandler {
	return &RegistrationHandler{svc: svc, logger: logger}
}

// HandleRegister is the Gin handler for POST /api/register.
func (h *RegistrationHandler) HandleRegister(c *gin.Context) { h.Register(c) }

// HandleVerifyOTP is the Gin handler for POST /api/verify-otp.
func (h *RegistrationHandler) HandleVerifyOTP(c *gin.Context) { h.VerifyOTP(c) }

// Register godoc
// @Summary     Регистрация на мероприятие
// @Tags        Registration
// @Accept      json
// @Produce     json
// @Param       body body model.RegisterRequest true "Данные участника"
// @Success     200 {object} service.RegisterResult
// @Failure     422 {object} model.ErrorResponse
// @Router      /api/register [post]
func (h *RegistrationHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, model.ErrorResponse{
			Status:  "error",
			Message: "Заполните обязательные поля.",
		})
		return
	}

	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()

	result, err := h.svc.Register(c.Request.Context(), req, ip, ua)
	if err != nil {
		h.logger.Error("registration failed",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// VerifyOTP godoc
// @Summary     Подтверждение OTP кода
// @Tags        Registration
// @Accept      json
// @Produce     json
// @Param       body body model.VerifyOTPRequest true "Email + OTP"
// @Success     200 {object} map[string]string
// @Failure     400 {object} model.ErrorResponse
// @Router      /api/verify-otp [post]
func (h *RegistrationHandler) VerifyOTP(c *gin.Context) {
	var req model.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Неверный формат запроса.",
		})
		return
	}

	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()

	cabinetURL, token, err := h.svc.VerifyOTP(c.Request.Context(), req, ip, ua)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      "Регистрация подтверждена! Данные для входа отправлены на почту.",
		"redirect_url": cabinetURL,
		"cabinet_url":  cabinetURL,
		// token allows the frontend to store the JWT and redirect without
		// requiring the user to log in again via POST /auth/login.
		"token": token,
	})
}
