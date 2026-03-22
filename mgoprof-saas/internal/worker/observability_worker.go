package worker

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// WelcomeMailGapThreshold is how long we wait after OTP verification before
// flagging a missing welcome email as anomalous.
const WelcomeMailGapThreshold = 5 * time.Minute

// RunObservabilityWorker starts a background loop that periodically checks for
// known anomaly patterns and logs structured warnings for operator visibility.
//
// Current detectors:
//  1. welcome_email_sent_at IS NULL more than WelcomeMailGapThreshold after
//     otp_verified_at — user is verified but welcome mail never went out.
//
// The loop stops when ctx is cancelled.
func RunObservabilityWorker(
	ctx context.Context,
	db *sqlx.DB,
	tickInterval time.Duration,
	logger *zap.Logger,
) {
	logger.Info("observability worker started", zap.Duration("interval", tickInterval))

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("observability worker stopped")
			return
		case <-ticker.C:
			checkWelcomeMailGap(ctx, db, logger)
		}
	}
}

// checkWelcomeMailGap detects verified registrations where welcome_email_sent_at
// is still NULL after WelcomeMailGapThreshold.
//
// Decision: fail-open behaviour for welcome mail is intentional and documented.
// This check makes the failure *observable* without changing the business logic.
func checkWelcomeMailGap(ctx context.Context, db *sqlx.DB, logger *zap.Logger) {
	type anomaly struct {
		RegID     int    `db:"reg_id"`
		UserEmail string `db:"user_email"`
		EventID   int    `db:"event_id"`
	}

	var rows []anomaly
	err := db.SelectContext(ctx, &rows, `
		SELECT r.id AS reg_id, u.email AS user_email, r.event_id
		FROM reg_registrations r
		JOIN reg_users u ON u.id = r.user_id
		WHERE r.status = 'verified'
		  AND r.otp_verified_at IS NOT NULL
		  AND r.welcome_email_sent_at IS NULL
		  AND r.otp_verified_at < NOW() - $1::interval
		LIMIT 50`,
		WelcomeMailGapThreshold.String(),
	)
	if err != nil {
		logger.Error("observability: welcome_mail_gap query failed", zap.Error(err))
		return
	}

	for _, row := range rows {
		logger.Warn("observability: welcome_email_gap detected",
			zap.Int("reg_id", row.RegID),
			zap.String("user_email", row.UserEmail),
			zap.Int("event_id", row.EventID),
			zap.String("detector", "welcome_email_gap"),
			// operator_action: investigate mailer logs for this registration
		)
	}

	if len(rows) > 0 {
		logger.Warn("observability: welcome_email_gap summary",
			zap.Int("affected_registrations", len(rows)),
			zap.String("detector", "welcome_email_gap"),
			zap.String("action", "check mailer logs; consider resend"),
		)
	}
}
