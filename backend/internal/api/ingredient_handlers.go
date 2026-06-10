package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/fuzzy"
	"github.com/morphatrix/campmenu/internal/models"
)

// resolveIngredient finds an existing ingredient by canonical name
// (case-insensitive) or creates a new referential entry. This is the single
// place ingredients are created, which is what makes global rename possible.
func (s *Server) resolveIngredient(name, unit string) (*models.Ingredient, error) {
	name = strings.TrimSpace(name)
	var ing models.Ingredient
	err := s.DB.Where("LOWER(canonical_name) = LOWER(?)", name).First(&ing).Error
	if err == nil {
		return &ing, nil
	}
	ing = models.Ingredient{CanonicalName: name, DefaultUnit: unit}
	if err := s.DB.Create(&ing).Error; err != nil {
		return nil, err
	}
	return &ing, nil
}

func (s *Server) handleListIngredients(w http.ResponseWriter, r *http.Request) {
	var ings []models.Ingredient
	s.DB.Order("canonical_name asc").Find(&ings)
	writeJSON(w, http.StatusOK, ings)
}

// handleSuggestIngredients powers the fuzzy autocomplete to avoid duplicates
// (e.g. "beurre salé" -> "beurre demi-sel").
func (s *Server) handleSuggestIngredients(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 8
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	var ings []models.Ingredient
	s.DB.Find(&ings)
	names := make([]string, len(ings))
	byName := make(map[string]models.Ingredient, len(ings))
	for i, ing := range ings {
		names[i] = ing.CanonicalName
		byName[ing.CanonicalName] = ing
	}
	matches := fuzzy.Suggest(q, names, limit, 4)
	out := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		ing := byName[m.Value]
		out = append(out, map[string]any{
			"id":          ing.ID,
			"name":        ing.CanonicalName,
			"defaultUnit": ing.DefaultUnit,
			"distance":    m.Distance,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type createIngredientReq struct {
	CanonicalName string `json:"canonicalName"`
	DefaultUnit   string `json:"defaultUnit"`
}

func (s *Server) handleCreateIngredient(w http.ResponseWriter, r *http.Request) {
	var req createIngredientReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if strings.TrimSpace(req.CanonicalName) == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	ing, err := s.resolveIngredient(req.CanonicalName, req.DefaultUnit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "création impossible")
		return
	}
	writeJSON(w, http.StatusCreated, ing)
}

type renameIngredientReq struct {
	CanonicalName string `json:"canonicalName"`
	DefaultUnit   string `json:"defaultUnit"`
}

// handleRenameIngredient renames the referential entry. Because recipes and
// articles reference the ingredient by id, the new name propagates everywhere
// automatically (admin only).
func (s *Server) handleRenameIngredient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req renameIngredientReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if strings.TrimSpace(req.CanonicalName) != "" {
		updates["canonical_name"] = strings.TrimSpace(req.CanonicalName)
	}
	if req.DefaultUnit != "" {
		updates["default_unit"] = req.DefaultUnit
	}
	if err := s.DB.Model(&models.Ingredient{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		writeError(w, http.StatusConflict, "ce nom existe déjà")
		return
	}
	var ing models.Ingredient
	s.DB.First(&ing, "id = ?", id)
	writeJSON(w, http.StatusOK, ing)
}

// handleListUnits returns the canonical units referential.
func (s *Server) handleListUnits(w http.ResponseWriter, r *http.Request) {
	var units []models.Unit
	s.DB.Order("name asc").Find(&units)
	writeJSON(w, http.StatusOK, units)
}
