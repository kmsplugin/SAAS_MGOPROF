package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// OnlineSessionRepository manages online_sessions rows.
type OnlineSessionRepository struct {
	db *sqlx.DB
}

func NewOnlineSessionRepository(db *sqlx.DB) *OnlineSessionRepository {
	return &OnlineSessionRepository{db: db}
}

// Create inserts a new open session and returns it with the generated session_uuid.
func (r *OnlineSessionRepository) Create(
	ctx context.Context,
	eventID, userID int,
	regID *int,
) (*model.OnlineSession, error) {
	var s model.OnlineSession
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO online_sessions (event_id, user_id, registration_id)
		VALUES ($1, $2, $3)
		RETURNING id, event_id, user_id, registration_id,
		          session_uuid, started_at, last_ping_at,
		          ended_at, duration_seconds, end_reason`,
		eventID, userID, regID,
	).Scan(
		&s.ID, &s.EventID, &s.UserID, &s.RegistrationID,
		&s.SessionUUID, &s.StartedAt, &s.LastPingAt,
		&s.EndedAt, &s.DurationSeconds, &s.EndReason,
	)
	if err != nil {
		return nil, fmt.Errorf("online_session Create: %w", err)
	}
	return &s, nil
}

// FindOpen returns the most recent open session (ended_at IS NULL) for a
// given user+event combination, or nil if none exists.
func (r *OnlineSessionRepository) FindOpen(
	ctx context.Context,
	eventID, userID int,
) (*model.OnlineSession, error) {
	var s model.OnlineSession
	err := r.db.GetContext(ctx, &s, `
		SELECT id, event_id, user_id, registration_id,
		       session_uuid, started_at, last_ping_at,
		       ended_at, duration_seconds, end_reason
		FROM online_sessions
		WHERE event_id = $1 AND user_id = $2 AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1`,
		eventID, userID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("online_session FindOpen: %w", err)
	}
	return &s, nil
}

// FindByUUID returns a session by its UUID.
func (r *OnlineSessionRepository) FindByUUID(
	ctx context.Context,
	sessionUUID string,
) (*model.OnlineSession, error) {
	var s model.OnlineSession
	err := r.db.GetContext(ctx, &s, `
		SELECT id, event_id, user_id, registration_id,
		       session_uuid, started_at, last_ping_at,
		       ended_at, duration_seconds, end_reason
		FROM online_sessions
		WHERE session_uuid = $1`,
		sessionUUID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("online_session FindByUUID: %w", err)
	}
	return &s, nil
}

// UpdatePing sets last_ping_at to now for the given session.
// Returns false (no error) if the session is already closed.
func (r *OnlineSessionRepository) UpdatePing(ctx context.Context, sessionUUID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE online_sessions
		SET last_ping_at = NOW()
		WHERE session_uuid = $1 AND ended_at IS NULL`,
		sessionUUID,
	)
	if err != nil {
		return fmt.Errorf("online_session UpdatePing: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session not found or already closed: %s", sessionUUID)
	}
	return nil
}

// Close finalises a session: sets ended_at, duration_seconds, end_reason.
func (r *OnlineSessionRepository) Close(
	ctx context.Context,
	sessionUUID string,
	reason string,
) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE online_sessions
		SET ended_at         = NOW(),
		    duration_seconds = GREATEST(1, EXTRACT(EPOCH FROM (NOW() - started_at))::int),
		    end_reason       = $2
		WHERE session_uuid = $1 AND ended_at IS NULL`,
		sessionUUID, reason,
	)
	if err != nil {
		return fmt.Errorf("online_session Close: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Already closed — not an error; idempotent.
		return nil
	}
	return nil
}

// TimeoutStale closes all open sessions whose last_ping_at is older than
// threshold. Returns the number of sessions timed out.
func (r *OnlineSessionRepository) TimeoutStale(
	ctx context.Context,
	threshold time.Duration,
) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE online_sessions
		SET ended_at         = NOW(),
		    duration_seconds = GREATEST(1, EXTRACT(EPOCH FROM (last_ping_at - started_at))::int),
		    end_reason       = 'timeout'
		WHERE ended_at IS NULL
		  AND last_ping_at < NOW() - $1::interval`,
		threshold.String(),
	)
	if err != nil {
		return 0, fmt.Errorf("online_session TimeoutStale: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// SummaryByEvent returns the online_session_summary view rows for a given event.
func (r *OnlineSessionRepository) SummaryByEvent(
	ctx context.Context,
	eventID int,
) ([]model.OnlineSessionSummary, error) {
	var rows []model.OnlineSessionSummary
	err := r.db.SelectContext(ctx, &rows, `
		SELECT event_id, user_id, registration_id,
		       session_count, first_join_at, last_seen_at, total_seconds
		FROM online_session_summary
		WHERE event_id = $1
		ORDER BY total_seconds DESC`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("online_session SummaryByEvent: %w", err)
	}
	return rows, nil
}

// SummaryWithUsers returns per-participant session aggregates enriched with
// user identity (email, first_name, last_name) for the admin attendance table.
func (r *OnlineSessionRepository) SummaryWithUsers(
	ctx context.Context,
	eventID int,
) ([]model.OnlineParticipantDetail, error) {
	var rows []model.OnlineParticipantDetail
	err := r.db.SelectContext(ctx, &rows, `
		SELECT
		    u.id                                             AS user_id,
		    u.email,
		    u.last_name,
		    u.first_name,
		    r.id                                             AS registration_id,
		    COUNT(os.id)                                     AS session_count,
		    MIN(os.started_at)                               AS first_join_at,
		    MAX(COALESCE(os.ended_at, os.last_ping_at))      AS last_seen_at,
		    COALESCE(SUM(os.duration_seconds), 0)            AS total_seconds,
		    BOOL_OR(os.ended_at IS NULL)                     AS is_active
		FROM online_sessions os
		JOIN reg_users u ON u.id = os.user_id
		LEFT JOIN reg_registrations r
		    ON r.event_id = os.event_id AND r.user_id = os.user_id
		WHERE os.event_id = $1
		GROUP BY u.id, u.email, u.last_name, u.first_name, r.id
		ORDER BY total_seconds DESC`,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("online_session SummaryWithUsers: %w", err)
	}
	return rows, nil
}

// ActiveCount returns the number of currently open sessions for an event.
func (r *OnlineSessionRepository) ActiveCount(ctx context.Context, eventID int) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM online_sessions
		WHERE event_id = $1 AND ended_at IS NULL`,
		eventID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("online_session ActiveCount: %w", err)
	}
	return n, nil
}
