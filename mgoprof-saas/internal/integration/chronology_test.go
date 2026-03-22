package integration

// Scenario 7: Admin chronology consistency.
//   Verify that after the full registration flow the participant timeline is
//   coherent: reg_datetime < otp_sent_at ≤ otp_verified_at ≤ welcome_email_sent_at.
//   Also verifies otp_send_count increments correctly on resend.

import (
	"testing"
	"time"

	"mgoprof-saas/internal/model"
)

func TestChronology_TimelineIsOrdered(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "chronology@example.com"

	t0 := time.Now()

	// Register (creates reg, sends OTP).
	post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})

	otp := env.FM.LastOTPFor(email)

	// Verify OTP (verifies reg, sends welcome email).
	post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})

	t1 := time.Now()

	// Read timestamps from DB.
	type timeline struct {
		CreatedAt          time.Time  `db:"created_at"`
		OTPSentAt          *time.Time `db:"otp_sent_at"`
		OTPVerifiedAt      *time.Time `db:"otp_verified_at"`
		WelcomeEmailSentAt *time.Time `db:"welcome_email_sent_at"`
		OTPSendCount       int        `db:"otp_send_count"`
	}
	var tl timeline
	err := env.DB.QueryRowContext(t.Context(), `
		SELECT r.created_at,
		       r.otp_sent_at,
		       r.otp_verified_at,
		       r.welcome_email_sent_at,
		       COALESCE(r.otp_send_count, 0) AS otp_send_count
		FROM reg_registrations r
		JOIN reg_users u ON u.id = r.user_id
		WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&tl.CreatedAt, &tl.OTPSentAt, &tl.OTPVerifiedAt, &tl.WelcomeEmailSentAt, &tl.OTPSendCount)
	if err != nil {
		t.Fatalf("query timeline: %v", err)
	}

	// All timestamps must fall within the [t0, t1] window.
	if tl.CreatedAt.Before(t0) || tl.CreatedAt.After(t1) {
		t.Errorf("created_at=%v is outside [%v, %v]", tl.CreatedAt, t0, t1)
	}

	if tl.OTPSentAt == nil {
		t.Error("otp_sent_at must be set")
	} else if tl.OTPSentAt.Before(tl.CreatedAt) {
		t.Errorf("otp_sent_at=%v must not be before created_at=%v", tl.OTPSentAt, tl.CreatedAt)
	}

	if tl.OTPVerifiedAt == nil {
		t.Error("otp_verified_at must be set after successful OTP verification")
	} else {
		if tl.OTPSentAt != nil && tl.OTPVerifiedAt.Before(*tl.OTPSentAt) {
			t.Errorf("otp_verified_at=%v must not be before otp_sent_at=%v",
				tl.OTPVerifiedAt, tl.OTPSentAt)
		}
	}

	if tl.WelcomeEmailSentAt == nil {
		t.Error("welcome_email_sent_at must be set after successful email send")
	} else if tl.OTPVerifiedAt != nil && tl.WelcomeEmailSentAt.Before(*tl.OTPVerifiedAt) {
		t.Errorf("welcome_email_sent_at=%v must not be before otp_verified_at=%v",
			tl.WelcomeEmailSentAt, tl.OTPVerifiedAt)
	}

	// First registration: otp_send_count must be 1.
	if tl.OTPSendCount != 1 {
		t.Errorf("otp_send_count must be 1 after first registration, got %d", tl.OTPSendCount)
	}
}

func TestChronology_OTPSendCountIncrementsOnResend(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "chronology_resend@example.com"

	reqBody := model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	}

	// First registration: otp_send_count = 1.
	post(t, env.Server, "/api/register", reqBody)

	// Force cooldown to expire by backdating updated_at.
	env.DB.ExecContext(t.Context(), `
		UPDATE reg_registrations r
		SET updated_at = NOW() - INTERVAL '1 minute'
		FROM reg_users u
		WHERE u.id = r.user_id AND u.email = $1 AND r.event_id = $2`,
		email, eventID,
	) //nolint:errcheck

	// Second registration after cooldown expiry: otp_send_count = 2.
	post(t, env.Server, "/api/register", reqBody)

	var count int
	env.DB.QueryRowContext(t.Context(),
		`SELECT COALESCE(r.otp_send_count, 0)
		 FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&count) //nolint:errcheck

	if count < 2 {
		t.Errorf("otp_send_count must be ≥ 2 after resend, got %d", count)
	}
}

func TestChronology_OTPVerifiedAt_NullBeforeVerification(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "chronology_unverified@example.com"

	post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})

	var verifiedAt *time.Time
	env.DB.QueryRowContext(t.Context(),
		`SELECT r.otp_verified_at FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&verifiedAt) //nolint:errcheck

	if verifiedAt != nil {
		t.Errorf("otp_verified_at must be NULL before OTP is submitted, got %v", verifiedAt)
	}
}

func TestChronology_StatusTransition_PendingToVerified(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "chronology_status@example.com"

	post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})

	var status string
	env.DB.QueryRowContext(t.Context(),
		`SELECT r.status FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&status) //nolint:errcheck
	if status != "pending" {
		t.Errorf("status must be pending right after register, got %q", status)
	}

	otp := env.FM.LastOTPFor(email)
	post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})

	env.DB.QueryRowContext(t.Context(),
		`SELECT r.status FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&status) //nolint:errcheck
	if status != "verified" {
		t.Errorf("status must be verified after OTP confirmation, got %q", status)
	}
}
