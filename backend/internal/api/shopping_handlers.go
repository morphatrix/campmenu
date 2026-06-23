package api

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/ai"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm/clause"
)

// deaccentLower lowercases and strips French accents for unit/name matching.
func deaccentLower(s string) string {
	return strings.NewReplacer(
		"à", "a", "â", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
		"î", "i", "ï", "i", "ô", "o", "ö", "o", "û", "u", "ù", "u", "ü", "u", "ç", "c",
	).Replace(strings.ToLower(s))
}

// liquidWords flags ingredients measured by volume; everything else is weight.
var liquidWords = map[string]bool{
	"huile": true, "huiles": true, "lait": true, "eau": true, "eaux": true,
	"vinaigre": true, "creme": true, "jus": true, "sirop": true, "sirops": true,
	"vin": true, "vins": true, "biere": true, "bieres": true, "sauce": true, "sauces": true,
	"bouillon": true, "coulis": true, "liqueur": true, "rhum": true, "vodka": true,
	"whisky": true, "cognac": true, "alcool": true, "vinaigrette": true, "soja": true,
}

func isLiquid(name string) bool {
	for _, tok := range strings.FieldsFunc(deaccentLower(name), func(r rune) bool { return !unicode.IsLetter(r) }) {
		if liquidWords[tok] {
			return true
		}
	}
	return false
}

// spoonFactor returns the average quantity (grams or ml) for one spoon of the
// given unit, or 0 when the unit isn't a spoon.
func spoonFactor(unit string) float64 {
	u := strings.ReplaceAll(deaccentLower(unit), ".", "")
	u = strings.TrimSpace(u)
	switch {
	case strings.Contains(u, "soupe"), u == "cas", u == "cs", u == "tbsp":
		return 15
	case strings.Contains(u, "cafe"), u == "cac", u == "cc", u == "tsp":
		return 5
	case strings.HasPrefix(u, "cuillere") || u == "cuil": // unspecified spoon → average
		return 12
	}
	return 0
}

// canonicalUnit is normalizeUnit's resulting unit (without a quantity), used to
// key persisted entries onto the same line as the computed quantities.
func canonicalUnit(name, unit string) string {
	u, _ := normalizeUnit(name, unit, 1)
	return u
}

// normalizeUnit converts spoons to a coherent base unit so the same ingredient
// merges: weight (g) for solids, volume (ml) for liquids.
func normalizeUnit(name, unit string, qty float64) (string, float64) {
	f := spoonFactor(unit)
	if f == 0 {
		return unit, qty
	}
	if isLiquid(name) {
		return "ml", qty * f
	}
	return "g", qty * f
}

// shoppingLine is one consolidated row of the shopping list.
type shoppingLine struct {
	Section      string     `json:"section"`
	Name         string     `json:"name"`
	Unit         string     `json:"unit"`
	Quantity       float64    `json:"quantity"`
	IngredientID   *uuid.UUID `json:"ingredientId"`
	Source         string     `json:"source"`
	Observation    string     `json:"observation"`
	Bought         bool       `json:"bought"`         // derived: bought quantity covers the total
	BoughtQuantity float64    `json:"boughtQuantity"` // how much is already bought
	BroughtBy      *uuid.UUID `json:"broughtBy"`
	Aisle          string     `json:"aisle"` // supermarket section (AI-classified, may be empty)
}

// applyAisles fills each line's Aisle from the cache and asynchronously asks the
// AI to classify any name not seen yet (so it's ready next time).
func (s *Server) applyAisles(lines []shoppingLine) {
	if len(lines) == 0 {
		return
	}
	keys := make([]string, 0, len(lines))
	for _, l := range lines {
		keys = append(keys, strings.ToLower(strings.TrimSpace(l.Name)))
	}
	var rows []models.AisleCache
	s.DB.Where("name IN ?", keys).Find(&rows)
	cached := make(map[string]string, len(rows))
	for _, r := range rows {
		cached[r.Name] = r.Aisle
	}
	missing := []string{}
	for i := range lines {
		k := strings.ToLower(strings.TrimSpace(lines[i].Name))
		if a, ok := cached[k]; ok {
			lines[i].Aisle = a
		} else {
			missing = append(missing, lines[i].Name)
		}
	}
	s.queueAisleClassification(missing)
}

// queueAisleClassification classifies the given names in the background (one at a
// time), de-duplicating in-flight names. No-op when AI is not configured.
func (s *Server) queueAisleClassification(names []string) {
	cfg := s.aiConfig()
	if !cfg.Enabled() || len(names) == 0 {
		return
	}
	s.aisleMu.Lock()
	todo := []string{}
	for _, n := range names {
		k := strings.ToLower(strings.TrimSpace(n))
		if k == "" || s.aisleInProgress[k] {
			continue
		}
		s.aisleInProgress[k] = true
		todo = append(todo, n)
	}
	s.aisleMu.Unlock()
	if len(todo) == 0 {
		return
	}
	go func() {
		for _, n := range todo {
			k := strings.ToLower(strings.TrimSpace(n))
			if aisle, err := ai.ClassifyAisle(context.Background(), cfg, n); err == nil && aisle != "" {
				s.DB.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "name"}},
					DoUpdates: clause.AssignmentColumns([]string{"aisle", "updated_at"}),
				}).Create(&models.AisleCache{Name: k, Aisle: aisle, UpdatedAt: time.Now()})
			}
			s.aisleMu.Lock()
			delete(s.aisleInProgress, k)
			s.aisleMu.Unlock()
		}
	}()
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
		// Convert spoons to g/ml so the same ingredient consolidates instead of
		// splitting into separate "cuillère" lines.
		unit, qty = normalizeUnit(name, unit, qty)
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
		// Match the entry to the normalized line (spoon → g/ml) so its metadata
		// attaches to the real line instead of spawning a phantom unit row.
		u := canonicalUnit(e.Name, e.Unit)
		k := lineKey(e.Section, e.Name, u)
		l, ok := agg[k]
		if !ok {
			l = &shoppingLine{Section: e.Section, Name: e.Name, Unit: u}
			agg[k] = l
		}
		l.Source = e.Source
		l.Observation = e.Observation
		l.BoughtQuantity = e.BoughtQuantity
		l.BroughtBy = e.BroughtBy
		if l.IngredientID == nil {
			l.IngredientID = e.IngredientID
		}
	}

	out := make([]shoppingLine, 0, len(agg))
	for _, l := range agg {
		l.Quantity = math.Round(l.Quantity*100) / 100
		if l.Quantity == 0 {
			continue // nothing to buy (e.g. a stale override for a removed item)
		}
		l.BoughtQuantity = math.Round(l.BoughtQuantity*100) / 100
		if l.BoughtQuantity > l.Quantity {
			l.BoughtQuantity = l.Quantity
		}
		// Fully bought only when the bought quantity covers the current total.
		l.Bought = l.Quantity > 0 && l.BoughtQuantity >= l.Quantity
		out = append(out, *l)
	}
	s.applyAisles(out)
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
	BoughtQuantity *float64   `json:"boughtQuantity"`
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
	if req.BoughtQuantity != nil {
		bq := *req.BoughtQuantity
		if bq < 0 {
			bq = 0
		}
		updates["bought_quantity"] = bq
		// Keep the legacy boolean consistent so the migration fallback never fires.
		updates["bought"] = bq > 0
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
