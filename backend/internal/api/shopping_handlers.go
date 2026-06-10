package api

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
)

// shoppingLine is one consolidated row of the shopping list.
type shoppingLine struct {
	Section      string     `json:"section"`
	Name         string     `json:"name"`
	Unit         string     `json:"unit"`
	Quantity     float64    `json:"quantity"`
	IngredientID *uuid.UUID `json:"ingredientId"`
	Source       string     `json:"source"`
	Observation  string     `json:"observation"`
	Bought       bool       `json:"bought"`
	BroughtBy    *uuid.UUID `json:"broughtBy"`
}

func lineKey(section, name, unit string) string {
	return strings.ToLower(strings.TrimSpace(section)) + "|" +
		strings.ToLower(strings.TrimSpace(name)) + "|" +
		strings.ToLower(strings.TrimSpace(unit))
}

// computeShoppingList aggregates planned recipes, raw meal items and every tab
// (voted matrix, organizer-set totals, attached cocktails) into deduplicated
// lines grouped by section, then merges the stored manual overrides.
func (s *Server) computeShoppingList(eventID uuid.UUID) []shoppingLine {
	var event models.Event
	if err := s.DB.First(&event, "id = ?", eventID).Error; err != nil {
		return nil
	}
	days := int(event.EndDate.Sub(event.StartDate).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	effective := s.effectiveParticipantCount(eventID, event.InitialParticipants)

	agg := map[string]*shoppingLine{}
	add := func(section, name, unit string, ingredientID *uuid.UUID, qty float64) {
		if strings.TrimSpace(name) == "" || qty == 0 {
			return
		}
		k := lineKey(section, name, unit)
		if l, ok := agg[k]; ok {
			l.Quantity += qty
			if l.IngredientID == nil {
				l.IngredientID = ingredientID
			}
			return
		}
		agg[k] = &shoppingLine{Section: strings.TrimSpace(section), Name: strings.TrimSpace(name), Unit: unit, Quantity: qty, IngredientID: ingredientID}
	}

	// addRecipe expands a recipe's ingredients scaled to a serving count.
	addRecipe := func(section string, recipe *models.Recipe, count int) {
		if recipe == nil || count <= 0 {
			return
		}
		bp := recipe.BasePersons
		if bp <= 0 {
			bp = 1
		}
		coef := recipe.Coefficient
		if coef <= 0 {
			coef = 1
		}
		for _, ri := range recipe.Ingredients {
			ingName := ""
			if ri.Ingredient != nil {
				ingName = ri.Ingredient.CanonicalName
			}
			id := ri.IngredientID
			add(section, ingName, ri.Unit, &id, ri.Quantity/float64(bp)*coef*float64(count))
		}
	}

	// 1. Planned meals: recipes (scaled) + raw items (absolute).
	var meals []models.Meal
	s.DB.Preload("Recipes.Recipe.Ingredients.Ingredient").Preload("RawItems").
		Where("event_id = ?", eventID).Find(&meals)
	for _, meal := range meals {
		base := effective
		if meal.ParticipantCount != nil {
			base = *meal.ParticipantCount
		}
		for _, mr := range meal.Recipes {
			weight := mr.ParticipantCount
			if weight <= 0 {
				weight = base
			}
			addRecipe("", mr.Recipe, weight)
		}
		for _, raw := range meal.RawItems {
			add("", raw.Name, raw.Unit, raw.IngredientID, raw.Quantity)
		}
	}

	// 2. Tabs: voted (participant consumption × days) or non-voted (organizer
	//    totals), plus attached recipes (cocktails), grouped by article section.
	var tabs []models.EventTab
	s.DB.Preload("Articles").Preload("Recipes.Recipe.Ingredients.Ingredient").
		Where("event_id = ? AND kind = ?", eventID, models.TabMatrix).Find(&tabs)
	for _, tab := range tabs {
		if tab.Voted {
			var cons []models.TabConsumption
			s.DB.Where("tab_id = ?", tab.ID).Find(&cons)
			levelByArticle := map[uuid.UUID][]int{}
			for _, c := range cons {
				levelByArticle[c.ArticleID] = append(levelByArticle[c.ArticleID], c.Level)
			}
			for _, art := range tab.Articles {
				total := 0.0
				for _, lvl := range levelByArticle[art.ID] {
					if lvl > 0 {
						if q, ok := art.QtyPerLevel[strconv.Itoa(lvl)]; ok {
							total += q
						}
					}
				}
				add(art.Section, art.Name, art.Unit, art.IngredientID, total*float64(days))
			}
		} else {
			for _, art := range tab.Articles {
				add(art.Section, art.Name, art.Unit, art.IngredientID, art.Quantity)
			}
		}
		for _, tr := range tab.Recipes {
			count := tr.ParticipantCount
			if count <= 0 {
				count = effective
			}
			addRecipe(tr.Section, tr.Recipe, count)
		}
	}

	// 3. Merge persisted overrides (source, observation, bought, broughtBy).
	var entries []models.ShoppingEntry
	s.DB.Where("event_id = ?", eventID).Find(&entries)
	for _, e := range entries {
		k := lineKey(e.Section, e.Name, e.Unit)
		l, ok := agg[k]
		if !ok {
			l = &shoppingLine{Section: e.Section, Name: e.Name, Unit: e.Unit}
			agg[k] = l
		}
		l.Source = e.Source
		l.Observation = e.Observation
		l.Bought = e.Bought
		l.BroughtBy = e.BroughtBy
		if l.IngredientID == nil {
			l.IngredientID = e.IngredientID
		}
	}

	out := make([]shoppingLine, 0, len(agg))
	for _, l := range agg {
		l.Quantity = math.Round(l.Quantity*100) / 100
		out = append(out, *l)
	}
	return out
}

func (s *Server) handleGetShoppingList(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	writeJSON(w, http.StatusOK, s.computeShoppingList(eventID))
}

type updateShoppingReq struct {
	Section        string     `json:"section"`
	Name           string     `json:"name"`
	Unit           string     `json:"unit"`
	IngredientID   *uuid.UUID `json:"ingredientId"`
	Source         *string    `json:"source"`
	Observation    *string    `json:"observation"`
	Bought         *bool      `json:"bought"`
	BroughtBy      *uuid.UUID `json:"broughtBy"`
	ClearBroughtBy bool       `json:"clearBroughtBy"`
}

// handleUpdateShoppingLine upserts the manual metadata for a consolidated line,
// keyed by (event, section, name, unit). Quantities stay computed.
func (s *Server) handleUpdateShoppingLine(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var req updateShoppingReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "nom requis")
		return
	}
	var entry models.ShoppingEntry
	s.DB.Where("event_id = ? AND section = ? AND LOWER(name) = LOWER(?) AND unit = ?", eventID, req.Section, req.Name, req.Unit).
		FirstOrCreate(&entry, models.ShoppingEntry{
			EventID: eventID, Section: req.Section, Name: strings.TrimSpace(req.Name), Unit: req.Unit, IngredientID: req.IngredientID,
		})
	updates := map[string]any{}
	if req.Source != nil {
		updates["source"] = *req.Source
	}
	if req.Observation != nil {
		updates["observation"] = *req.Observation
	}
	if req.Bought != nil {
		updates["bought"] = *req.Bought
	}
	if req.ClearBroughtBy {
		updates["brought_by"] = nil
	} else if req.BroughtBy != nil {
		updates["brought_by"] = *req.BroughtBy
	}
	if len(updates) > 0 {
		s.DB.Model(&entry).Updates(updates)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
