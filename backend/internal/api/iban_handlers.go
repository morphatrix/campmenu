package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm/clause"
)

// ---- IBAN visibility helpers ----

// grantedOwners returns the set of owner ids that allow `viewer` to see their IBAN.
func (s *Server) grantedOwners(viewer uuid.UUID) map[uuid.UUID]bool {
	var rows []models.IBANGrant
	s.DB.Where("viewer_id = ?", viewer).Find(&rows)
	m := make(map[uuid.UUID]bool, len(rows))
	for _, r := range rows {
		m[r.OwnerID] = true
	}
	return m
}

// redactIBANs blanks the IBAN of any user the viewer isn't allowed to see (and
// marks IBANHidden so the UI can offer to request access).
func (s *Server) redactIBANs(viewer uuid.UUID, users []*models.User) {
	granted := s.grantedOwners(viewer)
	for _, u := range users {
		if u == nil || u.ID == viewer || u.IBAN == "" || u.IBANVisibility == "public" || granted[u.ID] {
			continue
		}
		u.IBAN = ""
		u.IBANHidden = true
	}
}

// ---- own visibility settings ----

type ibanVisibilityReq struct {
	Visibility string      `json:"visibility"`
	ViewerIDs  []uuid.UUID `json:"viewerIds"`
}

func validIBANVisibility(v string) bool {
	return v == "public" || v == "selected" || v == "request"
}

// handleSetIBANVisibility updates the caller's IBAN visibility mode and, for the
// "selected" mode, replaces the explicit grant list.
func (s *Server) handleSetIBANVisibility(w http.ResponseWriter, r *http.Request) {
	var req ibanVisibilityReq
	if err := decode(r, &req); err != nil || !validIBANVisibility(req.Visibility) {
		writeError(w, http.StatusBadRequest, "réglage de visibilité invalide")
		return
	}
	uid := userIDFrom(r)
	s.DB.Model(&models.User{}).Where("id = ?", uid).Update("iban_visibility", req.Visibility)
	if req.Visibility == "selected" && req.ViewerIDs != nil {
		s.DB.Where("owner_id = ?", uid).Delete(&models.IBANGrant{})
		for _, vid := range req.ViewerIDs {
			if vid == uid {
				continue
			}
			s.DB.Create(&models.IBANGrant{OwnerID: uid, ViewerID: vid})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleGetIBANAccess returns the caller's visibility mode and the users they
// currently grant access to.
func (s *Server) handleGetIBANAccess(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	var u models.User
	s.DB.Select("iban_visibility").First(&u, "id = ?", uid)

	var grants []models.IBANGrant
	s.DB.Where("owner_id = ?", uid).Find(&grants)
	ids := make([]uuid.UUID, 0, len(grants))
	for _, g := range grants {
		ids = append(ids, g.ViewerID)
	}
	granted := []models.User{}
	if len(ids) > 0 {
		s.DB.Select("id", "first_name", "last_name", "nickname", "photo_url").Find(&granted, "id IN ?", ids)
	}
	writeJSON(w, http.StatusOK, map[string]any{"visibility": u.IBANVisibility, "granted": granted})
}

// handleRevokeIBANGrant removes a user from the caller's grant list.
func (s *Server) handleRevokeIBANGrant(w http.ResponseWriter, r *http.Request) {
	viewerID, err := uuid.Parse(chi.URLParam(r, "viewerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Where("owner_id = ? AND viewer_id = ?", userIDFrom(r), viewerID).Delete(&models.IBANGrant{})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- requests ----

// handleCreateIBANRequest lets the caller ask an owner to reveal their IBAN.
func (s *Server) handleCreateIBANRequest(w http.ResponseWriter, r *http.Request) {
	ownerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	requester := userIDFrom(r)
	if ownerID == requester {
		writeError(w, http.StatusBadRequest, "requête invalide")
		return
	}
	var owner models.User
	if err := s.DB.First(&owner, "id = ?", ownerID).Error; err != nil {
		writeError(w, http.StatusNotFound, "utilisateur introuvable")
		return
	}
	req := models.IBANRequest{OwnerID: ownerID, RequesterID: requester, Status: "pending"}
	if err := s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "owner_id"}, {Name: "requester_id"}},
		DoUpdates: clause.Assignments(map[string]any{"status": "pending"}),
	}).Create(&req).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "demande impossible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleListIBANRequests returns the caller's pending incoming requests (for the
// notification bell), with the requester's public identity.
func (s *Server) handleListIBANRequests(w http.ResponseWriter, r *http.Request) {
	var reqs []models.IBANRequest
	s.DB.Preload("Requester").
		Where("owner_id = ? AND status = ?", userIDFrom(r), "pending").
		Order("created_at desc").Find(&reqs)
	for i := range reqs {
		if u := reqs[i].Requester; u != nil { // never leak the requester's private fields
			u.IBAN, u.Email = "", ""
			u.Weight, u.BirthDate, u.ShoeSize = nil, nil, nil
		}
	}
	writeJSON(w, http.StatusOK, reqs)
}

func (s *Server) loadOwnedRequest(w http.ResponseWriter, r *http.Request) (*models.IBANRequest, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return nil, false
	}
	var req models.IBANRequest
	if err := s.DB.First(&req, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "demande introuvable")
		return nil, false
	}
	if req.OwnerID != userIDFrom(r) {
		writeError(w, http.StatusForbidden, "non autorisé")
		return nil, false
	}
	return &req, true
}

// handleAcceptIBANRequest grants the requester access and marks the ask accepted.
func (s *Server) handleAcceptIBANRequest(w http.ResponseWriter, r *http.Request) {
	req, ok := s.loadOwnedRequest(w, r)
	if !ok {
		return
	}
	s.DB.Model(req).Update("status", "accepted")
	s.DB.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&models.IBANGrant{OwnerID: req.OwnerID, ViewerID: req.RequesterID})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleDenyIBANRequest marks the ask denied (no grant created).
func (s *Server) handleDenyIBANRequest(w http.ResponseWriter, r *http.Request) {
	req, ok := s.loadOwnedRequest(w, r)
	if !ok {
		return
	}
	s.DB.Model(req).Update("status", "denied")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- directory ----

// handleDirectory lists every user's public identity (no email/IBAN) so any
// member can build a grant list or recognise a participant.
func (s *Server) handleDirectory(w http.ResponseWriter, _ *http.Request) {
	var users []models.User
	s.DB.Select("id", "first_name", "last_name", "nickname", "photo_url").
		Order("first_name asc").Find(&users)
	writeJSON(w, http.StatusOK, users)
}
