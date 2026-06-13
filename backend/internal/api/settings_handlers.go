package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/morphatrix/campmenu/internal/db"
	"github.com/morphatrix/campmenu/internal/models"
	"github.com/morphatrix/campmenu/internal/settings"
	"gorm.io/gorm/clause"
)

// ---- external database configuration ----

// handleGetDBConfig returns the external database pointer (stored in the primary
// DB) and whether the app is currently running on it.
func (s *Server) handleGetDBConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"externalDsn":   db.ExternalDSN(s.Primary),
		"usingExternal": s.UsingExternal,
	})
}

type dbConfigReq struct {
	ExternalDsn string `json:"externalDsn"`
}

// handleUpdateDBConfig validates and persists the external DSN to the PRIMARY
// database. It takes effect on the next restart (the working connection is bound
// at boot); an empty value reverts to the primary database.
func (s *Server) handleUpdateDBConfig(w http.ResponseWriter, r *http.Request) {
	var req dbConfigReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	dsn := strings.TrimSpace(req.ExternalDsn)
	if dsn != "" {
		if err := db.Ping(dsn); err != nil {
			writeError(w, http.StatusBadRequest, "connexion à la base externe impossible : "+err.Error())
			return
		}
	}
	row := models.AppSetting{Key: db.ExternalDSNKey, Value: dsn, UpdatedAt: time.Now()}
	if err := s.Primary.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&row).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "enregistrement impossible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "restartRequired": true})
}

// ---- SMTP test ----

// handleSendTestEmail sends a test message to the requesting admin's own address
// using the live SMTP settings. The delivery outcome is returned as 200 + an
// {ok,error} body on purpose: a real SMTP failure must reach the browser as
// JSON, but a 5xx would be swapped for an HTML error page by the ingress.
func (s *Server) handleSendTestEmail(w http.ResponseWriter, r *http.Request) {
	if s.Settings.Get(settings.KeySMTPHost) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "SMTP non configuré (renseignez l'hôte SMTP puis enregistrez)"})
		return
	}
	var user models.User
	if err := s.DB.First(&user, "id = ?", userIDFrom(r)).Error; err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "utilisateur introuvable"})
		return
	}
	siteName := s.Settings.Get(settings.KeySiteName)
	subject := fmt.Sprintf("[%s] Email de test", siteName)
	body := fmt.Sprintf("Ceci est un email de test envoyé depuis %s.\n\nSi vous le recevez, la configuration SMTP fonctionne correctement.\n", siteName)
	if err := s.Mailer.Send(user.Email, subject, body); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "to": user.Email})
}
