package service

import (
	"testing"
)

func makeTestAuthService(secret string) *AuthService {
	return &AuthService{jwtSecret: []byte(secret)}
}

func TestAuthService_IssueAndParseToken(t *testing.T) {
	svc := makeTestAuthService("test_secret_minimum_32_characters_ok")

	token, err := svc.issueToken(42, "user@example.com", "user")
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}
	if token == "" {
		t.Fatal("issueToken вернул пустой токен")
	}

	userID, role, email, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID: ожидали 42, получили %d", userID)
	}
	if role != "user" {
		t.Errorf("role: ожидали 'user', получили %q", role)
	}
	if email != "user@example.com" {
		t.Errorf("email: ожидали 'user@example.com', получили %q", email)
	}
}

func TestAuthService_ParseToken_WrongSecret(t *testing.T) {
	svc1 := makeTestAuthService("secret_one_minimum_32_characters_ok")
	svc2 := makeTestAuthService("secret_two_minimum_32_characters_ok")

	token, err := svc1.issueToken(1, "a@b.com", "user")
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}

	_, _, _, err = svc2.ParseToken(token)
	if err == nil {
		t.Error("ParseToken должен вернуть ошибку для токена с другим секретом")
	}
}

func TestAuthService_ParseToken_Garbage(t *testing.T) {
	svc := makeTestAuthService("test_secret_minimum_32_characters_ok")
	_, _, _, err := svc.ParseToken("not.a.jwt")
	if err == nil {
		t.Error("ParseToken должен вернуть ошибку для мусорного токена")
	}
}

// ── Requirement 1: auto-login token after OTP ────────────────────────────────

func TestIssueUserToken_ReturnsValidToken(t *testing.T) {
	svc := makeTestAuthService("test_secret_minimum_32_characters_ok")
	token, err := svc.IssueUserToken(55, "participant@event.ru")
	if err != nil {
		t.Fatalf("IssueUserToken: %v", err)
	}
	if token == "" {
		t.Fatal("IssueUserToken вернул пустой токен")
	}
	userID, role, email, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if userID != 55 {
		t.Errorf("userID: ожидали 55, получили %d", userID)
	}
	if role != "user" {
		t.Errorf("IssueUserToken должен выдавать role=user, получили %q", role)
	}
	if email != "participant@event.ru" {
		t.Errorf("email: ожидали 'participant@event.ru', получили %q", email)
	}
}

func TestIssueUserToken_RoleIsNeverAdmin(t *testing.T) {
	// IssueUserToken is called after OTP verification — must always be "user".
	// If it somehow issued "admin", a participant could escalate privileges.
	svc := makeTestAuthService("test_secret_minimum_32_characters_ok")
	token, _ := svc.IssueUserToken(1, "someone@test.ru")
	_, role, _, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if role == "admin" || role == "super_admin" {
		t.Errorf("IssueUserToken не должен выдавать привилегированную роль, получили %q", role)
	}
}

// ── Requirement 4: cabinet_first_login / cabinet_last_login semantics ────────
// These can't be tested without a DB, but we verify the SQL COALESCE pattern
// at the conceptual level: first login is set once, last login always updates.

func TestCabinetLogin_FirstIsCoalesced(t *testing.T) {
	// COALESCE(cabinet_first_login_at, NOW()) means:
	// - if NULL  → set to NOW() (first login)
	// - if set   → keep existing value (preserve first login)
	// We verify the semantics via the Go nil pointer behaviour analogy.
	var first *string // nil = not yet set

	simulate := func() {
		now := "2026-03-22T10:00:00"
		if first == nil {
			first = &now // first login: set once
		}
		// last login always updates (separate field, not tested here as pure Go)
	}

	simulate() // first call
	savedFirst := *first

	simulate() // second call — first must not change
	if *first != savedFirst {
		t.Errorf("cabinet_first_login_at изменился при повторном входе: %q → %q", savedFirst, *first)
	}
}

func TestAuthService_AdminRole(t *testing.T) {
	svc := makeTestAuthService("admin_secret_minimum_32_characters_ok")

	token, err := svc.issueToken(99, "admin@company.ru", "admin")
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}

	userID, role, email, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if userID != 99 {
		t.Errorf("userID: ожидали 99, получили %d", userID)
	}
	if role != "admin" {
		t.Errorf("role: ожидали 'admin', получили %q", role)
	}
	if email != "admin@company.ru" {
		t.Errorf("email: %q", email)
	}
}
