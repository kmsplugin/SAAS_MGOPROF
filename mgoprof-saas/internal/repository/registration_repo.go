package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// RegistrationRepository handles all DB operations for reg_registrations.
type RegistrationRepository struct {
	db *sqlx.DB
}

func NewRegistrationRepository(db *sqlx.DB) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

func (r *RegistrationRepository) FindByEventAndUser(
	ctx context.Context,
	eventID, userID int,
) (*model.Registration, error) {
	var reg model.Registration
	err := r.db.GetContext(ctx, &reg,
		`SELECT * FROM reg_registrations WHERE event_id = $1 AND user_id = $2 LIMIT 1`,
		eventID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reg FindByEventAndUser: %w", err)
	}
	return &reg, nil
}

func (r *RegistrationRepository) FindByEventAndUserTx(
	ctx context.Context,
	tx *sqlx.Tx,
	eventID, userID int,
) (*model.Registration, error) {
	var reg model.Registration
	err := tx.GetContext(ctx, &reg,
		`SELECT * FROM reg_registrations WHERE event_id = $1 AND user_id = $2 LIMIT 1`,
		eventID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reg FindByEventAndUserTx: %w", err)
	}
	return &reg, nil
}

func (r *RegistrationRepository) CreateTx(
	ctx context.Context,
	tx *sqlx.Tx,
	eventID, userID int,
	otp string,
	expiresAt time.Time,
	ip string,
	geo model.Geo,
) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO reg_registrations
			(event_id, user_id, otp_code, otp_expires_at, status,
			 ip_address, geo_country, geo_region, geo_city)
		 VALUES ($1,$2,$3,$4,'pending',$5,$6,$7,$8)`,
		eventID, userID, otp, expiresAt, ip, geo.Country, geo.Region, geo.City,
	)
	if err != nil {
		return fmt.Errorf("reg CreateTx: %w", err)
	}
	return nil
}

func (r *RegistrationRepository) UpdateOTPTx(
	ctx context.Context,
	tx *sqlx.Tx,
	id int,
	otp string,
	expiresAt time.Time,
	ip string,
	geo model.Geo,
) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE reg_registrations
		 SET otp_code=$1, otp_expires_at=$2, otp_verified_at=NULL,
		     status='pending', ip_address=$3, geo_country=$4,
		     geo_region=$5, geo_city=$6, updated_at=NOW()
		 WHERE id=$7`,
		otp, expiresAt, ip, geo.Country, geo.Region, geo.City, id,
	)
	if err != nil {
		return fmt.Errorf("reg UpdateOTPTx: %w", err)
	}
	return nil
}

func (r *RegistrationRepository) SetVerified(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reg_registrations
		 SET status='verified', otp_verified_at=NOW(), updated_at=NOW()
		 WHERE id=$1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("reg SetVerified: %w", err)
	}
	return nil
}

// ListAll returns all registrations joined with events and users (for admin).
func (r *RegistrationRepository) ListAll(ctx context.Context) ([]model.RegistrationRow, error) {
	var rows []model.RegistrationRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT
			r.created_at AS reg_datetime,
			e.title      AS event_title,
			u.last_name,
			u.first_name,
			u.patronymic,
			u.organization,
			u.district,
			u.email,
			u.is_union_member,
			COALESCE(u.union_ticket, '')  AS union_ticket,
			COALESCE(u.extra_info, '')    AS extra_info,
			COALESCE(r.ip_address, '')    AS ip_address,
			COALESCE(r.geo_country, '')   AS geo_country,
			COALESCE(r.geo_region, '')    AS geo_region,
			COALESCE(r.geo_city, '')      AS geo_city,
			r.status
		FROM reg_registrations r
		INNER JOIN reg_events e ON e.id = r.event_id
		INNER JOIN reg_users  u ON u.id = r.user_id
		ORDER BY r.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("reg ListAll: %w", err)
	}
	return rows, nil
}

// ListRecent returns the N most recent registrations (for admin dashboard).
func (r *RegistrationRepository) ListRecent(ctx context.Context, limit int) ([]model.RegistrationRow, error) {
	var rows []model.RegistrationRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT
			r.created_at AS reg_datetime,
			e.title      AS event_title,
			u.last_name,
			u.first_name,
			u.patronymic,
			u.organization,
			u.district,
			u.email,
			u.is_union_member,
			COALESCE(u.union_ticket, '') AS union_ticket,
			COALESCE(u.extra_info, '')   AS extra_info,
			COALESCE(r.ip_address, '')   AS ip_address,
			COALESCE(r.geo_country, '')  AS geo_country,
			COALESCE(r.geo_region, '')   AS geo_region,
			COALESCE(r.geo_city, '')     AS geo_city,
			r.status
		FROM reg_registrations r
		INNER JOIN reg_events e ON e.id = r.event_id
		INNER JOIN reg_users  u ON u.id = r.user_id
		ORDER BY r.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("reg ListRecent: %w", err)
	}
	return rows, nil
}

// ListByUserVerified returns verified registrations for a given user (cabinet).
func (r *RegistrationRepository) ListByUserVerified(ctx context.Context, userID int) ([]model.Event, error) {
	var events []model.Event
	err := r.db.SelectContext(ctx, &events, `
		SELECT e.id, e.title, e.description, e.event_date, e.event_time,
		       e.cabinet_link, e.is_active, e.created_at, e.updated_at
		FROM reg_events e
		INNER JOIN reg_registrations r ON r.event_id = e.id
		WHERE r.user_id = $1 AND r.status = 'verified'
		ORDER BY e.event_date ASC, e.event_time ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("reg ListByUserVerified: %w", err)
	}
	return events, nil
}

// GetStats fetches dashboard counters.
func (r *RegistrationRepository) GetStats(ctx context.Context) (*model.Stats, error) {
	var s model.Stats
	err := r.db.GetContext(ctx, &s, `
		SELECT
			(SELECT COUNT(*) FROM reg_users) AS total_users,
			(SELECT COUNT(*) FROM reg_registrations WHERE status = 'verified') AS verified_regs,
			(SELECT COUNT(*) FROM reg_registrations WHERE status = 'pending')  AS pending_regs,
			(SELECT COUNT(*) FROM reg_events WHERE is_active = TRUE)           AS active_events`)
	if err != nil {
		return nil, fmt.Errorf("reg GetStats: %w", err)
	}
	return &s, nil
}
