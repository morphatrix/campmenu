package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

// Ad-hoc lists are participant-created, event-private top-up lists used to
// complete the shopping during the event. They are matrix tabs flagged Adhoc
// (never voted, hidden from the desktop tab bar) so their articles flow into the
// shopping list like any other. Unlike regular tabs these endpoints are open to
// every event participant, not just staff.

// adhocTab loads an ad-hoc tab and verifies the requester may access its event.
func (s *Server) adhocTab(w http.ResponseWriter, r *http.Request) (*models.EventTab, bool) {
	tabID, err := uuid.Parse(chi.URLParam(r, "tabID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return nil, false
	}
	var tab models.EventTab
	if err := s.DB.First(&tab, "id = ? AND adhoc = ?", tabID, true).Error; err != nil {
		writeError(w, http.StatusNotFound, "liste introuvable")
		return nil, false
	}
	if !s.canAccessEvent(r, tab.EventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return nil, false
	}
	return &tab, true
}

// adhocItemTab resolves the ad-hoc tab owning an article and checks access.
func (s *Server) adhocItemTab(w http.ResponseWriter, r *http.Request) (*models.TabArticle, bool) {
	articleID, err := uuid.Parse(chi.URLParam(r, "articleID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return nil, false
	}
	var art models.TabArticle
	if err := s.DB.First(&art, "id = ?", articleID).Error; err != nil {
		writeError(w, http.StatusNotFound, "article introuvable")
		return nil, false
	}
	var tab models.EventTab
	if err := s.DB.First(&tab, "id = ? AND adhoc = ?", art.TabID, true).Error; err != nil {
		writeError(w, http.StatusNotFound, "liste introuvable")
		return nil, false
	}
	if !s.canAccessEvent(r, tab.EventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return nil, false
	}
	return &art, true
}

func (s *Server) handleListAdhocLists(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé à cet événement")
		return
	}
	var tabs []models.EventTab
	s.DB.Preload("Articles", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Where("event_id = ? AND adhoc = ?", eventID, true).
		Order("created_at asc").Find(&tabs)
	writeJSON(w, http.StatusOK, tabs)
}

type adhocListReq struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateAdhocList(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé à cet événement")
		return
	}
	var req adhocListReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	tab := models.EventTab{
		EventID: eventID, Kind: models.TabMatrix, Name: name, Icon: "list-plus",
		Position: s.nextTabPosition(eventID), Removable: true, Voted: false, Adhoc: true,
	}
	if err := s.DB.Create(&tab).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "création de la liste impossible")
		return
	}
	// GORM omits a zero value (false) when the column has a default (voted has
	// default:true), so force voted=false — an ad-hoc list is never voted.
	s.DB.Model(&models.EventTab{}).Where("id = ?", tab.ID).Update("voted", false)
	tab.Voted = false
	writeJSON(w, http.StatusCreated, tab)
}

func (s *Server) handleDeleteAdhocList(w http.ResponseWriter, r *http.Request) {
	tab, ok := s.adhocTab(w, r)
	if !ok {
		return
	}
	s.DB.Where("tab_id = ?", tab.ID).Delete(&models.TabArticle{})
	s.DB.Delete(&models.EventTab{}, "id = ?", tab.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type adhocItemReq struct {
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	Quantity float64 `json:"quantity"`
}

func (s *Server) handleAddAdhocItem(w http.ResponseWriter, r *http.Request) {
	tab, ok := s.adhocTab(w, r)
	if !ok {
		return
	}
	var req adhocItemReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	qty := req.Quantity
	if qty <= 0 {
		qty = 1 // a top-up item with no quantity must still reach the shopping list
	}
	var maxPos int
	s.DB.Model(&models.TabArticle{}).Where("tab_id = ?", tab.ID).
		Select("COALESCE(MAX(position),0)").Scan(&maxPos)
	art := models.TabArticle{
		TabID: tab.ID, Name: name, Unit: strings.TrimSpace(req.Unit),
		Quantity: qty, Position: maxPos + 1,
	}
	if err := s.DB.Create(&art).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "ajout impossible")
		return
	}
	writeJSON(w, http.StatusCreated, art)
}

func (s *Server) handleUpdateAdhocItem(w http.ResponseWriter, r *http.Request) {
	art, ok := s.adhocItemTab(w, r)
	if !ok {
		return
	}
	var req adhocItemReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{"name": strings.TrimSpace(req.Name), "unit": strings.TrimSpace(req.Unit), "quantity": req.Quantity}
	s.DB.Model(&models.TabArticle{}).Where("id = ?", art.ID).Updates(updates)
	var fresh models.TabArticle
	s.DB.First(&fresh, "id = ?", art.ID)
	writeJSON(w, http.StatusOK, fresh)
}

func (s *Server) handleDeleteAdhocItem(w http.ResponseWriter, r *http.Request) {
	art, ok := s.adhocItemTab(w, r)
	if !ok {
		return
	}
	s.DB.Delete(&models.TabArticle{}, "id = ?", art.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
