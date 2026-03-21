package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"platform/api/internal/model"
)

type EventRepository struct{ db *sqlx.DB }

func NewEventRepository(db *sqlx.DB) *EventRepository { return &EventRepository{db: db} }

func (r *EventRepository) FindByID(ctx context.Context, tenantID, id string) (*model.Event, error) {
	var e model.Event
	err := r.db.GetContext(ctx, &e,
		`SELECT * FROM events WHERE id = $1 AND tenant_id = $2 LIMIT 1`, id, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("event FindByID: %w", err)
	}
	return &e, nil
}

func (r *EventRepository) ListByTenant(ctx context.Context, tenantID, status string, limit, offset int) ([]model.Event, error) {
	query := `SELECT * FROM events WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	idx := 2
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, status)
		idx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	var events []model.Event
	if err := r.db.SelectContext(ctx, &events, query, args...); err != nil {
		return nil, fmt.Errorf("event ListByTenant: %w", err)
	}
	return events, nil
}

func (r *EventRepository) Create(ctx context.Context, tenantID, createdBy string, req model.CreateEventRequest) (*model.Event, error) {
	var startAt, endAt *time.Time
	if req.StartAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartAt)
		if err == nil {
			startAt = &t
		}
	}
	if req.EndAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndAt)
		if err == nil {
			endAt = &t
		}
	}
	tz := req.Timezone
	if tz == "" {
		tz = "UTC"
	}
	cap := req.ViewerCapacity
	if cap == 0 {
		cap = 10000
	}

	var e model.Event
	err := r.db.GetContext(ctx, &e,
		`INSERT INTO events
		   (tenant_id, title, description, type, start_at, end_at, timezone,
		    cover_url, capacity, viewer_capacity, is_public, registration_required, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING *`,
		tenantID, req.Title, req.Description, req.Type, startAt, endAt, tz,
		req.CoverURL, req.Capacity, cap, req.IsPublic, req.RegistrationRequired, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("event Create: %w", err)
	}
	return &e, nil
}

func (r *EventRepository) Update(ctx context.Context, tenantID, id string, req model.UpdateEventRequest) (*model.Event, error) {
	var startAt, endAt *time.Time
	if req.StartAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartAt)
		if err == nil {
			startAt = &t
		}
	}
	if req.EndAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndAt)
		if err == nil {
			endAt = &t
		}
	}
	var e model.Event
	err := r.db.GetContext(ctx, &e,
		`UPDATE events SET
		   title               = COALESCE(NULLIF($1,''), title),
		   description         = COALESCE(NULLIF($2,''), description),
		   start_at            = COALESCE($3, start_at),
		   end_at              = COALESCE($4, end_at),
		   timezone            = COALESCE(NULLIF($5,''), timezone),
		   cover_url           = COALESCE(NULLIF($6,''), cover_url),
		   capacity            = COALESCE($7, capacity),
		   viewer_capacity     = COALESCE($8, viewer_capacity),
		   is_public           = COALESCE($9, is_public),
		   registration_required = COALESCE($10, registration_required),
		   updated_at          = NOW()
		 WHERE id = $11 AND tenant_id = $12
		 RETURNING *`,
		req.Title, req.Description, startAt, endAt, req.Timezone, req.CoverURL,
		req.Capacity, req.ViewerCapacity, req.IsPublic, req.RegistrationRequired,
		id, tenantID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("event Update: %w", err)
	}
	return &e, nil
}

func (r *EventRepository) UpdateStatus(ctx context.Context, tenantID, id, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE events SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		status, id, tenantID)
	if err != nil {
		return fmt.Errorf("event UpdateStatus: %w", err)
	}
	return nil
}
