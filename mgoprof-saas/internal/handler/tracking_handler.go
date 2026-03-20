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

// TrackingHandler exposes tracking endpoints for participants and admin.
type TrackingHandler struct {
	svc    *service.TrackingService
	logger *zap.Logger
}

func NewTrackingHandler(svc *service.TrackingService, logger *zap.Logger) *TrackingHandler {
	return &TrackingHandler{svc: svc, logger: logger}
}

// RegisterRoutes wires participant tracking routes (JWT-protected) and admin route.
func (h *TrackingHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, adminAuth ...gin.HandlerFunc) {
	// Participant: must be authenticated (any role)
	track := r.Group("/track", auth)
	{
		// POST /api/track/:event_id  body: {"action":"visit"} etc.
		track.POST("/:event_id", h.Record)
	}

	// Admin: list all tracking events for an event
	if len(adminAuth) > 0 {
		admin := r.Group("/admin/events/:id/tracking", adminAuth...)
		admin.GET("", h.AdminList)
	}
}

// Record godoc
// @Summary     Записать событие отслеживания (визит / стрим)
// @Tags        Tracking
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       event_id path  int                  true "ID мероприятия"
// @Param       body     body  model.TrackActionRequest true "Действие"
// @Success     200 {object} map[string]string
// @Router      /api/track/{event_id} [post]
func (h *TrackingHandler) Record(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID мероприятия."})
		return
	}

	userID, _ := c.Get("user_id")
	uid, ok := userID.(int)
	if !ok || uid < 1 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Status: "error", Message: "требуется авторизация"})
		return
	}

	var req model.TrackActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный запрос."})
		return
	}

	ip := middleware.ExtractIP(c)
	ua := c.Request.UserAgent()

	if err := h.svc.RecordAction(c.Request.Context(), uid, eventID, req.Action, ip, ua); err != nil {
		h.logger.Warn("tracking record", zap.Error(err))
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// AdminList godoc
// @Summary     Список событий трекинга для мероприятия (admin)
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "ID мероприятия"
// @Success     200 {object} []model.TrackingEvent
// @Router      /api/admin/events/{id}/tracking [get]
func (h *TrackingHandler) AdminList(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "Неверный ID."})
		return
	}
	rows, err := h.svc.ListByEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tracking": rows, "total": len(rows)})
}
