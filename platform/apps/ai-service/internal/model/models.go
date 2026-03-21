package model

import (
	"time"

	"github.com/google/uuid"
)

// SummaryType перечисляет виды AI-контента.
type SummaryType string

const (
	SummaryTypeTranscript   SummaryType = "transcript"
	SummaryTypeSummary      SummaryType = "summary"
	SummaryTypeHighlights   SummaryType = "highlights"
	SummaryTypeChapters     SummaryType = "chapters"
	SummaryTypeActionItems  SummaryType = "action_items"
)

// AISummary — запись из таблицы ai_summaries.
type AISummary struct {
	ID           uuid.UUID   `db:"id"            json:"id"`
	EventID      uuid.UUID   `db:"event_id"      json:"event_id"`
	TenantID     uuid.UUID   `db:"tenant_id"     json:"tenant_id"`
	Type         SummaryType `db:"type"          json:"type"`
	Content      string      `db:"content"       json:"content"`
	Language     string      `db:"language"      json:"language"`
	ModelVersion string      `db:"model_version" json:"model_version"`
	GeneratedAt  time.Time   `db:"generated_at"  json:"generated_at"`
}

// AIJob — задача обработки записи события.
type AIJob struct {
	EventID    uuid.UUID
	TenantID   uuid.UUID
	AudioURL   string // HTTP URL или локальный путь к файлу
	Language   string // "ru", "en", "auto"
	RecordingID string // egress ID (опционально)
}

// JobStatus — текущий статус задачи.
type JobStatus struct {
	EventID  uuid.UUID   `json:"event_id"`
	Status   string      `json:"status"` // "queued" | "processing" | "done" | "failed"
	Progress []SummaryType `json:"completed_steps"`
	Error    string      `json:"error,omitempty"`
}

// ProcessRequest — тело запроса для ручного запуска обработки.
type ProcessRequest struct {
	AudioURL string `json:"audio_url" binding:"required"`
	Language string `json:"language"`
}

// LiveKitEgressWebhook — упрощённая структура вебхука LiveKit Egress.
type LiveKitEgressWebhook struct {
	Event   string      `json:"event"`
	EgressInfo EgressInfo `json:"egressInfo"`
}

type EgressInfo struct {
	EgressID    string        `json:"egressId"`
	RoomName    string        `json:"roomName"`
	Status      string        `json:"status"`
	FileResults []FileResult  `json:"fileResults"`
	Error       string        `json:"error"`
}

type FileResult struct {
	Filename string `json:"filename"`
	Location string `json:"location"` // s3:// или file://
	Duration int64  `json:"duration"` // секунды
	Size     int64  `json:"size"`     // байты
}
