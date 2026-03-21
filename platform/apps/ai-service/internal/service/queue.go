package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"platform/ai-service/internal/model"
)

// JobQueue — очередь задач AI-обработки на основе каналов.
// Запускает фиксированный пул воркеров.
type JobQueue struct {
	jobs     chan model.AIJob
	pipeline *Pipeline
	logger   *zap.Logger
	wg       sync.WaitGroup

	// Статусы задач по event_id (только в памяти — для простого API статуса)
	mu       sync.RWMutex
	statuses map[uuid.UUID]*model.JobStatus
}

func NewJobQueue(pipeline *Pipeline, workers, bufSize int, logger *zap.Logger) *JobQueue {
	return &JobQueue{
		jobs:     make(chan model.AIJob, bufSize),
		pipeline: pipeline,
		logger:   logger,
		statuses: make(map[uuid.UUID]*model.JobStatus),
	}
}

// Start запускает воркеры. Должен быть вызван один раз при старте сервиса.
func (q *JobQueue) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i)
	}
}

// Stop ждёт завершения всех воркеров.
func (q *JobQueue) Stop() {
	close(q.jobs)
	q.wg.Wait()
}

// Enqueue добавляет задачу в очередь. Если очередь полна — возвращает ошибку.
func (q *JobQueue) Enqueue(job model.AIJob) error {
	q.mu.Lock()
	q.statuses[job.EventID] = &model.JobStatus{
		EventID: job.EventID,
		Status:  "queued",
	}
	q.mu.Unlock()

	select {
	case q.jobs <- job:
		q.logger.Info("queue: enqueued job", zap.String("event_id", job.EventID.String()))
		return nil
	default:
		return fmt.Errorf("queue: full, try again later")
	}
}

// Status возвращает текущий статус обработки события.
func (q *JobQueue) Status(eventID uuid.UUID) *model.JobStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()
	s := q.statuses[eventID]
	if s == nil {
		return nil
	}
	copy := *s
	return &copy
}

func (q *JobQueue) worker(ctx context.Context, id int) {
	defer q.wg.Done()
	q.logger.Info("queue: worker started", zap.Int("id", id))

	for job := range q.jobs {
		q.setStatus(job.EventID, "processing", "", nil)

		err := q.pipeline.Process(ctx, job)
		if err != nil {
			q.logger.Error("queue: job failed",
				zap.Int("worker", id),
				zap.String("event_id", job.EventID.String()),
				zap.Error(err),
			)
			q.setStatus(job.EventID, "failed", err.Error(), nil)
			continue
		}

		q.setStatus(job.EventID, "done", "", []model.SummaryType{
			model.SummaryTypeTranscript,
			model.SummaryTypeSummary,
			model.SummaryTypeHighlights,
			model.SummaryTypeChapters,
			model.SummaryTypeActionItems,
		})
	}

	q.logger.Info("queue: worker stopped", zap.Int("id", id))
}

func (q *JobQueue) setStatus(eventID uuid.UUID, status, errMsg string, steps []model.SummaryType) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.statuses[eventID] = &model.JobStatus{
		EventID:  eventID,
		Status:   status,
		Progress: steps,
		Error:    errMsg,
	}
}
