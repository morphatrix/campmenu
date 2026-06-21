package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/morphatrix/campmenu/internal/ai"
	"github.com/morphatrix/campmenu/internal/db"
	"github.com/morphatrix/campmenu/internal/models"
	"github.com/morphatrix/campmenu/internal/settings"
)

// ---- AI test ----

type aiTestReq struct {
	Prompt string `json:"prompt"`
}

// handleTestAI sends a free-form prompt to the configured AI (saved settings)
// and returns its raw reply, so the admin can validate provider/key/model.
func (s *Server) handleTestAI(w http.ResponseWriter, r *http.Request) {
	var req aiTestReq
	if err := decode(r, &req); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "corps de requête invalide"})
		return
	}
	cfg := s.aiConfig()
	if !cfg.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "IA non configurée (enregistrez d'abord)"})
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "Bonjour, réponds en une phrase pour confirmer que tu fonctionnes."
	}
	resp, err := ai.Complete(r.Context(), cfg, prompt)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "response": resp})
}

const dsnPasswordMask = "••••••••"

// maskDSNPassword replaces the password value in a key=value DSN with bullets so
// it is never sent back to the browser.
func maskDSNPassword(dsn string) string {
	fields := strings.Fields(dsn)
	for i, f := range fields {
		if strings.HasPrefix(f, "password=") && len(f) > len("password=") {
			fields[i] = "password=" + dsnPasswordMask
		}
	}
	return strings.Join(fields, " ")
}

// mergeDSNPassword restores the stored password when the submitted DSN still
// carries the bullet mask (i.e. the admin didn't retype it).
func mergeDSNPassword(incoming, stored string) string {
	var storedPwd string
	for _, f := range strings.Fields(stored) {
		if strings.HasPrefix(f, "password=") {
			storedPwd = f
			break
		}
	}
	fields := strings.Fields(incoming)
	for i, f := range fields {
		if strings.HasPrefix(f, "password=") {
			val := strings.TrimPrefix(f, "password=")
			if isAllBullets(val) && storedPwd != "" {
				fields[i] = storedPwd
			}
		}
	}
	return strings.Join(fields, " ")
}

func isAllBullets(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '•' {
			return false
		}
	}
	return true
}

// ---- external database configuration ----

// handleGetDBConfig returns the external database pointer (password masked) and
// whether the app is currently running on it.
func (s *Server) handleGetDBConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"externalDsn":   maskDSNPassword(db.ExternalDSN(s.Primary, s.Crypto)),
		"usingExternal": s.UsingExternal,
		"encrypted":     s.Crypto.Enabled(),
	})
}

type dbConfigReq struct {
	ExternalDsn string `json:"externalDsn"`
}

// handleUpdateDBConfig validates and persists the external DSN (encrypted at
// rest) to the PRIMARY database. It takes effect on the next restart; an empty
// value reverts to the primary database. A masked password is preserved.
func (s *Server) handleUpdateDBConfig(w http.ResponseWriter, r *http.Request) {
	var req dbConfigReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	dsn := strings.TrimSpace(req.ExternalDsn)
	if dsn != "" {
		dsn = mergeDSNPassword(dsn, db.ExternalDSN(s.Primary, s.Crypto))
		if err := db.Ping(dsn); err != nil {
			writeError(w, http.StatusBadRequest, "connexion à la base externe impossible : "+err.Error())
			return
		}
	}
	if err := db.SetExternalDSN(s.Primary, s.Crypto, dsn); err != nil {
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
