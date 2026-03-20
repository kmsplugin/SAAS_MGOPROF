package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// ConsentRepository handles personal-data consent records.
// Legal basis: RF 152-ФЗ ст.9, GDPR Art.7, CCPA §1798.135.
type ConsentRepository struct {
	db *sqlx.DB
}

func NewConsentRepository(db *sqlx.DB) *ConsentRepository {
	return &ConsentRepository{db: db}
}

const defaultConsentVersion = "1.0"

// consentText is the canonical wording stored at consent time.
// Storing the exact text satisfies GDPR Art.7(1) burden-of-proof requirement.
const consentText = "Я даю согласие на обработку моих персональных данных " +
	"в соответствии с Федеральным законом № 152-ФЗ «О персональных данных» " +
	"в целях участия в мероприятии и получения информационных сообщений."

// SaveTx records the user's consent inside an existing transaction.
// Uses INSERT ON CONFLICT DO NOTHING so re-registrations don't fail on the UNIQUE constraint.
func (r *ConsentRepository) SaveTx(
	ctx context.Context,
	tx *sqlx.Tx,
	userID, eventID int,
	ip, userAgent string,
) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO reg_consents (user_id, event_id, consent_text, version, ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, event_id, version) DO NOTHING`,
		userID, eventID, consentText, defaultConsentVersion, ip, userAgent,
	)
	if err != nil {
		return fmt.Errorf("consent SaveTx: %w", err)
	}
	return nil
}

// ListByUser returns all consent records for a given user (for data export).
func (r *ConsentRepository) ListByUser(ctx context.Context, userID int) ([]model.ConsentRecord, error) {
	var records []model.ConsentRecord
	err := r.db.SelectContext(ctx, &records, `
		SELECT c.event_id, e.title AS event_title, c.version, c.consented_at, c.withdrawn_at
		FROM   reg_consents c
		JOIN   reg_events e ON e.id = c.event_id
		WHERE  c.user_id = $1
		ORDER  BY c.consented_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("consent ListByUser: %w", err)
	}
	return records, nil
}

// WithdrawAll marks all active consents as withdrawn (for account deletion).
func (r *ConsentRepository) WithdrawAll(ctx context.Context, userID int) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE reg_consents SET withdrawn_at = $1
		 WHERE user_id = $2 AND withdrawn_at IS NULL`,
		now, userID,
	)
	if err != nil {
		return fmt.Errorf("consent WithdrawAll: %w", err)
	}
	return nil
}

// RequestDeletion creates a pending deletion request for the user.
// Uses INSERT ON CONFLICT DO NOTHING so duplicate requests are idempotent.
func (r *ConsentRepository) RequestDeletion(ctx context.Context, userID int, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reg_deletion_requests (user_id, reason)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO NOTHING`,
		userID, reason,
	)
	if err != nil {
		return fmt.Errorf("consent RequestDeletion: %w", err)
	}
	return nil
}
