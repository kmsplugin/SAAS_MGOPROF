package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"platform/ai-service/internal/config"
	"platform/ai-service/internal/handler"
	"platform/ai-service/internal/repository"
	"platform/ai-service/internal/service"
)

func main() {
	// ── Logger ──────────────────────────────────────────────────
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// ── Config ──────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("config load failed", zap.Error(err))
	}

	// ── Database ────────────────────────────────────────────────
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	defer db.Close()
	logger.Info("database connected")

	// ── Services ────────────────────────────────────────────────
	repo := repository.NewAIRepository(db)

	var transcriber service.TranscriptionProvider
	if cfg.OpenAIAPIKey != "" {
		transcriber = service.NewWhisperTranscriber(cfg.OpenAIAPIKey, logger)
		logger.Info("transcription: using OpenAI Whisper")
	} else {
		// Заглушка для разработки без Whisper API
		transcriber = &noopTranscriber{logger: logger}
		logger.Warn("transcription: OPENAI_API_KEY not set, using noop transcriber (dev mode)")
	}

	claudeClient := service.NewClaudeClient(cfg.AnthropicAPIKey, cfg.AnthropicModel, logger)
	pipeline := service.NewPipeline(transcriber, claudeClient, repo, cfg.MaxAudioSize, logger)
	queue := service.NewJobQueue(pipeline, cfg.WorkerCount, cfg.QueueSize, logger)

	// Запускаем воркеры
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.Start(ctx, cfg.WorkerCount)

	// ── Handlers ────────────────────────────────────────────────
	webhookH := handler.NewWebhookHandler(queue, repo, logger)
	summaryH := handler.NewSummaryHandler(queue, repo, logger)

	// ── Router ──────────────────────────────────────────────────
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ginZapLogger(logger))

	r.GET("/health", handler.HealthCheck)

	// Вебхуки от LiveKit Egress
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/livekit-egress", webhookH.LiveKitEgress)
	}

	// REST API для клиентов (main API проксирует с добавлением X-Tenant-ID)
	v1 := r.Group("/api/v1")
	{
		events := v1.Group("/events/:event_id/ai")
		events.GET("", summaryH.GetEventSummaries)
		events.GET("/:type", summaryH.GetSummaryByType)
		events.POST("/process", summaryH.TriggerProcessing)
		events.GET("/status", summaryH.GetStatus)
	}

	// ── HTTP Server ─────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("ai-service started", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// ── Graceful Shutdown ───────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	cancel()
	queue.Stop()
	logger.Info("stopped")
}

// ginZapLogger — минимальный middleware для логирования запросов через zap.
func ginZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

// noopTranscriber — заглушка транскрипции для dev-режима без OpenAI ключа.
type noopTranscriber struct {
	logger *zap.Logger
}

func (n *noopTranscriber) Transcribe(_ context.Context, audioPath, _ string) (string, error) {
	n.logger.Warn("noop transcriber: returning placeholder transcript",
		zap.String("path", audioPath),
	)
	return "[Транскрипция недоступна: не задан OPENAI_API_KEY. " +
		"Установите переменную окружения для активации Whisper транскрипции.]", nil
}
