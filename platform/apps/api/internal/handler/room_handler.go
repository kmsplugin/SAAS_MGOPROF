package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/api/internal/middleware"
	"platform/api/internal/model"
	"platform/api/internal/service"
)

type RoomHandler struct {
	roomSvc *service.RoomService
	logger  *zap.Logger
}

func NewRoomHandler(roomSvc *service.RoomService, logger *zap.Logger) *RoomHandler {
	return &RoomHandler{roomSvc: roomSvc, logger: logger}
}

func (h *RoomHandler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	// Admin — create / end rooms
	admin := r.Group("/rooms", authMW, middleware.RequireRole("event_admin", "tenant_owner", "super_admin"))
	admin.POST("", h.Create)
	admin.DELETE("/:id", h.End)

	// List rooms for an event (any authenticated user)
	r.GET("/events/:id/rooms", authMW, h.ListByEvent)

	// Join room — any authenticated user (registration enforced inside service)
	r.POST("/rooms/:id/join", authMW, h.Join)
}

// Create creates a new room for an event.
func (h *RoomHandler) Create(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req model.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	room, err := h.roomSvc.CreateRoom(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		h.logger.Error("create room", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "room": room})
}

// ListByEvent returns all rooms for a given event.
func (h *RoomHandler) ListByEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	eventID := c.Param("id")

	rooms, err := h.roomSvc.GetRooms(c.Request.Context(), tenantID, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

// Join issues a LiveKit token for the caller to join a room.
func (h *RoomHandler) Join(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	roomID := c.Param("id")

	var req model.JoinRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}

	resp, err := h.roomSvc.JoinRoom(c.Request.Context(), tenantID, roomID, userID, req.DisplayName, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "join": resp})
}

// End closes a room (sets status = ended).
func (h *RoomHandler) End(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	roomID := c.Param("id")

	if err := h.roomSvc.EndRoom(c.Request.Context(), tenantID, roomID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Status: "error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ended"})
}
