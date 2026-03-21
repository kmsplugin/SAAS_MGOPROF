package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// ReportService builds per-event analytics reports.
type ReportService struct {
	reportRepo *repository.ReportRepository
	eventRepo  *repository.EventRepository
	logger     *zap.Logger
}

func NewReportService(
	reportRepo *repository.ReportRepository,
	eventRepo *repository.EventRepository,
	logger *zap.Logger,
) *ReportService {
	return &ReportService{
		reportRepo: reportRepo,
		eventRepo:  eventRepo,
		logger:     logger,
	}
}

// EventReportFull bundles event metadata with analytics data.
type EventReportFull struct {
	Event       *model.Event
	Report      *model.EventReport
	GeneratedAt time.Time
}

// GetEventReport returns the full analytics package for one event.
func (s *ReportService) GetEventReport(ctx context.Context, eventID int) (*EventReportFull, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("event lookup: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("мероприятие не найдено")
	}

	report, err := s.reportRepo.GetEventReport(ctx, eventID)
	if err != nil {
		s.logger.Error("report query failed", zap.Int("event_id", eventID), zap.Error(err))
		return nil, fmt.Errorf("формирование отчёта: %w", err)
	}

	return &EventReportFull{
		Event:       event,
		Report:      report,
		GeneratedAt: time.Now(),
	}, nil
}

// GetEventExport returns raw rows for a per-event CSV export.
func (s *ReportService) GetEventExport(ctx context.Context, eventID int) (*model.Event, []model.RegistrationRow, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, nil, fmt.Errorf("event lookup: %w", err)
	}
	if event == nil {
		return nil, nil, fmt.Errorf("мероприятие не найдено")
	}
	rows, err := s.reportRepo.GetEventRegistrationsForExport(ctx, eventID)
	if err != nil {
		return nil, nil, err
	}
	return event, rows, nil
}

// GetEventExportFiltered returns rows filtered by optional date range.
func (s *ReportService) GetEventExportFiltered(ctx context.Context, eventID int, from, to string) (*model.Event, []model.RegistrationRow, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, nil, fmt.Errorf("event lookup: %w", err)
	}
	if event == nil {
		return nil, nil, fmt.Errorf("мероприятие не найдено")
	}
	rows, err := s.reportRepo.GetEventRegistrationsFiltered(ctx, eventID, from, to)
	if err != nil {
		return nil, nil, err
	}
	return event, rows, nil
}

// GetEventBadges returns event metadata and badge data for all verified participants.
func (s *ReportService) GetEventBadges(ctx context.Context, eventID int) (*model.Event, []model.BadgeData, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, nil, fmt.Errorf("event lookup: %w", err)
	}
	if event == nil {
		return nil, nil, fmt.Errorf("мероприятие не найдено")
	}
	badges, err := s.reportRepo.GetBadgesForEvent(ctx, eventID)
	if err != nil {
		return nil, nil, err
	}
	return event, badges, nil
}

// GetMultiEventStats returns aggregate stats for all active events.
func (s *ReportService) GetMultiEventStats(ctx context.Context) ([]model.MultiEventStats, error) {
	return s.reportRepo.GetMultiEventStats(ctx)
}
