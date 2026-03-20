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

	"mgoprof-saas/internal/config"
	"mgoprof-saas/internal/handler"
	"mgoprof-saas/internal/mailer"
	"mgoprof-saas/internal/middleware"
	"mgoprof-saas/internal/repository"
	"mgoprof-saas/internal/service"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer logger.Sync() //nolint:errcheck

	// Database
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	defer db.Close()

	// Repositories
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)
	regRepo := repository.NewRegistrationRepository(db)
	logRepo := repository.NewLogRepository(db)
	reportRepo := repository.NewReportRepository(db)

	// Mailer (SMTP — compatible with Resend, Yandex, Mail.ru, etc.)
	mail := mailer.New(mailer.Config{
		Host:      cfg.SMTPHost,
		Port:      cfg.SMTPPort,
		Username:  cfg.SMTPUsername,
		Password:  cfg.SMTPPassword,
		FromEmail: cfg.MailFromEmail,
		FromName:  cfg.MailFromName,
		SiteURL:   cfg.SiteURL,
	})

	// GeoIP resolver (graceful degradation if no DB file)
	geo := service.NewGeoResolver(cfg.GeoDBPath, logger)
	defer geo.Close()

	// Services
	authSvc := service.NewAuthService(userRepo, logRepo, mail, geo, logger, cfg.JWTSecret, cfg.SiteURL)
	regSvc := service.NewRegistrationService(userRepo, eventRepo, regRepo, logRepo, mail, geo, logger, cfg.SiteURL)
	eventSvc := service.NewEventService(eventRepo, logger)
	adminSvc := service.NewAdminService(regRepo, eventRepo, logRepo, mail, authSvc, logger,
		cfg.AdminEmail, cfg.AdminPasswordHash, cfg.AdminName)
	reportSvc := service.NewReportService(reportRepo, eventRepo, logger)

	// Handlers
	regHandler := handler.NewRegistrationHandler(regSvc, logger)
	authHandler := handler.NewAuthHandler(authSvc, regRepo, logger)
	eventHandler := handler.NewEventHandler(eventSvc, logger)
	adminHandler := handler.NewAdminHandler(adminSvc, eventSvc, authSvc, logger)
	reportHandler := handler.NewReportHandler(reportSvc, logger)

	// Router
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	// Rate limiters
	registerLimiter := middleware.NewRateLimiter(5, 30*time.Second)   // 5 reg attempts / 30 s per IP
	otpLimiter := middleware.NewRateLimiter(10, time.Minute)           // 10 OTP tries / min per IP
	adminLoginLimiter := middleware.NewRateLimiter(5, time.Minute)     // 5 admin login tries / min

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.Logger(logger))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	authMW := middleware.Auth(authSvc)
	api := r.Group("/api")
	{
		// Public registration routes (rate-limited)
		api.POST("/register", registerLimiter.Limit(), regHandler.HandleRegister)
		api.POST("/verify-otp", otpLimiter.Limit(), regHandler.HandleVerifyOTP)

		// Public event list
		eventHandler.RegisterRoutes(api)

		// Cabinet (JWT-protected)
		authHandler.RegisterRoutes(api, authMW)

		// Admin auth (rate-limited) + admin panel
		api.POST("/admin/login", adminLoginLimiter.Limit(), adminHandler.Login)
		api.POST("/admin/verify-otp", otpLimiter.Limit(), adminHandler.VerifyOTP)
		adminHandler.RegisterProtectedRoutes(api, authMW)

		// Per-event reports
		reportHandler.RegisterRoutes(api, authMW)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server stopped")
}
