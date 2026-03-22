package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// SessionTimeoutThreshold is the maximum silence window before a session is
// considered abandoned. Clients must send a ping at least every 30 s;
// we allow 3 missed pings before closing the session automatically.
const SessionTimeoutThreshold = 90 * time.Second

// OnlineSessionService manages the lifecycle of online_sessions.
type OnlineSessionService struct {
	repo    *repository.OnlineSessionRepository
	regRepo *repository.RegistrationRepository
	logger  *zap.Logger
}

func NewOnlineSessionService(
	repo *repository.OnlineSessionRepository,
	regRepo *repository.RegistrationRepository,
	logger *zap.Logger,
) *OnlineSessionService {
	return &OnlineSessionService{repo: repo, regRepo: regRepo, logger: logger}
}

// Connect creates a new session for a user joining an online event stream.
//
// Reconnect behaviour: if the user has an open session (e.g. after a reload
// or network hiccup), that session is closed with end_reason=error before a
// fresh one is created. This avoids phantom open sessions from duplicate tabs.
func (s *OnlineSessionService) Connect(
	ctx context.Context,
	userID, eventID int,
) (*model.OnlineSession, error) {
	// Look up optional registration_id for richer aggregation.
	var regID *int
	reg, _ := s.regRepo.FindByEventAndUser(ctx, eventID, userID)
	if reg != nil {
		id := reg.ID
		regID = &id
	}

	// Close any stale open session for this user+event before creating a new
	// one. This handles reload / accidental reconnect / dual-tab scenarios.
	existing, err := s.repo.FindOpen(ctx, eventID, userID)
	if err != nil {
		return nil, fmt.Errorf("online_session Connect: find open: %w", err)
	}
	if existing != nil {
		if err := s.repo.Close(ctx, existing.SessionUUID, "error"); err != nil {
			s.logger.Warn("online_session: could not close stale session",
				zap.String("uuid", existing.SessionUUID),
				zap.Error(err))
			// Non-fatal — proceed to create new session.
		}
	}

	session, err := s.repo.Create(ctx, eventID, userID, regID)
	if err != nil {
		return nil, fmt.Errorf("online_session Connect: create: %w", err)
	}

	s.logger.Info("online_session: connect",
		zap.Int("user_id", userID),
		zap.Int("event_id", eventID),
		zap.String("uuid", session.SessionUUID))

	return session, nil
}

// Ping updates the heartbeat timestamp for an open session.
// Returns an error if the session does not exist or is already closed.
func (s *OnlineSessionService) Ping(ctx context.Context, sessionUUID string) error {
	if err := s.repo.UpdatePing(ctx, sessionUUID); err != nil {
		return fmt.Errorf("online_session Ping: %w", err)
	}
	return nil
}

// Disconnect explicitly closes a session when the participant leaves.
func (s *OnlineSessionService) Disconnect(ctx context.Context, sessionUUID string) error {
	if err := s.repo.Close(ctx, sessionUUID, "explicit"); err != nil {
		return fmt.Errorf("online_session Disconnect: %w", err)
	}
	s.logger.Info("online_session: disconnect", zap.String("uuid", sessionUUID))
	return nil
}

// TimeoutStaleSessions closes sessions that have not sent a ping for longer
// than SessionTimeoutThreshold. Intended to be called by a background worker.
// Returns the number of sessions that were closed.
func (s *OnlineSessionService) TimeoutStaleSessions(ctx context.Context) (int64, error) {
	n, err := s.repo.TimeoutStale(ctx, SessionTimeoutThreshold)
	if err != nil {
		return 0, fmt.Errorf("online_session TimeoutStaleSessions: %w", err)
	}
	if n > 0 {
		s.logger.Info("online_session: timed out stale sessions", zap.Int64("count", n))
	}
	return n, nil
}

// AdminStats returns per-registration participation totals for an event.
func (s *OnlineSessionService) AdminStats(
	ctx context.Context,
	eventID int,
) (*model.OnlineStatsResponse, error) {
	summary, err := s.repo.SummaryByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("online_session AdminStats: %w", err)
	}
	active, err := s.repo.ActiveCount(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("online_session AdminStats active count: %w", err)
	}
	return &model.OnlineStatsResponse{
		EventID:       eventID,
		ActiveNow:     active,
		Registrations: summary,
	}, nil
}
