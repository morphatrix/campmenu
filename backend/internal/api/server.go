package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/config"
	"github.com/morphatrix/campmenu/internal/mail"
	"github.com/morphatrix/campmenu/internal/models"
	"github.com/morphatrix/campmenu/internal/secrets"
	"github.com/morphatrix/campmenu/internal/settings"
	"github.com/morphatrix/campmenu/internal/sse"
	"gorm.io/gorm"
)

// Server bundles the dependencies shared by every handler.
type Server struct {
	DB       *gorm.DB // working database (external when configured, else primary)
	Cfg      *config.Config
	Mailer   *mail.Mailer
	Settings *settings.Store
	Hub      *sse.Hub

	// Primary is the env-configured database; it stores the pointer to an
	// optional external database. Equals DB when no external DB is configured.
	Primary *gorm.DB
	// UsingExternal reports whether DB is currently the external database.
	UsingExternal bool
	// Crypto encrypts/decrypts sensitive settings stored at rest.
	Crypto *secrets.Cipher

	// aisleInProgress dedupes in-flight AI aisle classifications by name.
	aisleMu         sync.Mutex
	aisleInProgress map[string]bool
}

func New(db *gorm.DB, cfg *config.Config, mailer *mail.Mailer, st *settings.Store, hub *sse.Hub) *Server {
	return &Server{DB: db, Cfg: cfg, Mailer: mailer, Settings: st, Hub: hub, aisleInProgress: map[string]bool{}}
}

// ---- request context ----

type ctxKey string

const (
	ctxUserID ctxKey = "userID"
	ctxRole   ctxKey = "role"
	ctxImp    ctxKey = "imp"
)

func userIDFrom(r *http.Request) uuid.UUID {
	if v, ok := r.Context().Value(ctxUserID).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

// impFrom returns the impersonating admin id (uuid.Nil if not impersonating).
func impFrom(r *http.Request) uuid.UUID {
	if v, ok := r.Context().Value(ctxImp).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

func roleFrom(r *http.Request) models.Role {
	if v, ok := r.Context().Value(ctxRole).(models.Role); ok {
		return v
	}
	return models.RoleUser
}

// ---- response helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

// decode parses a JSON body into dst, rejecting unknown fields.
func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
