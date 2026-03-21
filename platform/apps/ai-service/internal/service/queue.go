package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"platform/ai-service/internal/model"
	"platform/ai-service/internal/repository"
)

// JobQueue — очередь задач AI-обработки на основе каналов с PostgreSQL-персистентностью.
// Предотвращает дублирование: одно активное задание на событие в любой момент.
type JobQueue struct {
	jobs     chan model.AIJob
	pipeline *Pipeline
	repo     *repository.AIRepository
	logger   *zap.Logger
	wg       sync.WaitGroup

	// In-memory cache статусов для быстрого GET /status
	mu       sync.RWMutex
	statuses map[uuid.UUID]*model.JobStatus
}

func NewJobQueue(pipeline *Pipeline, repo *repository.AIRepository, workers, bufSize int, logger *zap.Logger) *JobQueue {
	return &JobQueue{
		jobs:     make(chan model.AIJob, bufSize),
		pipeline: pipeline,
		repo:     repo,
		logger:   logger,
		statuses: make(map[uuid.UUID]*model.JobStatus),
	}
}

// Start запускает воркеры и восстанавливает незавершённые задачи из БД.
func (q *JobQueue) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i)
	}
	go q.recoverPendingJobs(ctx)
}

// Stop ждёт завершения всех воркеров.
func (q *JobQueue) Stop() {
	close(q.jobs)
	q.wg.Wait()
}

// Enqueue добавляет задачу в очередь.
// Возвращает ошибку, если активное задание для события уже существует или очередь полна.
func (q *JobQueue) Enqueue(job model.AIJob) error {
	ctx := context.Background()

	// Duplicate prevention via DB
	exists, err := q.repo.JobExistsForEvent(ctx, job.EventID)
	if err != nil {
		q.logger.Warn("queue: duplicate check failed, proceeding", zap.Error(err))
	} else if exists {
		return fmt.Errorf("queue: active job already exists for event %s", job.EventID)
	}

	// Persist to DB
	jobDBID, err := q.repo.CreateJob(ctx, job)
	if err != nil {
		return fmt.Errorf("queue: persist job: %w", err)
	}
	job.JobDBID = jobDBID

	q.mu.Lock()
	q.statuses[job.EventID] = &model.JobStatus{
		EventID: job.EventID,
		Status:  "queued",
	}
	q.mu.Unlock()

	select {
	case q.jobs <- job:
		q.logger.Info("queue: enqueued job",
			zap.String("event_id", job.EventID.String()),
			zap.String("db_job_id", jobDBID.String()),
		)
		return nil
	default:
		// Channel full — update DB back to failed
		_ = q.repo.UpdateJobStatus(ctx, jobDBID, "failed", "queue channel full")
		return fmt.Errorf("queue: channel full, try again later")
	}
}

// Status возвращает текущий статус обработки события из in-memory кэша.
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
		if job.JobDBID != uuid.Nil {
			_ = q.repo.UpdateJobStatus(ctx, job.JobDBID, "processing", "")
		}

		err := q.pipeline.Process(ctx, job)
		if err != nil {
			q.logger.Error("queue: job failed",
				zap.Int("worker", id),
				zap.String("event_id", job.EventID.String()),
				zap.Error(err),
			)
			q.setStatus(job.EventID, "failed", err.Error(), nil)
			if job.JobDBID != uuid.Nil {
				_ = q.repo.UpdateJobStatus(ctx, job.JobDBID, "failed", err.Error())
			}
			continue
		}

		steps := []model.SummaryType{
			model.SummaryTypeTranscript,
			model.SummaryTypeSummary,
			model.SummaryTypeHighlights,
			model.SummaryTypeChapters,
			model.SummaryTypeActionItems,
		}
		q.setStatus(job.EventID, "done", "", steps)
		if job.JobDBID != uuid.Nil {
			_ = q.repo.UpdateJobStatus(ctx, job.JobDBID, "done", "")
		}
	}

	q.logger.Info("queue: worker stopped", zap.Int("id", id))
}

// recoverPendingJobs re-enqueues unfinished jobs after a service restart.
func (q *JobQueue) recoverPendingJobs(ctx context.Context) {
	pending, err := q.repo.GetPendingJobs(ctx)
	if err != nil {
		q.logger.Error("queue: recover pending jobs", zap.Error(err))
		return
	}
	if len(pending) == 0 {
		return
	}
	q.logger.Info("queue: recovering jobs", zap.Int("count", len(pending)))
	for _, dbJob := range pending {
		// Mark interrupted processing jobs back to queued in DB
		if dbJob.Status == "processing" {
			_ = q.repo.UpdateJobStatus(ctx, dbJob.ID, "queued", "")
		}
		job := model.AIJob{
			EventID:     dbJob.EventID,
			TenantID:    dbJob.TenantID,
			AudioURL:    dbJob.AudioURL,
			Language:    dbJob.Language,
			RecordingID: dbJob.RecordingID,
			JobDBID:     dbJob.ID,
		}
		q.mu.Lock()
		q.statuses[job.EventID] = &model.JobStatus{EventID: job.EventID, Status: "queued"}
		q.mu.Unlock()

		select {
		case q.jobs <- job:
			q.logger.Info("queue: recovered job", zap.String("event_id", job.EventID.String()))
		default:
			q.logger.Warn("queue: channel full during recovery, job skipped",
				zap.String("event_id", job.EventID.String()))
		}
	}
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
