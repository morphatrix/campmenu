package db

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/morphatrix/campmenu/internal/models"
	"github.com/morphatrix/campmenu/internal/secrets"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
)

// ExternalDSNKey is the app_settings key (stored in the PRIMARY database) that
// points at an optional external database. When set, the app uses that database
// for all data; the primary only keeps this pointer.
const ExternalDSNKey = "EXTERNAL_DATABASE_DSN"

func newLogger() gormlogger.Interface {
	// ErrRecordNotFound is part of normal flow (existence checks before insert),
	// so ignore it to avoid noisy logs; keep real warnings/errors.
	return gormlogger.New(log.New(os.Stdout, "", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
}

// connect opens a gorm connection, retrying up to `attempts` times with a
// linear backoff so a not-yet-ready database doesn't crash the boot.
func connect(dsn string, attempts int) (*gorm.DB, error) {
	gl := newLogger()
	var gdb *gorm.DB
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		gdb, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gl})
		if err == nil {
			return gdb, nil
		}
		if attempt < attempts {
			slog.Warn("database not ready, retrying", "attempt", attempt, "error", err)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, err
}

// migrate runs AutoMigrate plus the idempotent schema fixes AutoMigrate can't
// express. Safe to run on any (primary or external) database.
func migrate(gdb *gorm.DB) error {
	if err := gdb.AutoMigrate(models.AllModels()...); err != nil {
		return err
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
	return nil
}

// Open connects to PostgreSQL with retry and runs AutoMigrate. Used for the
// primary database at boot.
func Open(dsn string) (*gorm.DB, error) {
	gdb, err := connect(dsn, 10)
	if err != nil {
		return nil, err
	}
	if err := migrate(gdb); err != nil {
		return nil, err
	}
	return gdb, nil
}

// OpenOnce connects in a single attempt and migrates. Used for an admin-supplied
// external database at boot, where we prefer to fail fast and fall back to the
// primary rather than block startup for a misconfigured host.
func OpenOnce(dsn string) (*gorm.DB, error) {
	gdb, err := connect(dsn, 1)
	if err != nil {
		return nil, err
	}
	if err := migrate(gdb); err != nil {
		return nil, err
	}
	return gdb, nil
}

// Ping verifies a DSN is reachable without migrating anything, so the admin UI
// can validate an external DSN before saving it.
func Ping(dsn string) error {
	gdb, err := connect(dsn, 1)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return sqlDB.Ping()
}

// ExternalDSN returns the configured external database DSN stored in the primary
// database (decrypted), or "" when none is set (the app then stays on primary).
func ExternalDSN(primary *gorm.DB, cipher *secrets.Cipher) string {
	var row models.AppSetting
	if err := primary.Where("key = ?", ExternalDSNKey).First(&row).Error; err != nil {
		return ""
	}
	return cipher.Decrypt(row.Value)
}

// ExternalDSNEncrypted reports whether the stored external DSN is already
// ciphertext (used to migrate a legacy plaintext value on boot).
func ExternalDSNEncrypted(primary *gorm.DB) bool {
	var row models.AppSetting
	if err := primary.Where("key = ?", ExternalDSNKey).First(&row).Error; err != nil {
		return false
	}
	return secrets.IsEncrypted(row.Value)
}

// SetExternalDSN persists the external DSN (encrypted at rest) to the primary
// database. An empty dsn clears the pointer (revert to the primary database).
func SetExternalDSN(primary *gorm.DB, cipher *secrets.Cipher, dsn string) error {
	row := models.AppSetting{Key: ExternalDSNKey, Value: cipher.Encrypt(dsn), UpdatedAt: time.Now()}
	return primary.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&row).Error
}
