package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"platform/media-service/internal/service"
)

// TokenRequest is the body for POST /token.
type TokenRequest struct {
	// Room to join — created by the platform API before issuing a token.
	RoomName string `json:"room_name" binding:"required"`
	// Unique identity per participant per room (e.g. "user_42" or "guest_uuid").
	Identity string `json:"identity"  binding:"required"`
	// Display name shown to other participants.
	DisplayName string `json:"display_name" binding:"required"`
	// Role determines publish/subscribe capabilities.
	// Values: host | speaker | moderator | viewer
	Role string `json:"role" binding:"required,oneof=host speaker moderator viewer"`
}

// TokenResponse is returned on success.
type TokenResponse struct {
	Token    string `json:"token"`
	RoomName string `json:"room_name"`
	LiveKitURL string `json:"livekit_url"`
}

// TokenHandler issues LiveKit access tokens.
type TokenHandler struct {
	tokenSvc   *service.TokenService
	livekitURL string
	logger     *zap.Logger
}

func NewTokenHandler(tokenSvc *service.TokenService, livekitURL string, logger *zap.Logger) *TokenHandler {
	return &TokenHandler{tokenSvc: tokenSvc, livekitURL: livekitURL, logger: logger}
}

// RegisterRoutes wires token endpoints to the router group.
func (h *TokenHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/token", h.IssueToken)
}

// IssueToken godoc
// @Summary     Выпустить токен доступа к комнате LiveKit
// @Tags        Media
// @Accept      json
// @Produce     json
// @Param       body body TokenRequest true "Параметры токена"
// @Success     200 {object} TokenResponse
// @Failure     400 {object} map[string]string
// @Router      /media/token [post]
func (h *TokenHandler) IssueToken(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	var (
		token string
		err   error
	)

	switch req.Role {
	case "host":
		token, err = h.tokenSvc.IssueHostToken(req.RoomName, req.Identity, req.DisplayName)
	case "speaker":
		token, err = h.tokenSvc.IssueSpeakerToken(req.RoomName, req.Identity, req.DisplayName)
	case "moderator":
		token, err = h.tokenSvc.IssueModeratorToken(req.RoomName, req.Identity, req.DisplayName)
	case "viewer":
		token, err = h.tokenSvc.IssueViewerToken(req.RoomName, req.Identity, req.DisplayName)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown role: %s", req.Role)})
		return
	}

	if err != nil {
		h.logger.Error("token issue failed", zap.String("room", req.RoomName), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	h.logger.Info("token issued",
		zap.String("room", req.RoomName),
		zap.String("identity", req.Identity),
		zap.String("role", req.Role),
	)

	c.JSON(http.StatusOK, TokenResponse{
		Token:      token,
		RoomName:   req.RoomName,
		LiveKitURL: h.livekitURL,
	})
}
