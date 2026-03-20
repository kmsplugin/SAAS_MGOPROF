package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
	"mgoprof-saas/internal/service"
)

// AuthHandler exposes cabinet authentication endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
	regRepo *repository.RegistrationRepository
	logger  *zap.Logger
}

func NewAuthHandler(
	authSvc *service.AuthService,
	regRepo *repository.RegistrationRepository,
	logger *zap.Logger,
) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, regRepo: regRepo, logger: logger}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	r.POST("/auth/login", h.Login)
	r.POST("/auth/forgot-password", h.ForgotPassword)
	r.GET("/cabinet/me", auth, h.Me)
	r.GET("/cabinet/events", auth, h.MyEvents)
}

// Login godoc
// @Summary     Вход в личный кабинет
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body model.LoginRequest true "Email + пароль"
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} model.ErrorResponse
// @Router      /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Укажите email и пароль.",
		})
		return
	}

	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()

	token, user, err := h.authSvc.Login(c.Request.Context(), req, ip, ua)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"token":  token,
		"user": gin.H{
			"id":           user.ID,
			"email":        user.Email,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"patronymic":   user.Patronymic,
			"organization": user.Organization,
			"district":     user.District,
		},
	})
}

// ForgotPassword godoc
// @Summary     Запрос нового пароля
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body model.ForgotPasswordRequest true "Email"
// @Success     200 {object} map[string]string
// @Router      /api/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req model.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Укажите email.",
		})
		return
	}

	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()

	if err := h.authSvc.ForgotPassword(c.Request.Context(), req, ip, ua); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Если email зарегистрирован, новый пароль отправлен на почту.",
	})
}

// Me godoc
// @Summary     Данные текущего пользователя
// @Tags        Cabinet
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} model.User
// @Failure     401 {object} model.ErrorResponse
// @Router      /api/cabinet/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetInt("user_id")
	user, err := h.authSvc.Me(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, user)
}

// MyEvents godoc
// @Summary     Подтверждённые мероприятия пользователя
// @Tags        Cabinet
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} []model.Event
// @Failure     401 {object} model.ErrorResponse
// @Router      /api/cabinet/events [get]
func (h *AuthHandler) MyEvents(c *gin.Context) {
	userID := c.GetInt("user_id")
	events, err := h.regRepo.ListByUserVerified(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("cabinet events failed", zap.Int("user_id", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Ошибка получения мероприятий.",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}
