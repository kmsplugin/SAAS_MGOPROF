package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/api/internal/middleware"
	"platform/api/internal/model"
	"platform/api/internal/service"
)

type EventHandler struct {
	eventSvc *service.EventService
	logger   *zap.Logger
}

func NewEventHandler(eventSvc *service.EventService, logger *zap.Logger) *EventHandler {
	return &EventHandler{eventSvc: eventSvc, logger: logger}
}

func (h *EventHandler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	// Public — listing published events requires auth (tenant context needed)
	r.GET("/events", authMW, h.List)
	r.GET("/events/:id", authMW, h.Get)

	// Participant — register for an event
	r.POST("/events/:id/register", authMW, h.RegisterForEvent)

	// Admin/owner — event management
	admin := r.Group("/events", authMW, middleware.RequireRole("event_admin", "tenant_owner", "super_admin"))
	admin.POST("", h.Create)
	admin.POST("/:id/publish", h.Publish)
}

func (h *EventHandler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	events, err := h.eventSvc.List(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events, "total": len(events)})
}

func (h *EventHandler) Get(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	event, err := h.eventSvc.Get(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "мероприятие не найдено"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"event": event})
}

func (h *EventHandler) Create(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req model.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	event, err := h.eventSvc.Create(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		h.logger.Error("create event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "event": event})
}

func (h *EventHandler) Publish(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if err := h.eventSvc.Publish(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "published"})
}

func (h *EventHandler) RegisterForEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	eventID := c.Param("id")

	var req model.RegisterEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	req.EventID = eventID

	reg, err := h.eventSvc.Register(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "registration": reg})
}
