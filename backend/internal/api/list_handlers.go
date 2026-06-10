package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

func (s *Server) handleListProductLists(w http.ResponseWriter, _ *http.Request) {
	var lists []models.ProductList
	s.DB.
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc, name asc") }).
		Order("name asc").Find(&lists)
	writeJSON(w, http.StatusOK, lists)
}

type productListReq struct {
	Name     string   `json:"name"`
	Voted    *bool    `json:"voted"`
	Sections []string `json:"sections"`
}

func (s *Server) handleCreateProductList(w http.ResponseWriter, r *http.Request) {
	var req productListReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	voted := true
	if req.Voted != nil {
		voted = *req.Voted
	}
	list := models.ProductList{Name: strings.TrimSpace(req.Name), Voted: voted, Sections: req.Sections}
	if err := s.DB.Create(&list).Error; err != nil {
		writeError(w, http.StatusConflict, "une liste porte déjà ce nom")
		return
	}
	if !voted { // GORM omits false when the column has a default
		s.DB.Model(&models.ProductList{}).Where("id = ?", list.ID).Update("voted", false)
	}
	writeJSON(w, http.StatusCreated, list)
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
	updates := map[string]any{}
	if strings.TrimSpace(req.Name) != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Voted != nil {
		updates["voted"] = *req.Voted
	}
	if req.Sections != nil {
		updates["sections"] = models.JSONStrings(req.Sections)
	}
	if len(updates) > 0 {
		if err := s.DB.Model(&models.ProductList{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			writeError(w, http.StatusConflict, "une liste porte déjà ce nom")
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
