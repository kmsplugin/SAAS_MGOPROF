package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string

	AnthropicAPIKey string
	AnthropicModel  string

	// speech-service — self-hosted STT/diarization/NLP (заменяет OpenAI Whisper)
	SpeechServiceURL   string
	SpeechServiceToken string

	LiveKitWebhookSecret string

	// Worker settings
	WorkerCount  int
	QueueSize    int
	MaxAudioSize int64 // bytes
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                 getEnv("PORT", "8020"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://platform:platform_secret@localhost:5432/platform?sslmode=disable"),
		AnthropicAPIKey:      getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:       getEnv("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
		SpeechServiceURL:     getEnv("SPEECH_SERVICE_URL", "http://localhost:8030"),
		SpeechServiceToken:   getEnv("SPEECH_SERVICE_TOKEN", "dev_internal_token"),
		LiveKitWebhookSecret: getEnv("LIVEKIT_API_SECRET", "devsecret"),
		WorkerCount:          3,
		QueueSize:            50,
		MaxAudioSize:         500 * 1024 * 1024, // 500 MB
	}

	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
