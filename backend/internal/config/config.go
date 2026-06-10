package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every externalized parameter. Nothing is hard-coded elsewhere.
type Config struct {
	AppURL   string // public URL of the frontend, used in invitation/confirmation links
	Port     string
	LogLevel string

	DatabaseDSN string

	JWTSecret string
	JWTExpiry time.Duration

	BcryptCost int

	// SMTP. If SMTPHost is empty, emails are logged to stdout instead of sent.
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// When false, accounts are auto-confirmed (handy for local/dev).
	EmailConfirmRequired bool

	// Branding / graphical customization (all overridable via env).
	SiteName       string
	LogoURL        string
	DefaultTheme   string
	DefaultPalette string

	CORSOrigins []string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// Load reads configuration from the environment, applying safe defaults.
func Load() *Config {
	c := &Config{
		AppURL:               getenv("APP_URL", "http://localhost:5173"),
		Port:                 getenv("PORT", "8080"),
		LogLevel:             getenv("LOG_LEVEL", "info"),
		DatabaseDSN:          getenv("DATABASE_DSN", "host=localhost user=campmenu password=campmenu dbname=campmenu port=5432 sslmode=disable TimeZone=UTC"),
		JWTSecret:            getenv("JWT_SECRET", "change-me-in-production"),
		JWTExpiry:            time.Duration(getenvInt("JWT_EXPIRY_HOURS", 168)) * time.Hour,
		BcryptCost:           getenvInt("BCRYPT_COST", 12),
		SMTPHost:             getenv("SMTP_HOST", ""),
		SMTPPort:             getenvInt("SMTP_PORT", 587),
		SMTPUser:             getenv("SMTP_USER", ""),
		SMTPPass:             getenv("SMTP_PASS", ""),
		SMTPFrom:             getenv("SMTP_FROM", "no-reply@campmenu.local"),
		EmailConfirmRequired: getenvBool("EMAIL_CONFIRM_REQUIRED", true),
		SiteName:             getenv("SITE_NAME", "CampMenu"),
		LogoURL:              getenv("LOGO_URL", ""),
		DefaultTheme:         getenv("DEFAULT_THEME", "auto"),
		DefaultPalette:       getenv("DEFAULT_PALETTE", "default"),
		CORSOrigins:          splitAndTrim(getenv("CORS_ORIGINS", "http://localhost:5173")),
	}
	return c
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
