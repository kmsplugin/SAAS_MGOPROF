package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// TrackingRepository persists visitor/stream tracking events.
type TrackingRepository struct {
	db *sqlx.DB
}

func NewTrackingRepository(db *sqlx.DB) *TrackingRepository {
	return &TrackingRepository{db: db}
}

// Record inserts one tracking row.
func (r *TrackingRepository) Record(ctx context.Context, t model.TrackingEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reg_tracking
			(event_id, user_id, registration_id, action,
			 ip_address, geo_country, geo_region, geo_city,
			 isp_name, isp_asn, device_type, os_name, browser_name, user_agent)
		VALUES
			($1, $2, $3, $4,
			 $5, $6, $7, $8,
			 $9, $10, $11, $12, $13, $14)`,
		t.EventID, t.UserID, t.RegistrationID, t.Action,
		t.IPAddress, t.GeoCountry, t.GeoRegion, t.GeoCity,
		t.ISPName, t.ISPASN, t.DeviceType, t.OSName, t.BrowserName, t.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("tracking record: %w", err)
	}
	return nil
}

// ListByEvent returns all tracking events for one event, newest first.
func (r *TrackingRepository) ListByEvent(ctx context.Context, eventID int) ([]model.TrackingEvent, error) {
	var rows []model.TrackingEvent
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, event_id, user_id, registration_id, action,
		       ip_address, geo_country, geo_region, geo_city,
		       isp_name, isp_asn, device_type, os_name, browser_name, user_agent,
		       occurred_at
		FROM reg_tracking
		WHERE event_id = $1
		ORDER BY occurred_at DESC`, eventID)
	if err != nil {
		return nil, fmt.Errorf("tracking list: %w", err)
	}
	return rows, nil
}

// DeviceStatsByEvent returns device-type breakdown for one event.
func (r *TrackingRepository) DeviceStatsByEvent(ctx context.Context, eventID int, col string) ([]model.DeviceStat, error) {
	var rows []model.DeviceStat
	// col is safe — only called from trusted code with "device_type"/"os_name"/"browser_name"
	query := fmt.Sprintf(`
		SELECT COALESCE(NULLIF(%s,''),'unknown') AS name, COUNT(*) AS total
		FROM reg_tracking
		WHERE event_id = $1
		GROUP BY name
		ORDER BY total DESC`, col)
	err := r.db.SelectContext(ctx, &rows, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("device stats (%s): %w", col, err)
	}
	return rows, nil
}
