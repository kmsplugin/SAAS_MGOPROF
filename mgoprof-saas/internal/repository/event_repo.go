package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// eventCacheEntry wraps a cached event with its expiry time.
type eventCacheEntry struct {
	event     *model.Event
	expiresAt time.Time
}

// EventRepository handles all DB operations for reg_events.
// It includes an in-memory TTL cache for FindActiveByID to avoid a DB round-trip
// on every registration request during high-concurrency bursts.
type EventRepository struct {
	db    *sqlx.DB
	mu    sync.RWMutex
	cache map[int]*eventCacheEntry
	ttl   time.Duration
}

func NewEventRepository(db *sqlx.DB) *EventRepository {
	return &EventRepository{
		db:    db,
		cache: make(map[int]*eventCacheEntry),
		ttl:   60 * time.Second,
	}
}

func (r *EventRepository) cacheGet(id int) (*model.Event, bool) {
	r.mu.RLock()
	entry, ok := r.cache[id]
	r.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.event, true
}

func (r *EventRepository) cacheSet(id int, event *model.Event) {
	r.mu.Lock()
	r.cache[id] = &eventCacheEntry{event: event, expiresAt: time.Now().Add(r.ttl)}
	r.mu.Unlock()
}

func (r *EventRepository) cacheInvalidate(id int) {
	r.mu.Lock()
	delete(r.cache, id)
	r.mu.Unlock()
}

func (r *EventRepository) FindActiveByID(ctx context.Context, id int) (*model.Event, error) {
	if e, ok := r.cacheGet(id); ok {
		return e, nil
	}
	var e model.Event
	err := r.db.GetContext(ctx, &e,
		`SELECT * FROM reg_events WHERE id = $1 AND is_active = TRUE LIMIT 1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("event FindActiveByID: %w", err)
	}
	r.cacheSet(id, &e)
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
		`SELECT * FROM reg_events WHERE is_active = TRUE ORDER BY COALESCE(start_at, NOW()) ASC, event_date ASC, event_time ASC`)
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
			COUNT(r.id)                                                  AS total_regs,
			COUNT(r.id) FILTER (WHERE r.status = 'verified')            AS verified_regs
		FROM reg_events e
		LEFT JOIN reg_registrations r ON r.event_id = e.id
		GROUP BY e.id
		ORDER BY e.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("event ListWithStats: %w", err)
	}
	return events, nil
}

// CountVerified returns the number of verified registrations for capacity checks.
func (r *EventRepository) CountVerified(ctx context.Context, eventID int) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reg_registrations WHERE event_id = $1 AND status = 'verified'`, eventID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("event CountVerified: %w", err)
	}
	return n, nil
}

func (r *EventRepository) Create(ctx context.Context, req model.CreateEventRequest) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO reg_events
		   (title, description, event_date, event_time, start_at, venue, address,
		    capacity, cover_url, cabinet_link, is_active, is_online)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		req.Title, req.Description, req.EventDate, req.EventTime,
		req.StartAt, req.Venue, req.Address,
		req.Capacity, req.CoverURL, req.CabinetLink,
		req.IsActive, req.IsOnline,
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
		     start_at=$5, venue=$6, address=$7, capacity=$8,
		     cover_url=$9, cabinet_link=$10, is_active=$11, is_online=$12,
		     updated_at=NOW()
		 WHERE id=$13`,
		req.Title, req.Description, req.EventDate, req.EventTime,
		req.StartAt, req.Venue, req.Address, req.Capacity,
		req.CoverURL, req.CabinetLink, req.IsActive, req.IsOnline, id,
	)
	if err != nil {
		return fmt.Errorf("event Update: %w", err)
	}
	r.cacheInvalidate(id)
	return nil
}
