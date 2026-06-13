package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/morphatrix/campmenu/internal/api"
	"github.com/morphatrix/campmenu/internal/config"
	"github.com/morphatrix/campmenu/internal/db"
	"github.com/morphatrix/campmenu/internal/mail"
	"github.com/morphatrix/campmenu/internal/seed"
	"github.com/morphatrix/campmenu/internal/settings"
	"github.com/morphatrix/campmenu/internal/sse"
)

func main() {
	_ = godotenv.Load() // optional .env for local dev

	cfg := config.Load()
	setupLogger(cfg.LogLevel)

	if cfg.JWTSecret == "change-me-in-production" || len(cfg.JWTSecret) < 32 {
		slog.Warn("weak JWT_SECRET — set a strong random value of at least 32 characters in production")
	}

	// Primary database (from env). It always holds the pointer to an optional
	// external database configured later from the admin UI.
	primary, err := db.Open(cfg.DatabaseDSN)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	// If an external database is configured, use it for all data; on failure we
	// log and stay on the primary so the app still boots and the DSN can be fixed.
	database := primary
	usingExternal := false
	if dsn := db.ExternalDSN(primary); dsn != "" {
		if ext, err := db.OpenOnce(dsn); err != nil {
			slog.Error("external database unreachable, using primary", "error", err)
		} else {
			database, usingExternal = ext, true
			slog.Info("using external database")
		}
	}

	seed.Run(database, cfg)

	// Settings store: DB-backed, seeded from env defaults on first run.
	store, err := settings.New(database, map[string]string{
		settings.KeySiteName:             cfg.SiteName,
		settings.KeyLogoURL:              cfg.LogoURL,
		settings.KeyDefaultTheme:         cfg.DefaultTheme,
		settings.KeyDefaultPalette:       cfg.DefaultPalette,
		settings.KeyAppURL:               cfg.AppURL,
		settings.KeyCORSOrigins:          strings.Join(cfg.CORSOrigins, ","),
		settings.KeySMTPHost:             cfg.SMTPHost,
		settings.KeySMTPPort:             strconv.Itoa(cfg.SMTPPort),
		settings.KeySMTPUser:             cfg.SMTPUser,
		settings.KeySMTPPass:             cfg.SMTPPass,
		settings.KeySMTPFrom:             cfg.SMTPFrom,
		settings.KeyEmailConfirmRequired: strconv.FormatBool(cfg.EmailConfirmRequired),
	})
	if err != nil {
		slog.Error("settings init failed", "error", err)
		os.Exit(1)
	}

	srv := api.New(database, cfg, mail.New(store), store, sse.NewHub())
	// The primary handle lets the admin persist the external-DB pointer even when
	// the app is currently running on the external database.
	srv.Primary = primary
	srv.UsingExternal = usingExternal
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "port", cfg.Port, "site", cfg.SiteName)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)
}
