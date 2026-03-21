package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/api/internal/middleware"
	"platform/api/internal/model"
	"platform/api/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
	logger  *zap.Logger
}

func NewAuthHandler(authSvc *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, logger: logger}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.GET("/auth/me", authMW, h.Me)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	ip := middleware.ExtractIP(c)
	token, user, err := h.authSvc.Register(c.Request.Context(), req, ip)
	if err != nil {
		h.logger.Warn("register failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "token": token, "user": user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	ip := middleware.ExtractIP(c)
	token, user, err := h.authSvc.Login(c.Request.Context(), req, ip)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "token": token, "user": user})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	user, err := h.authSvc.Me(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}
