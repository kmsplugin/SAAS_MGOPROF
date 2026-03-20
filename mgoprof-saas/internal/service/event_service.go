package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// EventService handles public event operations.
type EventService struct {
	eventRepo *repository.EventRepository
	logger    *zap.Logger
}

func NewEventService(eventRepo *repository.EventRepository, logger *zap.Logger) *EventService {
	return &EventService{eventRepo: eventRepo, logger: logger}
}

// ListActive returns all active events.
func (s *EventService) ListActive(ctx context.Context) ([]model.Event, error) {
	events, err := s.eventRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("список мероприятий: %w", err)
	}
	return events, nil
}

// GetByID returns a single event by ID.
func (s *EventService) GetByID(ctx context.Context, id int) (*model.Event, error) {
	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("мероприятие: %w", err)
	}
	return event, nil
}

// Create creates a new event.
func (s *EventService) Create(ctx context.Context, req model.CreateEventRequest) (*model.Event, error) {
	id, err := s.eventRepo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("создание мероприятия: %w", err)
	}
	return s.eventRepo.FindByID(ctx, id)
}

// Update updates an existing event.
func (s *EventService) Update(ctx context.Context, id int, req model.CreateEventRequest) (*model.Event, error) {
	existing, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("поиск мероприятия: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("мероприятие не найдено")
	}
	if err := s.eventRepo.Update(ctx, id, req); err != nil {
		return nil, fmt.Errorf("обновление мероприятия: %w", err)
	}
	return s.eventRepo.FindByID(ctx, id)
}
