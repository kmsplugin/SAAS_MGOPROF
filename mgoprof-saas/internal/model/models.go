package model

import "time"

// ParticipantRole defines the allowed roles for a registration.
// Roles determine which event link is shown in the cabinet and
// which tracking label is applied.
const (
	ParticipantRoleSpeaker   = "speaker"
	ParticipantRoleModerator = "moderator"
	ParticipantRoleViewer    = "viewer"
	ParticipantRoleDelegate  = "delegate"
	ParticipantRoleGuest     = "guest"
)

// SpeakerRoles is the set of roles that resolve to speaker_link.
var SpeakerRoles = map[string]bool{
	ParticipantRoleSpeaker:   true,
	ParticipantRoleModerator: true,
}

// ── Report analytics types ────────────────────────────────────────────────────

// ReportStats holds aggregate counters for a single event.
type ReportStats struct {
	TotalRegs    int        `db:"total_regs"    json:"total_regs"`
	VerifiedRegs int        `db:"verified_regs" json:"verified_regs"`
	PendingRegs  int        `db:"pending_regs"  json:"pending_regs"`
	UnionMembers int        `db:"union_members" json:"union_members"`
	NonUnion     int        `db:"non_union"     json:"non_union"`
	FirstRegAt   *time.Time `db:"first_reg_at"  json:"first_reg_at,omitempty"`
	LastRegAt    *time.Time `db:"last_reg_at"   json:"last_reg_at,omitempty"`
}

// DistrictStat holds registration counts per district.
type DistrictStat struct {
	District string `db:"district" json:"district"`
	Total    int    `db:"total"    json:"total"`
	Verified int    `db:"verified" json:"verified"`
}

// TimelinePoint represents registrations in a 5-minute bucket.
type TimelinePoint struct {
	Bucket   time.Time `db:"bucket"   json:"bucket"`
	Count    int       `db:"count"    json:"count"`
	Verified int       `db:"verified" json:"verified"`
}

// OrgStat holds registration counts per organization (top-N).
type OrgStat struct {
	Organization string `db:"organization" json:"organization"`
	Total        int    `db:"total"        json:"total"`
}

// EventReport bundles all analytics for one event.
type EventReport struct {
	Stats     ReportStats     `json:"stats"`
	Districts []DistrictStat  `json:"districts"`
	Timeline  []TimelinePoint `json:"timeline"`
	Orgs      []OrgStat       `json:"orgs"`
	Devices   []DeviceStat    `json:"devices"`
	OSes      []DeviceStat    `json:"oses"`
	Browsers  []DeviceStat    `json:"browsers"`
}

// ──────────────────────────────────────────────────────────────────────────────

// Event represents a reg_events row.
type Event struct {
	ID                    int        `db:"id"                      json:"id"`
	Title                 string     `db:"title"                   json:"title"`
	Description           string     `db:"description"             json:"description"`
	EventDate             string     `db:"event_date"              json:"event_date"`
	EventTime             string     `db:"event_time"              json:"event_time"`
	StartAt               *time.Time `db:"start_at"                json:"start_at,omitempty"`
	EndAt                 *time.Time `db:"end_at"                  json:"end_at,omitempty"`
	Venue                 string     `db:"venue"                   json:"venue"`
	Address               string     `db:"address"                 json:"address"`
	Capacity              int        `db:"capacity"                json:"capacity"` // 0 = unlimited
	CoverURL              string     `db:"cover_url"               json:"cover_url"`
	CabinetLink           string     `db:"cabinet_link"            json:"cabinet_link"`
	SpeakerLink           string     `db:"speaker_link"            json:"speaker_link"`
	ViewerLink            string     `db:"viewer_link"             json:"viewer_link"`
	IsActive              bool       `db:"is_active"               json:"is_active"`
	IsOnline              bool       `db:"is_online"               json:"is_online"`
	EventType             string     `db:"event_type"              json:"event_type"`              // online|offline|hybrid
	CheckInMode           string     `db:"check_in_mode"           json:"check_in_mode"`           // none|entry_only|entry_exit
	RegistrationOpensAt   *time.Time `db:"registration_opens_at"   json:"registration_opens_at,omitempty"`
	RegistrationClosesAt  *time.Time `db:"registration_closes_at"  json:"registration_closes_at,omitempty"`
	BadgeTemplateID       *int       `db:"badge_template_id"       json:"badge_template_id,omitempty"`
	MaxScansPerTicket     int        `db:"max_scans_per_ticket"    json:"max_scans_per_ticket"`
	CreatedAt             time.Time  `db:"created_at"              json:"created_at"`
	UpdatedAt             *time.Time `db:"updated_at"              json:"updated_at,omitempty"`
}

// EventWithStats extends Event with aggregated registration counts.
type EventWithStats struct {
	Event
	TotalRegs    int `db:"total_regs"    json:"total_regs"`
	VerifiedRegs int `db:"verified_regs" json:"verified_regs"`
}

// User represents a reg_users row.
type User struct {
	ID                int        `db:"id"                   json:"id"`
	Email             string     `db:"email"                json:"email"`
	LastName          string     `db:"last_name"            json:"last_name"`
	FirstName         string     `db:"first_name"           json:"first_name"`
	Patronymic        string     `db:"patronymic"           json:"patronymic"`
	Organization      string     `db:"organization"         json:"organization"`
	District          string     `db:"district"             json:"district"`
	IsUnionMember     bool       `db:"is_union_member"      json:"is_union_member"`
	UnionTicket       string     `db:"union_ticket"         json:"union_ticket"`
	ExtraInfo         string     `db:"extra_info"           json:"extra_info"`
	LastIP            string     `db:"last_ip"              json:"-"`
	GeoCountry        string     `db:"geo_country"          json:"geo_country"`
	GeoRegion         string     `db:"geo_region"           json:"geo_region"`
	GeoCity           string     `db:"geo_city"             json:"geo_city"`
	UserAgent         string     `db:"user_agent"           json:"-"`
	PasswordHash      string     `db:"password_hash"         json:"-"`
	PasswordUpdatedAt *time.Time `db:"password_updated_at"   json:"-"`
	CabinetFirstLoginAt *time.Time `db:"cabinet_first_login_at" json:"cabinet_first_login_at,omitempty"`
	CabinetLastLoginAt  *time.Time `db:"cabinet_last_login_at"  json:"cabinet_last_login_at,omitempty"`
	CreatedAt         time.Time  `db:"created_at"            json:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at"            json:"-"`
}

// Registration represents a reg_registrations row.
type Registration struct {
	ID               int        `db:"id"                json:"id"`
	EventID          int        `db:"event_id"          json:"event_id"`
	UserID           int        `db:"user_id"           json:"user_id"`
	OTPCode          string     `db:"otp_code"          json:"-"`
	OTPExpiresAt     time.Time  `db:"otp_expires_at"    json:"-"`
	OTPSentAt           *time.Time `db:"otp_sent_at"            json:"-"`
	OTPSendCount        int        `db:"otp_send_count"         json:"-"`
	OTPVerifiedAt       *time.Time `db:"otp_verified_at"        json:"otp_verified_at,omitempty"`
	WelcomeEmailSentAt  *time.Time `db:"welcome_email_sent_at"  json:"welcome_email_sent_at,omitempty"`
	Status              string     `db:"status"                 json:"status"`
	IPAddress        string     `db:"ip_address"        json:"-"`
	GeoCountry       string     `db:"geo_country"       json:"geo_country"`
	GeoRegion        string     `db:"geo_region"        json:"geo_region"`
	GeoCity          string     `db:"geo_city"          json:"geo_city"`
	ISPName          string     `db:"isp_name"          json:"-"`
	ISPASN           string     `db:"isp_asn"           json:"-"`
	DeviceType       string     `db:"device_type"       json:"device_type"`
	OSName           string     `db:"os_name"           json:"os_name"`
	BrowserName      string     `db:"browser_name"      json:"browser_name"`
	UserAgent        string     `db:"user_agent"        json:"-"`
	ParticipantToken *string    `db:"participant_token"  json:"participant_token,omitempty"`
	ParticipantRole  string     `db:"participant_role"   json:"participant_role"`
	CheckedInAt      *time.Time `db:"checked_in_at"      json:"checked_in_at,omitempty"`
	CreatedAt        time.Time  `db:"created_at"         json:"created_at"`
	UpdatedAt        *time.Time `db:"updated_at"         json:"updated_at,omitempty"`
}

// OnlineSession represents one continuous participation window in an online event.
type OnlineSession struct {
	ID             int64      `db:"id"              json:"id"`
	EventID        int        `db:"event_id"        json:"event_id"`
	UserID         int        `db:"user_id"         json:"user_id"`
	RegistrationID *int       `db:"registration_id" json:"registration_id,omitempty"`
	SessionUUID    string     `db:"session_uuid"    json:"session_uuid"`
	StartedAt      time.Time  `db:"started_at"      json:"started_at"`
	LastPingAt     time.Time  `db:"last_ping_at"    json:"last_ping_at"`
	EndedAt        *time.Time `db:"ended_at"        json:"ended_at,omitempty"`
	DurationSeconds *int      `db:"duration_seconds" json:"duration_seconds,omitempty"`
	EndReason      *string    `db:"end_reason"      json:"end_reason,omitempty"`
}

// OnlineSessionSummary maps to the online_session_summary view.
type OnlineSessionSummary struct {
	EventID        int        `db:"event_id"        json:"event_id"`
	UserID         int        `db:"user_id"         json:"user_id"`
	RegistrationID *int       `db:"registration_id" json:"registration_id,omitempty"`
	SessionCount   int        `db:"session_count"   json:"session_count"`
	FirstJoinAt    *time.Time `db:"first_join_at"   json:"first_join_at,omitempty"`
	LastSeenAt     *time.Time `db:"last_seen_at"    json:"last_seen_at,omitempty"`
	TotalSeconds   int        `db:"total_seconds"   json:"total_seconds"`
}

// OnlineParticipantDetail enriches OnlineSessionSummary with user identity fields.
// Returned by the admin attendance page for the per-participant breakdown table.
type OnlineParticipantDetail struct {
	UserID         int        `db:"user_id"         json:"user_id"`
	Email          string     `db:"email"           json:"email"`
	LastName       string     `db:"last_name"       json:"last_name"`
	FirstName      string     `db:"first_name"      json:"first_name"`
	RegistrationID *int       `db:"registration_id" json:"registration_id,omitempty"`
	SessionCount   int        `db:"session_count"   json:"session_count"`
	FirstJoinAt    *time.Time `db:"first_join_at"   json:"first_join_at,omitempty"`
	LastSeenAt     *time.Time `db:"last_seen_at"    json:"last_seen_at,omitempty"`
	TotalSeconds   int        `db:"total_seconds"   json:"total_seconds"`
	IsActive       bool       `db:"is_active"       json:"is_active"`
}

// OnlineStatsResponse is returned by the admin stats endpoint.
type OnlineStatsResponse struct {
	EventID       int                       `json:"event_id"`
	ActiveNow     int                       `json:"active_now"`
	Registrations []OnlineSessionSummary    `json:"registrations"`
	Participants  []OnlineParticipantDetail `json:"participants"`
}

// SessionConnectRequest is the optional request body for stream/connect.
// Fields are informational — the server derives identity from the JWT.
type SessionConnectRequest struct {
	// Reserved for future client metadata (e.g. player version).
}

// SessionPingRequest carries the session identifier.
type SessionPingRequest struct {
	SessionUUID string `json:"session_uuid" binding:"required"`
}

// SessionDisconnectRequest carries the session identifier.
type SessionDisconnectRequest struct {
	SessionUUID string `json:"session_uuid" binding:"required"`
}

// ResolveEventLink returns the canonical event link for a participant role.
// This is the single source of truth for role→link resolution in Go code.
// The same logic is mirrored in the SQL CASE expression inside ListByUserVerified
// for bulk queries — keep the two in sync.
func ResolveEventLink(role, speakerLink, viewerLink, cabinetLink string) string {
	if SpeakerRoles[role] && speakerLink != "" {
		return speakerLink
	}
	if viewerLink != "" {
		return viewerLink
	}
	return cabinetLink
}

// TicketInfo bundles everything needed to render a participant ticket.
type TicketInfo struct {
	Registration Registration
	Event        Event
	User         User
	TicketURL    string // full URL encoded in the QR code
	EventLink    string // role-resolved stream/event link
}

// CheckInRequest is the admin body for scanning / manually entering a token.
type CheckInRequest struct {
	Token string `json:"token" binding:"required"`
}

// CheckInResult is returned after a successful or duplicate check-in.
type CheckInResult struct {
	Status       string       `json:"status"` // "ok" | "already"
	CheckedInAt  time.Time    `json:"checked_in_at"`
	Registration Registration `json:"registration"`
	User         User         `json:"user"`
	Event        Event        `json:"event"`
}

// RegistrationRow is used for admin list/export queries joining all tables.
type RegistrationRow struct {
	RegID          int        `db:"reg_id"          json:"reg_id"`
	RegDatetime    time.Time  `db:"reg_datetime"    json:"reg_datetime"`
	EventID        int        `db:"event_id"        json:"event_id"`
	EventTitle     string     `db:"event_title"     json:"event_title"`
	LastName       string     `db:"last_name"       json:"last_name"`
	FirstName      string     `db:"first_name"      json:"first_name"`
	Patronymic     string     `db:"patronymic"      json:"patronymic"`
	Organization   string     `db:"organization"    json:"organization"`
	District       string     `db:"district"        json:"district"`
	Email          string     `db:"email"           json:"email"`
	IsUnionMember  bool       `db:"is_union_member" json:"is_union_member"`
	UnionTicket    string     `db:"union_ticket"    json:"union_ticket"`
	ExtraInfo      string     `db:"extra_info"      json:"extra_info"`
	IPAddress      string     `db:"ip_address"      json:"ip_address"`
	GeoCountry     string     `db:"geo_country"     json:"geo_country"`
	GeoRegion      string     `db:"geo_region"      json:"geo_region"`
	GeoCity        string     `db:"geo_city"        json:"geo_city"`
	ISPName        string     `db:"isp_name"        json:"isp_name"`
	ISPASN         string     `db:"isp_asn"         json:"isp_asn"`
	DeviceType     string     `db:"device_type"     json:"device_type"`
	OSName         string     `db:"os_name"         json:"os_name"`
	BrowserName    string     `db:"browser_name"    json:"browser_name"`
	Status         string     `db:"status"           json:"status"`
	StatusExtended string     `db:"status_extended"  json:"status_extended"`
	ScanCount      int        `db:"scan_count"       json:"scan_count"`
	OTPSentAt          *time.Time `db:"otp_sent_at"           json:"otp_sent_at,omitempty"`
	OTPSendCount       int        `db:"otp_send_count"        json:"otp_send_count"`
	OTPVerifiedAt      *time.Time `db:"otp_verified_at"       json:"otp_verified_at,omitempty"`
	WelcomeEmailSentAt *time.Time `db:"welcome_email_sent_at" json:"welcome_email_sent_at,omitempty"`
	CheckedInAt        *time.Time `db:"checked_in_at"         json:"checked_in_at,omitempty"`
	FirstEntryAt   *time.Time `db:"first_entry_at"   json:"first_entry_at,omitempty"`
	LastExitAt     *time.Time `db:"last_exit_at"     json:"last_exit_at,omitempty"`
}

// Log represents a reg_logs row.
type Log struct {
	ID        int       `db:"id"         json:"id"`
	EventType string    `db:"event_type" json:"event_type"`
	UserEmail string    `db:"user_email" json:"user_email"`
	IPAddress string    `db:"ip_address" json:"ip_address"`
	Message   string    `db:"message"    json:"message"`
	UserAgent string    `db:"user_agent" json:"user_agent"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Stats aggregates dashboard counters.
type Stats struct {
	TotalUsers    int `db:"total_users"    json:"total_users"`
	VerifiedRegs  int `db:"verified_regs"  json:"verified_regs"`
	PendingRegs   int `db:"pending_regs"   json:"pending_regs"`
	ActiveEvents  int `db:"active_events"  json:"active_events"`
	CheckedIn     int `db:"checked_in"     json:"checked_in"`
}

// Geo holds geographic information resolved from an IP address.
type Geo struct {
	Country string
	Region  string
	City    string
	ISPName string
	ISPASN  string
}

// DeviceInfo holds parsed user-agent breakdown.
type DeviceInfo struct {
	DeviceType  string // mobile | tablet | desktop | bot | unknown
	OSName      string // Windows | macOS | Linux | Android | iOS | …
	BrowserName string // Chrome | Firefox | Safari | Edge | Opera | YandexBrowser | …
}

// TrackingEvent represents a reg_tracking row.
type TrackingEvent struct {
	ID             int        `db:"id"              json:"id"`
	EventID        int        `db:"event_id"        json:"event_id"`
	UserID         int        `db:"user_id"         json:"user_id"`
	RegistrationID *int       `db:"registration_id" json:"registration_id,omitempty"`
	Action         string     `db:"action"          json:"action"` // visit | stream_connect | stream_disconnect | stream_error
	IPAddress      string     `db:"ip_address"      json:"ip_address"`
	GeoCountry     string     `db:"geo_country"     json:"geo_country"`
	GeoRegion      string     `db:"geo_region"      json:"geo_region"`
	GeoCity        string     `db:"geo_city"        json:"geo_city"`
	ISPName        string     `db:"isp_name"        json:"isp_name"`
	ISPASN         string     `db:"isp_asn"         json:"isp_asn"`
	DeviceType     string     `db:"device_type"     json:"device_type"`
	OSName         string     `db:"os_name"         json:"os_name"`
	BrowserName    string     `db:"browser_name"    json:"browser_name"`
	UserAgent      string     `db:"user_agent"      json:"user_agent"`
	OccurredAt     time.Time  `db:"occurred_at"     json:"occurred_at"`
}

// EventField is a custom registration field defined by admin per event.
type EventField struct {
	ID              int        `db:"id"               json:"id"`
	EventID         int        `db:"event_id"         json:"event_id"`
	Label           string     `db:"label"            json:"label"`
	FieldType       string     `db:"field_type"       json:"field_type"` // text|textarea|select|multiselect|checkbox|radio|phone|number|date|hidden|masked|file
	Options         []string   `db:"options"          json:"options,omitempty"` // for select/radio/checkbox
	Placeholder     string     `db:"placeholder"      json:"placeholder"`
	HelperText      string     `db:"helper_text"      json:"helper_text"`
	ValidationRegex string     `db:"validation_regex" json:"validation_regex,omitempty"`
	IsRequired      bool       `db:"is_required"      json:"is_required"`
	InBadge         bool       `db:"in_badge"         json:"in_badge"`
	InReport        bool       `db:"in_report"        json:"in_report"`
	InExport        bool       `db:"in_export"        json:"in_export"`
	ListID          *int       `db:"list_id"          json:"list_id,omitempty"` // FK to ref_lists
	MinValue        *float64   `db:"min_value"        json:"min_value,omitempty"`
	MaxValue        *float64   `db:"max_value"        json:"max_value,omitempty"`
	MaxLength       *int       `db:"max_length"       json:"max_length,omitempty"`
	SortOrder       int        `db:"sort_order"       json:"sort_order"`
	CreatedAt       time.Time  `db:"created_at"       json:"created_at"`
}

// FieldAnswer is a registrant's answer to one custom field.
type FieldAnswer struct {
	ID             int    `db:"id"              json:"id"`
	RegistrationID int    `db:"registration_id" json:"registration_id"`
	FieldID        int    `db:"field_id"        json:"field_id"`
	Value          string `db:"value"           json:"value"`
}

// DeviceStat holds registration counts per device/OS/browser.
type DeviceStat struct {
	Name  string `db:"name"  json:"name"`
	Total int    `db:"total" json:"total"`
}

// BadgeData contains everything needed to render one participant badge.
type BadgeData struct {
	RegistrationID   int        `db:"registration_id"   json:"registration_id"`
	ParticipantToken string     `db:"participant_token" json:"participant_token"`
	LastName         string     `db:"last_name"         json:"last_name"`
	FirstName        string     `db:"first_name"        json:"first_name"`
	Patronymic       string     `db:"patronymic"        json:"patronymic"`
	Organization     string     `db:"organization"      json:"organization"`
	District         string     `db:"district"          json:"district"`
	IsUnionMember    bool       `db:"is_union_member"   json:"is_union_member"`
	Email            string     `db:"email"             json:"email"`
	CheckedInAt      *time.Time `db:"checked_in_at"     json:"checked_in_at,omitempty"`
}

// MultiEventStats holds aggregate stats across multiple events.
type MultiEventStats struct {
	EventID      int    `db:"event_id"      json:"event_id"`
	EventTitle   string `db:"event_title"   json:"event_title"`
	EventDate    string `db:"event_date"    json:"event_date"`
	IsOnline     bool   `db:"is_online"     json:"is_online"`
	TotalRegs    int    `db:"total_regs"    json:"total_regs"`
	VerifiedRegs int    `db:"verified_regs" json:"verified_regs"`
	UnionMembers int    `db:"union_members" json:"union_members"`
	CheckedIn    int    `db:"checked_in"    json:"checked_in"`
}

// --- Request / Response DTOs ---

type AnswerInput struct {
	FieldID int    `json:"field_id" binding:"required,min=1"`
	Value   string `json:"value"`
}

type RegisterRequest struct {
	Email         string        `json:"email"          binding:"required,email"`
	EventID       int           `json:"event_id"       binding:"required,min=1"`
	FirstName     string        `json:"first_name"     binding:"required"`
	LastName      string        `json:"last_name"      binding:"required"`
	Patronymic    string        `json:"patronymic"`
	Organization  string        `json:"organization"   binding:"required"`
	District      string        `json:"district"       binding:"required"`
	IsUnionMember bool          `json:"is_union_member"`
	UnionTicket   string        `json:"union_ticket"`
	ExtraInfo     string        `json:"extra_info"`
	Answers       []AnswerInput `json:"answers"`
	// ConsentGiven must be true; required by 152-ФЗ ст.9, GDPR Art.7, CCPA §1798.135.
	ConsentGiven  bool          `json:"consent_given"  binding:"required"`
}

type TrackActionRequest struct {
	Action string `json:"action" binding:"required"` // visit | stream_connect | stream_disconnect | stream_error
}

type CreateFieldRequest struct {
	Label           string   `json:"label"            binding:"required"`
	FieldType       string   `json:"field_type"       binding:"required,oneof=text textarea select multiselect checkbox radio phone number date hidden masked file"`
	Options         []string `json:"options"`
	Placeholder     string   `json:"placeholder"`
	HelperText      string   `json:"helper_text"`
	ValidationRegex string   `json:"validation_regex"`
	IsRequired      bool     `json:"is_required"`
	InBadge         bool     `json:"in_badge"`
	InReport        bool     `json:"in_report"`
	InExport        bool     `json:"in_export"`
	ListID          *int     `json:"list_id"`
	MinValue        *float64 `json:"min_value"`
	MaxValue        *float64 `json:"max_value"`
	MaxLength       *int     `json:"max_length"`
	SortOrder       int      `json:"sort_order"`
}

type VerifyOTPRequest struct {
	Email   string `json:"email"    binding:"required,email"`
	EventID int    `json:"event_id" binding:"required,min=1"`
	OTP     string `json:"otp"      binding:"required,len=6"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type AdminLoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AdminVerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp"   binding:"required,len=6"`
}

type CreateEventRequest struct {
	Title                string  `json:"title"                   binding:"required"`
	Description          string  `json:"description"`
	EventDate            string  `json:"event_date"              binding:"required"`
	EventTime            string  `json:"event_time"              binding:"required"`
	StartAt              *string `json:"start_at"`               // RFC3339 or null
	EndAt                *string `json:"end_at"`                 // RFC3339 or null
	Venue                string  `json:"venue"`
	Address              string  `json:"address"`
	Capacity             int     `json:"capacity"`               // 0 = unlimited
	CoverURL             string  `json:"cover_url"`
	CabinetLink          string  `json:"cabinet_link"`
	IsActive             bool    `json:"is_active"`
	IsOnline             bool    `json:"is_online"`
	EventType            string  `json:"event_type"`             // online|offline|hybrid
	CheckInMode          string  `json:"check_in_mode"`          // none|entry_only|entry_exit
	RegistrationOpensAt  *string `json:"registration_opens_at"`  // RFC3339 or null
	RegistrationClosesAt *string `json:"registration_closes_at"` // RFC3339 or null
	BadgeTemplateID      *int    `json:"badge_template_id"`
	MaxScansPerTicket    int     `json:"max_scans_per_ticket"`
}

// UpdateProfileRequest is used by PUT /api/cabinet/profile.
type UpdateProfileRequest struct {
	FirstName     string `json:"first_name"     binding:"required"`
	LastName      string `json:"last_name"      binding:"required"`
	Patronymic    string `json:"patronymic"`
	Organization  string `json:"organization"   binding:"required"`
	District      string `json:"district"       binding:"required"`
	IsUnionMember bool   `json:"is_union_member"`
	UnionTicket   string `json:"union_ticket"`
	ExtraInfo     string `json:"extra_info"`
}

// ChangePasswordRequest is used by PUT /api/cabinet/password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password"     binding:"required,min=6"`
}

// ConsentRecord is returned in data-export responses.
type ConsentRecord struct {
	EventID     int        `db:"event_id"    json:"event_id"`
	EventTitle  string     `db:"event_title" json:"event_title"`
	Version     string     `db:"version"     json:"version"`
	ConsentedAt time.Time  `db:"consented_at" json:"consented_at"`
	WithdrawnAt *time.Time `db:"withdrawn_at" json:"withdrawn_at,omitempty"`
}

// DataExport bundles all personal data for a user (GDPR Art.15 / 152-ФЗ ст.14).
type DataExport struct {
	User          User                 `json:"user"`
	Registrations []RegistrationExport `json:"registrations"`
	Consents      []ConsentRecord      `json:"consents"`
	ExportedAt    time.Time            `json:"exported_at"`
}

// RegistrationExport is a simplified registration row for data export.
type RegistrationExport struct {
	EventID   int       `db:"event_id"    json:"event_id"`
	EventTitle string   `db:"event_title" json:"event_title"`
	Status    string    `db:"status"      json:"status"`
	CreatedAt time.Time `db:"created_at"  json:"created_at"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ── Scan & Attendance models ──────────────────────────────────────────────────

// ScanLog represents one QR scan attempt (immutable audit).
type ScanLog struct {
	ID             int64      `db:"id"              json:"id"`
	EventID        int        `db:"event_id"        json:"event_id"`
	RegistrationID *int       `db:"registration_id" json:"registration_id,omitempty"`
	ScannedToken   string     `db:"scanned_token"   json:"scanned_token"`
	ScanMode       string     `db:"scan_mode"       json:"scan_mode"`   // entry|exit|verify
	ScanResult     string     `db:"scan_result"     json:"scan_result"` // ok|duplicate|not_found|wrong_event|cancelled|error
	OperatorID     *int       `db:"operator_id"     json:"operator_id,omitempty"`
	DeviceInfo     string     `db:"device_info"     json:"device_info"`
	IPAddress      string     `db:"ip_address"      json:"ip_address"`
	Note           string     `db:"note"            json:"note"`
	ScannedAt      time.Time  `db:"scanned_at"      json:"scanned_at"`
}

// AttendanceEvent records a single entry or exit action.
type AttendanceEvent struct {
	ID             int64     `db:"id"              json:"id"`
	EventID        int       `db:"event_id"        json:"event_id"`
	RegistrationID int       `db:"registration_id" json:"registration_id"`
	Action         string    `db:"action"          json:"action"` // entry|exit
	ScanLogID      *int64    `db:"scan_log_id"     json:"scan_log_id,omitempty"`
	OccurredAt     time.Time `db:"occurred_at"     json:"occurred_at"`
}

// RegistrationStatusLog records every status FSM transition.
type RegistrationStatusLog struct {
	ID             int64     `db:"id"              json:"id"`
	RegistrationID int       `db:"registration_id" json:"registration_id"`
	EventID        int       `db:"event_id"        json:"event_id"`
	PrevStatus     string    `db:"prev_status"     json:"prev_status"`
	NewStatus      string    `db:"new_status"      json:"new_status"`
	ChangedBy      *int      `db:"changed_by"      json:"changed_by,omitempty"`
	ChangeSource   string    `db:"change_source"   json:"change_source"` // system|admin|scanner|api|user
	Note           string    `db:"note"            json:"note"`
	ChangedAt      time.Time `db:"changed_at"      json:"changed_at"`
}

// ScanRequest is the body for POST /admin/events/:id/scan.
type ScanRequest struct {
	Token    string `json:"token"     binding:"required"`
	Mode     string `json:"mode"      binding:"required,oneof=entry exit verify"`
	DeviceInfo string `json:"device_info"`
}

// ScanResponse is returned after a scan attempt.
type ScanResponse struct {
	Result         string       `json:"result"` // ok|duplicate|not_found|wrong_event|cancelled|error
	Message        string       `json:"message"`
	Registration   *Registration `json:"registration,omitempty"`
	User           *User         `json:"user,omitempty"`
	StatusExtended string        `json:"status_extended,omitempty"`
	ScanCount      int           `json:"scan_count,omitempty"`
}

// ── Reference list models ─────────────────────────────────────────────────────

// RefList represents a managed dropdown list.
type RefList struct {
	ID          int        `db:"id"          json:"id"`
	Slug        string     `db:"slug"        json:"slug"`
	Title       string     `db:"title"       json:"title"`
	Description string     `db:"description" json:"description"`
	IsActive    bool       `db:"is_active"   json:"is_active"`
	CreatedAt   time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"  json:"updated_at,omitempty"`
}

// RefListItem is a single entry in a reference list.
type RefListItem struct {
	ID        int    `db:"id"         json:"id"`
	ListID    int    `db:"list_id"    json:"list_id"`
	Value     string `db:"value"      json:"value"`
	Label     string `db:"label"      json:"label"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
	IsActive  bool   `db:"is_active"  json:"is_active"`
}

// ── Badge template models ─────────────────────────────────────────────────────

// BadgeTemplate stores a per-event badge design.
type BadgeTemplate struct {
	ID            int        `db:"id"             json:"id"`
	EventID       *int       `db:"event_id"       json:"event_id,omitempty"`
	Name          string     `db:"name"           json:"name"`
	IsDefault     bool       `db:"is_default"     json:"is_default"`
	AccentColor   string     `db:"accent_color"   json:"accent_color"`
	LogoURL       string     `db:"logo_url"       json:"logo_url"`
	BackgroundURL string     `db:"background_url" json:"background_url"`
	PaperSize     string     `db:"paper_size"     json:"paper_size"`
	Orientation   string     `db:"orientation"    json:"orientation"`
	HTMLTemplate  string     `db:"html_template"  json:"html_template"`
	FieldsConfig  []byte     `db:"fields_config"  json:"fields_config"` // JSONB
	CreatedAt     time.Time  `db:"created_at"     json:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"     json:"updated_at,omitempty"`
}

// CreateRefListRequest is the body for POST /admin/reflists.
type CreateRefListRequest struct {
	Slug        string `json:"slug"        binding:"required"`
	Title       string `json:"title"       binding:"required"`
	Description string `json:"description"`
}

// CreateRefListItemRequest is the body for POST /admin/reflists/:id/items.
type CreateRefListItemRequest struct {
	Value     string `json:"value"      binding:"required"`
	Label     string `json:"label"      binding:"required"`
	SortOrder int    `json:"sort_order"`
}

// ── Admin / RBAC models ───────────────────────────────────────────────────────

// AdminUser represents an admin_users row.
type AdminUser struct {
	ID           int        `db:"id"            json:"id"`
	Email        string     `db:"email"         json:"email"`
	Name         string     `db:"name"          json:"name"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Role         string     `db:"role"          json:"role"`
	IsActive     bool       `db:"is_active"     json:"is_active"`
	CreatedBy    *int       `db:"created_by"    json:"created_by,omitempty"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"    json:"updated_at,omitempty"`
}

// Permission represents a granular permission key.
type Permission struct {
	ID          int    `db:"id"          json:"id"`
	Key         string `db:"key"         json:"key"`
	Description string `db:"description" json:"description"`
	Category    string `db:"category"    json:"category"`
}

// AdminActionLog represents a row in admin_action_logs.
type AdminActionLog struct {
	ID         int64      `db:"id"          json:"id"`
	AdminID    *int       `db:"admin_id"    json:"admin_id,omitempty"`
	AdminEmail string     `db:"admin_email" json:"admin_email"`
	Action     string     `db:"action"      json:"action"`
	TargetType string     `db:"target_type" json:"target_type"`
	TargetID   *int       `db:"target_id"   json:"target_id,omitempty"`
	OldValue   []byte     `db:"old_value"   json:"old_value,omitempty"`
	NewValue   []byte     `db:"new_value"   json:"new_value,omitempty"`
	IP         string     `db:"ip"          json:"ip"`
	CreatedAt  time.Time  `db:"created_at"  json:"created_at"`
}

// UserProfileChangeLog is a single field-level audit entry.
type UserProfileChangeLog struct {
	ID        int64      `db:"id"         json:"id"`
	UserID    int        `db:"user_id"    json:"user_id"`
	ChangedBy *int       `db:"changed_by" json:"changed_by,omitempty"`
	FieldName string     `db:"field_name" json:"field_name"`
	OldValue  string     `db:"old_value"  json:"old_value"`
	NewValue  string     `db:"new_value"  json:"new_value"`
	ChangedAt time.Time  `db:"changed_at" json:"changed_at"`
}

// ── Q&A models ────────────────────────────────────────────────────────────────

// UserQuestion represents a user_questions row.
type UserQuestion struct {
	ID           int        `db:"id"             json:"id"`
	UserID       int        `db:"user_id"        json:"user_id"`
	EventID      int        `db:"event_id"       json:"event_id"`
	Subject      string     `db:"subject"        json:"subject"`
	Status       string     `db:"status"         json:"status"`
	Priority     int        `db:"priority"       json:"priority"`
	AssignedTo   *int       `db:"assigned_to"    json:"assigned_to,omitempty"`
	FirstReplyAt *time.Time `db:"first_reply_at" json:"first_reply_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at"     json:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"     json:"updated_at,omitempty"`
	ClosedAt     *time.Time `db:"closed_at"      json:"closed_at,omitempty"`
}

// UserQuestionWithMeta extends UserQuestion with joined display fields.
type UserQuestionWithMeta struct {
	UserQuestion
	EventTitle  string `db:"event_title"  json:"event_title"`
	UserEmail   string `db:"user_email"   json:"user_email"`
	UserName    string `db:"user_name"    json:"user_name"`
	UnreadCount int    `db:"unread_count" json:"unread_count"`
}

// QuestionMessage represents a single message in a question thread.
type QuestionMessage struct {
	ID         int       `db:"id"          json:"id"`
	QuestionID int       `db:"question_id" json:"question_id"`
	SenderID   int       `db:"sender_id"   json:"sender_id"`
	SenderRole string    `db:"sender_role" json:"sender_role"`
	Body       string    `db:"body"        json:"body"`
	IsRead     bool      `db:"is_read"     json:"is_read"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}

// QuestionStats holds analytics for the Q&A dashboard.
type QuestionStats struct {
	Total          int     `db:"total"           json:"total"`
	New            int     `db:"new_count"       json:"new"`
	InProgress     int     `db:"in_progress"     json:"in_progress"`
	Answered       int     `db:"answered"        json:"answered"`
	Closed         int     `db:"closed"          json:"closed"`
	AvgReplyMinutes float64 `db:"avg_reply_min"  json:"avg_reply_minutes"`
}

// ── Request DTOs ──────────────────────────────────────────────────────────────

// CreateQuestionRequest is the body for POST /api/cabinet/questions.
type CreateQuestionRequest struct {
	EventID int    `json:"event_id" binding:"required,min=1"`
	Subject string `json:"subject"  binding:"required,min=5,max=200"`
	Body    string `json:"body"     binding:"required,min=10"`
}

// SendMessageRequest is the body for POST /api/cabinet/questions/:id/messages.
type SendMessageRequest struct {
	Body string `json:"body" binding:"required,min=1,max=5000"`
}

// CreateAdminRequest is used by super_admin to create a new admin user.
type CreateAdminRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Name     string `json:"name"     binding:"required"`
	Role     string `json:"role"     binding:"required,oneof=admin operator support viewer"`
	Password string `json:"password" binding:"required,min=8"`
}

// UpdateAdminUserRequest allows editing an admin's name/role/active state.
type UpdateAdminUserRequest struct {
	Name     string `json:"name"`
	Role     string `json:"role"      binding:"omitempty,oneof=admin operator support viewer"`
	IsActive *bool  `json:"is_active"`
}

// AssignQuestionRequest assigns a question to an operator.
type AssignQuestionRequest struct {
	AdminID int `json:"admin_id" binding:"required,min=1"`
}

// UpdateQuestionStatusRequest changes question status.
type UpdateQuestionStatusRequest struct {
	Status  string `json:"status"  binding:"required,oneof=in_progress answered closed"`
	Comment string `json:"comment"`
}

