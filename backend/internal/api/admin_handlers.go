package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/auth"
	"github.com/morphatrix/campmenu/internal/models"
	"github.com/morphatrix/campmenu/internal/settings"
)

// ---- site settings ----

func (s *Server) handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Settings.All())
}

// handleUpdateSettings persists only the whitelisted editable keys.
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]string{}
	for _, k := range settings.EditableKeys {
		if v, ok := req[k]; ok {
			updates[k] = v
		}
	}
	if len(updates) > 0 {
		if err := s.Settings.Set(updates); err != nil {
			writeError(w, http.StatusInternalServerError, "enregistrement impossible")
			return
		}
	}
	writeJSON(w, http.StatusOK, s.Settings.All())
}

// ---- admin user management ----

type adminUpdateUserReq struct {
	FirstName *string      `json:"firstName"`
	LastName  *string      `json:"lastName"`
	Nickname  *string      `json:"nickname"`
	IBAN      *string      `json:"iban"`
	Email     *string      `json:"email"`
	Role      *models.Role `json:"role"`
}

func validRole(r models.Role) bool {
	return r == models.RoleAdmin || r == models.RoleCollaborator || r == models.RoleUser
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req adminUpdateUserReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.IBAN != nil {
		updates["iban"] = *req.IBAN
	}
	if req.Email != nil {
		updates["email"] = strings.ToLower(strings.TrimSpace(*req.Email))
	}
	if req.Role != nil && validRole(*req.Role) {
		// Don't let the last admin be demoted into lockout.
		if *req.Role != models.RoleAdmin {
			var current models.User
			s.DB.Select("role").First(&current, "id = ?", id)
			if current.Role == models.RoleAdmin {
				var admins int64
				s.DB.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&admins)
				if admins <= 1 {
					writeError(w, http.StatusConflict, "impossible de rétrograder le dernier administrateur")
					return
				}
			}
		}
		updates["role"] = *req.Role
	}
	if len(updates) > 0 {
		if err := s.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			writeError(w, http.StatusConflict, "mise à jour impossible (email déjà utilisé ?)")
			return
		}
	}
	var user models.User
	s.DB.First(&user, "id = ?", id)
	writeJSON(w, http.StatusOK, user)
}

type resetPasswordReq struct {
	Password string `json:"password"`
}

func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req resetPasswordReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if err := validatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := auth.HashPassword(req.Password, s.Cfg.BcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erreur serveur")
		return
	}
	s.DB.Model(&models.User{}).Where("id = ?", id).Update("password_hash", hash)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handlePromoteCollaborator (staff) promotes a USER to COLLABORATOR.
func (s *Server) handlePromoteCollaborator(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var u models.User
	if err := s.DB.First(&u, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "utilisateur introuvable")
		return
	}
	if u.Role == models.RoleAdmin {
		writeError(w, http.StatusConflict, "déjà administrateur")
		return
	}
	s.DB.Model(&models.User{}).Where("id = ?", id).Update("role", models.RoleCollaborator)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleAdminConfirmUser (admin) manually marks a user's email as confirmed.
func (s *Server) handleAdminConfirmUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Model(&models.User{}).Where("id = ?", id).
		Updates(map[string]any{"email_confirmed": true, "confirmation_token": ""})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if id == userIDFrom(r) {
		writeError(w, http.StatusConflict, "vous ne pouvez pas supprimer votre propre compte")
		return
	}
	s.DB.Where("user_id = ?", id).Delete(&models.EventParticipant{})
	s.DB.Delete(&models.User{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
