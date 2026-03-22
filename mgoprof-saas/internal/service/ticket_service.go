package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// TicketService handles participant QR tickets and entrance check-in.
type TicketService struct {
	regRepo   *repository.RegistrationRepository
	eventRepo *repository.EventRepository
	userRepo  *repository.UserRepository
	logger    *zap.Logger
	siteURL   string
}

func NewTicketService(
	regRepo *repository.RegistrationRepository,
	eventRepo *repository.EventRepository,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
	siteURL string,
) *TicketService {
	return &TicketService{regRepo: regRepo, eventRepo: eventRepo, userRepo: userRepo, logger: logger, siteURL: siteURL}
}

// GetTicket returns all data needed to render the ticket page for a participant.
func (s *TicketService) GetTicket(ctx context.Context, userID, eventID int) (*model.TicketInfo, error) {
	reg, err := s.regRepo.FindByEventAndUser(ctx, eventID, userID)
	if err != nil {
		return nil, fmt.Errorf("регистрация: %w", err)
	}
	if reg == nil {
		return nil, fmt.Errorf("регистрация не найдена")
	}
	if reg.Status != "verified" {
		return nil, fmt.Errorf("регистрация не подтверждена — сначала введите код из письма")
	}
	if reg.ParticipantToken == nil || *reg.ParticipantToken == "" {
		return nil, fmt.Errorf("билет ещё не сформирован")
	}

	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("мероприятие не найдено")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}

	// Build the URL that will be encoded in the QR code.
	// Admin will open this URL or the scanner will POST the token from it.
	ticketURL := fmt.Sprintf("%s/api/admin/checkin?token=%s", s.siteURL, *reg.ParticipantToken)

	eventLink := model.ResolveEventLink(
		reg.ParticipantRole,
		event.SpeakerLink,
		event.ViewerLink,
		event.CabinetLink,
	)

	return &model.TicketInfo{
		Registration: *reg,
		Event:        *event,
		User:         *user,
		TicketURL:    ticketURL,
		EventLink:    eventLink,
	}, nil
}

// CheckIn marks a participant as checked-in by their token.
func (s *TicketService) CheckIn(ctx context.Context, token string) (*model.CheckInResult, error) {
	if token == "" {
		return nil, fmt.Errorf("токен не указан")
	}

	reg, err := s.regRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("поиск по токену: %w", err)
	}
	if reg == nil {
		return nil, fmt.Errorf("билет не найден — неверный токен")
	}
	if reg.Status != "verified" {
		return nil, fmt.Errorf("регистрация не подтверждена (статус: %s)", reg.Status)
	}

	event, err := s.eventRepo.FindByID(ctx, reg.EventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("мероприятие не найдено")
	}

	user, err := s.userRepo.FindByID(ctx, reg.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}

	// Already checked in — return duplicate status
	if reg.CheckedInAt != nil {
		return &model.CheckInResult{
			Status:       "already",
			CheckedInAt:  *reg.CheckedInAt,
			Registration: *reg,
			User:         *user,
			Event:        *event,
		}, nil
	}

	if err := s.regRepo.SetCheckedIn(ctx, reg.ID); err != nil {
		return nil, fmt.Errorf("отметка входа: %w", err)
	}
	now := time.Now()
	reg.CheckedInAt = &now

	s.logger.Info("check-in",
		zap.String("token", token),
		zap.Int("user_id", reg.UserID),
		zap.Int("event_id", reg.EventID),
	)

	return &model.CheckInResult{
		Status:       "ok",
		CheckedInAt:  now,
		Registration: *reg,
		User:         *user,
		Event:        *event,
	}, nil
}
