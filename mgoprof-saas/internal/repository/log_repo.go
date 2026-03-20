package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// LogRepository handles all DB operations for reg_logs.
type LogRepository struct {
	db *sqlx.DB
}

func NewLogRepository(db *sqlx.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) Write(
	ctx context.Context,
	eventType, userEmail, ipAddress, message, userAgent string,
) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reg_logs (event_type, user_email, ip_address, message, user_agent)
		 VALUES ($1, $2, $3, $4, $5)`,
		eventType, userEmail, ipAddress, message, userAgent,
	)
	if err != nil {
		return fmt.Errorf("log Write: %w", err)
	}
	return nil
}

func (r *LogRepository) ListRecent(ctx context.Context, limit int) ([]model.Log, error) {
	var logs []model.Log
	err := r.db.SelectContext(ctx, &logs,
		`SELECT * FROM reg_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("log ListRecent: %w", err)
	}
	return logs, nil
}
