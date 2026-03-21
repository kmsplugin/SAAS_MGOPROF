package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"platform/api/internal/config"
	"platform/api/internal/handler"
	"platform/api/internal/middleware"
	"platform/api/internal/repository"
	"platform/api/internal/service"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connect", zap.Error(err))
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// ── Repositories ──────────────────────────────────────────────────────────
	tenantRepo := repository.NewTenantRepository(db)
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	regRepo := repository.NewRegistrationRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, tenantRepo, logger, cfg.JWTSecret)

	eventSvc := service.NewEventService(eventRepo, regRepo, logger)

	roomSvc := service.NewRoomService(
		roomRepo, eventRepo, regRepo,
		cfg.MediaServiceURL, cfg.MediaServiceToken, cfg.LiveKitURL,
		logger,
	)

	// ── Handlers ──────────────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authSvc, logger)
	eventHandler := handler.NewEventHandler(eventSvc, logger)
	roomHandler := handler.NewRoomHandler(roomSvc, logger)

	// ── Router ────────────────────────────────────────────────────────────────
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.CORS())
	r.Use(middleware.Logger(logger))
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "platform-api"})
	})

	authMW := middleware.Auth(authSvc.ParseTokenFunc())

	v1 := r.Group("/api/v1")
	authHandler.RegisterRoutes(v1, authMW)
	eventHandler.RegisterRoutes(v1, authMW)
	roomHandler.RegisterRoutes(v1, authMW)

	// ── Server ────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("platform-api starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}
}
