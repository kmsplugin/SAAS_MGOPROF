package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// FieldHandler manages custom registration fields per event.
type FieldHandler struct {
	svc    *service.FieldService
	logger *zap.Logger
}

func NewFieldHandler(svc *service.FieldService, logger *zap.Logger) *FieldHandler {
	return &FieldHandler{svc: svc, logger: logger}
}

// RegisterPublicRoutes adds the public GET endpoint (used by the registration form).
func (h *FieldHandler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/events/:id/fields", h.ListPublic)
}

// RegisterAdminRoutes adds admin CRUD routes inside an already-protected group.
func (h *FieldHandler) RegisterAdminRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	admin := r.Group("/admin/events/:id/fields", auth, middleware.RequireRole("admin"))
	{
		admin.GET("", h.List)
		admin.POST("", h.Create)
		admin.PUT("/:field_id", h.Update)
		admin.DELETE("/:field_id", h.Delete)
	}
}

// ListPublic godoc
// @Summary     Список полей мероприятия (публичный)
// @Tags        Events
// @Produce     json
// @Param       id path int true "ID мероприятия"
// @Success     200 {object} []model.EventField
// @Router      /api/events/{id}/fields [get]
func (h *FieldHandler) ListPublic(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	fields, err := h.svc.ListByEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fields": fields})
}

// List godoc
// @Summary     Список полей мероприятия (admin)
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "ID мероприятия"
// @Success     200 {object} []model.EventField
// @Router      /api/admin/events/{id}/fields [get]
func (h *FieldHandler) List(c *gin.Context) {
	h.ListPublic(c)
}

// Create godoc
// @Summary     Создать поле мероприятия
// @Tags        Admin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id   path int                      true "ID мероприятия"
// @Param       body body model.CreateFieldRequest true "Данные поля"
// @Success     201 {object} model.EventField
// @Router      /api/admin/events/{id}/fields [post]
func (h *FieldHandler) Create(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	var req model.CreateFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос: " + err.Error()})
		return
	}
	field, err := h.svc.Create(c.Request.Context(), eventID, req)
	if err != nil {
		h.logger.Error("create field failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, field)
}

// Update godoc
// @Summary     Обновить поле мероприятия
// @Tags        Admin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id       path int                      true "ID мероприятия"
// @Param       field_id path int                      true "ID поля"
// @Param       body     body model.CreateFieldRequest true "Данные поля"
// @Success     200 {object} model.EventField
// @Router      /api/admin/events/{id}/fields/{field_id} [put]
func (h *FieldHandler) Update(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID мероприятия."})
		return
	}
	fieldID, err := strconv.Atoi(c.Param("field_id"))
	if err != nil || fieldID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID поля."})
		return
	}
	var req model.CreateFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос: " + err.Error()})
		return
	}
	field, err := h.svc.Update(c.Request.Context(), eventID, fieldID, req)
	if err != nil {
		h.logger.Error("update field failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, field)
}

// Delete godoc
// @Summary     Удалить поле мероприятия
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Param       id       path int true "ID мероприятия"
// @Param       field_id path int true "ID поля"
// @Success     200 {object} map[string]string
// @Router      /api/admin/events/{id}/fields/{field_id} [delete]
func (h *FieldHandler) Delete(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID мероприятия."})
		return
	}
	fieldID, err := strconv.Atoi(c.Param("field_id"))
	if err != nil || fieldID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID поля."})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), eventID, fieldID); err != nil {
		h.logger.Error("delete field failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
