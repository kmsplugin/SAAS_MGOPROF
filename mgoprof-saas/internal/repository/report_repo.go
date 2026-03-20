package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"mgoprof-saas/internal/model"
)

// ReportRepository provides analytics queries for per-event reports.
type ReportRepository struct {
	db *sqlx.DB
}

func NewReportRepository(db *sqlx.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// GetEventReport fetches all aggregated data for a single event report.
func (r *ReportRepository) GetEventReport(ctx context.Context, eventID int) (*model.EventReport, error) {
	// -- Basic stats ---------------------------------------------------------
	var stats model.ReportStats
	err := r.db.GetContext(ctx, &stats, `
		SELECT
			COUNT(*)                                              AS total_regs,
			COUNT(*) FILTER (WHERE r.status = 'verified')        AS verified_regs,
			COUNT(*) FILTER (WHERE r.status = 'pending')         AS pending_regs,
			COUNT(*) FILTER (WHERE u.is_union_member = TRUE AND r.status = 'verified') AS union_members,
			COUNT(*) FILTER (WHERE u.is_union_member = FALSE AND r.status = 'verified') AS non_union,
			MIN(r.created_at)                                    AS first_reg_at,
			MAX(r.created_at)                                    AS last_reg_at
		FROM reg_registrations r
		JOIN reg_users u ON u.id = r.user_id
		WHERE r.event_id = $1`, eventID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("report stats: %w", err)
	}

	// -- Districts -----------------------------------------------------------
	var districts []model.DistrictStat
	err = r.db.SelectContext(ctx, &districts, `
		SELECT
			COALESCE(NULLIF(TRIM(u.district), ''), 'Не указан') AS district,
			COUNT(*)                                             AS total,
			COUNT(*) FILTER (WHERE r.status = 'verified')       AS verified
		FROM reg_registrations r
		JOIN reg_users u ON u.id = r.user_id
		WHERE r.event_id = $1
		GROUP BY district
		ORDER BY total DESC`, eventID)
	if err != nil {
		return nil, fmt.Errorf("report districts: %w", err)
	}

	// -- Timeline (5-min buckets) --------------------------------------------
	var timeline []model.TimelinePoint
	err = r.db.SelectContext(ctx, &timeline, `
		SELECT
			date_trunc('hour', created_at) +
			  (EXTRACT(MINUTE FROM created_at)::int / 5) * interval '5 minutes' AS bucket,
			COUNT(*)                                                             AS count,
			COUNT(*) FILTER (WHERE status = 'verified')                         AS verified
		FROM reg_registrations
		WHERE event_id = $1
		GROUP BY bucket
		ORDER BY bucket`, eventID)
	if err != nil {
		return nil, fmt.Errorf("report timeline: %w", err)
	}

	// -- Organizations (top 10) ----------------------------------------------
	var orgs []model.OrgStat
	err = r.db.SelectContext(ctx, &orgs, `
		SELECT
			COALESCE(NULLIF(TRIM(u.organization), ''), 'Не указана') AS organization,
			COUNT(*)                                                  AS total
		FROM reg_registrations r
		JOIN reg_users u ON u.id = r.user_id
		WHERE r.event_id = $1
		GROUP BY organization
		ORDER BY total DESC
		LIMIT 10`, eventID)
	if err != nil {
		return nil, fmt.Errorf("report orgs: %w", err)
	}

	return &model.EventReport{
		Stats:     stats,
		Districts: districts,
		Timeline:  timeline,
		Orgs:      orgs,
	}, nil
}

// GetEventRegistrationsForExport returns all registrations for one event (for CSV).
func (r *ReportRepository) GetEventRegistrationsForExport(ctx context.Context, eventID int) ([]model.RegistrationRow, error) {
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
		WHERE r.event_id = $1
		ORDER BY r.status DESC, u.last_name, u.first_name`, eventID)
	if err != nil {
		return nil, fmt.Errorf("export registrations: %w", err)
	}
	return rows, nil
}

// GetAllEventsExport returns all registrations for all events grouped by event (for global CSV).
func (r *ReportRepository) GetAllEventsExport(ctx context.Context) ([]model.RegistrationRow, error) {
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
		ORDER BY e.event_date DESC, e.event_time DESC, r.status DESC, u.last_name`)
	if err != nil {
		return nil, fmt.Errorf("export all: %w", err)
	}
	return rows, nil
}

