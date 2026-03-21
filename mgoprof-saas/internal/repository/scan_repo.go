package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// ScanRepository handles persistence for QR scan logs and attendance events.
type ScanRepository struct {
	db *sqlx.DB
}

func NewScanRepository(db *sqlx.DB) *ScanRepository {
	return &ScanRepository{db: db}
}

// FindRegistrationByToken resolves a participant_token to a registration + user row.
func (r *ScanRepository) FindRegistrationByToken(ctx context.Context, token string) (*model.Registration, *model.User, error) {
	type row struct {
		model.Registration
		UserID        int        `db:"user_id_u"`
		Email         string     `db:"email"`
		LastName      string     `db:"last_name"`
		FirstName     string     `db:"first_name"`
		Patronymic    string     `db:"patronymic"`
		Organization  string     `db:"organization"`
		District      string     `db:"district"`
		IsUnionMember bool       `db:"is_union_member"`
		UnionTicket   string     `db:"union_ticket"`
		ExtraInfo     string     `db:"extra_info"`
		GeoCountry    string     `db:"geo_country_u"`
		GeoRegion     string     `db:"geo_region_u"`
		GeoCity       string     `db:"geo_city_u"`
	}

	var res row
	err := r.db.GetContext(ctx, &res, `
		SELECT
			r.id, r.event_id, r.user_id, r.otp_code, r.otp_expires_at, r.otp_verified_at,
			r.status, r.ip_address, r.geo_country, r.geo_region, r.geo_city,
			r.isp_name, r.isp_asn, r.device_type, r.os_name, r.browser_name, r.user_agent,
			r.participant_token, r.checked_in_at, r.created_at, r.updated_at,
			u.id AS user_id_u, u.email, u.last_name, u.first_name, u.patronymic,
			u.organization, u.district, u.is_union_member, u.union_ticket, u.extra_info,
			u.geo_country AS geo_country_u, u.geo_region AS geo_region_u, u.geo_city AS geo_city_u
		FROM reg_registrations r
		JOIN reg_users u ON u.id = r.user_id
		WHERE r.participant_token = $1`, token)
	if err != nil {
		return nil, nil, err
	}

	reg := res.Registration
	reg.UserID = res.UserID

	user := &model.User{
		ID:            res.UserID,
		Email:         res.Email,
		LastName:      res.LastName,
		FirstName:     res.FirstName,
		Patronymic:    res.Patronymic,
		Organization:  res.Organization,
		District:      res.District,
		IsUnionMember: res.IsUnionMember,
		UnionTicket:   res.UnionTicket,
		ExtraInfo:     res.ExtraInfo,
		GeoCountry:    res.GeoCountry,
		GeoRegion:     res.GeoRegion,
		GeoCity:       res.GeoCity,
	}

	return &reg, user, nil
}

// GetStatusExtended returns the status_extended value for a registration.
func (r *ScanRepository) GetStatusExtended(ctx context.Context, regID int) (string, int, error) {
	var row struct {
		StatusExtended string `db:"status_extended"`
		ScanCount      int    `db:"scan_count"`
	}
	err := r.db.GetContext(ctx, &row,
		`SELECT COALESCE(status_extended, 'registered') AS status_extended,
		        COALESCE(scan_count, 0) AS scan_count
		 FROM reg_registrations WHERE id = $1`, regID)
	return row.StatusExtended, row.ScanCount, err
}

// SetStatusExtended updates status_extended and increments scan_count.
func (r *ScanRepository) SetStatusExtended(ctx context.Context, regID int, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE reg_registrations
		SET status_extended = $1,
		    scan_count       = COALESCE(scan_count, 0) + 1,
		    updated_at       = NOW()
		WHERE id = $2`, status, regID)
	return err
}

// SetFirstEntry records the first_entry_at timestamp (only if not yet set).
func (r *ScanRepository) SetFirstEntry(ctx context.Context, regID int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE reg_registrations
		SET first_entry_at = COALESCE(first_entry_at, NOW()),
		    updated_at     = NOW()
		WHERE id = $1`, regID)
	return err
}

// SetLastExit records the last_exit_at timestamp and updates participation_seconds.
func (r *ScanRepository) SetLastExit(ctx context.Context, regID int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE reg_registrations
		SET last_exit_at           = NOW(),
		    participation_seconds  = COALESCE(participation_seconds, 0) +
		        EXTRACT(EPOCH FROM (NOW() - COALESCE(first_entry_at, NOW())))::int,
		    updated_at             = NOW()
		WHERE id = $1`, regID)
	return err
}

// InsertScanLog writes an immutable scan log row and returns the new ID.
func (r *ScanRepository) InsertScanLog(ctx context.Context, log *model.ScanLog) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO scan_logs
		    (event_id, registration_id, scanned_token, scan_mode, scan_result,
		     operator_id, device_info, ip_address, note, scanned_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		log.EventID, log.RegistrationID, log.ScannedToken, log.ScanMode, log.ScanResult,
		log.OperatorID, log.DeviceInfo, log.IPAddress, log.Note, time.Now(),
	).Scan(&id)
	return id, err
}

// InsertAttendanceEvent writes one entry/exit row.
func (r *ScanRepository) InsertAttendanceEvent(ctx context.Context, evt *model.AttendanceEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO attendance_events (event_id, registration_id, action, scan_log_id, occurred_at)
		VALUES ($1,$2,$3,$4,$5)`,
		evt.EventID, evt.RegistrationID, evt.Action, evt.ScanLogID, time.Now())
	return err
}

// InsertStatusLog records a status FSM transition.
func (r *ScanRepository) InsertStatusLog(ctx context.Context, log *model.RegistrationStatusLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO registration_status_log
		    (registration_id, event_id, prev_status, new_status, changed_by, change_source, note, changed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())`,
		log.RegistrationID, log.EventID, log.PrevStatus, log.NewStatus,
		log.ChangedBy, log.ChangeSource, log.Note)
	return err
}

// GetScanLogs returns the most recent scan logs for an event (up to limit rows).
func (r *ScanRepository) GetScanLogs(ctx context.Context, eventID, limit int) ([]model.ScanLog, error) {
	var rows []model.ScanLog
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, event_id, registration_id, scanned_token, scan_mode, scan_result,
		       operator_id, COALESCE(device_info,'') AS device_info,
		       COALESCE(ip_address,'') AS ip_address, COALESCE(note,'') AS note, scanned_at
		FROM scan_logs
		WHERE event_id = $1
		ORDER BY scanned_at DESC
		LIMIT $2`, eventID, limit)
	return rows, err
}

// GetAttendanceSummary returns total entries, exits, and current presence count for an event.
func (r *ScanRepository) GetAttendanceSummary(ctx context.Context, eventID int) (entries, exits, present int, err error) {
	type summary struct {
		Entries int `db:"entries"`
		Exits   int `db:"exits"`
	}
	var s summary
	err = r.db.GetContext(ctx, &s, `
		SELECT
		    COUNT(*) FILTER (WHERE action = 'entry') AS entries,
		    COUNT(*) FILTER (WHERE action = 'exit')  AS exits
		FROM attendance_events
		WHERE event_id = $1`, eventID)
	if err != nil {
		return
	}
	entries = s.Entries
	exits = s.Exits
	if entries > exits {
		present = entries - exits
	}
	return
}
