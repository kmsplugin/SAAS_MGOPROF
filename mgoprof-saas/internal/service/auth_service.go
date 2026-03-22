package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"mgoprof-saas/internal/mailer"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

const jwtTTL = 72 * time.Hour

// AuthService handles user cabinet authentication and self-service.
type AuthService struct {
	userRepo    *repository.UserRepository
	regRepo     *repository.RegistrationRepository
	consentRepo *repository.ConsentRepository
	logRepo     *repository.LogRepository
	mailer      *mailer.Mailer
	geo         *GeoResolver
	logger      *zap.Logger
	jwtSecret   []byte
	siteURL     string
}

func NewAuthService(
	userRepo *repository.UserRepository,
	regRepo *repository.RegistrationRepository,
	consentRepo *repository.ConsentRepository,
	logRepo *repository.LogRepository,
	m *mailer.Mailer,
	geo *GeoResolver,
	logger *zap.Logger,
	jwtSecret, siteURL string,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		regRepo:     regRepo,
		consentRepo: consentRepo,
		logRepo:     logRepo,
		mailer:      m,
		geo:         geo,
		logger:      logger,
		jwtSecret:   []byte(jwtSecret),
		siteURL:     siteURL,
	}
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, req model.LoginRequest, ip, userAgent string) (string, *model.User, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, fmt.Errorf("поиск пользователя: %w", err)
	}
	if user == nil || user.PasswordHash == "" {
		_ = s.logRepo.Write(ctx, "cabinet_login_failed", email, ip, "user not found or no password", userAgent)
		return "", nil, fmt.Errorf("неверный email или пароль")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		_ = s.logRepo.Write(ctx, "cabinet_login_failed", email, ip, "wrong password", userAgent)
		return "", nil, fmt.Errorf("неверный email или пароль")
	}

	token, err := s.issueToken(user.ID, email, "user")
	if err != nil {
		return "", nil, fmt.Errorf("выпуск токена: %w", err)
	}

	// Record first/last login timestamps for the admin participant timeline.
	if err := s.userRepo.RecordLogin(ctx, user.ID); err != nil {
		s.logger.Warn("record login failed", zap.String("email", email), zap.Error(err))
	}

	_ = s.logRepo.Write(ctx, "cabinet_login_success", email, ip, "logged in", userAgent)
	return token, user, nil
}

// ForgotPassword generates and mails a new password.
func (s *AuthService) ForgotPassword(ctx context.Context, req model.ForgotPasswordRequest, ip, userAgent string) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("поиск пользователя: %w", err)
	}
	// Return success even if user not found to prevent email enumeration.
	if user == nil {
		return nil
	}

	pwd := generatePassword(passwordLength)
	hash, err := hashPassword(pwd)
	if err != nil {
		return err
	}
	if err := s.userRepo.SetPassword(ctx, user.ID, hash); err != nil {
		return fmt.Errorf("сохранение пароля: %w", err)
	}

	loginURL := fmt.Sprintf("%s/cabinet/login?email=%s", s.siteURL, email)
	if err := s.mailer.SendNewPassword(email, user.FirstName, pwd, loginURL); err != nil {
		s.logger.Error("forgot password email failed", zap.String("email", email), zap.Error(err))
		return fmt.Errorf("не удалось отправить письмо. Попробуйте позже.")
	}

	_ = s.logRepo.Write(ctx, "password_reset", email, ip, "new password issued", userAgent)
	return nil
}

// Me returns the user from a JWT token.
func (s *AuthService) Me(ctx context.Context, userID int) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("поиск пользователя: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}
	return user, nil
}

// IssueUserToken generates a signed JWT for a regular cabinet user.
// Called after successful OTP verification to provide an auto-login token.
func (s *AuthService) IssueUserToken(userID int, email string) (string, error) {
	return s.issueToken(userID, email, "user")
}

func (s *AuthService) issueToken(userID int, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(jwtTTL).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}

// UpdateProfile updates non-sensitive profile fields.
func (s *AuthService) UpdateProfile(ctx context.Context, userID int, req model.UpdateProfileRequest, ip, ua string) error {
	if err := s.userRepo.UpdateProfile(ctx, userID, req); err != nil {
		return fmt.Errorf("обновление профиля: %w", err)
	}
	_ = s.logRepo.Write(ctx, "profile_updated", "", ip, fmt.Sprintf("user #%d updated profile", userID), ua)
	return nil
}

// ChangePassword verifies the current password and sets a new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID int, req model.ChangePasswordRequest, ip, ua string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("пользователь не найден")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		_ = s.logRepo.Write(ctx, "password_change_failed", user.Email, ip, "wrong current password", ua)
		return fmt.Errorf("неверный текущий пароль")
	}
	hash, err := hashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err := s.userRepo.SetPassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("сохранение пароля: %w", err)
	}
	_ = s.logRepo.Write(ctx, "password_changed", user.Email, ip, "password changed by user", ua)
	return nil
}

// ExportData assembles all personal data for a user and emails it to them.
// Required by: RF 152-ФЗ ст.14, GDPR Art.15, CCPA §1798.100.
func (s *AuthService) ExportData(ctx context.Context, userID int, ip, ua string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("пользователь не найден")
	}
	regs, err := s.regRepo.ListRegistrationsForExport(ctx, userID)
	if err != nil {
		return fmt.Errorf("получение регистраций: %w", err)
	}
	consents, err := s.consentRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("получение согласий: %w", err)
	}

	export := model.DataExport{
		User:          *user,
		Registrations: regs,
		Consents:      consents,
		ExportedAt:    time.Now(),
	}
	raw, _ := json.MarshalIndent(export, "", "  ")

	if err := s.mailer.SendDataExport(user.Email, user.FirstName, string(raw)); err != nil {
		return fmt.Errorf("отправка данных: %w", err)
	}
	_ = s.logRepo.Write(ctx, "data_export_requested", user.Email, ip, "GDPR/152-ФЗ data export sent", ua)
	return nil
}

// DeleteAccount anonymizes the user's data (right to erasure).
// Required by: RF 152-ФЗ ст.21, GDPR Art.17, CCPA §1798.105.
// The user row is anonymized in-place; foreign-key integrity is preserved.
func (s *AuthService) DeleteAccount(ctx context.Context, userID int, password, ip, ua string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("пользователь не найден")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		_ = s.logRepo.Write(ctx, "account_delete_failed", user.Email, ip, "wrong password", ua)
		return fmt.Errorf("неверный пароль")
	}
	// Withdraw all consents
	if err := s.consentRepo.WithdrawAll(ctx, userID); err != nil {
		s.logger.Warn("withdraw consents failed", zap.Error(err))
	}
	// Log deletion request for DPA audit trail
	if err := s.consentRepo.RequestDeletion(ctx, userID, "user self-delete"); err != nil {
		s.logger.Warn("deletion request record failed", zap.Error(err))
	}
	// Anonymize personal data
	if err := s.userRepo.Anonymize(ctx, userID); err != nil {
		return fmt.Errorf("анонимизация данных: %w", err)
	}
	_ = s.logRepo.Write(ctx, "account_deleted", user.Email, ip, "user account anonymized", ua)
	return nil
}

// CancelRegistration cancels the user's registration for an event.
func (s *AuthService) CancelRegistration(ctx context.Context, userID, eventID int, ip, ua string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("пользователь не найден")
	}
	if err := s.regRepo.CancelByUser(ctx, userID, eventID); err != nil {
		return err
	}
	_ = s.logRepo.Write(ctx, "registration_cancelled", user.Email, ip,
		fmt.Sprintf("cancelled event #%d", eventID), ua)
	return nil
}

// ParseToken validates a JWT and returns userID, role, and email.
func (s *AuthService) ParseToken(tokenStr string) (int, string, string, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return 0, "", "", fmt.Errorf("invalid token claims")
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, "", "", fmt.Errorf("invalid token sub")
	}
	role, _ := claims["role"].(string)
	email, _ := claims["email"].(string)
	return int(sub), role, email, nil
}
