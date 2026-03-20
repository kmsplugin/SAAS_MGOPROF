package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
	"mgoprof-saas/internal/service"
)

// AuthHandler exposes cabinet authentication and self-service endpoints.
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
	// Public
	r.POST("/auth/login", h.Login)
	r.POST("/auth/forgot-password", h.ForgotPassword)

	// JWT-protected
	cab := r.Group("/cabinet", auth)
	{
		cab.GET("/me", h.Me)
		cab.PUT("/profile", h.UpdateProfile)
		cab.PUT("/password", h.ChangePassword)
		cab.GET("/events", h.MyEvents)
		cab.DELETE("/events/:event_id/registration", h.CancelRegistration)
		cab.GET("/data-export", h.DataExport)
		cab.DELETE("/account", h.DeleteAccount)
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Укажите email и пароль."})
		return
	}
	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()
	token, user, err := h.authSvc.Login(c.Request.Context(), req, ip, ua)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Status: "error", Message: err.Error()})
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

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req model.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Укажите email."})
		return
	}
	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()
	if err := h.authSvc.ForgotPassword(c.Request.Context(), req, ip, ua); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Если email зарегистрирован, новый пароль отправлен на почту.",
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetInt("user_id")
	user, err := h.authSvc.Me(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateProfile godoc
// @Summary     Обновить профиль пользователя
// @Tags        Cabinet
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body model.UpdateProfileRequest true "Данные профиля"
// @Success     200 {object} map[string]string
// @Router      /api/cabinet/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetInt("user_id")
	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	if err := h.authSvc.UpdateProfile(c.Request.Context(), userID, req, middleware.ExtractIP(c), c.Request.UserAgent()); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Профиль обновлён."})
}

// ChangePassword godoc
// @Summary     Смена пароля
// @Tags        Cabinet
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body model.ChangePasswordRequest true "Текущий и новый пароль"
// @Success     200 {object} map[string]string
// @Failure     400 {object} model.ErrorResponse
// @Router      /api/cabinet/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetInt("user_id")
	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}
	if err := h.authSvc.ChangePassword(c.Request.Context(), userID, req, middleware.ExtractIP(c), c.Request.UserAgent()); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Пароль изменён."})
}

func (h *AuthHandler) MyEvents(c *gin.Context) {
	userID := c.GetInt("user_id")
	events, err := h.regRepo.ListByUserVerified(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("cabinet events failed", zap.Int("user_id", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка получения мероприятий."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// CancelRegistration godoc
// @Summary     Отмена регистрации на мероприятие
// @Tags        Cabinet
// @Security    BearerAuth
// @Produce     json
// @Param       event_id path int true "ID мероприятия"
// @Success     200 {object} map[string]string
// @Failure     400 {object} model.ErrorResponse
// @Router      /api/cabinet/events/{event_id}/registration [delete]
func (h *AuthHandler) CancelRegistration(c *gin.Context) {
	userID := c.GetInt("user_id")
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID мероприятия."})
		return
	}
	if err := h.authSvc.CancelRegistration(c.Request.Context(), userID, eventID,
		middleware.ExtractIP(c), c.Request.UserAgent()); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Регистрация отменена."})
}

// DataExport godoc
// @Summary     Запросить экспорт персональных данных (152-ФЗ ст.14 / GDPR Art.15)
// @Tags        Cabinet
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} map[string]string
// @Router      /api/cabinet/data-export [get]
func (h *AuthHandler) DataExport(c *gin.Context) {
	userID := c.GetInt("user_id")
	if err := h.authSvc.ExportData(c.Request.Context(), userID,
		middleware.ExtractIP(c), c.Request.UserAgent()); err != nil {
		h.logger.Error("data export failed", zap.Int("user_id", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Данные отправлены на ваш email.",
	})
}

// DeleteAccount godoc
// @Summary     Удалить аккаунт (152-ФЗ ст.21 / GDPR Art.17)
// @Tags        Cabinet
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body object{password=string} true "Текущий пароль для подтверждения"
// @Success     200 {object} map[string]string
// @Failure     400 {object} model.ErrorResponse
// @Router      /api/cabinet/account [delete]
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	userID := c.GetInt("user_id")
	var body struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Укажите пароль для подтверждения."})
		return
	}
	if err := h.authSvc.DeleteAccount(c.Request.Context(), userID, body.Password,
		middleware.ExtractIP(c), c.Request.UserAgent()); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Аккаунт удалён. Ваши данные анонимизированы в соответствии с 152-ФЗ.",
	})
}
