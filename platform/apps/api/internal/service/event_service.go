package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"platform/api/internal/model"
	"platform/api/internal/repository"
)

type EventService struct {
	eventRepo *repository.EventRepository
	regRepo   *repository.RegistrationRepository
	logger    *zap.Logger
}

func NewEventService(
	eventRepo *repository.EventRepository,
	regRepo *repository.RegistrationRepository,
	logger *zap.Logger,
) *EventService {
	return &EventService{eventRepo: eventRepo, regRepo: regRepo, logger: logger}
}

func (s *EventService) List(ctx context.Context, tenantID, status string, limit, offset int) ([]model.Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	events, err := s.eventRepo.ListByTenant(ctx, tenantID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("список мероприятий: %w", err)
	}
	return events, nil
}

func (s *EventService) Get(ctx context.Context, tenantID, id string) (*model.Event, error) {
	event, err := s.eventRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("мероприятие: %w", err)
	}
	return event, nil
}

func (s *EventService) Create(ctx context.Context, tenantID, userID string, req model.CreateEventRequest) (*model.Event, error) {
	event, err := s.eventRepo.Create(ctx, tenantID, userID, req)
	if err != nil {
		return nil, fmt.Errorf("создание мероприятия: %w", err)
	}
	s.logger.Info("event created", zap.String("id", event.ID), zap.String("tenant", tenantID))
	return event, nil
}

func (s *EventService) Publish(ctx context.Context, tenantID, id string) error {
	if err := s.eventRepo.UpdateStatus(ctx, tenantID, id, "published"); err != nil {
		return fmt.Errorf("публикация: %w", err)
	}
	return nil
}

// Register registers a user for an event.
func (s *EventService) Register(ctx context.Context, tenantID, userID string, req model.RegisterEventRequest) (*model.Registration, error) {
	if !req.ConsentGiven {
		return nil, fmt.Errorf("необходимо дать согласие на обработку данных")
	}

	event, err := s.eventRepo.FindByID(ctx, tenantID, req.EventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("мероприятие не найдено")
	}
	if event.Status == "ended" || event.Status == "archived" {
		return nil, fmt.Errorf("регистрация закрыта")
	}

	// Check capacity
	if event.Capacity > 0 {
		count, err := s.regRepo.CountByEvent(ctx, tenantID, req.EventID)
		if err != nil {
			return nil, fmt.Errorf("проверка мест: %w", err)
		}
		if count >= event.Capacity {
			return nil, fmt.Errorf("мест нет — мероприятие заполнено")
		}
	}

	// Idempotent — return existing registration if already registered
	existing, err := s.regRepo.Find(ctx, tenantID, req.EventID, userID)
	if err != nil {
		return nil, fmt.Errorf("проверка регистрации: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	reg, err := s.regRepo.Create(ctx, tenantID, req.EventID, userID, req.ConsentGiven)
	if err != nil {
		return nil, fmt.Errorf("создание регистрации: %w", err)
	}
	return reg, nil
}
