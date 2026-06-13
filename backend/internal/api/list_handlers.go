package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

// handleListProductLists returns the shared catalog (EventID IS NULL). With
// ?eventId=<id> it also includes that event's own private lists, so they can be
// resolved by tabs without leaking into the global catalog or other events.
func (s *Server) handleListProductLists(w http.ResponseWriter, r *http.Request) {
	q := s.DB.
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc, name asc") }).
		Order("name asc")
	if eid, err := uuid.Parse(r.URL.Query().Get("eventId")); err == nil {
		q = q.Where("event_id IS NULL OR event_id = ?", eid)
	} else {
		q = q.Where("event_id IS NULL")
	}
	var lists []models.ProductList
	q.Find(&lists)
	writeJSON(w, http.StatusOK, lists)
}

type productListReq struct {
	Name     string     `json:"name"`
	Voted    *bool      `json:"voted"`
	Sections []string   `json:"sections"`
	EventID  *uuid.UUID `json:"eventId"` // set => list private to that event
}

// nameTakenInScope reports whether a list with the same (case-insensitive) name
// already exists in the same scope (the global catalog, or one event).
func (s *Server) nameTakenInScope(name string, eventID *uuid.UUID, excludeID uuid.UUID) bool {
	q := s.DB.Model(&models.ProductList{}).Where("LOWER(name) = LOWER(?)", name)
	if eventID != nil {
		q = q.Where("event_id = ?", *eventID)
	} else {
		q = q.Where("event_id IS NULL")
	}
	if excludeID != uuid.Nil {
		q = q.Where("id <> ?", excludeID)
	}
	var cnt int64
	q.Count(&cnt)
	return cnt > 0
}

func (s *Server) handleCreateProductList(w http.ResponseWriter, r *http.Request) {
	var req productListReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	if s.nameTakenInScope(name, req.EventID, uuid.Nil) {
		writeError(w, http.StatusConflict, "une liste porte déjà ce nom")
		return
	}
	voted := true
	if req.Voted != nil {
		voted = *req.Voted
	}
	list := models.ProductList{Name: name, Voted: voted, Sections: req.Sections, EventID: req.EventID}
	if err := s.DB.Create(&list).Error; err != nil {
		writeError(w, http.StatusConflict, "une liste porte déjà ce nom")
		return
	}
	if !voted { // GORM omits false when the column has a default
		s.DB.Model(&models.ProductList{}).Where("id = ?", list.ID).Update("voted", false)
	}
	writeJSON(w, http.StatusCreated, list)
}

// handleSaveProductList promotes an event-private list to the shared catalog by
// clearing its EventID, after which it appears on the Lists page like the others.
func (s *Server) handleSaveProductList(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var list models.ProductList
	if err := s.DB.First(&list, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "liste introuvable")
		return
	}
	if list.EventID == nil { // already global
		s.DB.Preload("Items").First(&list, "id = ?", id)
		writeJSON(w, http.StatusOK, list)
		return
	}
	if s.nameTakenInScope(list.Name, nil, id) {
		writeError(w, http.StatusConflict, "une liste du catalogue porte déjà ce nom")
		return
	}
	s.DB.Model(&models.ProductList{}).Where("id = ?", id).Update("event_id", nil)
	s.DB.Preload("Items").First(&list, "id = ?", id)
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleUpdateProductList(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req productListReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	var current models.ProductList
	if err := s.DB.First(&current, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "liste introuvable")
		return
	}
	updates := map[string]any{}
	if name := strings.TrimSpace(req.Name); name != "" && name != current.Name {
		if s.nameTakenInScope(name, current.EventID, id) {
			writeError(w, http.StatusConflict, "une liste porte déjà ce nom")
			return
		}
		updates["name"] = name
	}
	if req.Voted != nil {
		updates["voted"] = *req.Voted
	}
	if req.Sections != nil {
		updates["sections"] = models.JSONStrings(req.Sections)
	}
	if len(updates) > 0 {
		if err := s.DB.Model(&models.ProductList{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			writeError(w, http.StatusConflict, "mise à jour impossible")
			return
		}
	}
	var list models.ProductList
	s.DB.Preload("Items").First(&list, "id = ?", id)
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleDeleteProductList(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	// Detach tabs that referenced this list (keep their copied articles).
	s.DB.Model(&models.EventTab{}).Where("list_id = ?", id).Update("list_id", nil)
	s.DB.Delete(&models.ProductList{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- items ----

type listItemReq struct {
	Name        string         `json:"name"`
	Unit        string         `json:"unit"`
	Section     string         `json:"section"`
	QtyPerLevel models.JSONNum `json:"qtyPerLevel"`
	Quantity    float64        `json:"quantity"`
}

func (s *Server) handleAddListItem(w http.ResponseWriter, r *http.Request) {
	listID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req listItemReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	// Avoid duplicates (case-insensitive) within the same list.
	var existing models.ProductListItem
	if err := s.DB.Where("list_id = ? AND LOWER(name) = LOWER(?)", listID, strings.TrimSpace(req.Name)).
		First(&existing).Error; err == nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	var maxPos int
	s.DB.Model(&models.ProductListItem{}).Where("list_id = ?", listID).
		Select("COALESCE(MAX(position),0)").Scan(&maxPos)
	item := models.ProductListItem{
		ListID: listID, Name: strings.TrimSpace(req.Name), Unit: req.Unit, Section: req.Section,
		QtyPerLevel: req.QtyPerLevel, Quantity: req.Quantity, Position: maxPos + 1,
	}
	if err := s.DB.Create(&item).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "ajout impossible")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateListItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "itemID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req listItemReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{"name": req.Name, "unit": req.Unit, "section": req.Section, "quantity": req.Quantity}
	if req.QtyPerLevel != nil {
		updates["qty_per_level"] = req.QtyPerLevel
	}
	s.DB.Model(&models.ProductListItem{}).Where("id = ?", id).Updates(updates)
	var item models.ProductListItem
	s.DB.First(&item, "id = ?", id)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteListItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "itemID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Delete(&models.ProductListItem{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
