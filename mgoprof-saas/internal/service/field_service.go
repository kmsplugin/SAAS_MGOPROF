package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"mgoprof-saas/internal/model"
	"mgoprof-saas/internal/repository"
)

// FieldService manages custom registration fields per event.
type FieldService struct {
	repo   *repository.FieldRepository
	logger *zap.Logger
}

func NewFieldService(repo *repository.FieldRepository, logger *zap.Logger) *FieldService {
	return &FieldService{repo: repo, logger: logger}
}

func (s *FieldService) ListByEvent(ctx context.Context, eventID int) ([]model.EventField, error) {
	return s.repo.ListByEvent(ctx, eventID)
}

func (s *FieldService) Create(ctx context.Context, eventID int, req model.CreateFieldRequest) (*model.EventField, error) {
	f := model.EventField{
		EventID:         eventID,
		Label:           req.Label,
		FieldType:       req.FieldType,
		Options:         req.Options,
		Placeholder:     req.Placeholder,
		HelperText:      req.HelperText,
		ValidationRegex: req.ValidationRegex,
		IsRequired:      req.IsRequired,
		InBadge:         req.InBadge,
		InReport:        req.InReport,
		InExport:        req.InExport,
		ListID:          req.ListID,
		MinValue:        req.MinValue,
		MaxValue:        req.MaxValue,
		MaxLength:       req.MaxLength,
		SortOrder:       req.SortOrder,
	}
	if f.Options == nil {
		f.Options = []string{}
	}
	return s.repo.Create(ctx, f)
}

func (s *FieldService) Update(ctx context.Context, eventID, fieldID int, req model.CreateFieldRequest) (*model.EventField, error) {
	f := model.EventField{
		ID:              fieldID,
		EventID:         eventID,
		Label:           req.Label,
		FieldType:       req.FieldType,
		Options:         req.Options,
		Placeholder:     req.Placeholder,
		HelperText:      req.HelperText,
		ValidationRegex: req.ValidationRegex,
		IsRequired:      req.IsRequired,
		InBadge:         req.InBadge,
		InReport:        req.InReport,
		InExport:        req.InExport,
		ListID:          req.ListID,
		MinValue:        req.MinValue,
		MaxValue:        req.MaxValue,
		MaxLength:       req.MaxLength,
		SortOrder:       req.SortOrder,
	}
	if f.Options == nil {
		f.Options = []string{}
	}
	updated, err := s.repo.Update(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("update field %d: %w", fieldID, err)
	}
	return updated, nil
}

func (s *FieldService) Delete(ctx context.Context, eventID, fieldID int) error {
	return s.repo.Delete(ctx, fieldID, eventID)
}

// ValidateAnswers checks that all required fields for an event are present in the answers.
func (s *FieldService) ValidateAnswers(ctx context.Context, eventID int, answers []model.AnswerInput) error {
	fields, err := s.repo.ListByEvent(ctx, eventID)
	if err != nil {
		return fmt.Errorf("load fields: %w", err)
	}

	// Index answers by field_id for O(1) lookup
	ansMap := make(map[int]string, len(answers))
	for _, a := range answers {
		ansMap[a.FieldID] = a.Value
	}

	for _, f := range fields {
		if f.IsRequired {
			v, ok := ansMap[f.ID]
			if !ok || v == "" {
				return fmt.Errorf("обязательное поле не заполнено: %s", f.Label)
			}
		}
	}
	return nil
}

// ToFieldAnswers converts AnswerInput slice to FieldAnswer slice for storage.
func ToFieldAnswers(regID int, answers []model.AnswerInput) []model.FieldAnswer {
	out := make([]model.FieldAnswer, 0, len(answers))
	for _, a := range answers {
		out = append(out, model.FieldAnswer{
			RegistrationID: regID,
			FieldID:        a.FieldID,
			Value:          a.Value,
		})
	}
	return out
}

// LogFieldError logs non-critical field save errors without failing the request.
func (s *FieldService) LogError(msg string, err error) {
	s.logger.Error(msg, zap.Error(err))
}
