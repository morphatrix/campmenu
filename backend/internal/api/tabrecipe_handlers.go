package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
)

type tabRecipeReq struct {
	RecipeID         uuid.UUID `json:"recipeId"`
	Section          string    `json:"section"`
	ParticipantCount int       `json:"participantCount"`
}

// handleAddTabRecipe attaches a recipe (e.g. a cocktail) to a tab section.
func (s *Server) handleAddTabRecipe(w http.ResponseWriter, r *http.Request) {
	tabID, err := uuid.Parse(chi.URLParam(r, "tabID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req tabRecipeReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	var maxPos int
	s.DB.Model(&models.TabRecipe{}).Where("tab_id = ?", tabID).
		Select("COALESCE(MAX(position),0)").Scan(&maxPos)
	tr := models.TabRecipe{
		TabID: tabID, RecipeID: req.RecipeID, Section: req.Section,
		ParticipantCount: req.ParticipantCount, Position: maxPos + 1,
	}
	if err := s.DB.Create(&tr).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "ajout impossible")
		return
	}
	s.DB.Preload("Recipe").First(&tr, "id = ?", tr.ID)
	writeJSON(w, http.StatusCreated, tr)
}

type updateTabRecipeReq struct {
	Section          *string `json:"section"`
	ParticipantCount *int    `json:"participantCount"`
}

func (s *Server) handleUpdateTabRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req updateTabRecipeReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if req.Section != nil {
		updates["section"] = *req.Section
	}
	if req.ParticipantCount != nil {
		updates["participant_count"] = *req.ParticipantCount
	}
	if len(updates) > 0 {
		s.DB.Model(&models.TabRecipe{}).Where("id = ?", id).Updates(updates)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteTabRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Delete(&models.TabRecipe{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
