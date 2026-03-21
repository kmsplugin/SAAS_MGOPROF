package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"mgoprof-saas/internal/mailer"
	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

const adminOTPTTL = 10 * time.Minute

// pendingAdminLogin is an in-memory store for admin OTP state.
// In production consider Redis; for single-instance deployment this is sufficient.
type pendingAdmin struct {
	email     string
	adminID   int
	name      string
	otpHash   string
	expiresAt time.Time
}

// AdminService handles admin authentication and dashboard data.
type AdminService struct {
	regRepo    *repository.RegistrationRepository
	eventRepo  *repository.EventRepository
	logRepo    *repository.LogRepository
	adminRepo  *repository.AdminRepository
	mailer     *mailer.Mailer
	authSvc    *AuthService
	logger     *zap.Logger
	adminEmail string
	adminHash  string
	adminName  string
	pending    map[string]*pendingAdmin // keyed by email
}

func NewAdminService(
	regRepo *repository.RegistrationRepository,
	eventRepo *repository.EventRepository,
	logRepo *repository.LogRepository,
	adminRepo *repository.AdminRepository,
	m *mailer.Mailer,
	authSvc *AuthService,
	logger *zap.Logger,
	adminEmail, adminHash, adminName string,
) *AdminService {
	return &AdminService{
		regRepo:    regRepo,
		eventRepo:  eventRepo,
		logRepo:    logRepo,
		adminRepo:  adminRepo,
		mailer:     m,
		authSvc:    authSvc,
		logger:     logger,
		adminEmail: strings.ToLower(adminEmail),
		adminHash:  adminHash,
		adminName:  adminName,
		pending:    make(map[string]*pendingAdmin),
	}
}

// Login step 1: verify password against admin_users table (or env-var fallback), send OTP.
func (s *AdminService) Login(ctx context.Context, req model.AdminLoginRequest, ip, userAgent string) error {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var adminID int
	var adminName string

	// 1. Check admin_users table first (multi-admin support).
	if s.adminRepo != nil {
		admin, err := s.adminRepo.FindByEmail(ctx, email)
		if err != nil {
			s.logger.Error("admin lookup failed", zap.String("email", email), zap.Error(err))
			return fmt.Errorf("ошибка аутентификации")
		}
		if admin != nil {
			if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
				_ = s.logRepo.Write(ctx, "admin_login_failed", email, ip, "wrong password (db)", userAgent)
				return fmt.Errorf("неверный логин или пароль")
			}
			adminID = admin.ID
			adminName = admin.Name
		}
	}

	// 2. Fall back to env-var single-admin (backward compatibility).
	if adminID == 0 {
		if email != s.adminEmail {
			_ = s.logRepo.Write(ctx, "admin_login_failed", email, ip, "wrong email", userAgent)
			return fmt.Errorf("неверный логин или пароль")
		}
		if s.adminHash == "" {
			return fmt.Errorf("пароль администратора не задан")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(s.adminHash), []byte(req.Password)); err != nil {
			_ = s.logRepo.Write(ctx, "admin_login_failed", email, ip, "wrong password (env)", userAgent)
			return fmt.Errorf("неверный логин или пароль")
		}
		adminName = s.adminName
	}

	otp := generateOTP()
	hash, err := hashPassword(otp)
	if err != nil {
		return err
	}

	s.pending[email] = &pendingAdmin{
		email:     email,
		adminID:   adminID,
		name:      adminName,
		otpHash:   hash,
		expiresAt: time.Now().Add(adminOTPTTL),
	}

	if err := s.mailer.SendAdminOTP(email, adminName, otp); err != nil {
		s.logger.Error("admin OTP email failed", zap.String("email", email), zap.Error(err))
		delete(s.pending, email)
		return fmt.Errorf("не удалось отправить OTP. Попробуйте позже.")
	}

	_ = s.logRepo.Write(ctx, "admin_otp_sent", email, ip, "admin OTP issued", userAgent)
	return nil
}

// VerifyOTP step 2: verify OTP and issue JWT with real admin ID.
func (s *AdminService) VerifyOTP(ctx context.Context, req model.AdminVerifyOTPRequest, ip, userAgent string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	p, ok := s.pending[email]
	if !ok {
		return "", fmt.Errorf("сессия входа истекла. Введите логин и пароль заново.")
	}
	if time.Now().After(p.expiresAt) {
		delete(s.pending, email)
		return "", fmt.Errorf("срок действия OTP истёк. Войдите заново.")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(p.otpHash), []byte(req.OTP)); err != nil {
		_ = s.logRepo.Write(ctx, "admin_otp_failed", email, ip, "invalid admin OTP", userAgent)
		return "", fmt.Errorf("неверный OTP-код")
	}

	delete(s.pending, email)

	// Issue token with real admin ID (0 only for env-var fallback admin).
	token, err := s.authSvc.issueToken(p.adminID, email, "admin")
	if err != nil {
		return "", fmt.Errorf("выпуск токена: %w", err)
	}

	_ = s.logRepo.Write(ctx, "admin_login_success", email, ip, "admin logged in", userAgent)
	return token, nil
}

// BootstrapSuperAdmin seeds the first super_admin from env vars if admin_users is empty.
// Called once on startup; idempotent.
func (s *AdminService) BootstrapSuperAdmin(ctx context.Context) {
	if s.adminRepo == nil || s.adminEmail == "" || s.adminHash == "" {
		return
	}
	if err := s.adminRepo.BootstrapSuperAdmin(ctx, s.adminEmail, s.adminName, s.adminHash); err != nil {
		s.logger.Error("bootstrap super_admin failed", zap.Error(err))
	}
}

// GetStats returns dashboard counters.
func (s *AdminService) GetStats(ctx context.Context) (*model.Stats, error) {
	stats, err := s.regRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("статистика: %w", err)
	}
	return stats, nil
}

// GetRegistrations returns all registrations (admin list).
func (s *AdminService) GetRegistrations(ctx context.Context) ([]model.RegistrationRow, error) {
	rows, err := s.regRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("список регистраций: %w", err)
	}
	return rows, nil
}

// GetRecentRegistrations returns N most recent registrations.
func (s *AdminService) GetRecentRegistrations(ctx context.Context, limit int) ([]model.RegistrationRow, error) {
	rows, err := s.regRepo.ListRecent(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("последние регистрации: %w", err)
	}
	return rows, nil
}

// GetLogs returns recent log entries.
func (s *AdminService) GetLogs(ctx context.Context, limit int) ([]model.Log, error) {
	logs, err := s.logRepo.ListRecent(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("логи: %w", err)
	}
	return logs, nil
}

// GetEventsWithStats returns events with registration counters.
func (s *AdminService) GetEventsWithStats(ctx context.Context) ([]model.EventWithStats, error) {
	events, err := s.eventRepo.ListWithStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("мероприятия: %w", err)
	}
	return events, nil
}
