package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"platform/api/internal/model"
	"platform/api/internal/repository"
)

type QuestionService struct {
	qRepo    *repository.QuestionRepository
	msgRepo  *repository.QuestionMessageRepository
	eventRepo *repository.EventRepository
	logger   *zap.Logger
}

func NewQuestionService(
	qRepo *repository.QuestionRepository,
	msgRepo *repository.QuestionMessageRepository,
	eventRepo *repository.EventRepository,
	logger *zap.Logger,
) *QuestionService {
	return &QuestionService{qRepo: qRepo, msgRepo: msgRepo, eventRepo: eventRepo, logger: logger}
}

func (s *QuestionService) ListQuestions(ctx context.Context, tenantID, eventID, role string) ([]model.Question, error) {
	// Non-admins see only public questions.
	publicOnly := role != "event_admin" && role != "tenant_owner" && role != "super_admin"
	return s.qRepo.ListByEvent(ctx, tenantID, eventID, publicOnly)
}

func (s *QuestionService) CreateQuestion(ctx context.Context, tenantID, eventID, userID string, req model.CreateQuestionRequest) (*model.Question, error) {
	event, err := s.eventRepo.FindByID(ctx, tenantID, eventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("мероприятие не найдено")
	}
	if event.Status == "ended" || event.Status == "archived" {
		return nil, fmt.Errorf("мероприятие завершено")
	}
	return s.qRepo.Create(ctx, tenantID, eventID, userID, req.Subject, req.IsPublic)
}

func (s *QuestionService) ListMessages(ctx context.Context, tenantID, questionID, userID, role string) ([]model.QuestionMessage, error) {
	q, err := s.qRepo.FindByID(ctx, tenantID, questionID)
	if err != nil || q == nil {
		return nil, fmt.Errorf("вопрос не найден")
	}
	// Participants can only read their own question's messages.
	isAdmin := role == "event_admin" || role == "tenant_owner" || role == "super_admin"
	if !isAdmin && q.UserID != userID {
		return nil, fmt.Errorf("доступ запрещён")
	}
	return s.msgRepo.ListByQuestion(ctx, questionID)
}

func (s *QuestionService) AddMessage(ctx context.Context, tenantID, questionID, senderID, role string, req model.CreateMessageRequest) (*model.QuestionMessage, error) {
	q, err := s.qRepo.FindByID(ctx, tenantID, questionID)
	if err != nil || q == nil {
		return nil, fmt.Errorf("вопрос не найден")
	}
	if q.Status == "closed" {
		return nil, fmt.Errorf("вопрос закрыт")
	}
	// Only question owner or admins can send messages.
	isAdmin := role == "event_admin" || role == "tenant_owner" || role == "super_admin"
	if !isAdmin && q.UserID != senderID {
		return nil, fmt.Errorf("доступ запрещён")
	}
	senderRole := "participant"
	if isAdmin {
		senderRole = "admin"
	}
	msg, err := s.msgRepo.Create(ctx, questionID, senderID, senderRole, req.Body)
	if err != nil {
		return nil, fmt.Errorf("отправка сообщения: %w", err)
	}
	// Auto-mark question as in-progress when admin replies.
	if isAdmin && q.Status == "open" {
		_ = s.qRepo.UpdateStatus(ctx, tenantID, questionID, "in_progress")
	}
	return msg, nil
}
