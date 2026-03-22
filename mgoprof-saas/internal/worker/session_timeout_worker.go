// Package worker contains background goroutines that run periodic maintenance tasks.
package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"mgoprof-saas/internal/service"
)

// RunSessionTimeoutWorker starts a background loop that closes stale online
// sessions. It ticks every tickInterval and calls
// OnlineSessionService.TimeoutStaleSessions.
//
// The loop stops when ctx is cancelled (i.e. on graceful shutdown).
func RunSessionTimeoutWorker(
	ctx context.Context,
	svc *service.OnlineSessionService,
	tickInterval time.Duration,
	logger *zap.Logger,
) {
	logger.Info("session timeout worker started",
		zap.Duration("interval", tickInterval))

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("session timeout worker stopped")
			return
		case <-ticker.C:
			n, err := svc.TimeoutStaleSessions(ctx)
			if err != nil {
				logger.Error("session timeout worker error", zap.Error(err))
			} else if n > 0 {
				logger.Info("session timeout worker: closed stale sessions",
					zap.Int64("count", n))
			}
		}
	}
}
