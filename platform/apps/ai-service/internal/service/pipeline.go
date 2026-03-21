package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"platform/ai-service/internal/model"
	"platform/ai-service/internal/repository"
)

// Pipeline — главный оркестратор AI-обработки.
// Транскрибирует аудио, затем параллельно генерирует все типы AI-контента.
type Pipeline struct {
	transcriber TranscriptionProvider
	claude      *ClaudeClient
	repo        *repository.AIRepository
	maxAudioSize int64
	logger      *zap.Logger
}

func NewPipeline(
	transcriber TranscriptionProvider,
	claude *ClaudeClient,
	repo *repository.AIRepository,
	maxAudioSize int64,
	logger *zap.Logger,
) *Pipeline {
	return &Pipeline{
		transcriber:  transcriber,
		claude:       claude,
		repo:         repo,
		maxAudioSize: maxAudioSize,
		logger:       logger,
	}
}

// Process запускает полный цикл обработки для одной задачи.
func (p *Pipeline) Process(ctx context.Context, job model.AIJob) error {
	p.logger.Info("pipeline: start",
		zap.String("event_id", job.EventID.String()),
		zap.String("audio_url", job.AudioURL),
	)

	// Проверяем, что у тенанта включён AI
	enabled, err := p.repo.IsTenantAIEnabled(ctx, job.TenantID)
	if err != nil {
		return fmt.Errorf("pipeline: check ai_enabled: %w", err)
	}
	if !enabled {
		return fmt.Errorf("pipeline: ai not enabled for tenant %s", job.TenantID)
	}

	// 1. Передаём URL напрямую в speech-service — он скачивает и обрабатывает сам
	audioPath := job.AudioURL

	// 2. Транскрипция
	lang := job.Language
	if lang == "" {
		lang = "ru"
	}
	transcript, err := p.transcriber.Transcribe(ctx, audioPath, lang)
	if err != nil {
		return fmt.Errorf("pipeline: transcribe: %w", err)
	}

	// 3. Сохраняем транскрипт
	if err = p.saveSummary(ctx, job, model.SummaryTypeTranscript, transcript); err != nil {
		return err
	}
	p.logger.Info("pipeline: transcript saved", zap.String("event_id", job.EventID.String()))

	// 4. Параллельно генерируем остальные типы AI-контента
	type result struct {
		t    model.SummaryType
		text string
		err  error
	}

	tasks := []struct {
		t      model.SummaryType
		prompt string
	}{
		{model.SummaryTypeSummary, SummaryPrompt(transcript)},
		{model.SummaryTypeHighlights, HighlightsPrompt(transcript)},
		{model.SummaryTypeChapters, ChaptersPrompt(transcript)},
		{model.SummaryTypeActionItems, ActionItemsPrompt(transcript)},
	}

	results := make(chan result, len(tasks))
	for _, task := range tasks {
		t, prompt := task.t, task.prompt
		go func() {
			text, err := p.claude.Complete(ctx, systemPrompt, prompt)
			results <- result{t: t, text: text, err: err}
		}()
	}

	var firstErr error
	for range tasks {
		r := <-results
		if r.err != nil {
			p.logger.Error("pipeline: claude error",
				zap.String("type", string(r.t)),
				zap.Error(r.err),
			)
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		if err = p.saveSummary(ctx, job, r.t, r.text); err != nil {
			p.logger.Error("pipeline: save error",
				zap.String("type", string(r.t)),
				zap.Error(err),
			)
			if firstErr == nil {
				firstErr = err
			}
		} else {
			p.logger.Info("pipeline: saved", zap.String("type", string(r.t)))
		}
	}

	if firstErr != nil {
		return fmt.Errorf("pipeline: partial failure: %w", firstErr)
	}

	p.logger.Info("pipeline: done", zap.String("event_id", job.EventID.String()))
	return nil
}

func (p *Pipeline) saveSummary(ctx context.Context, job model.AIJob, t model.SummaryType, content string) error {
	s := &model.AISummary{
		ID:           uuid.New(),
		EventID:      job.EventID,
		TenantID:     job.TenantID,
		Type:         t,
		Content:      content,
		Language:     job.Language,
		ModelVersion: p.claude.model,
		GeneratedAt:  time.Now().UTC(),
	}
	return p.repo.Save(ctx, s)
}
