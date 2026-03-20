package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// EventHandler exposes public event endpoints.
type EventHandler struct {
	svc    *service.EventService
	logger *zap.Logger
}

func NewEventHandler(svc *service.EventService, logger *zap.Logger) *EventHandler {
	return &EventHandler{svc: svc, logger: logger}
}

func (h *EventHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/events", h.List)
	r.GET("/events/:id", h.Get)
}

// List godoc
// @Summary     Список активных мероприятий
// @Tags        Events
// @Produce     json
// @Success     200 {object} []model.Event
// @Router      /api/events [get]
func (h *EventHandler) List(c *gin.Context) {
	events, err := h.svc.ListActive(c.Request.Context())
	if err != nil {
		h.logger.Error("event list failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Ошибка получения мероприятий.",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// Get godoc
// @Summary     Получить мероприятие по ID
// @Tags        Events
// @Produce     json
// @Param       id path int true "ID мероприятия"
// @Success     200 {object} model.Event
// @Failure     404 {object} model.ErrorResponse
// @Router      /api/events/{id} [get]
func (h *EventHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Неверный ID мероприятия.",
		})
		return
	}

	event, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Status: "error", Message: "Мероприятие не найдено."})
		return
	}
	c.JSON(http.StatusOK, event)
}
