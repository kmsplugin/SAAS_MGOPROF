package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"platform/media-service/internal/handler"
	"platform/media-service/internal/service"
)

func main() {
	_ = godotenv.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	livekitHost   := mustEnv("LIVEKIT_HOST")
	livekitURL    := getEnv("LIVEKIT_URL", livekitHost)
	apiKey        := mustEnv("LIVEKIT_API_KEY")
	apiSecret     := mustEnv("LIVEKIT_API_SECRET")
	port          := getEnv("PORT", "8010")
	internalToken := mustEnv("MEDIA_SERVICE_TOKEN")

	tokenSvc := service.NewTokenService(apiKey, apiSecret)
	roomSvc  := service.NewRoomService(livekitHost, tokenSvc)

	tokenHandler := handler.NewTokenHandler(tokenSvc, livekitURL, logger)
	roomHandler  := handler.NewRoomHandler(roomSvc, logger)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "media-service"})
	})

	media := r.Group("/media", bearerAuth(internalToken))
	tokenHandler.RegisterRoutes(media)
	roomHandler.RegisterRoutes(media)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		logger.Info("media-service started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	logger.Info("media-service stopped")
}

func bearerAuth(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " || auth[7:] != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env not set: " + key)
	}
	return v
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
