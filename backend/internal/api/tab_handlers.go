package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

type createTabReq struct {
	Name              string         `json:"name"`
	Icon              string         `json:"icon"`
	Kind              string         `json:"kind"` // MENUS | LOCATIONS | (empty => MATRIX)
	WithRecipes       bool           `json:"withRecipes"`
	Voted             *bool          `json:"voted"`
	ListID            *uuid.UUID     `json:"listId"`
	Sections          []string       `json:"sections"`
	ConsumptionLabels models.JSONMap `json:"consumptionLabels"`
}

// nextTabPosition returns a position just before the (always-last) shopping tab.
func (s *Server) nextTabPosition(eventID uuid.UUID) int {
	var maxPos int
	s.DB.Model(&models.EventTab{}).Where("event_id = ? AND kind <> ?", eventID, models.TabShopping).
		Select("COALESCE(MAX(position),0)").Scan(&maxPos)
	return maxPos + 1
}

// handleCreateTab adds a custom MATRIX tab to an event (admin only).
func (s *Server) handleCreateTab(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req createTabReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}

	// Special single-instance tabs (Menus, Locations) added after creation.
	if req.Kind == string(models.TabMenus) || req.Kind == string(models.TabLocations) {
		kind := models.TabKind(req.Kind)
		var cnt int64
		s.DB.Model(&models.EventTab{}).Where("event_id = ? AND kind = ?", eventID, kind).Count(&cnt)
		if cnt > 0 {
			writeError(w, http.StatusConflict, "cet onglet est déjà présent")
			return
		}
		name, icon := req.Name, req.Icon
		if kind == models.TabMenus {
			if name == "" {
				name = "Menus"
			}
			if icon == "" {
				icon = "utensils"
			}
		} else {
			if name == "" {
				name = "Locations"
			}
			if icon == "" {
				icon = "map-pin"
			}
		}
		pos := s.nextTabPosition(eventID)
		tab := models.EventTab{EventID: eventID, Kind: kind, Name: name, Icon: icon, Position: pos, Removable: true}
		if err := s.DB.Create(&tab).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "création de l'onglet impossible")
			return
		}
		s.DB.Model(&models.EventTab{}).Where("event_id = ? AND kind = ?", eventID, models.TabShopping).Update("position", pos+1)
		writeJSON(w, http.StatusCreated, tab)
		return
	}

	// A tab is created either from a sub-list (name/mode/sections inherited) or
	// with a free name.
	var list *models.ProductList
	if req.ListID != nil {
		var l models.ProductList
		if err := s.DB.Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
			First(&l, "id = ?", *req.ListID).Error; err == nil {
			list = &l
		}
	}
	if req.Name == "" && list != nil {
		req.Name = list.Name
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}

	voted := true
	if req.Voted != nil {
		voted = *req.Voted
	} else if list != nil {
		voted = list.Voted
	}
	var sections models.JSONStrings
	if req.Sections != nil {
		sections = req.Sections
	} else if list != nil {
		sections = list.Sections
	}
	labels := req.ConsumptionLabels
	if len(labels) == 0 {
		labels = defaultConsumptionLabels
	}

	pos := s.nextTabPosition(eventID)
	tab := models.EventTab{
		EventID: eventID, Kind: models.TabMatrix, Name: req.Name, Icon: req.Icon,
		Position: pos, Removable: true, WithRecipes: req.WithRecipes,
		Voted: voted, ListID: req.ListID, Sections: sections, ConsumptionLabels: labels,
	}
	if err := s.DB.Create(&tab).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "création de l'onglet impossible")
		return
	}
	// GORM omits a zero value (false) when the column has a default, so force it.
	if !voted {
		s.DB.Model(&models.EventTab{}).Where("id = ?", tab.ID).Update("voted", false)
	}
	// Non-voted tabs copy the list's items so organizers set totals directly.
	if !voted && list != nil && len(list.Items) > 0 {
		arts := make([]models.TabArticle, 0, len(list.Items))
		for i, it := range list.Items {
			arts = append(arts, models.TabArticle{
				TabID: tab.ID, Name: it.Name, Unit: it.Unit, Section: it.Section,
				Quantity: it.Quantity, QtyPerLevel: it.QtyPerLevel, Position: i,
			})
		}
		s.DB.Create(&arts)
	}
	// Keep shopping last.
	s.DB.Model(&models.EventTab{}).Where("event_id = ? AND kind = ?", eventID, models.TabShopping).
		Update("position", pos+1)
	writeJSON(w, http.StatusCreated, tab)
}

type updateTabReq struct {
	Name              *string        `json:"name"`
	Icon              *string        `json:"icon"`
	Voted             *bool          `json:"voted"`
	Sections          []string       `json:"sections"`
	ConsumptionLabels models.JSONMap `json:"consumptionLabels"`
}

func (s *Server) handleUpdateTab(w http.ResponseWriter, r *http.Request) {
	tabID, err := uuid.Parse(chi.URLParam(r, "tabID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req updateTabReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}
	if req.ConsumptionLabels != nil {
		updates["consumption_labels"] = req.ConsumptionLabels
	}
	if req.Voted != nil {
		updates["voted"] = *req.Voted
	}
	if req.Sections != nil {
		updates["sections"] = models.JSONStrings(req.Sections)
	}
	if len(updates) > 0 {
		s.DB.Model(&models.EventTab{}).Where("id = ?", tabID).Updates(updates)
	}
	var tab models.EventTab
	s.DB.First(&tab, "id = ?", tabID)
	writeJSON(w, http.StatusOK, tab)
}

func (s *Server) handleDeleteTab(w http.ResponseWriter, r *http.Request) {
	tabID, err := uuid.Parse(chi.URLParam(r, "tabID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var tab models.EventTab
	if err := s.DB.First(&tab, "id = ?", tabID).Error; err != nil {
		writeError(w, http.StatusNotFound, "onglet introuvable")
		return
	}
	if !tab.Removable {
		writeError(w, http.StatusForbidden, "cet onglet ne peut pas être supprimé")
		return
	}
	s.DB.Where("tab_id = ?", tabID).Delete(&models.TabConsumption{})
	s.DB.Where("tab_id = ?", tabID).Delete(&models.TabArticle{})
	s.DB.Where("tab_id = ?", tabID).Delete(&models.TabRecipe{})
	s.DB.Delete(&models.EventTab{}, "id = ?", tabID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type reorderTabsReq struct {
	Order []uuid.UUID `json:"order"`
}

// handleReorderTabs persists a new drag-and-drop tab order (admin only).
func (s *Server) handleReorderTabs(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req reorderTabsReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	for i, id := range req.Order {
		s.DB.Model(&models.EventTab{}).Where("id = ? AND event_id = ?", id, eventID).
			Update("position", i)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- articles within a matrix tab ----

type articleReq struct {
	Name         string         `json:"name"`
	Unit         string         `json:"unit"`
	Section      string         `json:"section"`
	IngredientID *uuid.UUID     `json:"ingredientId"`
	QtyPerLevel  models.JSONNum `json:"qtyPerLevel"`
	Quantity     float64        `json:"quantity"`
}

func (s *Server) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	tabID, err := uuid.Parse(chi.URLParam(r, "tabID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req articleReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	var maxPos int
	s.DB.Model(&models.TabArticle{}).Where("tab_id = ?", tabID).
		Select("COALESCE(MAX(position),0)").Scan(&maxPos)
	art := models.TabArticle{
		TabID: tabID, Name: req.Name, Unit: req.Unit, Section: req.Section, IngredientID: req.IngredientID,
		QtyPerLevel: req.QtyPerLevel, Quantity: req.Quantity, Position: maxPos + 1,
	}
	if err := s.DB.Create(&art).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "création de l'article impossible")
		return
	}
	writeJSON(w, http.StatusCreated, art)
}

func (s *Server) handleUpdateArticle(w http.ResponseWriter, r *http.Request) {
	articleID, err := uuid.Parse(chi.URLParam(r, "articleID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req articleReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{"name": req.Name, "unit": req.Unit, "section": req.Section, "quantity": req.Quantity}
	if req.QtyPerLevel != nil {
		updates["qty_per_level"] = req.QtyPerLevel
	}
	s.DB.Model(&models.TabArticle{}).Where("id = ?", articleID).Updates(updates)
	var art models.TabArticle
	s.DB.First(&art, "id = ?", articleID)
	writeJSON(w, http.StatusOK, art)
}

func (s *Server) handleDeleteArticle(w http.ResponseWriter, r *http.Request) {
	articleID, err := uuid.Parse(chi.URLParam(r, "articleID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Where("article_id = ?", articleID).Delete(&models.TabConsumption{})
	s.DB.Delete(&models.TabArticle{}, "id = ?", articleID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- consumption (per participant) ----

func (s *Server) handleGetConsumption(w http.ResponseWriter, r *http.Request) {
	tabID, err := uuid.Parse(chi.URLParam(r, "tabID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var rows []models.TabConsumption
	s.DB.Where("tab_id = ?", tabID).Find(&rows)
	writeJSON(w, http.StatusOK, rows)
}

type setConsumptionReq struct {
	Level int `json:"level"`
}

// handleSetConsumption sets the level for the CURRENT user only — a participant
// can never edit another participant's choices.
func (s *Server) handleSetConsumption(w http.ResponseWriter, r *http.Request) {
	tabID, _ := uuid.Parse(chi.URLParam(r, "tabID"))
	articleID, err := uuid.Parse(chi.URLParam(r, "articleID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req setConsumptionReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if req.Level < 0 || req.Level > 3 {
		writeError(w, http.StatusBadRequest, "niveau hors plage (0-3)")
		return
	}
	uid := userIDFrom(r)
	row := models.TabConsumption{TabID: tabID, ArticleID: articleID, UserID: uid}
	s.DB.Where("tab_id = ? AND article_id = ? AND user_id = ?", tabID, articleID, uid).
		FirstOrCreate(&row)
	s.DB.Model(&row).Update("level", req.Level)
	row.Level = req.Level
	writeJSON(w, http.StatusOK, row)
}
