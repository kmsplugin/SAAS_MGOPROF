package model

import "time"

// ── Tenant ────────────────────────────────────────────────────────────────────

type Tenant struct {
	ID             string     `db:"id"                   json:"id"`
	Slug           string     `db:"slug"                 json:"slug"`
	Name           string     `db:"name"                 json:"name"`
	Plan           string     `db:"plan"                 json:"plan"`
	LogoURL        string     `db:"logo_url"             json:"logo_url"`
	PrimaryColor   string     `db:"primary_color"        json:"primary_color"`
	CustomDomain   string     `db:"custom_domain"        json:"custom_domain,omitempty"`
	IsActive       bool       `db:"is_active"            json:"is_active"`
	AIEnabled      bool       `db:"ai_enabled"           json:"ai_enabled"`
	RecordingEnabled bool     `db:"recording_enabled"    json:"recording_enabled"`
	MaxParticipants int       `db:"max_participants"     json:"max_participants"`
	MaxViewers      int       `db:"max_viewers"          json:"max_viewers"`
	CreatedAt      time.Time  `db:"created_at"           json:"created_at"`
}

// ── User ──────────────────────────────────────────────────────────────────────

type User struct {
	ID            string     `db:"id"             json:"id"`
	TenantID      string     `db:"tenant_id"      json:"tenant_id"`
	Email         string     `db:"email"          json:"email"`
	EmailVerified bool       `db:"email_verified" json:"email_verified"`
	FirstName     string     `db:"first_name"     json:"first_name"`
	LastName      string     `db:"last_name"      json:"last_name"`
	AvatarURL     string     `db:"avatar_url"     json:"avatar_url,omitempty"`
	PasswordHash  string     `db:"password_hash"  json:"-"`
	Role          string     `db:"role"           json:"role"`
	IsActive      bool       `db:"is_active"      json:"is_active"`
	LastLoginAt   *time.Time `db:"last_login_at"  json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `db:"created_at"     json:"created_at"`
}

// ── Event ─────────────────────────────────────────────────────────────────────

type Event struct {
	ID                   string     `db:"id"                    json:"id"`
	TenantID             string     `db:"tenant_id"             json:"tenant_id"`
	Title                string     `db:"title"                 json:"title"`
	Description          string     `db:"description"           json:"description"`
	Type                 string     `db:"type"                  json:"type"`
	Status               string     `db:"status"                json:"status"`
	StartAt              *time.Time `db:"start_at"              json:"start_at,omitempty"`
	EndAt                *time.Time `db:"end_at"                json:"end_at,omitempty"`
	Timezone             string     `db:"timezone"              json:"timezone"`
	CoverURL             string     `db:"cover_url"             json:"cover_url,omitempty"`
	Capacity             int        `db:"capacity"              json:"capacity"`
	ViewerCapacity       int        `db:"viewer_capacity"       json:"viewer_capacity"`
	IsPublic             bool       `db:"is_public"             json:"is_public"`
	RegistrationRequired bool       `db:"registration_required" json:"registration_required"`
	CreatedBy            string     `db:"created_by"            json:"created_by"`
	CreatedAt            time.Time  `db:"created_at"            json:"created_at"`
	UpdatedAt            *time.Time `db:"updated_at"            json:"updated_at,omitempty"`
}

// ── Room ──────────────────────────────────────────────────────────────────────

type Room struct {
	ID              string     `db:"id"               json:"id"`
	EventID         string     `db:"event_id"         json:"event_id"`
	TenantID        string     `db:"tenant_id"        json:"tenant_id"`
	LiveKitRoomName string     `db:"livekit_room_name" json:"livekit_room_name"`
	Mode            string     `db:"mode"             json:"mode"`
	Status          string     `db:"status"           json:"status"`
	MaxParticipants int        `db:"max_participants"  json:"max_participants"`
	IsRecording     bool       `db:"is_recording"     json:"is_recording"`
	HLSURL          string     `db:"hls_url"          json:"hls_url,omitempty"`
	StartedAt       *time.Time `db:"started_at"       json:"started_at,omitempty"`
	EndedAt         *time.Time `db:"ended_at"         json:"ended_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at"       json:"created_at"`
}

// ── Registration ──────────────────────────────────────────────────────────────

type Registration struct {
	ID             string     `db:"id"              json:"id"`
	EventID        string     `db:"event_id"        json:"event_id"`
	UserID         string     `db:"user_id"         json:"user_id"`
	TenantID       string     `db:"tenant_id"       json:"tenant_id"`
	Status         string     `db:"status"          json:"status"`
	TicketCode     string     `db:"ticket_code"     json:"ticket_code"`
	CheckInAt      *time.Time `db:"check_in_at"     json:"check_in_at,omitempty"`
	ConsentGiven   bool       `db:"consent_given"   json:"consent_given"`
	ConsentVersion string     `db:"consent_version" json:"consent_version"`
	CreatedAt      time.Time  `db:"created_at"      json:"created_at"`
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type RegisterRequest struct {
	Email         string `json:"email"          binding:"required,email"`
	Password      string `json:"password"       binding:"required,min=8"`
	FirstName     string `json:"first_name"     binding:"required"`
	LastName      string `json:"last_name"      binding:"required"`
	TenantSlug    string `json:"tenant_slug"    binding:"required"`
	ConsentGiven  bool   `json:"consent_given"  binding:"required"`
}

type LoginRequest struct {
	Email      string `json:"email"       binding:"required,email"`
	Password   string `json:"password"    binding:"required"`
	TenantSlug string `json:"tenant_slug" binding:"required"`
}

type CreateEventRequest struct {
	Title                string  `json:"title"                 binding:"required"`
	Description          string  `json:"description"`
	Type                 string  `json:"type"                  binding:"required,oneof=webinar conference broadcast hybrid meeting"`
	StartAt              *string `json:"start_at"`
	EndAt                *string `json:"end_at"`
	Timezone             string  `json:"timezone"`
	CoverURL             string  `json:"cover_url"`
	Capacity             int     `json:"capacity"`
	ViewerCapacity       int     `json:"viewer_capacity"`
	IsPublic             bool    `json:"is_public"`
	RegistrationRequired bool    `json:"registration_required"`
}

type CreateRoomRequest struct {
	EventID         string `json:"event_id"         binding:"required"`
	Mode            string `json:"mode"             binding:"required,oneof=meeting webinar broadcast stage"`
	MaxParticipants int    `json:"max_participants"`
}

type JoinRoomRequest struct {
	Role        string `json:"role"         binding:"required,oneof=host speaker moderator viewer"`
	DisplayName string `json:"display_name" binding:"required"`
}

type RoomTokenResponse struct {
	Token      string `json:"token"`
	RoomName   string `json:"room_name"`
	LiveKitURL string `json:"livekit_url"`
}

type RegisterEventRequest struct {
	EventID      string `json:"event_id"      binding:"required"`
	ConsentGiven bool   `json:"consent_given" binding:"required"`
}
