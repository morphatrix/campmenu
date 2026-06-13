package db

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Open connects to PostgreSQL with retry and runs AutoMigrate.
func Open(dsn string) (*gorm.DB, error) {
	// ErrRecordNotFound is part of normal flow (existence checks before insert),
	// so ignore it to avoid noisy logs; keep real warnings/errors.
	gl := gormlogger.New(log.New(os.Stdout, "", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})

	var gdb *gorm.DB
	var err error
	for attempt := 1; attempt <= 10; attempt++ {
		gdb, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gl})
		if err == nil {
			break
		}
		slog.Warn("database not ready, retrying", "attempt", attempt, "error", err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	if err != nil {
		return nil, err
	}

	if err := gdb.AutoMigrate(models.AllModels()...); err != nil {
		return nil, err
	}

	// AutoMigrate does not reliably widen an existing varchar column, so force
	// the role columns to text (idempotent) — otherwise writing "COLLABORATOR"
	// into a legacy varchar(10) fails.
	gdb.Exec(`ALTER TABLE users ALTER COLUMN role TYPE text`)
	gdb.Exec(`ALTER TABLE invites ALTER COLUMN role TYPE text`)
	// Shopping tabs must never be removable (a GORM default:true once let false slip).
	gdb.Exec(`UPDATE event_tabs SET removable = false WHERE kind = 'SHOPPING'`)
	// Backfill recipe tags from the legacy kind column.
	gdb.Exec(`UPDATE recipes SET tags = jsonb_build_array(kind) WHERE (tags IS NULL OR tags = '[]'::jsonb) AND COALESCE(kind,'') <> ''`)
	// product_lists.name is no longer globally unique (now scoped global vs
	// per-event); drop the legacy unique index/constraint if it lingers so two
	// events can each have e.g. a "Petit-déjeuner" list.
	gdb.Exec(`DROP INDEX IF EXISTS idx_product_lists_name`)
	gdb.Exec(`ALTER TABLE product_lists DROP CONSTRAINT IF EXISTS uni_product_lists_name`)

	return gdb, nil
}
