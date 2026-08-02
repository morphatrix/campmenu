package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
)

// apiTokenPrefix marks a raw API token (vs. a signed JWT) so requireAuth can
// tell the two apart without trying to parse one as the other.
const apiTokenPrefix = "cmk_"

func hashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// lookupAPIToken resolves a raw bearer token to (userID, role), or ok=false
// if it doesn't match any live token. Also best-effort bumps LastUsedAt.
func (s *Server) lookupAPIToken(token string) (uuid.UUID, models.Role, bool) {
	var at models.ApiToken
	if err := s.DB.Where("token_hash = ?", hashAPIToken(token)).First(&at).Error; err != nil {
		return uuid.Nil, "", false
	}
	var u models.User
	if err := s.DB.Select("id, role").First(&u, "id = ?", at.UserID).Error; err != nil {
		return uuid.Nil, "", false
	}
	now := time.Now()
	s.DB.Model(&at).Update("last_used_at", now)
	return u.ID, u.Role, true
}

type apiTokenResp struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

func (s *Server) handleListAPITokens(w http.ResponseWriter, r *http.Request) {
	var tokens []models.ApiToken
	s.DB.Where("user_id = ?", userIDFrom(r)).Order("created_at desc").Find(&tokens)
	out := make([]apiTokenResp, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, apiTokenResp{ID: t.ID.String(), Label: t.Label, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

type createAPITokenReq struct {
	Label string `json:"label"`
}

func (s *Server) handleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	var req createAPITokenReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if strings.TrimSpace(req.Label) == "" {
		writeError(w, http.StatusBadRequest, "libellé requis")
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		writeError(w, http.StatusInternalServerError, "génération du token impossible")
		return
	}
	plaintext := apiTokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	at := models.ApiToken{UserID: userIDFrom(r), Label: req.Label, TokenHash: hashAPIToken(plaintext)}
	if err := s.DB.Create(&at).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "création du token impossible")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"id": at.ID.String(), "label": at.Label, "token": plaintext,
	})
}

func (s *Server) handleRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if err := s.DB.Where("id = ? AND user_id = ?", id, userIDFrom(r)).Delete(&models.ApiToken{}).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "révocation impossible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
