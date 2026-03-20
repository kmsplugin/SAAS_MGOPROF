package model

import "time"

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
	ID          int        `db:"id"           json:"id"`
	Title       string     `db:"title"        json:"title"`
	Description string     `db:"description"  json:"description"`
	EventDate   string     `db:"event_date"   json:"event_date"`
	EventTime   string     `db:"event_time"   json:"event_time"`
	CabinetLink string     `db:"cabinet_link" json:"cabinet_link"`
	IsActive    bool       `db:"is_active"    json:"is_active"`
	CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"   json:"updated_at,omitempty"`
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
	PasswordHash      string     `db:"password_hash"        json:"-"`
	PasswordUpdatedAt *time.Time `db:"password_updated_at"  json:"-"`
	CreatedAt         time.Time  `db:"created_at"           json:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at"           json:"-"`
}

// Registration represents a reg_registrations row.
type Registration struct {
	ID            int        `db:"id"              json:"id"`
	EventID       int        `db:"event_id"        json:"event_id"`
	UserID        int        `db:"user_id"         json:"user_id"`
	OTPCode       string     `db:"otp_code"        json:"-"`
	OTPExpiresAt  time.Time  `db:"otp_expires_at"  json:"-"`
	OTPVerifiedAt *time.Time `db:"otp_verified_at" json:"otp_verified_at,omitempty"`
	Status        string     `db:"status"          json:"status"`
	IPAddress     string     `db:"ip_address"      json:"-"`
	GeoCountry    string     `db:"geo_country"     json:"geo_country"`
	GeoRegion     string     `db:"geo_region"      json:"geo_region"`
	GeoCity       string     `db:"geo_city"        json:"geo_city"`
	ISPName       string     `db:"isp_name"        json:"-"`
	ISPASN        string     `db:"isp_asn"         json:"-"`
	DeviceType    string     `db:"device_type"     json:"device_type"`
	OSName        string     `db:"os_name"         json:"os_name"`
	BrowserName   string     `db:"browser_name"    json:"browser_name"`
	UserAgent     string     `db:"user_agent"      json:"-"`
	CreatedAt     time.Time  `db:"created_at"      json:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"      json:"updated_at,omitempty"`
}

// RegistrationRow is used for admin list/export queries joining all tables.
type RegistrationRow struct {
	RegDatetime   time.Time `db:"reg_datetime"   json:"reg_datetime"`
	EventTitle    string    `db:"event_title"    json:"event_title"`
	LastName      string    `db:"last_name"      json:"last_name"`
	FirstName     string    `db:"first_name"     json:"first_name"`
	Patronymic    string    `db:"patronymic"     json:"patronymic"`
	Organization  string    `db:"organization"   json:"organization"`
	District      string    `db:"district"       json:"district"`
	Email         string    `db:"email"          json:"email"`
	IsUnionMember bool      `db:"is_union_member" json:"is_union_member"`
	UnionTicket   string    `db:"union_ticket"   json:"union_ticket"`
	ExtraInfo     string    `db:"extra_info"     json:"extra_info"`
	IPAddress     string    `db:"ip_address"     json:"ip_address"`
	GeoCountry    string    `db:"geo_country"    json:"geo_country"`
	GeoRegion     string    `db:"geo_region"     json:"geo_region"`
	GeoCity       string    `db:"geo_city"       json:"geo_city"`
	ISPName       string    `db:"isp_name"       json:"isp_name"`
	ISPASN        string    `db:"isp_asn"        json:"isp_asn"`
	DeviceType    string    `db:"device_type"    json:"device_type"`
	OSName        string    `db:"os_name"        json:"os_name"`
	BrowserName   string    `db:"browser_name"   json:"browser_name"`
	Status        string    `db:"status"         json:"status"`
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
	ID         int        `db:"id"          json:"id"`
	EventID    int        `db:"event_id"    json:"event_id"`
	Label      string     `db:"label"       json:"label"`
	FieldType  string     `db:"field_type"  json:"field_type"` // text|textarea|select|checkbox|radio
	Options    []string   `db:"options"     json:"options,omitempty"` // for select/radio/checkbox
	Placeholder string    `db:"placeholder" json:"placeholder"`
	IsRequired  bool      `db:"is_required" json:"is_required"`
	SortOrder   int       `db:"sort_order"  json:"sort_order"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
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
}

type TrackActionRequest struct {
	Action string `json:"action" binding:"required"` // visit | stream_connect | stream_disconnect | stream_error
}

type CreateFieldRequest struct {
	Label       string   `json:"label"       binding:"required"`
	FieldType   string   `json:"field_type"  binding:"required,oneof=text textarea select checkbox radio"`
	Options     []string `json:"options"`
	Placeholder string   `json:"placeholder"`
	IsRequired  bool     `json:"is_required"`
	SortOrder   int      `json:"sort_order"`
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
	Title       string `json:"title"        binding:"required"`
	Description string `json:"description"`
	EventDate   string `json:"event_date"   binding:"required"`
	EventTime   string `json:"event_time"   binding:"required"`
	CabinetLink string `json:"cabinet_link"`
	IsActive    bool   `json:"is_active"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
