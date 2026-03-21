package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateOTP_Length(t *testing.T) {
	for i := 0; i < 20; i++ {
		otp := generateOTP()
		if len(otp) != 6 {
			t.Errorf("ожидали длину OTP 6, получили %d (otp=%q)", len(otp), otp)
		}
	}
}

func TestGenerateOTP_OnlyDigits(t *testing.T) {
	for i := 0; i < 50; i++ {
		otp := generateOTP()
		for _, ch := range otp {
			if ch < '0' || ch > '9' {
				t.Errorf("OTP содержит не-цифровой символ %q в %q", ch, otp)
			}
		}
	}
}

func TestGeneratePassword_Length(t *testing.T) {
	pwd := generatePassword(passwordLength)
	if len(pwd) != passwordLength {
		t.Errorf("ожидали длину %d, получили %d", passwordLength, len(pwd))
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		p := generatePassword(passwordLength)
		if seen[p] {
			t.Errorf("generatePassword вернул дубль: %q", p)
		}
		seen[p] = true
	}
}

func TestHashPassword_ValidBcrypt(t *testing.T) {
	pwd := "TestPassword123"
	hash, err := hashPassword(pwd)
	if err != nil {
		t.Fatalf("hashPassword вернул ошибку: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd)); err != nil {
		t.Errorf("хеш не совпадает с паролем: %v", err)
	}
}

func TestHashPassword_Cost(t *testing.T) {
	hash, err := hashPassword("anypassword")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost: %v", err)
	}
	if cost < 12 {
		t.Errorf("bcrypt cost должен быть >= 12, получили %d", cost)
	}
}

func TestGenerateParticipantToken_Length(t *testing.T) {
	token := generateParticipantToken()
	if len(token) != 12 {
		t.Errorf("ожидали длину токена 12, получили %d", len(token))
	}
}

func TestGenerateParticipantToken_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		tok := generateParticipantToken()
		if seen[tok] {
			t.Errorf("generateParticipantToken вернул дубль: %q", tok)
		}
		seen[tok] = true
	}
}
