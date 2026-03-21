package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"platform/api/internal/model"
	"platform/api/internal/repository"
)

const jwtTTL = 72 * time.Hour

type AuthService struct {
	userRepo   *repository.UserRepository
	tenantRepo *repository.TenantRepository
	logger     *zap.Logger
	jwtSecret  []byte
}

func NewAuthService(
	userRepo *repository.UserRepository,
	tenantRepo *repository.TenantRepository,
	logger *zap.Logger,
	jwtSecret string,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
		logger:     logger,
		jwtSecret:  []byte(jwtSecret),
	}
}

// Register creates a new user within a tenant.
func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest, ip string) (string, *model.User, error) {
	if !req.ConsentGiven {
		return "", nil, fmt.Errorf("согласие на обработку персональных данных обязательно")
	}

	tenant, err := s.tenantRepo.FindBySlug(ctx, req.TenantSlug)
	if err != nil {
		return "", nil, fmt.Errorf("поиск организации: %w", err)
	}
	if tenant == nil {
		return "", nil, fmt.Errorf("организация не найдена")
	}

	existing, err := s.userRepo.FindByEmail(ctx, tenant.ID, req.Email)
	if err != nil {
		return "", nil, fmt.Errorf("проверка email: %w", err)
	}
	if existing != nil {
		return "", nil, fmt.Errorf("пользователь с таким email уже существует")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("хеширование пароля: %w", err)
	}

	user, err := s.userRepo.Create(ctx, tenant.ID, req.Email, req.FirstName, req.LastName, string(hash), "participant")
	if err != nil {
		return "", nil, fmt.Errorf("создание пользователя: %w", err)
	}

	token, err := s.issueToken(user.ID, tenant.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, fmt.Errorf("выпуск токена: %w", err)
	}

	return token, user, nil
}

// Login authenticates a user and returns a JWT.
func (s *AuthService) Login(ctx context.Context, req model.LoginRequest, ip string) (string, *model.User, error) {
	tenant, err := s.tenantRepo.FindBySlug(ctx, req.TenantSlug)
	if err != nil {
		return "", nil, fmt.Errorf("поиск организации: %w", err)
	}
	if tenant == nil {
		return "", nil, fmt.Errorf("неверный логин или пароль")
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := s.userRepo.FindByEmail(ctx, tenant.ID, email)
	if err != nil {
		return "", nil, fmt.Errorf("поиск пользователя: %w", err)
	}
	if user == nil || user.PasswordHash == "" {
		return "", nil, fmt.Errorf("неверный email или пароль")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", nil, fmt.Errorf("неверный email или пароль")
	}

	token, err := s.issueToken(user.ID, tenant.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, fmt.Errorf("выпуск токена: %w", err)
	}

	_ = s.userRepo.SetLastLogin(ctx, tenant.ID, user.ID)
	return token, user, nil
}

// ParseToken validates a JWT and returns userID, tenantID, role, email.
func (s *AuthService) ParseToken(tokenStr string) (string, string, string, string, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", "", "", "", fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return "", "", "", "", fmt.Errorf("invalid token claims")
	}
	userID, _ := claims["sub"].(string)
	tenantID, _ := claims["tenant_id"].(string)
	role, _ := claims["role"].(string)
	email, _ := claims["email"].(string)
	return userID, tenantID, role, email, nil
}

func (s *AuthService) issueToken(userID, tenantID, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID,
		"tenant_id": tenantID,
		"email":     email,
		"role":      role,
		"exp":       time.Now().Add(jwtTTL).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}

func (s *AuthService) Me(ctx context.Context, tenantID, userID string) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("поиск пользователя: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}
	return user, nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, tenantID, userID, firstName, lastName, avatarURL string) (*model.User, error) {
	user, err := s.userRepo.UpdateProfile(ctx, tenantID, userID, firstName, lastName, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("обновление профиля: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}
	return user, nil
}

// ParseTokenFunc returns a function compatible with the middleware.Auth signature.
func (s *AuthService) ParseTokenFunc() func(string) (string, string, string, string, error) {
	return s.ParseToken
}

// HashPassword is a utility for seeding/testing.
func HashPassword(pwd string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}
