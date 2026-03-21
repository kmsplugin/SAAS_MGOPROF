package service

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// ScanService implements the QR scan FSM with full audit logging.
type ScanService struct {
	scanRepo  *repository.ScanRepository
	eventRepo *repository.EventRepository
	logger    *zap.Logger
}

func NewScanService(
	scanRepo *repository.ScanRepository,
	eventRepo *repository.EventRepository,
	logger *zap.Logger,
) *ScanService {
	return &ScanService{
		scanRepo:  scanRepo,
		eventRepo: eventRepo,
		logger:    logger,
	}
}

// ProcessScan is the main entry point for a QR scan.
// It validates the token, applies FSM transitions, writes audit rows, and returns a ScanResponse.
func (s *ScanService) ProcessScan(
	ctx context.Context,
	eventID int,
	req model.ScanRequest,
	operatorID *int,
	ipAddress string,
) (*model.ScanResponse, error) {
	// 1. Resolve token → registration + user
	reg, user, err := s.scanRepo.FindRegistrationByToken(ctx, req.Token)
	if err == sql.ErrNoRows {
		logRow := &model.ScanLog{
			EventID:      eventID,
			ScannedToken: req.Token,
			ScanMode:     req.Mode,
			ScanResult:   "not_found",
			OperatorID:   operatorID,
			DeviceInfo:   req.DeviceInfo,
			IPAddress:    ipAddress,
			Note:         "token not found",
		}
		_, _ = s.scanRepo.InsertScanLog(ctx, logRow)
		return &model.ScanResponse{
			Result:  "not_found",
			Message: "Участник не найден по данному QR-коду.",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan token lookup: %w", err)
	}

	// 2. Verify the ticket belongs to this event
	if reg.EventID != eventID {
		logRow := &model.ScanLog{
			EventID:        eventID,
			RegistrationID: &reg.ID,
			ScannedToken:   req.Token,
			ScanMode:       req.Mode,
			ScanResult:     "wrong_event",
			OperatorID:     operatorID,
			DeviceInfo:     req.DeviceInfo,
			IPAddress:      ipAddress,
			Note:           fmt.Sprintf("ticket belongs to event %d, scanned at event %d", reg.EventID, eventID),
		}
		_, _ = s.scanRepo.InsertScanLog(ctx, logRow)
		return &model.ScanResponse{
			Result:  "wrong_event",
			Message: "QR-код предназначен для другого мероприятия.",
		}, nil
	}

	// 3. Check basic registration status
	if reg.Status == "cancelled" {
		logRow := &model.ScanLog{
			EventID:        eventID,
			RegistrationID: &reg.ID,
			ScannedToken:   req.Token,
			ScanMode:       req.Mode,
			ScanResult:     "cancelled",
			OperatorID:     operatorID,
			DeviceInfo:     req.DeviceInfo,
			IPAddress:      ipAddress,
		}
		_, _ = s.scanRepo.InsertScanLog(ctx, logRow)
		return &model.ScanResponse{
			Result:       "cancelled",
			Message:      "Регистрация отменена.",
			Registration: reg,
			User:         user,
		}, nil
	}

	// 4. Get extended status and scan count
	statusExt, scanCount, err := s.scanRepo.GetStatusExtended(ctx, reg.ID)
	if err != nil {
		return nil, fmt.Errorf("get status extended: %w", err)
	}

	// 5. Handle verify mode (read-only, no FSM change)
	if req.Mode == "verify" {
		logRow := &model.ScanLog{
			EventID:        eventID,
			RegistrationID: &reg.ID,
			ScannedToken:   req.Token,
			ScanMode:       "verify",
			ScanResult:     "ok",
			OperatorID:     operatorID,
			DeviceInfo:     req.DeviceInfo,
			IPAddress:      ipAddress,
		}
		_, _ = s.scanRepo.InsertScanLog(ctx, logRow)
		return &model.ScanResponse{
			Result:         "ok",
			Message:        "Участник верифицирован.",
			Registration:   reg,
			User:           user,
			StatusExtended: statusExt,
			ScanCount:      scanCount,
		}, nil
	}

	// 6. Apply FSM transitions for entry / exit
	var (
		newStatus  string
		scanResult string
		message    string
	)

	switch req.Mode {
	case "entry":
		switch statusExt {
		case "entered":
			// Already inside — treat as duplicate but still log
			scanResult = "duplicate"
			message = "Участник уже отмечен как вошедший."
			newStatus = statusExt
		case "exited", "confirmed", "qr_issued", "arrived", "registered":
			newStatus = "entered"
			scanResult = "ok"
			message = "Вход зафиксирован."
		case "participated", "no_show", "cancelled", "duplicate":
			scanResult = "duplicate"
			message = fmt.Sprintf("Статус участника: %s. Повторный вход невозможен.", statusExt)
			newStatus = statusExt
		default:
			newStatus = "entered"
			scanResult = "ok"
			message = "Вход зафиксирован."
		}

	case "exit":
		switch statusExt {
		case "entered":
			newStatus = "exited"
			scanResult = "ok"
			message = "Выход зафиксирован."
		case "exited":
			scanResult = "duplicate"
			message = "Участник уже отмечен как вышедший."
			newStatus = statusExt
		default:
			scanResult = "error"
			message = fmt.Sprintf("Невозможно зафиксировать выход: статус '%s'.", statusExt)
			newStatus = statusExt
		}
	}

	// 7. Persist FSM change and audit rows (best-effort; log errors but don't fail scan)
	logRow := &model.ScanLog{
		EventID:        eventID,
		RegistrationID: &reg.ID,
		ScannedToken:   req.Token,
		ScanMode:       req.Mode,
		ScanResult:     scanResult,
		OperatorID:     operatorID,
		DeviceInfo:     req.DeviceInfo,
		IPAddress:      ipAddress,
	}
	scanLogID, logErr := s.scanRepo.InsertScanLog(ctx, logRow)
	if logErr != nil {
		s.logger.Error("insert scan log", zap.Error(logErr))
	}

	if scanResult == "ok" {
		// Update extended status
		if err := s.scanRepo.SetStatusExtended(ctx, reg.ID, newStatus); err != nil {
			s.logger.Error("set status extended", zap.Error(err))
		}

		// Status history
		statusLogRow := &model.RegistrationStatusLog{
			RegistrationID: reg.ID,
			EventID:        eventID,
			PrevStatus:     statusExt,
			NewStatus:      newStatus,
			ChangedBy:      operatorID,
			ChangeSource:   "scanner",
		}
		if err := s.scanRepo.InsertStatusLog(ctx, statusLogRow); err != nil {
			s.logger.Error("insert status log", zap.Error(err))
		}

		// Attendance stream event
		attEvt := &model.AttendanceEvent{
			EventID:        eventID,
			RegistrationID: reg.ID,
			Action:         req.Mode,
		}
		if logErr == nil {
			attEvt.ScanLogID = &scanLogID
		}
		if err := s.scanRepo.InsertAttendanceEvent(ctx, attEvt); err != nil {
			s.logger.Error("insert attendance event", zap.Error(err))
		}

		// Entry/exit specific timestamp bookkeeping
		if req.Mode == "entry" {
			if err := s.scanRepo.SetFirstEntry(ctx, reg.ID); err != nil {
				s.logger.Error("set first entry", zap.Error(err))
			}
		} else if req.Mode == "exit" {
			if err := s.scanRepo.SetLastExit(ctx, reg.ID); err != nil {
				s.logger.Error("set last exit", zap.Error(err))
			}
		}
	}

	// Update scan count for response
	_, scanCount, _ = s.scanRepo.GetStatusExtended(ctx, reg.ID)

	return &model.ScanResponse{
		Result:         scanResult,
		Message:        message,
		Registration:   reg,
		User:           user,
		StatusExtended: newStatus,
		ScanCount:      scanCount,
	}, nil
}

// GetScanHistory returns the most recent scan logs for an event.
func (s *ScanService) GetScanHistory(ctx context.Context, eventID, limit int) ([]model.ScanLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.scanRepo.GetScanLogs(ctx, eventID, limit)
}

// GetAttendanceSummary returns real-time presence stats for an event.
func (s *ScanService) GetAttendanceSummary(ctx context.Context, eventID int) (entries, exits, present int, err error) {
	return s.scanRepo.GetAttendanceSummary(ctx, eventID)
}
