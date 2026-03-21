package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"platform/api/internal/model"
)

type RoomRepository struct{ db *sqlx.DB }

func NewRoomRepository(db *sqlx.DB) *RoomRepository { return &RoomRepository{db: db} }

func (r *RoomRepository) FindByID(ctx context.Context, tenantID, id string) (*model.Room, error) {
	var room model.Room
	err := r.db.GetContext(ctx, &room,
		`SELECT * FROM rooms WHERE id = $1 AND tenant_id = $2 LIMIT 1`, id, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("room FindByID: %w", err)
	}
	return &room, nil
}

func (r *RoomRepository) FindByEvent(ctx context.Context, tenantID, eventID string) ([]model.Room, error) {
	var rooms []model.Room
	err := r.db.SelectContext(ctx, &rooms,
		`SELECT * FROM rooms WHERE event_id = $1 AND tenant_id = $2 ORDER BY created_at DESC`,
		eventID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("room FindByEvent: %w", err)
	}
	return rooms, nil
}

func (r *RoomRepository) FindActiveByEvent(ctx context.Context, tenantID, eventID string) (*model.Room, error) {
	var room model.Room
	err := r.db.GetContext(ctx, &room,
		`SELECT * FROM rooms WHERE event_id = $1 AND tenant_id = $2 AND status = 'active' LIMIT 1`,
		eventID, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("room FindActiveByEvent: %w", err)
	}
	return &room, nil
}

func (r *RoomRepository) Create(ctx context.Context, tenantID, eventID, livekitName, mode string, maxParticipants int) (*model.Room, error) {
	var room model.Room
	err := r.db.GetContext(ctx, &room,
		`INSERT INTO rooms (event_id, tenant_id, livekit_room_name, mode, max_participants)
		 VALUES ($1, $2, $3, $4, $5) RETURNING *`,
		eventID, tenantID, livekitName, mode, maxParticipants,
	)
	if err != nil {
		return nil, fmt.Errorf("room Create: %w", err)
	}
	return &room, nil
}

func (r *RoomRepository) SetStatus(ctx context.Context, id, status string) error {
	query := `UPDATE rooms SET status = $1 WHERE id = $2`
	if status == "active" {
		query = `UPDATE rooms SET status = $1, started_at = NOW() WHERE id = $2`
	} else if status == "ended" {
		query = `UPDATE rooms SET status = $1, ended_at = NOW() WHERE id = $2`
	}
	_, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("room SetStatus: %w", err)
	}
	return nil
}
