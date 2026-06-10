package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
)

const maxRecipesPerMeal = 3

// canAccessMeal checks the requester may act on the meal's parent event.
func (s *Server) canAccessMeal(r *http.Request, mealID uuid.UUID) bool {
	var meal models.Meal
	if err := s.DB.Select("event_id").First(&meal, "id = ?", mealID).Error; err != nil {
		return false
	}
	return s.canAccessEvent(r, meal.EventID)
}

func (s *Server) handleListMeals(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var meals []models.Meal
	s.DB.Preload("Recipes.Recipe").Preload("RawItems").
		Where("event_id = ?", eventID).
		Order("day_index asc, type asc").Find(&meals)
	writeJSON(w, http.StatusOK, meals)
}

type updateMealReq struct {
	Variant          *string `json:"variant"`
	ParticipantCount *int    `json:"participantCount"`
}

func (s *Server) handleUpdateMeal(w http.ResponseWriter, r *http.Request) {
	mealID, err := uuid.Parse(chi.URLParam(r, "mealID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessMeal(r, mealID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var req updateMealReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if req.Variant != nil {
		updates["variant"] = *req.Variant
	}
	if req.ParticipantCount != nil {
		updates["participant_count"] = *req.ParticipantCount
	}
	if len(updates) > 0 {
		s.DB.Model(&models.Meal{}).Where("id = ?", mealID).Updates(updates)
	}
	var meal models.Meal
	s.DB.Preload("Recipes.Recipe").Preload("RawItems").First(&meal, "id = ?", mealID)
	writeJSON(w, http.StatusOK, meal)
}

type addMealRecipeReq struct {
	RecipeID         uuid.UUID `json:"recipeId"`
	ParticipantCount int       `json:"participantCount"`
}

// handleAddMealRecipe drops a recipe onto a slot (1-3 recipes per slot).
func (s *Server) handleAddMealRecipe(w http.ResponseWriter, r *http.Request) {
	mealID, err := uuid.Parse(chi.URLParam(r, "mealID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessMeal(r, mealID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var req addMealRecipeReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	var count int64
	s.DB.Model(&models.MealRecipe{}).Where("meal_id = ?", mealID).Count(&count)
	if count >= maxRecipesPerMeal {
		writeError(w, http.StatusConflict, "maximum 3 recettes par repas")
		return
	}
	mr := models.MealRecipe{
		MealID: mealID, RecipeID: req.RecipeID,
		ParticipantCount: req.ParticipantCount, Position: int(count),
	}
	if err := s.DB.Create(&mr).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "ajout impossible")
		return
	}
	s.DB.Preload("Recipe").First(&mr, "id = ?", mr.ID)
	writeJSON(w, http.StatusCreated, mr)
}

type updateMealRecipeReq struct {
	ParticipantCount *int `json:"participantCount"`
	Position         *int `json:"position"`
}

func (s *Server) handleUpdateMealRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req updateMealRecipeReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if req.ParticipantCount != nil {
		updates["participant_count"] = *req.ParticipantCount
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}
	if len(updates) > 0 {
		s.DB.Model(&models.MealRecipe{}).Where("id = ?", id).Updates(updates)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteMealRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Delete(&models.MealRecipe{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type rawItemReq struct {
	Name         string     `json:"name"`
	Quantity     float64    `json:"quantity"`
	Unit         string     `json:"unit"`
	IngredientID *uuid.UUID `json:"ingredientId"`
}

// handleAddRawItem adds an ad-hoc ingredient line to a meal (recipe-less entry).
func (s *Server) handleAddRawItem(w http.ResponseWriter, r *http.Request) {
	mealID, err := uuid.Parse(chi.URLParam(r, "mealID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessMeal(r, mealID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var req rawItemReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	item := models.MealRawItem{
		MealID: mealID, Name: req.Name, Quantity: req.Quantity,
		Unit: req.Unit, IngredientID: req.IngredientID,
	}
	if err := s.DB.Create(&item).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "ajout impossible")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleDeleteRawItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Delete(&models.MealRawItem{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
