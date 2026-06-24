package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

var defaultConsumptionLabels = models.JSONMap{
	"0": "aucun",
	"1": "1/jour",
	"2": "2/jour",
	"3": "3/jour",
}

// dayMealTypes is the set of meal columns generated per day.
var dayMealTypes = []models.MealType{
	models.MealBreakfast, models.MealLunch, models.MealDinner,
	models.MealAperitif, models.MealDessert,
}

type createEventReq struct {
	Name                string    `json:"name"`
	StartDate           time.Time `json:"startDate"`
	EndDate             time.Time `json:"endDate"`
	InitialParticipants int       `json:"initialParticipants"`
	PhotoURL            string    `json:"photoUrl"`
	IncludeMenus        *bool     `json:"includeMenus"`
	IncludeBreakfast    *bool     `json:"includeBreakfast"`
	IncludeSlopes       *bool     `json:"includeSlopes"`
	IncludeLocations    *bool     `json:"includeLocations"`
}

func boolOr(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

// handleCreateEvent provisions an event with its default tabs and meal grid.
func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if req.Name == "" || req.EndDate.Before(req.StartDate) {
		writeError(w, http.StatusBadRequest, "nom requis et dates cohérentes")
		return
	}

	event := models.Event{
		Name:                req.Name,
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		InitialParticipants: req.InitialParticipants,
		PhotoURL:            req.PhotoURL,
		CreatedBy:           userIDFrom(r),
	}

	days := int(req.EndDate.Sub(req.StartDate).Hours()/24) + 1
	if days < 1 {
		days = 1
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		// Tabs selected by the admin at creation (Shopping is always present).
		pos := 0
		var tabs []models.EventTab
		add := func(t models.EventTab) {
			t.EventID = event.ID
			t.Position = pos
			pos++
			tabs = append(tabs, t)
		}
		if boolOr(req.IncludeMenus, true) {
			add(models.EventTab{Kind: models.TabMenus, Name: "Menus", Icon: "utensils", Removable: true})
		}
		if boolOr(req.IncludeBreakfast, true) {
			add(models.EventTab{Kind: models.TabMatrix, Name: "Petit-déjeuner", Icon: "coffee", Removable: true, ConsumptionLabels: defaultConsumptionLabels})
		}
		if boolOr(req.IncludeSlopes, false) {
			add(models.EventTab{Kind: models.TabMatrix, Name: "Sur les pistes", Icon: "mountain", Removable: true, ConsumptionLabels: defaultConsumptionLabels})
		}
		if boolOr(req.IncludeLocations, false) {
			add(models.EventTab{Kind: models.TabLocations, Name: "Locations", Icon: "map-pin", Removable: true})
		}
		add(models.EventTab{Kind: models.TabShopping, Name: "Liste de courses", Icon: "shopping-cart", Removable: false})
		if err := tx.Create(&tabs).Error; err != nil {
			return err
		}
		// Removable:false is omitted by GORM (default:true), so force it for shopping.
		tx.Model(&models.EventTab{}).Where("event_id = ? AND kind = ?", event.ID, models.TabShopping).
			Update("removable", false)
		// Meal grid: one slot per (day, meal type).
		meals := make([]models.Meal, 0, days*len(dayMealTypes))
		for d := 0; d < days; d++ {
			for _, mt := range dayMealTypes {
				meals = append(meals, models.Meal{EventID: event.ID, DayIndex: d, Type: mt})
			}
		}
		return tx.Create(&meals).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "création de l'événement impossible")
		return
	}
	s.DB.Preload("Tabs").First(&event, "id = ?", event.ID)
	writeJSON(w, http.StatusCreated, event)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	var events []models.Event
	q := s.DB.Order("start_date desc")
	if !isStaff(r) {
		q = q.Where("id IN (?)",
			s.DB.Model(&models.EventParticipant{}).
				Select("event_id").Where("user_id = ?", userIDFrom(r)))
	}
	q.Find(&events)
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, id) {
		writeError(w, http.StatusForbidden, "accès refusé à cet événement")
		return
	}
	var event models.Event
	if err := s.DB.
		Preload("Tabs", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Preload("Tabs.Articles", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Preload("Tabs.Recipes.Recipe.Ingredients.Ingredient").
		Preload("Participants.User").
		First(&event, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "événement introuvable")
		return
	}
	// Privacy: hide other participants' weight, birth date and shoe size, and
	// hide the IBAN unless its owner allows this viewer (public / granted).
	self := userIDFrom(r)
	others := make([]*models.User, 0, len(event.Participants))
	for i := range event.Participants {
		if u := event.Participants[i].User; u != nil && u.ID != self {
			u.Weight, u.BirthDate, u.ShoeSize = nil, nil, nil
			others = append(others, u)
		}
	}
	s.redactIBANs(self, others)
	writeJSON(w, http.StatusOK, map[string]any{
		"event":                event,
		"effectiveParticipants": s.effectiveParticipantCount(id, event.InitialParticipants),
	})
}

type updateEventReq struct {
	Name                *string    `json:"name"`
	StartDate           *time.Time `json:"startDate"`
	EndDate             *time.Time `json:"endDate"`
	InitialParticipants *int       `json:"initialParticipants"`
	PhotoURL            *string    `json:"photoUrl"`
	VoteWeights         *string    `json:"voteWeights"`
	VenueAddress        *string    `json:"venueAddress"`
	VenueMapsURL        *string    `json:"venueMapsUrl"`
	VenuePhone          *string    `json:"venuePhone"`
	VenueInfo           *string    `json:"venueInfo"`
}

func (s *Server) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req updateEventReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.StartDate != nil {
		updates["start_date"] = *req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
	}
	if req.InitialParticipants != nil {
		updates["initial_participants"] = *req.InitialParticipants
	}
	if req.PhotoURL != nil {
		updates["photo_url"] = *req.PhotoURL
	}
	if req.VoteWeights != nil {
		updates["vote_weights"] = *req.VoteWeights
	}
	if req.VenueAddress != nil {
		updates["venue_address"] = *req.VenueAddress
	}
	if req.VenueMapsURL != nil {
		updates["venue_maps_url"] = *req.VenueMapsURL
	}
	if req.VenuePhone != nil {
		updates["venue_phone"] = *req.VenuePhone
	}
	if req.VenueInfo != nil {
		updates["venue_info"] = *req.VenueInfo
	}
	if len(updates) > 0 {
		s.DB.Model(&models.Event{}).Where("id = ?", id).Updates(updates)
	}
	var event models.Event
	s.DB.First(&event, "id = ?", id)
	writeJSON(w, http.StatusOK, event)
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Where("event_id = ?", id).Delete(&models.Meal{})
	s.DB.Where("event_id = ?", id).Delete(&models.ShoppingEntry{})
	s.DB.Where("event_id = ?", id).Delete(&models.LocationVote{})
	s.DB.Where("event_id = ?", id).Delete(&models.Location{})
	s.DB.Delete(&models.Event{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- participants ----

type addParticipantReq struct {
	UserID uuid.UUID `json:"userId"`
}

func (s *Server) handleAddParticipant(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req addParticipantReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	ep := models.EventParticipant{EventID: eventID, UserID: req.UserID, Counted: true}
	if err := s.DB.Where("event_id = ? AND user_id = ?", eventID, req.UserID).
		FirstOrCreate(&ep).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "ajout impossible")
		return
	}
	s.DB.Preload("User").First(&ep, "id = ?", ep.ID)
	writeJSON(w, http.StatusCreated, ep)
}

type updateParticipantReq struct {
	Counted *bool `json:"counted"`
}

func (s *Server) handleUpdateParticipant(w http.ResponseWriter, r *http.Request) {
	eventID, _ := uuid.Parse(chi.URLParam(r, "eventID"))
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var req updateParticipantReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if req.Counted != nil {
		s.DB.Model(&models.EventParticipant{}).
			Where("event_id = ? AND user_id = ?", eventID, userID).
			Update("counted", *req.Counted)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRemoveParticipant(w http.ResponseWriter, r *http.Request) {
	eventID, _ := uuid.Parse(chi.URLParam(r, "eventID"))
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Where("event_id = ? AND user_id = ?", eventID, userID).Delete(&models.EventParticipant{})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
