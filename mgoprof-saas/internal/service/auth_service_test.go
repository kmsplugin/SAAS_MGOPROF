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
