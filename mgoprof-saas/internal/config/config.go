package config

import "os"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	SiteURL     string

	// SMTP
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	MailFromEmail string
	MailFromName  string

	// Auth
	JWTSecret     string
	AllowedOrigin string

	// Admin credentials
	AdminEmail        string
	AdminPasswordHash string
	AdminName         string

	// Optional MaxMind GeoLite2 database files
	GeoDBPath    string // GeoLite2-City.mmdb
	GeoASNDBPath string // GeoLite2-ASN.mmdb
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: mustEnv("DATABASE_URL"),
		SiteURL:     getEnv("SITE_URL", "https://mgoprof.ru"),

		SMTPHost:      getEnv("SMTP_HOST", "smtp.resend.com"),
		SMTPPort:      getEnv("SMTP_PORT", "465"),
		SMTPUsername:  getEnv("SMTP_USERNAME", "resend"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		MailFromEmail: getEnv("MAIL_FROM_EMAIL", "noreply@mgoprof.ru"),
		MailFromName:  getEnv("MAIL_FROM_NAME", "MGOPROF"),

		JWTSecret:     mustEnv("JWT_SECRET"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),

		AdminEmail:        getEnv("ADMIN_EMAIL", ""),
		AdminPasswordHash: getEnv("ADMIN_PASSWORD_HASH", ""),
		AdminName:         getEnv("ADMIN_NAME", "Администратор"),

		GeoDBPath:    getEnv("GEO_DB_PATH", ""),
		GeoASNDBPath: getEnv("GEO_ASN_DB_PATH", ""),
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
