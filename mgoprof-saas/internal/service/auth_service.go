package service

import (
	"context"
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

// AuthService handles user cabinet authentication.
type AuthService struct {
	userRepo *repository.UserRepository
	logRepo  *repository.LogRepository
	mailer   *mailer.Mailer
	geo      *GeoResolver
	logger   *zap.Logger
	jwtSecret []byte
	siteURL  string
}

func NewAuthService(
	userRepo *repository.UserRepository,
	logRepo *repository.LogRepository,
	m *mailer.Mailer,
	geo *GeoResolver,
	logger *zap.Logger,
	jwtSecret, siteURL string,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		logRepo:   logRepo,
		mailer:    m,
		geo:       geo,
		logger:    logger,
		jwtSecret: []byte(jwtSecret),
		siteURL:   siteURL,
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

// ParseToken validates a JWT and returns userID and role.
func (s *AuthService) ParseToken(tokenStr string) (int, string, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return 0, "", fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return 0, "", fmt.Errorf("invalid token claims")
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("invalid token sub")
	}
	role, _ := claims["role"].(string)
	return int(sub), role, nil
}
