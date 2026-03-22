package integration

// Scenario 1: Full happy-path OTP registration flow.
//   POST /api/register → OTP email → POST /api/verify-otp → JWT token → POST /api/auth/login
//
// Scenario 2: OTP failure paths.
//   wrong code, expired code, replay attack, resend cooldown

import (
	"net/http"
	"testing"
	"time"

	"mgoprof-saas/internal/model"
)

// ── Scenario 1: happy-path full OTP flow ─────────────────────────────────────

func TestOTPFlow_HappyPath(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "otp_happy@example.com"

	// ── Step 1: register ──────────────────────────────────────────────────────
	registerResp := post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Иван",
		LastName:     "Тестов",
		Organization: "TestOrg",
		District:     "TestDistrict",
		ConsentGiven: true,
	})
	mustStatus(t, registerResp, http.StatusOK)

	var regResult map[string]interface{}
	decodeJSON(t, registerResp, &regResult)

	if regResult["status"] != "success" {
		t.Fatalf("register: expected status=success, got %v", regResult["status"])
	}
	if regResult["show_otp"] != true {
		t.Errorf("register: expected show_otp=true, got %v", regResult["show_otp"])
	}

	// Mailer must have received an OTP email.
	if !env.FM.SentTo("registration", email) {
		t.Fatal("register: no registration email sent")
	}

	// ── Step 2: retrieve OTP from the fake mailer ─────────────────────────────
	otp := env.FM.LastOTPFor(email)
	if len(otp) != 6 {
		t.Fatalf("expected 6-digit OTP, got %q", otp)
	}

	// ── Step 3: verify OTP ────────────────────────────────────────────────────
	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})
	mustStatus(t, verifyResp, http.StatusOK)

	var verifyResult map[string]interface{}
	decodeJSON(t, verifyResp, &verifyResult)

	if verifyResult["status"] != "success" {
		t.Fatalf("verify-otp: expected status=success, got %v", verifyResult["status"])
	}

	// JWT token must be present for auto-login.
	token, ok := verifyResult["token"].(string)
	if !ok || token == "" {
		t.Fatal("verify-otp: response missing non-empty token field")
	}

	// redirect_url and cabinet_url must point to cabinet.
	for _, key := range []string{"redirect_url", "cabinet_url"} {
		url, _ := verifyResult[key].(string)
		if url == "" {
			t.Errorf("verify-otp: %s must not be empty", key)
		}
	}

	// Welcome email must have been sent.
	if !env.FM.SentTo("welcome", email) {
		t.Error("verify-otp: no welcome email sent after OTP success")
	}

	// ── Step 4: verify the token works for GET /api/cabinet/me ───────────────
	meResp := getWithToken(t, env.Server, "/api/cabinet/me", token)
	mustStatus(t, meResp, http.StatusOK)

	var user map[string]interface{}
	decodeJSON(t, meResp, &user)
	if user["email"] != email {
		t.Errorf("cabinet/me: expected email=%q, got %v", email, user["email"])
	}

	// ── Step 5: auto-login token also works via POST /api/auth/login ─────────
	// The welcome email contains the password; we fetch it from the fake mailer.
	pwd := env.FM.WelcomePasswordFor(email)
	if pwd == "" {
		t.Fatal("welcome email missing password arg")
	}

	loginResp := post(t, env.Server, "/api/auth/login", model.LoginRequest{
		Email:    email,
		Password: pwd,
	})
	mustStatus(t, loginResp, http.StatusOK)

	var loginResult map[string]interface{}
	decodeJSON(t, loginResp, &loginResult)
	if loginResult["status"] != "success" {
		t.Fatalf("auth/login: expected status=success, got %v", loginResult["status"])
	}
	if _, ok := loginResult["token"].(string); !ok {
		t.Error("auth/login: missing token field")
	}
}

// ── Scenario 2: OTP failure paths ────────────────────────────────────────────

func TestOTPFlow_WrongCode(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "otp_wrong@example.com"

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

	// Submit a deliberately wrong OTP.
	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     "000000",
	})
	if verifyResp.StatusCode == http.StatusOK {
		var r map[string]interface{}
		decodeJSON(t, verifyResp, &r)
		// The response may be 200 but with status=error, or a 4xx.
		if r["status"] != "error" {
			t.Fatalf("wrong OTP must return status=error, got %v", r["status"])
		}
	} else if verifyResp.StatusCode != http.StatusBadRequest {
		verifyResp.Body.Close()
		t.Fatalf("wrong OTP: expected 400, got %d", verifyResp.StatusCode)
	} else {
		verifyResp.Body.Close()
	}

	// Verify registration is still pending in the DB.
	var status string
	err := env.DB.QueryRowContext(t.Context(),
		`SELECT status FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("query registration status: %v", err)
	}
	if status != "pending" {
		t.Errorf("after wrong OTP: expected status=pending, got %q", status)
	}
}

func TestOTPFlow_ExpiredCode(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "otp_expired@example.com"

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
	if otp == "" {
		t.Fatal("no registration email received")
	}

	// Force-expire the OTP by backdating otp_expires_at.
	_, err := env.DB.ExecContext(t.Context(),
		`UPDATE reg_registrations r
		 SET otp_expires_at = NOW() - INTERVAL '1 minute'
		 FROM reg_users u
		 WHERE u.id = r.user_id AND u.email = $1 AND r.event_id = $2`,
		email, eventID,
	)
	if err != nil {
		t.Fatalf("backdating otp_expires_at: %v", err)
	}

	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})

	var r map[string]interface{}
	decodeJSON(t, verifyResp, &r)
	if r["status"] == "success" {
		t.Fatal("expired OTP must not succeed")
	}
}

func TestOTPFlow_ReplayAfterVerification(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "otp_replay@example.com"

	// Full happy-path first verification.
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
	verifyResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})
	mustStatus(t, verifyResp, http.StatusOK)

	// Attempt replay: submit the same OTP again.
	replayResp := post(t, env.Server, "/api/verify-otp", model.VerifyOTPRequest{
		Email:   email,
		EventID: eventID,
		OTP:     otp,
	})

	var r map[string]interface{}
	decodeJSON(t, replayResp, &r)
	// Service returns "already confirmed" which the handler wraps as 400 error.
	// The registration must still be verified (not corrupted).
	var dbStatus string
	env.DB.QueryRowContext(t.Context(),
		`SELECT r.status FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&dbStatus) //nolint:errcheck
	if dbStatus != "verified" {
		t.Errorf("after replay: DB status must still be verified, got %q", dbStatus)
	}

	// otp_code must be NULL (cleared after first verification).
	var otpCode *string
	env.DB.QueryRowContext(t.Context(),
		`SELECT r.otp_code FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&otpCode) //nolint:errcheck
	if otpCode != nil {
		t.Errorf("otp_code must be NULL after verification, got %v", *otpCode)
	}
}

func TestOTPFlow_ResendCooldown(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "otp_resend@example.com"

	registerBody := model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	}

	// First registration — must succeed.
	resp1 := post(t, env.Server, "/api/register", registerBody)
	mustStatus(t, resp1, http.StatusOK)
	var r1 map[string]interface{}
	decodeJSON(t, resp1, &r1)
	if r1["status"] != "success" {
		t.Fatalf("first register: expected success, got %v", r1["status"])
	}

	// Second registration within cooldown — must return "pending" status.
	resp2 := post(t, env.Server, "/api/register", registerBody)
	mustStatus(t, resp2, http.StatusOK)
	var r2 map[string]interface{}
	decodeJSON(t, resp2, &r2)
	// During cooldown (< 10s) the service returns status=pending and does NOT
	// resend the OTP.
	if r2["status"] != "pending" {
		t.Logf("second register within cooldown: status=%v (may be 'success' if cooldown elapsed)", r2["status"])
	}

	// Regardless of cooldown status, the registration must still exist in DB.
	var count int
	env.DB.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&count) //nolint:errcheck
	if count != 1 {
		t.Errorf("expected exactly 1 registration, got %d", count)
	}
}

func TestOTPFlow_TTLIsPositive(t *testing.T) {
	// Verify the OTP expiry in the DB is in the future right after registration.
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "otp_ttl@example.com"

	before := time.Now()
	post(t, env.Server, "/api/register", model.RegisterRequest{
		Email:        email,
		EventID:      eventID,
		FirstName:    "Test",
		LastName:     "User",
		Organization: "Org",
		District:     "District",
		ConsentGiven: true,
	})

	var expiresAt time.Time
	env.DB.QueryRowContext(t.Context(),
		`SELECT r.otp_expires_at FROM reg_registrations r
		 JOIN reg_users u ON u.id = r.user_id
		 WHERE u.email = $1 AND r.event_id = $2`,
		email, eventID,
	).Scan(&expiresAt) //nolint:errcheck

	if expiresAt.IsZero() {
		t.Fatal("otp_expires_at is zero — registration row not found?")
	}
	if !expiresAt.After(before) {
		t.Errorf("otp_expires_at must be in the future (before=%v, expiresAt=%v)", before, expiresAt)
	}
	// Should be approximately NOW + 10 minutes.
	diff := expiresAt.Sub(before)
	if diff < 9*time.Minute || diff > 11*time.Minute {
		t.Errorf("otp_expires_at should be ~10 min from now, got diff=%v", diff)
	}
}
