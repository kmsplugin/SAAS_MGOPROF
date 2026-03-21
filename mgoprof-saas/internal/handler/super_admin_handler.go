package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// SuperAdminHandler manages admin users (super_admin only).
type SuperAdminHandler struct {
	adminRepo *repository.AdminRepository
	logger    *zap.Logger
}

func NewSuperAdminHandler(adminRepo *repository.AdminRepository, logger *zap.Logger) *SuperAdminHandler {
	return &SuperAdminHandler{adminRepo: adminRepo, logger: logger}
}

// RegisterRoutes registers all super_admin-only routes.
func (h *SuperAdminHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	superAdmin := r.Group("/superadmin", auth, middleware.RequireRole("super_admin"))
	{
		superAdmin.GET("/admins", h.ListAdmins)
		superAdmin.POST("/admins", h.CreateAdmin)
		superAdmin.PUT("/admins/:id", h.UpdateAdmin)
		superAdmin.DELETE("/admins/:id", h.DeactivateAdmin)
		superAdmin.GET("/audit-logs", h.AuditLogs)
	}
}

// ListAdmins godoc
// @Summary     Список администраторов
// @Tags        SuperAdmin
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} map[string]interface{}
// @Router      /api/superadmin/admins [get]
func (h *SuperAdminHandler) ListAdmins(c *gin.Context) {
	admins, err := h.adminRepo.ListAll(c.Request.Context())
	if err != nil {
		h.logger.Error("list admins failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	// Scrub password hashes before sending
	for i := range admins {
		admins[i].PasswordHash = ""
	}
	c.JSON(http.StatusOK, gin.H{"admins": admins, "total": len(admins)})
}

// CreateAdmin godoc
// @Summary     Создать администратора
// @Tags        SuperAdmin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body model.CreateAdminRequest true "Данные администратора"
// @Success     201 {object} map[string]interface{}
// @Router      /api/superadmin/admins [post]
func (h *SuperAdminHandler) CreateAdmin(c *gin.Context) {
	callerID := c.GetInt("user_id")
	callerEmail := c.GetString("email")

	var req model.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Проверьте заполнение полей."})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: "Ошибка хеширования пароля."})
		return
	}

	id, err := h.adminRepo.Create(c.Request.Context(), req, callerID, string(hash))
	if err != nil {
		h.logger.Error("create admin failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	ip := middleware.ExtractIP(c)
	_ = h.adminRepo.WriteAuditLog(c.Request.Context(),
		callerID, callerEmail, "admin_created", "admin_user", &id,
		nil, map[string]interface{}{"email": req.Email, "role": req.Role}, ip,
	)

	c.JSON(http.StatusCreated, gin.H{"status": "success", "id": id})
}

// UpdateAdmin godoc
// @Summary     Обновить администратора (имя, роль, статус)
// @Tags        SuperAdmin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id   path int                        true "ID"
// @Param       body body model.UpdateAdminUserRequest true "Поля для обновления"
// @Success     200 {object} map[string]string
// @Router      /api/superadmin/admins/{id} [put]
func (h *SuperAdminHandler) UpdateAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}

	var req model.UpdateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}

	// Snapshot old state for audit
	old, _ := h.adminRepo.FindByID(c.Request.Context(), id)

	if err := h.adminRepo.Update(c.Request.Context(), id, req); err != nil {
		h.logger.Error("update admin failed", zap.Int("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	callerID := c.GetInt("user_id")
	callerEmail := c.GetString("email")
	ip := middleware.ExtractIP(c)
	_ = h.adminRepo.WriteAuditLog(c.Request.Context(),
		callerID, callerEmail, "admin_updated", "admin_user", &id,
		old, req, ip,
	)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DeactivateAdmin godoc
// @Summary     Деактивировать администратора
// @Tags        SuperAdmin
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "ID"
// @Success     200 {object} map[string]string
// @Router      /api/superadmin/admins/{id} [delete]
func (h *SuperAdminHandler) DeactivateAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}

	// Prevent self-deactivation
	if id == c.GetInt("user_id") {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Нельзя деактивировать самого себя."})
		return
	}

	inactive := false
	if err := h.adminRepo.Update(c.Request.Context(), id, model.UpdateAdminUserRequest{IsActive: &inactive}); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	callerID := c.GetInt("user_id")
	callerEmail := c.GetString("email")
	ip := middleware.ExtractIP(c)
	_ = h.adminRepo.WriteAuditLog(c.Request.Context(),
		callerID, callerEmail, "admin_deactivated", "admin_user", &id,
		nil, nil, ip,
	)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// AuditLogs godoc
// @Summary     Журнал действий администраторов
// @Tags        SuperAdmin
// @Security    BearerAuth
// @Produce     json
// @Param       admin_id query int false "Фильтр по ID администратора"
// @Param       limit    query int false "Лимит (по умолчанию 100)"
// @Param       offset   query int false "Смещение"
// @Success     200 {object} map[string]interface{}
// @Router      /api/superadmin/audit-logs [get]
func (h *SuperAdminHandler) AuditLogs(c *gin.Context) {
	adminID, _ := strconv.Atoi(c.Query("admin_id"))
	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	offset, _ := strconv.Atoi(c.Query("offset"))

	logs, err := h.adminRepo.ListAuditLogs(c.Request.Context(), adminID, limit, offset)
	if err != nil {
		h.logger.Error("audit logs failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs)})
}
