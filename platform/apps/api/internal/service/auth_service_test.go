package service

import (
	"testing"

	"go.uber.org/zap"
)

func makeTestPlatformAuthService(secret string) *AuthService {
	return &AuthService{
		jwtSecret: []byte(secret),
		logger:    zap.NewNop(),
	}
}

func TestPlatformAuthService_IssueAndParseToken(t *testing.T) {
	svc := makeTestPlatformAuthService("platform_test_secret_min_32_chars_ok")

	token, err := svc.issueToken("user-uuid-123", "tenant-uuid-456", "user@example.com", "participant")
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}
	if token == "" {
		t.Fatal("issueToken вернул пустой токен")
	}

	userID, tenantID, role, email, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if userID != "user-uuid-123" {
		t.Errorf("userID: ожидали 'user-uuid-123', получили %q", userID)
	}
	if tenantID != "tenant-uuid-456" {
		t.Errorf("tenantID: ожидали 'tenant-uuid-456', получили %q", tenantID)
	}
	if role != "participant" {
		t.Errorf("role: ожидали 'participant', получили %q", role)
	}
	if email != "user@example.com" {
		t.Errorf("email: ожидали 'user@example.com', получили %q", email)
	}
}

func TestPlatformAuthService_ParseToken_WrongSecret(t *testing.T) {
	svc1 := makeTestPlatformAuthService("secret_one_for_platform_minimum_32_chars")
	svc2 := makeTestPlatformAuthService("secret_two_for_platform_minimum_32_chars")

	token, err := svc1.issueToken("uid", "tid", "a@b.com", "admin")
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}

	_, _, _, _, err = svc2.ParseToken(token)
	if err == nil {
		t.Error("ParseToken должен вернуть ошибку для токена с чужим секретом")
	}
}

func TestPlatformAuthService_ParseToken_Garbage(t *testing.T) {
	svc := makeTestPlatformAuthService("platform_test_secret_min_32_chars_ok")
	_, _, _, _, err := svc.ParseToken("not.a.valid.jwt.token")
	if err == nil {
		t.Error("ParseToken должен вернуть ошибку для невалидного токена")
	}
}

func TestPlatformAuthService_ParseTokenFunc(t *testing.T) {
	svc := makeTestPlatformAuthService("platform_test_secret_min_32_chars_ok")
	fn := svc.ParseTokenFunc()

	token, _ := svc.issueToken("u1", "t1", "test@test.com", "organizer")
	userID, tenantID, role, email, err := fn(token)
	if err != nil {
		t.Fatalf("ParseTokenFunc: %v", err)
	}
	if userID != "u1" || tenantID != "t1" || role != "organizer" || email != "test@test.com" {
		t.Errorf("ParseTokenFunc вернул неверные данные: %s %s %s %s", userID, tenantID, role, email)
	}
}

func TestHashPassword_RoundTrip(t *testing.T) {
	pwd := "SecurePassword!42"
	hash, err := HashPassword(pwd)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword вернул пустой хеш")
	}
	if hash == pwd {
		t.Error("хеш не должен совпадать с паролем")
	}
}
