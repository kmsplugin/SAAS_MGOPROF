package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"platform/ai-service/internal/model"
	"platform/ai-service/internal/repository"
	"platform/ai-service/internal/service"
)

// WebhookHandler обрабатывает входящие вебхуки от LiveKit Egress.
type WebhookHandler struct {
	queue     *service.JobQueue
	repo      *repository.AIRepository
	apiSecret string // LiveKit API secret для верификации подписи
	logger    *zap.Logger
}

func NewWebhookHandler(queue *service.JobQueue, repo *repository.AIRepository, apiSecret string, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{queue: queue, repo: repo, apiSecret: apiSecret, logger: logger}
}

// LiveKitEgress обрабатывает вебхук завершения записи.
// POST /webhooks/livekit-egress
func (h *WebhookHandler) LiveKitEgress(c *gin.Context) {
	// ── Signature verification ────────────────────────────────────────────────
	authHeader := c.GetHeader("Authorization")
	if !verifyLiveKitWebhook(authHeader, h.apiSecret) {
		h.logger.Warn("webhook: invalid signature",
			zap.String("remote", c.ClientIP()),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	var payload model.LiveKitEgressWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Нас интересует только событие завершения
	if payload.Event != "egress_ended" {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	info := payload.EgressInfo

	if info.Status == "EGRESS_FAILED" || info.Error != "" {
		h.logger.Warn("webhook: egress failed",
			zap.String("egress_id", info.EgressID),
			zap.String("room", info.RoomName),
			zap.String("error", info.Error),
		)
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	if len(info.FileResults) == 0 {
		h.logger.Warn("webhook: no file results", zap.String("egress_id", info.EgressID))
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	audioURL := info.FileResults[0].Location

	// Находим event_id и tenant_id по имени комнаты
	eventID, tenantID, err := h.repo.GetEventByRoom(c.Request.Context(), info.RoomName)
	if err != nil {
		h.logger.Error("webhook: room not found",
			zap.String("room", info.RoomName),
			zap.Error(err),
		)
		// Возвращаем 200 чтобы LiveKit не повторял вебхук
		c.JSON(http.StatusOK, gin.H{"ok": true, "warning": "room not found"})
		return
	}

	// ── Duplicate prevention ──────────────────────────────────────────────────
	exists, err := h.repo.JobExistsForEvent(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Warn("webhook: duplicate check error, proceeding", zap.Error(err))
	} else if exists {
		h.logger.Info("webhook: job already active, skipping",
			zap.String("event_id", eventID.String()),
		)
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "job already active"})
		return
	}

	// Сохраняем URL записи
	if err = h.repo.UpdateRoomRecordingURL(c.Request.Context(), info.RoomName, audioURL); err != nil {
		h.logger.Error("webhook: update recording url", zap.Error(err))
	}

	// Ставим задачу в очередь
	job := model.AIJob{
		EventID:     eventID,
		TenantID:    tenantID,
		AudioURL:    audioURL,
		Language:    "ru",
		RecordingID: info.EgressID,
	}

	if err = h.queue.Enqueue(job); err != nil {
		h.logger.Error("webhook: enqueue failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "queue full"})
		return
	}

	h.logger.Info("webhook: job enqueued",
		zap.String("event_id", eventID.String()),
		zap.String("room", info.RoomName),
	)
	c.JSON(http.StatusOK, gin.H{"ok": true, "event_id": eventID})
}

// verifyLiveKitWebhook verifies a LiveKit JWT webhook signature.
// LiveKit signs webhooks using HMAC-SHA256 with the API secret.
// Authorization: Bearer <header>.<payload>.<signature>
func verifyLiveKitWebhook(authHeader, apiSecret string) bool {
	if apiSecret == "" {
		// Dev mode: skip verification when secret is not configured
		return true
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return false // No "Bearer " prefix
	}
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return false
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(parts[2]))
}
