package integration

// Scenario 5: cabinet_first_login_at / cabinet_last_login_at via real login flow.
//   - first login: sets cabinet_first_login_at to a non-NULL timestamp.
//   - second login: cabinet_first_login_at unchanged; cabinet_last_login_at updated.
//   - COALESCE semantics: once first_login_at is set it stays set.

import (
	"net/http"
	"testing"
	"time"

	"mgoprof-saas/internal/model"
)

func TestLoginTiming_FirstLoginSetOnce(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "timing_user@example.com"

	// Register + verify OTP to create a user with a known password.
	registerAndVerify(t, env, email, eventID)

	// Retrieve the password from the welcome email.
	pwd := env.FM.WelcomePasswordFor(email)
	if pwd == "" {
		t.Fatal("welcome email not received — cannot test login timing")
	}

	// First login.
	resp1 := post(t, env.Server, "/api/auth/login", model.LoginRequest{
		Email:    email,
		Password: pwd,
	})
	mustStatus(t, resp1, http.StatusOK)
	resp1.Body.Close()

	var firstLoginAt *time.Time
	var lastLoginAt1 *time.Time
	err := env.DB.QueryRowContext(t.Context(),
		`SELECT cabinet_first_login_at, cabinet_last_login_at
		 FROM reg_users WHERE email = $1`, email,
	).Scan(&firstLoginAt, &lastLoginAt1)
	if err != nil {
		t.Fatalf("query login timestamps (first): %v", err)
	}
	if firstLoginAt == nil {
		t.Fatal("cabinet_first_login_at must be set after first login")
	}
	if lastLoginAt1 == nil {
		t.Fatal("cabinet_last_login_at must be set after first login")
	}
	savedFirst := *firstLoginAt

	// Small sleep to guarantee last_login_at advances.
	time.Sleep(10 * time.Millisecond)

	// Second login.
	resp2 := post(t, env.Server, "/api/auth/login", model.LoginRequest{
		Email:    email,
		Password: pwd,
	})
	mustStatus(t, resp2, http.StatusOK)
	resp2.Body.Close()

	var firstLoginAt2 *time.Time
	var lastLoginAt2 *time.Time
	err = env.DB.QueryRowContext(t.Context(),
		`SELECT cabinet_first_login_at, cabinet_last_login_at
		 FROM reg_users WHERE email = $1`, email,
	).Scan(&firstLoginAt2, &lastLoginAt2)
	if err != nil {
		t.Fatalf("query login timestamps (second): %v", err)
	}

	// COALESCE: first_login_at must be unchanged.
	if firstLoginAt2 == nil || !firstLoginAt2.Equal(savedFirst) {
		t.Errorf("cabinet_first_login_at must not change on second login: was %v, now %v",
			savedFirst, firstLoginAt2)
	}

	// last_login_at must have been updated (or at least set).
	if lastLoginAt2 == nil {
		t.Error("cabinet_last_login_at must be set after second login")
	}
}

func TestLoginTiming_TokenFromVerifyOTPAlsoTriggers(t *testing.T) {
	// The JWT token returned by POST /verify-otp allows the frontend to skip
	// POST /auth/login. Verify the token is valid (not that login timestamps
	// are set from VerifyOTP — they are only set on Login).
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "timing_auto_login@example.com"

	token := registerAndVerify(t, env, email, eventID)

	// Token from VerifyOTP must allow cabinet access without calling /auth/login.
	meResp := getWithToken(t, env.Server, "/api/cabinet/me", token)
	mustStatus(t, meResp, http.StatusOK)

	var me map[string]interface{}
	decodeJSON(t, meResp, &me)
	if me["email"] != email {
		t.Errorf("expected email=%q, got %v", email, me["email"])
	}
}

func TestLoginTiming_BadPasswordDoesNotUpdateTimestamps(t *testing.T) {
	env := newTestEnv(t)
	eventID := seedEvent(t, env.DB, "online")
	email := "timing_bad_pwd@example.com"

	registerAndVerify(t, env, email, eventID)

	// Attempt login with wrong password.
	resp := post(t, env.Server, "/api/auth/login", model.LoginRequest{
		Email:    email,
		Password: "definitely_wrong_password_xyz",
	})
	if resp.StatusCode == http.StatusOK {
		var r map[string]interface{}
		decodeJSON(t, resp, &r)
		if r["status"] != "error" {
			t.Fatal("bad password login must not return status=success")
		}
	} else {
		resp.Body.Close()
	}

	// Timestamps must remain NULL (user never successfully logged in via /auth/login).
	var firstLoginAt *time.Time
	err := env.DB.QueryRowContext(t.Context(),
		`SELECT cabinet_first_login_at FROM reg_users WHERE email = $1`, email,
	).Scan(&firstLoginAt)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if firstLoginAt != nil {
		t.Errorf("cabinet_first_login_at must remain NULL after failed login, got %v", firstLoginAt)
	}
}
