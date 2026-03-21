package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"platform/api/internal/model"
)

type RegistrationRepository struct{ db *sqlx.DB }

func NewRegistrationRepository(db *sqlx.DB) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

func (r *RegistrationRepository) Find(ctx context.Context, tenantID, eventID, userID string) (*model.Registration, error) {
	var reg model.Registration
	err := r.db.GetContext(ctx, &reg,
		`SELECT * FROM registrations WHERE tenant_id = $1 AND event_id = $2 AND user_id = $3 LIMIT 1`,
		tenantID, eventID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reg Find: %w", err)
	}
	return &reg, nil
}

func (r *RegistrationRepository) Create(ctx context.Context, tenantID, eventID, userID string, consentGiven bool) (*model.Registration, error) {
	var reg model.Registration
	err := r.db.GetContext(ctx, &reg,
		`INSERT INTO registrations (event_id, user_id, tenant_id, status, consent_given, consent_version)
		 VALUES ($1, $2, $3, 'confirmed', $4, '1.0') RETURNING *`,
		eventID, userID, tenantID, consentGiven,
	)
	if err != nil {
		return nil, fmt.Errorf("reg Create: %w", err)
	}
	return &reg, nil
}

func (r *RegistrationRepository) ListByUser(ctx context.Context, tenantID, userID string) ([]model.Registration, error) {
	var regs []model.Registration
	err := r.db.SelectContext(ctx, &regs,
		`SELECT * FROM registrations WHERE tenant_id = $1 AND user_id = $2 ORDER BY created_at DESC`,
		tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("reg ListByUser: %w", err)
	}
	return regs, nil
}

func (r *RegistrationRepository) CountByEvent(ctx context.Context, tenantID, eventID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM registrations WHERE tenant_id = $1 AND event_id = $2 AND status = 'confirmed'`,
		tenantID, eventID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("reg CountByEvent: %w", err)
	}
	return count, nil
}

func (r *RegistrationRepository) CheckIn(ctx context.Context, tenantID, eventID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE registrations SET status = 'attended', check_in_at = NOW()
		 WHERE tenant_id = $1 AND event_id = $2 AND user_id = $3`,
		tenantID, eventID, userID)
	if err != nil {
		return fmt.Errorf("reg CheckIn: %w", err)
	}
	return nil
}
