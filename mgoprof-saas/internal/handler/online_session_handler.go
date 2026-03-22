package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/service"
)

// OnlineSessionHandler exposes session lifecycle endpoints for participants
// (connect / ping / disconnect) and admin statistics.
type OnlineSessionHandler struct {
	svc    *service.OnlineSessionService
	logger *zap.Logger
}

func NewOnlineSessionHandler(svc *service.OnlineSessionService, logger *zap.Logger) *OnlineSessionHandler {
	return &OnlineSessionHandler{svc: svc, logger: logger}
}

// RegisterRoutes wires routes under /api/session/:event_id and /api/admin.
func (h *OnlineSessionHandler) RegisterRoutes(
	r *gin.RouterGroup,
	auth gin.HandlerFunc,
	adminAuth ...gin.HandlerFunc,
) {
	// Participant — requires JWT
	sess := r.Group("/session/:event_id", auth)
	{
		sess.POST("/connect",    h.Connect)
		sess.POST("/ping",       h.Ping)
		sess.POST("/disconnect", h.Disconnect)
	}

	// Admin — requires JWT + admin role
	if len(adminAuth) > 0 {
		admin := r.Group("/admin/events/:id/online-stats", adminAuth...)
		admin.GET("", h.AdminStats)
	}
}

// Connect godoc
// @Summary     Начать онлайн-сессию участника
// @Tags        Session
// @Security    BearerAuth
// @Produce     json
// @Param       event_id path int true "ID мероприятия"
// @Success     200 {object} model.OnlineSession
// @Failure     400 {object} model.ErrorResponse
// @Failure     401 {object} model.ErrorResponse
// @Router      /api/session/{event_id}/connect [post]
func (h *OnlineSessionHandler) Connect(c *gin.Context) {
	eventID, userID, ok := h.extractIDs(c)
	if !ok {
		return
	}

	session, err := h.svc.Connect(c.Request.Context(), userID, eventID)
	if err != nil {
		h.logger.Error("session connect failed",
			zap.Int("user_id", userID),
			zap.Int("event_id", eventID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status: "error", Message: "не удалось создать сессию",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"session_uuid": session.SessionUUID,
		"started_at":   session.StartedAt,
	})
}

// Ping godoc
// @Summary     Обновить heartbeat сессии
// @Tags        Session
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       event_id path  int                          true "ID мероприятия"
// @Param       body     body  model.SessionPingRequest      true "session_uuid"
// @Success     200 {object} map[string]string
// @Failure     400 {object} model.ErrorResponse
// @Router      /api/session/{event_id}/ping [post]
func (h *OnlineSessionHandler) Ping(c *gin.Context) {
	var req model.SessionPingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status: "error", Message: "требуется session_uuid",
		})
		return
	}

	if err := h.svc.Ping(c.Request.Context(), req.SessionUUID); err != nil {
		// Expired/unknown session — not a server error; client should reconnect.
		c.JSON(http.StatusGone, model.ErrorResponse{
			Status: "error", Message: "сессия не найдена или уже закрыта",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Disconnect godoc
// @Summary     Завершить онлайн-сессию
// @Tags        Session
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       event_id path  int                               true "ID мероприятия"
// @Param       body     body  model.SessionDisconnectRequest     true "session_uuid"
// @Success     200 {object} map[string]string
// @Router      /api/session/{event_id}/disconnect [post]
func (h *OnlineSessionHandler) Disconnect(c *gin.Context) {
	var req model.SessionDisconnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status: "error", Message: "требуется session_uuid",
		})
		return
	}

	if err := h.svc.Disconnect(c.Request.Context(), req.SessionUUID); err != nil {
		h.logger.Warn("session disconnect error",
			zap.String("uuid", req.SessionUUID),
			zap.Error(err))
	}

	// Always respond 200 — disconnect is best-effort (sendBeacon may fire late).
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// AdminStats godoc
// @Summary     Статистика онлайн-участия по мероприятию (admin)
// @Tags        Admin
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "ID мероприятия"
// @Success     200 {object} model.OnlineStatsResponse
// @Router      /api/admin/events/{id}/online-stats [get]
func (h *OnlineSessionHandler) AdminStats(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: "неверный ID"})
		return
	}

	stats, err := h.svc.AdminStats(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("admin online stats", zap.Int("event_id", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Status: "error", Message: "не удалось получить статистику",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *OnlineSessionHandler) extractIDs(c *gin.Context) (eventID, userID int, ok bool) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID < 1 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Status: "error", Message: "неверный ID мероприятия",
		})
		return 0, 0, false
	}

	rawUID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Status: "error", Message: "требуется авторизация",
		})
		return 0, 0, false
	}
	uid, ok := rawUID.(int)
	if !ok || uid < 1 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Status: "error", Message: "требуется авторизация",
		})
		return 0, 0, false
	}

	return eventID, uid, true
}
