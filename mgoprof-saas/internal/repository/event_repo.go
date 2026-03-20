package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// EventRepository handles all DB operations for reg_events.
type EventRepository struct {
	db *sqlx.DB
}

func NewEventRepository(db *sqlx.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) FindActiveByID(ctx context.Context, id int) (*model.Event, error) {
	var e model.Event
	err := r.db.GetContext(ctx, &e,
		`SELECT * FROM reg_events WHERE id = $1 AND is_active = TRUE LIMIT 1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("event FindActiveByID: %w", err)
	}
	return &e, nil
}

func (r *EventRepository) FindByID(ctx context.Context, id int) (*model.Event, error) {
	var e model.Event
	err := r.db.GetContext(ctx, &e,
		`SELECT * FROM reg_events WHERE id = $1 LIMIT 1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("event FindByID: %w", err)
	}
	return &e, nil
}

func (r *EventRepository) ListActive(ctx context.Context) ([]model.Event, error) {
	var events []model.Event
	err := r.db.SelectContext(ctx, &events,
		`SELECT * FROM reg_events WHERE is_active = TRUE ORDER BY event_date ASC, event_time ASC`)
	if err != nil {
		return nil, fmt.Errorf("event ListActive: %w", err)
	}
	return events, nil
}

func (r *EventRepository) ListWithStats(ctx context.Context) ([]model.EventWithStats, error) {
	var events []model.EventWithStats
	err := r.db.SelectContext(ctx, &events, `
		SELECT
			e.*,
			COUNT(r.id) AS total_regs,
			COUNT(r.id) FILTER (WHERE r.status = 'verified') AS verified_regs
		FROM reg_events e
		LEFT JOIN reg_registrations r ON r.event_id = e.id
		GROUP BY e.id
		ORDER BY e.event_date DESC, e.event_time DESC`)
	if err != nil {
		return nil, fmt.Errorf("event ListWithStats: %w", err)
	}
	return events, nil
}

func (r *EventRepository) Create(ctx context.Context, req model.CreateEventRequest) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO reg_events (title, description, event_date, event_time, cabinet_link, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		req.Title, req.Description, req.EventDate, req.EventTime, req.CabinetLink, req.IsActive,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("event Create: %w", err)
	}
	return id, nil
}

func (r *EventRepository) Update(ctx context.Context, id int, req model.CreateEventRequest) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reg_events
		 SET title=$1, description=$2, event_date=$3, event_time=$4,
		     cabinet_link=$5, is_active=$6, updated_at=NOW()
		 WHERE id=$7`,
		req.Title, req.Description, req.EventDate, req.EventTime,
		req.CabinetLink, req.IsActive, id,
	)
	if err != nil {
		return fmt.Errorf("event Update: %w", err)
	}
	return nil
}
