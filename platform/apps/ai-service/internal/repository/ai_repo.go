package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"platform/ai-service/internal/model"
)

type AIRepository struct {
	db *sqlx.DB
}

func NewAIRepository(db *sqlx.DB) *AIRepository {
	return &AIRepository{db: db}
}

// Save сохраняет одну запись AI-контента.
func (r *AIRepository) Save(ctx context.Context, s *model.AISummary) error {
	q := `
		INSERT INTO ai_summaries (id, event_id, tenant_id, type, content, language, model_version, generated_at)
		VALUES (:id, :event_id, :tenant_id, :type, :content, :language, :model_version, :generated_at)
		ON CONFLICT (event_id, type) DO UPDATE
		  SET content = EXCLUDED.content,
		      model_version = EXCLUDED.model_version,
		      generated_at = EXCLUDED.generated_at`
	_, err := r.db.NamedExecContext(ctx, q, s)
	if err != nil {
		return fmt.Errorf("ai_repo.Save: %w", err)
	}
	return nil
}

// FindByEvent возвращает все AI-записи для события, принадлежащего тенанту.
func (r *AIRepository) FindByEvent(ctx context.Context, tenantID, eventID uuid.UUID) ([]model.AISummary, error) {
	var out []model.AISummary
	err := r.db.SelectContext(ctx, &out,
		`SELECT * FROM ai_summaries WHERE tenant_id = $1 AND event_id = $2 ORDER BY generated_at`,
		tenantID, eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("ai_repo.FindByEvent: %w", err)
	}
	return out, nil
}

// FindOne возвращает конкретный тип AI-контента для события.
func (r *AIRepository) FindOne(ctx context.Context, tenantID, eventID uuid.UUID, t model.SummaryType) (*model.AISummary, error) {
	var s model.AISummary
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM ai_summaries WHERE tenant_id = $1 AND event_id = $2 AND type = $3`,
		tenantID, eventID, string(t),
	)
	if err != nil {
		return nil, fmt.Errorf("ai_repo.FindOne: %w", err)
	}
	return &s, nil
}

// GetEventTenantID возвращает tenant_id события — нужно для проверки из вебхука.
func (r *AIRepository) GetEventByRoom(ctx context.Context, livekitRoomName string) (eventID, tenantID uuid.UUID, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT r.event_id, r.tenant_id FROM rooms r WHERE r.livekit_room_name = $1`,
		livekitRoomName,
	)
	err = row.Scan(&eventID, &tenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("ai_repo.GetEventByRoom: %w", err)
	}
	return eventID, tenantID, nil
}

// UpdateRoomRecordingURL сохраняет URL записи в комнату.
func (r *AIRepository) UpdateRoomRecordingURL(ctx context.Context, livekitRoomName, recordingURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE rooms SET recording_url = $1 WHERE livekit_room_name = $2`,
		recordingURL, livekitRoomName,
	)
	if err != nil {
		return fmt.Errorf("ai_repo.UpdateRoomRecordingURL: %w", err)
	}
	return nil
}

// IsTenantAIEnabled проверяет, что у тенанта включён AI.
func (r *AIRepository) IsTenantAIEnabled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	var enabled bool
	err := r.db.GetContext(ctx, &enabled,
		`SELECT ai_enabled FROM tenants WHERE id = $1`,
		tenantID,
	)
	if err != nil {
		return false, fmt.Errorf("ai_repo.IsTenantAIEnabled: %w", err)
	}
	return enabled, nil
}
