package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/media-service/internal/service"
)

// CreateRoomRequest is the body for POST /rooms.
type CreateRoomRequest struct {
	Name            string `json:"name"             binding:"required"`
	MaxParticipants uint32 `json:"max_participants"`
	EmptyTimeout    uint32 `json:"empty_timeout"`    // seconds, default 300
}

// RoomHandler exposes room lifecycle endpoints.
type RoomHandler struct {
	roomSvc *service.RoomService
	logger  *zap.Logger
}

func NewRoomHandler(roomSvc *service.RoomService, logger *zap.Logger) *RoomHandler {
	return &RoomHandler{roomSvc: roomSvc, logger: logger}
}

// RegisterRoutes wires room endpoints.
func (h *RoomHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/rooms", h.CreateRoom)
	r.GET("/rooms/:name", h.GetRoom)
	r.DELETE("/rooms/:name", h.EndRoom)
	r.GET("/rooms/:name/participants", h.ListParticipants)
	r.DELETE("/rooms/:name/participants/:identity", h.RemoveParticipant)
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.EmptyTimeout == 0 {
		req.EmptyTimeout = 300 // 5 minutes default
	}

	room, err := h.roomSvc.CreateRoom(c.Request.Context(), req.Name, req.MaxParticipants, req.EmptyTimeout)
	if err != nil {
		h.logger.Error("create room failed", zap.String("name", req.Name), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"room": room})
}

func (h *RoomHandler) GetRoom(c *gin.Context) {
	room, err := h.roomSvc.GetRoom(c.Request.Context(), c.Param("name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"room": room})
}

func (h *RoomHandler) EndRoom(c *gin.Context) {
	if err := h.roomSvc.EndRoom(c.Request.Context(), c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ended"})
}

func (h *RoomHandler) ListParticipants(c *gin.Context) {
	participants, err := h.roomSvc.ListParticipants(c.Request.Context(), c.Param("name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"participants": participants, "total": len(participants)})
}

func (h *RoomHandler) RemoveParticipant(c *gin.Context) {
	err := h.roomSvc.RemoveParticipant(c.Request.Context(), c.Param("name"), c.Param("identity"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}
