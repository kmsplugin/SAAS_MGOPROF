package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

const (
	resendCooldownSeconds = 10
	otpTTLMinutes         = 10
	passwordLength        = 10
	passwordAlphabet      = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
)

// RegisterResult is returned to the handler after a registration attempt.
type RegisterResult struct {
	Status          string `json:"status"`
	Message         string `json:"message"`
	ShowOTP         bool   `json:"show_otp"`
	PasswordIssued  bool   `json:"password_issued"`
	CabinetLoginURL string `json:"cabinet_login"`
}

// AlreadyRegisteredError is returned when the user is already verified for an event.
type AlreadyRegisteredError struct {
	CabinetURL string
}

func (e *AlreadyRegisteredError) Error() string { return "already_registered" }

// RegistrationService orchestrates the full registration flow.
type RegistrationService struct {
	userRepo    *repository.UserRepository
	eventRepo   *repository.EventRepository
	regRepo     *repository.RegistrationRepository
	fieldRepo   *repository.FieldRepository
	logRepo     *repository.LogRepository
	consentRepo *repository.ConsentRepository
	mailer      RegistrationMailer
	authSvc     *AuthService
	geo         *GeoResolver
	logger      *zap.Logger
	siteURL     string
}

func NewRegistrationService(
	userRepo *repository.UserRepository,
	eventRepo *repository.EventRepository,
	regRepo *repository.RegistrationRepository,
	fieldRepo *repository.FieldRepository,
	logRepo *repository.LogRepository,
	consentRepo *repository.ConsentRepository,
	m RegistrationMailer,
	authSvc *AuthService,
	geo *GeoResolver,
	logger *zap.Logger,
	siteURL string,
) *RegistrationService {
	return &RegistrationService{
		userRepo:    userRepo,
		eventRepo:   eventRepo,
		regRepo:     regRepo,
		fieldRepo:   fieldRepo,
		logRepo:     logRepo,
		consentRepo: consentRepo,
		mailer:      m,
		authSvc:     authSvc,
		geo:         geo,
		logger:      logger,
		siteURL:     siteURL,
	}
}

// Register runs the full registration flow: find/create user, cooldown, send OTP.
func (s *RegistrationService) Register(
	ctx context.Context,
	req model.RegisterRequest,
	ip, userAgent string,
) (*RegisterResult, error) {
	// 152-ФЗ ст.9, GDPR Art.7: consent is mandatory before any data processing.
	if !req.ConsentGiven {
		return nil, fmt.Errorf("для регистрации необходимо согласие на обработку персональных данных")
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	cabinetURL := fmt.Sprintf("%s/cabinet/login?email=%s", s.siteURL, email)

	event, err := s.eventRepo.FindActiveByID(ctx, req.EventID)
	if err != nil {
		return nil, fmt.Errorf("проверка мероприятия: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("мероприятие недоступно для регистрации")
	}

	// Validate custom fields before starting TX
	if err := s.validateCustomAnswers(ctx, req.EventID, req.Answers); err != nil {
		return nil, err
	}

	otp := generateOTP()
	otpExpiresAt := time.Now().Add(otpTTLMinutes * time.Minute)
	geoInfo := s.geo.Resolve(ip)
	devInfo := ParseUserAgent(userAgent)

	var (
		passwordIssued bool
		needSendOTP    = true
		newRegID       int
		message        = "Код отправлен на почту."
	)

	txErr := s.userRepo.WithTx(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		// 1. Find or create user
		user, err := s.userRepo.FindByEmailTx(ctx, tx, email)
		if err != nil {
			return fmt.Errorf("поиск пользователя: %w", err)
		}

		var userID int
		if user != nil {
			userID = user.ID
			if err := s.userRepo.UpdateTx(ctx, tx, userID, req, ip, geoInfo, userAgent); err != nil {
				return fmt.Errorf("обновление пользователя: %w", err)
			}
			// Issue a placeholder password if not set (will be replaced at OTP verification)
			if user.PasswordHash == "" {
				pwd := generatePassword(passwordLength)
				hash, err := hashPassword(pwd)
				if err != nil {
					return err
				}
				if err := s.userRepo.SetPasswordTx(ctx, tx, userID, hash); err != nil {
					return err
				}
				passwordIssued = true
			}
		} else {
			pwd := generatePassword(passwordLength)
			hash, err := hashPassword(pwd)
			if err != nil {
				return err
			}
			userID, err = s.userRepo.CreateTx(ctx, tx, email, req, ip, geoInfo, userAgent, hash)
			if err != nil {
				return fmt.Errorf("создание пользователя: %w", err)
			}
			passwordIssued = true
		}

		// 2a. Capacity check — only for new registrations (existing re-register skips this)
		if event.Capacity > 0 {
			count, err := s.regRepo.CountVerifiedByEvent(ctx, tx, req.EventID)
			if err != nil {
				return fmt.Errorf("проверка вместимости: %w", err)
			}
			if count >= event.Capacity {
				return fmt.Errorf("мест нет — мероприятие заполнено (%d/%d)", count, event.Capacity)
			}
		}

		// 2b. Save consent (152-ФЗ ст.9, GDPR Art.7) — idempotent via ON CONFLICT DO NOTHING
		if err := s.consentRepo.SaveTx(ctx, tx, userID, req.EventID, ip, userAgent); err != nil {
			s.logger.Warn("consent save failed", zap.Error(err))
			// Non-fatal: log but don't block registration
		}

		// 3. Check existing registration
		existing, err := s.regRepo.FindByEventAndUserTx(ctx, tx, req.EventID, userID)
		if err != nil {
			return fmt.Errorf("поиск регистрации: %w", err)
		}

		if existing != nil {
			if existing.Status == "verified" {
				return &AlreadyRegisteredError{CabinetURL: cabinetURL}
			}
			// Cooldown check
			last := existing.UpdatedAt
			if last == nil {
				last = &existing.CreatedAt
			}
			if time.Since(*last).Seconds() < resendCooldownSeconds {
				needSendOTP = false
				message = "Регистрация уже начата. Проверьте письмо и введите код подтверждения."
				newRegID = existing.ID
			} else {
				if err := s.regRepo.UpdateOTPTx(ctx, tx, existing.ID, otp, otpExpiresAt, ip, geoInfo, devInfo); err != nil {
					return err
				}
				newRegID = existing.ID
				message = "Новый код отправлен на почту."
			}
		} else {
			id, err := s.regRepo.CreateTx(ctx, tx, req.EventID, userID, otp, otpExpiresAt, ip, geoInfo, devInfo)
			if err != nil {
				return err
			}
			newRegID = id
		}

		// 4. Save custom field answers
		if len(req.Answers) > 0 && newRegID > 0 {
			answers := ToFieldAnswers(newRegID, req.Answers)
			if err := s.fieldRepo.SaveAnswersTx(ctx, tx, newRegID, answers); err != nil {
				s.logger.Warn("save answers failed", zap.Error(err))
				// non-fatal — don't block registration
			}
		}
		return nil
	})

	if txErr != nil {
		if alreadyErr, ok := txErr.(*AlreadyRegisteredError); ok {
			_ = s.logRepo.Write(ctx, "registration_duplicate_verified", email, ip,
				fmt.Sprintf("verified registration exists for event #%d", req.EventID), userAgent)
			return &RegisterResult{
				Status:          "already_registered",
				Message:         "Вы уже зарегистрированы на это мероприятие. Войдите в кабинет.",
				CabinetLoginURL: alreadyErr.CabinetURL,
			}, nil
		}
		_ = s.logRepo.Write(ctx, "registration_error", email, ip, txErr.Error(), userAgent)
		return nil, txErr
	}

	if needSendOTP {
		if mailErr := s.mailer.SendRegistration(email, req.FirstName, otp, event.Title); mailErr != nil {
			s.logger.Error("registration email failed",
				zap.String("email", email),
				zap.Error(mailErr),
			)
			_ = s.logRepo.Write(ctx, "registration_mail_error", email, ip,
				"email failed: "+mailErr.Error(), userAgent)
			return nil, fmt.Errorf("регистрация сохранена, но письмо не отправлено. Попробуйте повторить позже.")
		}
		_ = s.logRepo.Write(ctx, "registration_created", email, ip,
			fmt.Sprintf("OTP sent for event #%d", req.EventID), userAgent)
	} else {
		_ = s.logRepo.Write(ctx, "registration_pending_reused", email, ip,
			fmt.Sprintf("pending reused for event #%d", req.EventID), userAgent)
	}

	if passwordIssued {
		message += " После подтверждения кода данные для входа придут на почту."
	}

	status := "success"
	if !needSendOTP {
		status = "pending"
	}

	return &RegisterResult{
		Status:          status,
		Message:         message,
		ShowOTP:         true,
		PasswordIssued:  passwordIssued,
		CabinetLoginURL: cabinetURL,
	}, nil
}

// VerifyOTP verifies the OTP, marks the registration as verified, issues fresh
// credentials and sends the welcome email.
// Returns cabinetURL (for redirect) and a JWT token (for auto-login).
func (s *RegistrationService) VerifyOTP(ctx context.Context, req model.VerifyOTPRequest, ip, userAgent string) (cabinetURL, token string, err error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	cabinetURL = fmt.Sprintf("%s/cabinet", s.siteURL)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("поиск пользователя: %w", err)
	}
	if user == nil {
		return "", "", fmt.Errorf("пользователь не найден")
	}

	reg, err := s.regRepo.FindByEventAndUser(ctx, req.EventID, user.ID)
	if err != nil {
		return "", "", fmt.Errorf("поиск регистрации: %w", err)
	}
	if reg == nil {
		return "", "", fmt.Errorf("регистрация не найдена")
	}
	if reg.Status == "verified" {
		// Already verified — issue a new token so the user can still auto-login.
		t, _ := s.authSvc.IssueUserToken(user.ID, email)
		return cabinetURL, t, fmt.Errorf("регистрация уже подтверждена")
	}
	if time.Now().After(reg.OTPExpiresAt) {
		return "", "", fmt.Errorf("код подтверждения истёк. Запросите новый.")
	}
	if reg.OTPCode != req.OTP {
		_ = s.logRepo.Write(ctx, "otp_failed", email, ip, "invalid OTP", userAgent)
		return "", "", fmt.Errorf("неверный код подтверждения")
	}

	// Mark verified and clear OTP
	participantToken := generateParticipantToken()
	if err := s.regRepo.SetVerified(ctx, reg.ID, participantToken); err != nil {
		return "", "", fmt.Errorf("подтверждение: %w", err)
	}
	_ = s.logRepo.Write(ctx, "registration_verified", email, ip,
		fmt.Sprintf("event #%d verified, token=%s", req.EventID, participantToken), userAgent)

	// Generate fresh password and update the user record
	pwd := generatePassword(passwordLength)
	hash, err := hashPassword(pwd)
	if err != nil {
		s.logger.Error("password hash failed", zap.String("email", email), zap.Error(err))
	} else if err := s.userRepo.SetPassword(ctx, user.ID, hash); err != nil {
		s.logger.Error("set password failed", zap.String("email", email), zap.Error(err))
	}

	// Issue JWT so the frontend can auto-login without a separate POST /auth/login.
	jwtToken, jwtErr := s.authSvc.IssueUserToken(user.ID, email)
	if jwtErr != nil {
		s.logger.Error("issue user token failed", zap.String("email", email), zap.Error(jwtErr))
		// Non-fatal: user can still log in manually with credentials from welcome email.
	}

	// Determine ticket URL (non-empty for offline/hybrid events)
	ticketURL := ""
	event, eventErr := s.eventRepo.FindByID(ctx, req.EventID)
	if eventErr == nil && event != nil {
		ticketURL = resolveTicketURL(s.siteURL, req.EventID, event.EventType)
	}

	// Send welcome email with credentials and cabinet link.
	// Record the timestamp so the admin timeline shows welcome_email_sent_at.
	if mailErr := s.mailer.SendWelcome(email, user.FirstName, pwd, user.ID, reg.ID, cabinetURL, ticketURL); mailErr != nil {
		s.logger.Warn("welcome email failed", zap.String("email", email), zap.Error(mailErr))
	} else {
		if err := s.regRepo.SetWelcomeEmailSent(ctx, reg.ID); err != nil {
			s.logger.Warn("set welcome_email_sent_at failed", zap.String("email", email), zap.Error(err))
		}
	}

	return cabinetURL, jwtToken, nil
}

// validateCustomAnswers checks required custom fields are filled.
func (s *RegistrationService) validateCustomAnswers(ctx context.Context, eventID int, answers []model.AnswerInput) error {
	fields, err := s.fieldRepo.ListByEvent(ctx, eventID)
	if err != nil {
		// Non-fatal: if we can't load fields, don't block registration
		return nil
	}
	ansMap := make(map[int]string, len(answers))
	for _, a := range answers {
		ansMap[a.FieldID] = a.Value
	}
	for _, f := range fields {
		if f.IsRequired {
			v, ok := ansMap[f.ID]
			if !ok || v == "" {
				return fmt.Errorf("обязательное поле не заполнено: %s", f.Label)
			}
		}
	}
	return nil
}

// resolveTicketURL returns a QR-ticket URL for offline/hybrid events.
// Online events have no QR ticket — participants join via event_link from cabinet.
func resolveTicketURL(siteURL string, eventID int, eventType string) string {
	if eventType == "online" {
		return ""
	}
	return fmt.Sprintf("%s/cabinet/events/%d/ticket", siteURL, eventID)
}

func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1_000_000))
	return fmt.Sprintf("%06d", n.Int64())
}

func generatePassword(length int) string {
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(passwordAlphabet))))
		b[i] = passwordAlphabet[n.Int64()]
	}
	return string(b)
}

func hashPassword(pwd string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(h), nil
}

// generateParticipantToken returns a URL-safe 12-character alphanumeric token.
// Enough entropy for a few hundred thousand registrations without collision.
const tokenAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

func generateParticipantToken() string {
	const length = 12
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(tokenAlphabet))))
		b[i] = tokenAlphabet[n.Int64()]
	}
	return string(b)
}
