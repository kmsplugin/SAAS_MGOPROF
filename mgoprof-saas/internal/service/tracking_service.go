package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// TrackingService records visit and stream events for participants.
type TrackingService struct {
	trackRepo *repository.TrackingRepository
	regRepo   *repository.RegistrationRepository
	geo       *GeoResolver
	logger    *zap.Logger
}

func NewTrackingService(
	trackRepo *repository.TrackingRepository,
	regRepo *repository.RegistrationRepository,
	geo *GeoResolver,
	logger *zap.Logger,
) *TrackingService {
	return &TrackingService{trackRepo: trackRepo, regRepo: regRepo, geo: geo, logger: logger}
}

// RecordAction persists one tracking event.
// userID and eventID come from the JWT claim + URL param (already validated by caller).
func (s *TrackingService) RecordAction(ctx context.Context, userID, eventID int, action, ip, ua string) error {
	allowed := map[string]bool{
		"visit": true, "stream_connect": true,
		"stream_disconnect": true, "stream_error": true,
	}
	if !allowed[action] {
		return fmt.Errorf("unknown action: %s", action)
	}

	geo := s.geo.Resolve(ip)
	dev := ParseUserAgent(ua)

	// Try to find the registration_id (optional — tracking can exist without a verified reg)
	var regID *int
	reg, _ := s.regRepo.FindByEventAndUser(ctx, eventID, userID)
	if reg != nil {
		id := reg.ID
		regID = &id
	}

	t := model.TrackingEvent{
		EventID:        eventID,
		UserID:         userID,
		RegistrationID: regID,
		Action:         action,
		IPAddress:      ip,
		GeoCountry:     geo.Country,
		GeoRegion:      geo.Region,
		GeoCity:        geo.City,
		ISPName:        geo.ISPName,
		ISPASN:         geo.ISPASN,
		DeviceType:     dev.DeviceType,
		OSName:         dev.OSName,
		BrowserName:    dev.BrowserName,
		UserAgent:      ua,
	}

	if err := s.trackRepo.Record(ctx, t); err != nil {
		s.logger.Error("tracking record failed", zap.Int("user", userID), zap.Int("event", eventID), zap.Error(err))
		return err
	}
	return nil
}

// ListByEvent returns all raw tracking rows for an event (admin view).
func (s *TrackingService) ListByEvent(ctx context.Context, eventID int) ([]model.TrackingEvent, error) {
	return s.trackRepo.ListByEvent(ctx, eventID)
}
