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

// CreateTx inserts a new pending registration and returns its ID.
func (r *RegistrationRepository) CreateTx(
	ctx context.Context,
	tx *sqlx.Tx,
	eventID, userID int,
	otp string,
	expiresAt time.Time,
	ip string,
	geo model.Geo,
	dev model.DeviceInfo,
) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx,
		`INSERT INTO reg_registrations
			(event_id, user_id, otp_code, otp_expires_at, otp_sent_at, status,
			 ip_address, geo_country, geo_region, geo_city,
			 isp_name, isp_asn, device_type, os_name, browser_name)
		 VALUES ($1,$2,$3,$4,NOW(),'pending',$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id`,
		eventID, userID, otp, expiresAt, ip,
		geo.Country, geo.Region, geo.City,
		geo.ISPName, geo.ISPASN,
		dev.DeviceType, dev.OSName, dev.BrowserName,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("reg CreateTx: %w", err)
	}
	return id, nil
}

// UpdateOTPTx refreshes OTP and device/geo info on an existing pending registration.
func (r *RegistrationRepository) UpdateOTPTx(
	ctx context.Context,
	tx *sqlx.Tx,
	id int,
	otp string,
	expiresAt time.Time,
	ip string,
	geo model.Geo,
	dev model.DeviceInfo,
) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE reg_registrations
		 SET otp_code=$1, otp_expires_at=$2, otp_sent_at=NOW(), otp_verified_at=NULL,
		     status='pending', ip_address=$3, geo_country=$4, geo_region=$5, geo_city=$6,
		     isp_name=$7, isp_asn=$8, device_type=$9, os_name=$10, browser_name=$11,
		     updated_at=NOW()
		 WHERE id=$12`,
		otp, expiresAt, ip, geo.Country, geo.Region, geo.City,
		geo.ISPName, geo.ISPASN, dev.DeviceType, dev.OSName, dev.BrowserName,
		id,
	)
	if err != nil {
		return fmt.Errorf("reg UpdateOTPTx: %w", err)
	}
	return nil
}

// SetVerified marks a registration as verified, assigns a participant token,
// and clears the OTP code so it cannot be reused even if status checks are bypassed.
func (r *RegistrationRepository) SetVerified(ctx context.Context, id int, token string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reg_registrations
		 SET status='verified', otp_verified_at=NOW(), updated_at=NOW(),
		     participant_token=$2,
		     otp_code=NULL, otp_expires_at=NOW()
		 WHERE id=$1`,
		id, token,
	)
	if err != nil {
		return fmt.Errorf("reg SetVerified: %w", err)
	}
	return nil
}

// FindByToken returns a registration by its participant_token (for QR scanning).
func (r *RegistrationRepository) FindByToken(ctx context.Context, token string) (*model.Registration, error) {
	var reg model.Registration
	err := r.db.GetContext(ctx, &reg,
		`SELECT * FROM reg_registrations WHERE participant_token=$1 LIMIT 1`, token)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reg FindByToken: %w", err)
	}
	return &reg, nil
}

// SetCheckedIn records the check-in timestamp for a registration.
func (r *RegistrationRepository) SetCheckedIn(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reg_registrations SET checked_in_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("reg SetCheckedIn: %w", err)
	}
	return nil
}

const regRowSelectSQL = `
	SELECT
		r.id                                            AS reg_id,
		r.created_at                                    AS reg_datetime,
		r.event_id,
		e.title                                         AS event_title,
		u.last_name,
		u.first_name,
		u.patronymic,
		u.organization,
		u.district,
		u.email,
		u.is_union_member,
		COALESCE(u.union_ticket, '')                    AS union_ticket,
		COALESCE(u.extra_info, '')                      AS extra_info,
		COALESCE(r.ip_address, '')                      AS ip_address,
		COALESCE(r.geo_country, '')                     AS geo_country,
		COALESCE(r.geo_region, '')                      AS geo_region,
		COALESCE(r.geo_city, '')                        AS geo_city,
		COALESCE(r.isp_name, '')                        AS isp_name,
		COALESCE(r.isp_asn, '')                         AS isp_asn,
		COALESCE(r.device_type, '')                     AS device_type,
		COALESCE(r.os_name, '')                         AS os_name,
		COALESCE(r.browser_name, '')                    AS browser_name,
		r.status,
		COALESCE(r.status_extended, r.status)           AS status_extended,
		COALESCE(r.scan_count, 0)                       AS scan_count,
		r.otp_sent_at,
		r.otp_verified_at,
		r.checked_in_at,
		r.first_entry_at,
		r.last_exit_at
	FROM reg_registrations r
	INNER JOIN reg_events e ON e.id = r.event_id
	INNER JOIN reg_users  u ON u.id = r.user_id`

// ListByEvent returns all registrations for a specific event (admin view).
func (r *RegistrationRepository) ListByEvent(ctx context.Context, eventID int) ([]model.RegistrationRow, error) {
	var rows []model.RegistrationRow
	err := r.db.SelectContext(ctx, &rows,
		regRowSelectSQL+` WHERE r.event_id = $1 ORDER BY r.created_at DESC`, eventID)
	if err != nil {
		return nil, fmt.Errorf("reg ListByEvent: %w", err)
	}
	return rows, nil
}

// ListAll returns all registrations joined with events and users (for admin).
func (r *RegistrationRepository) ListAll(ctx context.Context) ([]model.RegistrationRow, error) {
	var rows []model.RegistrationRow
	err := r.db.SelectContext(ctx, &rows, regRowSelectSQL+` ORDER BY r.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("reg ListAll: %w", err)
	}
	return rows, nil
}

// ListRecent returns the N most recent registrations (for admin dashboard).
func (r *RegistrationRepository) ListRecent(ctx context.Context, limit int) ([]model.RegistrationRow, error) {
	var rows []model.RegistrationRow
	err := r.db.SelectContext(ctx, &rows, regRowSelectSQL+` ORDER BY r.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("reg ListRecent: %w", err)
	}
	return rows, nil
}

// CabinetEventRow combines event data with the participant's role-specific link.
type CabinetEventRow struct {
	model.Event
	ParticipantRole string `db:"participant_role" json:"participant_role"`
	// EventLink is the role-resolved URL to show in the cabinet:
	//   speakers/moderators → speaker_link (if set)
	//   all others          → viewer_link (if set), falling back to cabinet_link
	EventLink string `db:"event_link" json:"event_link"`
}

// ListByUserVerified returns verified registrations for a given user (cabinet).
// The returned EventLink is resolved per participant role.
func (r *RegistrationRepository) ListByUserVerified(ctx context.Context, userID int) ([]CabinetEventRow, error) {
	var rows []CabinetEventRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT e.id, e.title, e.description, e.event_date, e.event_time,
		       e.cabinet_link, e.speaker_link, e.viewer_link,
		       e.is_active, e.is_online, e.event_type,
		       e.start_at, e.end_at, e.venue, e.address,
		       e.capacity, e.cover_url, e.check_in_mode,
		       e.registration_opens_at, e.registration_closes_at,
		       e.badge_template_id, e.max_scans_per_ticket,
		       e.created_at, e.updated_at,
		       r.participant_role,
		       CASE
		           WHEN r.participant_role IN ('speaker','moderator') AND e.speaker_link != ''
		               THEN e.speaker_link
		           WHEN e.viewer_link != ''
		               THEN e.viewer_link
		           ELSE e.cabinet_link
		       END AS event_link
		FROM reg_events e
		INNER JOIN reg_registrations r ON r.event_id = e.id
		WHERE r.user_id = $1 AND r.status = 'verified'
		ORDER BY e.event_date ASC, e.event_time ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("reg ListByUserVerified: %w", err)
	}
	return rows, nil
}

// CountVerifiedByEvent returns the number of verified registrations for an event.
// Used for capacity enforcement before inserting a new registration.
func (r *RegistrationRepository) CountVerifiedByEvent(ctx context.Context, tx *sqlx.Tx, eventID int) (int, error) {
	var n int
	err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reg_registrations WHERE event_id = $1 AND status = 'verified'`, eventID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("reg CountVerifiedByEvent: %w", err)
	}
	return n, nil
}

// CancelByUser sets a registration status to 'cancelled' for the given user and event.
// Returns false (no error) if the registration does not exist or is already cancelled.
func (r *RegistrationRepository) CancelByUser(ctx context.Context, userID, eventID int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE reg_registrations
		 SET status = 'cancelled', updated_at = NOW()
		 WHERE user_id = $1 AND event_id = $2 AND status != 'cancelled'`,
		userID, eventID,
	)
	if err != nil {
		return fmt.Errorf("reg CancelByUser: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("регистрация не найдена или уже отменена")
	}
	return nil
}

// GetStats fetches dashboard counters.
func (r *RegistrationRepository) GetStats(ctx context.Context) (*model.Stats, error) {
	var s model.Stats
	err := r.db.GetContext(ctx, &s, `
		SELECT
			(SELECT COUNT(*) FROM reg_users)                                              AS total_users,
			(SELECT COUNT(*) FROM reg_registrations WHERE status = 'verified')            AS verified_regs,
			(SELECT COUNT(*) FROM reg_registrations WHERE status = 'pending')             AS pending_regs,
			(SELECT COUNT(*) FROM reg_events       WHERE is_active = TRUE)                AS active_events,
			(SELECT COUNT(*) FROM reg_registrations WHERE checked_in_at IS NOT NULL)      AS checked_in`)
	if err != nil {
		return nil, fmt.Errorf("reg GetStats: %w", err)
	}
	return &s, nil
}

// ListRegistrationsForExport returns all registrations for a user (for GDPR data export).
func (r *RegistrationRepository) ListRegistrationsForExport(ctx context.Context, userID int) ([]model.RegistrationExport, error) {
	var rows []model.RegistrationExport
	err := r.db.SelectContext(ctx, &rows, `
		SELECT r.event_id, e.title AS event_title, r.status, r.created_at
		FROM   reg_registrations r
		JOIN   reg_events e ON e.id = r.event_id
		WHERE  r.user_id = $1
		ORDER  BY r.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("reg ListRegistrationsForExport: %w", err)
	}
	return rows, nil
}
