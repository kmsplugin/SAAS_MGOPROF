package integration

// Scenario 6: Welcome mail success/failure with controllable FakeMailer stub.
//   - When mailer succeeds: welcome_email_sent_at is set in DB; registration is verified.
//   - When mailer fails on welcome: registration is still verified (fail-open model);
//     welcome_email_sent_at remains NULL.
//   - When mailer fails on registration OTP: register returns an error.

import (
	"net/http"
	"testing"

	"mgoprof-saas/internal/model"
)

func TestMailer_WelcomeEmailSentAt_OnSuccess(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "mailer_success@example.com"

	// Mailer is in success mode (default).
	registerResp := post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})
	mustStatus(t, registerResp, http.StatusOK)

	otp := env.FM.LastOTPFor(email)
	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})
	mustStatus(t, verifyResp, http.StatusOK)

	// welcome_email_sent_at must be set.
	var sentAt *string
	err := env.DB.QueryRowContext(t.Context(),
		`SELECT to_char(r.welcome_email_sent_at, 'YYYY-MM-DD')
		 FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&sentAt)
	if err != nil {
		t.Fatalf("query welcome_email_sent_at: %v", err)
	}
	if sentAt == nil || *sentAt == "" {
		t.Error("welcome_email_sent_at must be set after successful welcome email")
	}
}

func TestMailer_WelcomeEmailSentAt_NullOnMailFailure(t *testing.T) {
	// When SendWelcome fails, the registration flow must still succeed (fail-open),
	// but welcome_email_sent_at must remain NULL.
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "mailer_fail@example.com"

	registerResp := post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})
	mustStatus(t, registerResp, http.StatusOK)

	otp := env.FM.LastOTPFor(email)

	// Force welcome-mail failure before calling verify-otp.
	env.FM.SetFailMode(true)

	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})

	// Restore so future tests are not affected.
	env.FM.SetFailMode(false)

	// Even with mailer failure, verify-otp must return 200 (fail-open model:
	// the registration is persisted and the token is issued).
	mustStatus(t, verifyResp, http.StatusOK)

	var result map[string]interface{}
	decodeJSON(t, verifyResp, &result)
	if result["status"] != "success" {
		t.Fatalf("verify-otp must succeed even if mailer fails, got %v", result["status"])
	}
	if _, ok := result["token"].(string); !ok {
		t.Error("verify-otp must still return a JWT token even if mailer fails")
	}

	// Registration must be verified in DB.
	var status string
	env.DB.QueryRowContext(t.Context(),
		`SELECT r.status FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&status) //nolint:errcheck
	if status != "verified" {
		t.Errorf("registration must be verified even if mailer fails, got %q", status)
	}

	// welcome_email_sent_at must be NULL because the mail was not sent.
	var sentAt *string
	env.DB.QueryRowContext(t.Context(),
		`SELECT to_char(r.welcome_email_sent_at, 'YYYY-MM-DD')
		 FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&sentAt) //nolint:errcheck
	if sentAt != nil && *sentAt != "" {
		t.Errorf("welcome_email_sent_at must be NULL when mailer fails, got %v", *sentAt)
	}
}

func TestMailer_RegistrationOTPFail_RegisterReturnsError(t *testing.T) {
	// When SendRegistration fails, POST /register must return an error.
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "mailer_otp_fail@example.com"

	env.FM.SetFailMode(true)
	registerResp := post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})
	env.FM.SetFailMode(false)

	// Service must propagate the mailer error.
	if registerResp.StatusCode == http.StatusOK {
		var r map[string]interface{}
		decodeJSON(t, registerResp, &r)
		if r["status"] != "error" {
			t.Fatalf("register must return status=error when OTP mail fails, got %v", r["status"])
		}
	} else {
		registerResp.Body.Close()
	}
}

func TestMailer_OTPEmailContainsSixDigitCode(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "otp_digits@example.com"

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
	if len(otp) != 6 {
		t.Errorf("OTP in email must be exactly 6 characters, got %d: %q", len(otp), otp)
	}
	for _, ch := range otp {
		if ch < '0' || ch > '9' {
			t.Errorf("OTP must contain only digits, got %q", otp)
			break
		}
	}
}
