package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	SiteURL     string
	JWTSecret   string

	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	MailFromEmail string
	MailFromName  string

	MediaServiceURL   string
	MediaServiceToken string

	LiveKitURL string

	AllowedOrigin string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: mustEnv("DATABASE_URL"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		SiteURL:     getEnv("SITE_URL", "http://localhost:3000"),
		JWTSecret:   mustEnv("JWT_SECRET"),

		SMTPHost:      getEnv("SMTP_HOST", "smtp.resend.com"),
		SMTPPort:      getEnv("SMTP_PORT", "465"),
		SMTPUsername:  getEnv("SMTP_USERNAME", "resend"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		MailFromEmail: getEnv("MAIL_FROM_EMAIL", "noreply@platform.local"),
		MailFromName:  getEnv("MAIL_FROM_NAME", "Platform"),

		MediaServiceURL:   getEnv("MEDIA_SERVICE_URL", "http://localhost:8010"),
		MediaServiceToken: getEnv("MEDIA_SERVICE_TOKEN", ""),

		LiveKitURL: getEnv("LIVEKIT_URL", "ws://localhost:7880"),

		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env variable not set: " + key)
	}
	return v
}
