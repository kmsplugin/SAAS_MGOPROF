package repository

import (
	"context"
	"database/sql"
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

// ── Job persistence ────────────────────────────────────────────────────────────

// CreateJob inserts a new job row and returns its generated ID.
func (r *AIRepository) CreateJob(ctx context.Context, job model.AIJob) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ai_jobs (id, event_id, tenant_id, audio_url, language, recording_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'queued')`,
		id, job.EventID, job.TenantID, job.AudioURL, job.Language, job.RecordingID,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("ai_repo.CreateJob: %w", err)
	}
	return id, nil
}

// UpdateJobStatus sets status (and optional error message) for a job.
func (r *AIRepository) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status, errMsg string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ai_jobs SET status = $1, error_msg = $2, updated_at = NOW() WHERE id = $3`,
		status, errMsg, jobID,
	)
	if err != nil {
		return fmt.Errorf("ai_repo.UpdateJobStatus: %w", err)
	}
	return nil
}

// JobExistsForEvent returns true when an active (queued/processing) job exists for this event.
func (r *AIRepository) JobExistsForEvent(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM ai_jobs WHERE event_id = $1 AND status IN ('queued', 'processing')`,
		eventID,
	)
	if err != nil {
		return false, fmt.Errorf("ai_repo.JobExistsForEvent: %w", err)
	}
	return count > 0, nil
}

// GetPendingJobs returns all queued/processing jobs for restart recovery.
func (r *AIRepository) GetPendingJobs(ctx context.Context) ([]model.AIJobDB, error) {
	var jobs []model.AIJobDB
	err := r.db.SelectContext(ctx, &jobs,
		`SELECT * FROM ai_jobs WHERE status IN ('queued', 'processing') ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("ai_repo.GetPendingJobs: %w", err)
	}
	return jobs, nil
}

// GetJobStatusByEvent returns the most recent job for an event within a tenant.
func (r *AIRepository) GetJobStatusByEvent(ctx context.Context, tenantID, eventID uuid.UUID) (*model.AIJobDB, error) {
	var job model.AIJobDB
	err := r.db.GetContext(ctx, &job,
		`SELECT * FROM ai_jobs WHERE tenant_id = $1 AND event_id = $2 ORDER BY created_at DESC LIMIT 1`,
		tenantID, eventID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ai_repo.GetJobStatusByEvent: %w", err)
	}
	return &job, nil
}
