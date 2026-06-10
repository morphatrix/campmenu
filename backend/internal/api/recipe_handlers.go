package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

type recipeIngredientReq struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type recipeReq struct {
	Name         string                `json:"name"`
	BasePersons  int                   `json:"basePersons"`
	Coefficient  float64               `json:"coefficient"`
	PhotoURL     string                `json:"photoUrl"`
	Instructions string                `json:"instructions"`
	Kind         string                `json:"kind"`
	Tags         []string              `json:"tags"`
	Ingredients  []recipeIngredientReq `json:"ingredients"`
}

// deriveKind keeps the legacy kind column consistent with tags.
func deriveKind(tags []string, fallback string) string {
	for _, t := range tags {
		if strings.EqualFold(t, "cocktail") {
			return "cocktail"
		}
	}
	if len(tags) > 0 {
		return tags[0]
	}
	return fallback
}

// handleListRecipes returns the shared library: approved recipes for everyone,
// plus the caller's own pending recipes.
func (s *Server) handleListRecipes(w http.ResponseWriter, r *http.Request) {
	var recipes []models.Recipe
	q := s.DB.Preload("Ingredients.Ingredient").Order("name asc")
	if !isStaff(r) {
		q = q.Where("approved = ? OR created_by = ?", true, userIDFrom(r))
	}
	q.Find(&recipes)
	writeJSON(w, http.StatusOK, recipes)
}

func (s *Server) handleListPendingRecipes(w http.ResponseWriter, r *http.Request) {
	var recipes []models.Recipe
	s.DB.Preload("Ingredients.Ingredient").Where("approved = ?", false).
		Order("created_at asc").Find(&recipes)
	writeJSON(w, http.StatusOK, recipes)
}

func (s *Server) handleGetRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var recipe models.Recipe
	if err := s.DB.Preload("Ingredients.Ingredient").First(&recipe, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "recette introuvable")
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

// buildRecipeIngredients resolves each ingredient name against the referential.
func (s *Server) buildRecipeIngredients(tx *gorm.DB, recipeID uuid.UUID, items []recipeIngredientReq) ([]models.RecipeIngredient, error) {
	out := make([]models.RecipeIngredient, 0, len(items))
	for _, it := range items {
		if strings.TrimSpace(it.Name) == "" {
			continue
		}
		var ing models.Ingredient
		err := tx.Where("LOWER(canonical_name) = LOWER(?)", strings.TrimSpace(it.Name)).First(&ing).Error
		if err != nil {
			ing = models.Ingredient{CanonicalName: strings.TrimSpace(it.Name), DefaultUnit: it.Unit}
			if err := tx.Create(&ing).Error; err != nil {
				return nil, err
			}
		}
		out = append(out, models.RecipeIngredient{
			RecipeID: recipeID, IngredientID: ing.ID, Quantity: it.Quantity, Unit: it.Unit,
		})
	}
	return out, nil
}

func (s *Server) handleCreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req recipeReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	if req.BasePersons <= 0 {
		req.BasePersons = 1
	}
	if req.Coefficient <= 0 {
		req.Coefficient = 1
	}
	recipe := models.Recipe{
		Name: req.Name, BasePersons: req.BasePersons, Coefficient: req.Coefficient,
		PhotoURL: req.PhotoURL, Instructions: req.Instructions,
		Kind: deriveKind(req.Tags, req.Kind), Tags: req.Tags,
		CreatedBy: userIDFrom(r),
		Approved:  isStaff(r), // staff-authored recipes are auto-approved
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&recipe).Error; err != nil {
			return err
		}
		ris, err := s.buildRecipeIngredients(tx, recipe.ID, req.Ingredients)
		if err != nil {
			return err
		}
		if len(ris) > 0 {
			return tx.Create(&ris).Error
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "création de la recette impossible")
		return
	}
	s.DB.Preload("Ingredients.Ingredient").First(&recipe, "id = ?", recipe.ID)
	writeJSON(w, http.StatusCreated, recipe)
}

func (s *Server) handleUpdateRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var recipe models.Recipe
	if err := s.DB.First(&recipe, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "recette introuvable")
		return
	}
	if recipe.CreatedBy != userIDFrom(r) && !isStaff(r) {
		writeError(w, http.StatusForbidden, "modification non autorisée")
		return
	}
	var req recipeReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"name": req.Name, "base_persons": req.BasePersons, "coefficient": req.Coefficient,
			"photo_url": req.PhotoURL, "instructions": req.Instructions,
			"kind": deriveKind(req.Tags, req.Kind), "tags": models.JSONStrings(req.Tags),
		}
		if err := tx.Model(&models.Recipe{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		if req.Ingredients != nil {
			if err := tx.Where("recipe_id = ?", id).Delete(&models.RecipeIngredient{}).Error; err != nil {
				return err
			}
			ris, err := s.buildRecipeIngredients(tx, id, req.Ingredients)
			if err != nil {
				return err
			}
			if len(ris) > 0 {
				return tx.Create(&ris).Error
			}
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mise à jour impossible")
		return
	}
	s.DB.Preload("Ingredients.Ingredient").First(&recipe, "id = ?", id)
	writeJSON(w, http.StatusOK, recipe)
}

func (s *Server) handleApproveRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Model(&models.Recipe{}).Where("id = ?", id).Update("approved", true)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var recipe models.Recipe
	if err := s.DB.First(&recipe, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "recette introuvable")
		return
	}
	if recipe.CreatedBy != userIDFrom(r) && !isStaff(r) {
		writeError(w, http.StatusForbidden, "suppression non autorisée")
		return
	}
	s.DB.Delete(&models.Recipe{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
