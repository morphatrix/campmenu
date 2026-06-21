package settings

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/morphatrix/campmenu/internal/models"
	"github.com/morphatrix/campmenu/internal/secrets"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// sensitiveKeys are encrypted at rest (decrypted transparently in memory).
var sensitiveKeys = map[string]bool{KeySMTPPass: true, KeyAIAPIKey: true}

// Setting keys (also the env var names used as defaults).
const (
	KeySiteName             = "SITE_NAME"
	KeyLogoURL              = "LOGO_URL"
	KeyDefaultTheme         = "DEFAULT_THEME"
	KeyDefaultPalette       = "DEFAULT_PALETTE"
	KeyAppURL               = "APP_URL"
	KeyCORSOrigins          = "CORS_ORIGINS"
	KeySMTPHost             = "SMTP_HOST"
	KeySMTPPort             = "SMTP_PORT"
	KeySMTPUser             = "SMTP_USER"
	KeySMTPPass             = "SMTP_PASS"
	KeySMTPFrom             = "SMTP_FROM"
	KeyEmailConfirmRequired = "EMAIL_CONFIRM_REQUIRED"
	KeyAIProvider           = "AI_PROVIDER"
	KeyAIBaseURL            = "AI_BASE_URL"
	KeyAIAPIKey             = "AI_API_KEY"
	KeyAIModel              = "AI_MODEL"
)

// EditableKeys are the settings the admin UI may change.
var EditableKeys = []string{
	KeySiteName, KeyLogoURL, KeyDefaultTheme, KeyDefaultPalette,
	KeyAppURL, KeyCORSOrigins,
	KeySMTPHost, KeySMTPPort, KeySMTPUser, KeySMTPPass, KeySMTPFrom,
	KeyEmailConfirmRequired,
	KeyAIProvider, KeyAIBaseURL, KeyAIAPIKey, KeyAIModel,
}

// Store is a thread-safe, DB-backed settings cache. Sensitive values are
// encrypted at rest and decrypted transparently into the in-memory cache.
type Store struct {
	db      *gorm.DB
	cipher  *secrets.Cipher
	mu      sync.RWMutex
	vals    map[string]string // always plaintext in memory
	encInDB map[string]bool   // keys whose stored value was already ciphertext
}

// New loads settings from the DB and seeds any missing key with its env default.
// The cipher encrypts sensitive keys at rest; pass a disabled cipher to keep
// plaintext behaviour.
func New(db *gorm.DB, defaults map[string]string, cipher *secrets.Cipher) (*Store, error) {
	s := &Store{db: db, cipher: cipher, vals: map[string]string{}, encInDB: map[string]bool{}}
	if err := s.reload(); err != nil {
		return nil, err
	}
	missing := map[string]string{}
	s.mu.RLock()
	for k, v := range defaults {
		if _, ok := s.vals[k]; !ok {
			missing[k] = v
		}
	}
	s.mu.RUnlock()
	if len(missing) > 0 {
		if err := s.Set(missing); err != nil {
			return nil, err
		}
	}
	// Migrate any sensitive value still stored in plaintext to ciphertext.
	if cipher.Enabled() {
		migrate := map[string]string{}
		s.mu.RLock()
		for k := range sensitiveKeys {
			if v := s.vals[k]; v != "" && !s.encInDB[k] {
				migrate[k] = v
			}
		}
		s.mu.RUnlock()
		if len(migrate) > 0 {
			if err := s.Set(migrate); err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

func (s *Store) reload() error {
	var rows []models.AppSetting
	if err := s.db.Find(&rows).Error; err != nil {
		return err
	}
	m := make(map[string]string, len(rows))
	enc := make(map[string]bool, len(rows))
	for _, r := range rows {
		enc[r.Key] = secrets.IsEncrypted(r.Value)
		// Decrypt sensitive values so the in-memory cache is plaintext for the app.
		m[r.Key] = s.cipher.Decrypt(r.Value)
	}
	s.mu.Lock()
	s.vals = m
	s.encInDB = enc
	s.mu.Unlock()
	return nil
}

func (s *Store) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vals[key]
}

func (s *Store) Bool(key string) bool {
	b, _ := strconv.ParseBool(s.Get(key))
	return b
}

func (s *Store) Int(key string, def int) int {
	if n, err := strconv.Atoi(s.Get(key)); err == nil {
		return n
	}
	return def
}

// Set upserts the given keys and updates the in-memory cache. Sensitive keys are
// encrypted before they touch the database; the cache keeps plaintext.
func (s *Store) Set(updates map[string]string) error {
	for k, v := range updates {
		stored := v
		if sensitiveKeys[k] {
			stored = s.cipher.Encrypt(v)
		}
		row := models.AppSetting{Key: k, Value: stored, UpdatedAt: time.Now()}
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return err
		}
	}
	s.mu.Lock()
	for k, v := range updates {
		s.vals[k] = v // cache stays plaintext
		if sensitiveKeys[k] {
			s.encInDB[k] = s.cipher.Enabled()
		}
	}
	s.mu.Unlock()
	return nil
}

// AllowedOrigins parses the comma-separated CORS origins list.
func (s *Store) AllowedOrigins() []string {
	parts := strings.Split(s.Get(KeyCORSOrigins), ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// All returns a copy of every setting (admin view).
func (s *Store) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := make(map[string]string, len(s.vals))
	for k, v := range s.vals {
		m[k] = v
	}
	return m
}
