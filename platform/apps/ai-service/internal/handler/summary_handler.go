package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"platform/ai-service/internal/model"
	"platform/ai-service/internal/repository"
	"platform/ai-service/internal/service"
)

// SummaryHandler — REST API для получения AI-контента и ручного запуска обработки.
type SummaryHandler struct {
	queue  *service.JobQueue
	repo   *repository.AIRepository
	logger *zap.Logger
}

func NewSummaryHandler(queue *service.JobQueue, repo *repository.AIRepository, logger *zap.Logger) *SummaryHandler {
	return &SummaryHandler{queue: queue, repo: repo, logger: logger}
}

// GetEventSummaries возвращает все AI-материалы для события.
// GET /api/v1/events/:event_id/ai
// Header: X-Tenant-ID: <uuid>
func (h *SummaryHandler) GetEventSummaries(c *gin.Context) {
	tenantID, ok := parseTenantID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_id"})
		return
	}

	summaries, err := h.repo.FindByEvent(c.Request.Context(), tenantID, eventID)
	if err != nil {
		h.logger.Error("get summaries", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Собираем в удобный map по типу
	out := make(map[string]*model.AISummary, len(summaries))
	for i := range summaries {
		out[string(summaries[i].Type)] = &summaries[i]
	}

	c.JSON(http.StatusOK, gin.H{"summaries": out})
}

// GetSummaryByType возвращает один тип AI-контента.
// GET /api/v1/events/:event_id/ai/:type
func (h *SummaryHandler) GetSummaryByType(c *gin.Context) {
	tenantID, ok := parseTenantID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_id"})
		return
	}

	summaryType := model.SummaryType(c.Param("type"))
	switch summaryType {
	case model.SummaryTypeTranscript,
		model.SummaryTypeSummary,
		model.SummaryTypeHighlights,
		model.SummaryTypeChapters,
		model.SummaryTypeActionItems:
		// ok
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown type", "valid_types": []string{
			"transcript", "summary", "highlights", "chapters", "action_items",
		}})
		return
	}

	s, err := h.repo.FindOne(c.Request.Context(), tenantID, eventID, summaryType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, s)
}

// TriggerProcessing запускает AI-обработку вручную (для администраторов).
// POST /api/v1/events/:event_id/ai/process
func (h *SummaryHandler) TriggerProcessing(c *gin.Context) {
	tenantID, ok := parseTenantID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_id"})
		return
	}

	var req model.ProcessRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lang := req.Language
	if lang == "" {
		lang = "ru"
	}

	job := model.AIJob{
		EventID:  eventID,
		TenantID: tenantID,
		AudioURL: req.AudioURL,
		Language: lang,
	}

	if err = h.queue.Enqueue(job); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "queue full, try again later"})
		return
	}

	h.logger.Info("manual trigger",
		zap.String("event_id", eventID.String()),
		zap.String("tenant_id", tenantID.String()),
	)
	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "event_id": eventID})
}

// GetStatus возвращает текущий статус обработки события.
// GET /api/v1/events/:event_id/ai/status
// Header: X-Tenant-ID required for tenant isolation
func (h *SummaryHandler) GetStatus(c *gin.Context) {
	tenantID, ok := parseTenantID(c)
	if !ok {
		return
	}

	eventID, err := uuid.Parse(c.Param("event_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_id"})
		return
	}

	// Check in-memory cache first (reflects live processing state)
	if status := h.queue.Status(eventID); status != nil {
		c.JSON(http.StatusOK, status)
		return
	}

	// Fall back to DB — also enforces tenant isolation
	job, err := h.repo.GetJobStatusByEvent(c.Request.Context(), tenantID, eventID)
	if err != nil {
		h.logger.Warn("get job status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if job == nil {
		c.JSON(http.StatusOK, gin.H{"event_id": eventID, "status": "no_job"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"event_id": job.EventID,
		"status":   job.Status,
		"error":    job.ErrorMsg,
	})
}

// HealthCheck — livez probe.
// GET /health
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ai-service"})
}

// parseTenantID читает X-Tenant-ID из заголовка.
func parseTenantID(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetHeader("X-Tenant-ID")
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return uuid.Nil, false
	}
	tid, err := uuid.Parse(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-Tenant-ID"})
		return uuid.Nil, false
	}
	return tid, true
}
