package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/ai"
	"github.com/morphatrix/campmenu/internal/models"
)

type importLocationReq struct {
	URL string `json:"url"`
}

// handleImportLocation fetches a lodging web page and asks the configured AI to
// extract the form fields. Returns 200 + {ok,error|draft} (a 5xx would be turned
// into an HTML page by the ingress).
func (s *Server) handleImportLocation(w http.ResponseWriter, r *http.Request) {
	var req importLocationReq
	if err := decode(r, &req); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "corps de requête invalide"})
		return
	}
	cfg := s.aiConfig()
	if !cfg.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "IA non configurée"})
		return
	}
	page, images, err := ai.FetchAndClean(r.Context(), strings.TrimSpace(req.URL))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	draft, err := ai.ExtractLocation(r.Context(), cfg, page)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	// All page photos populate the gallery (the model is unreliable for image URLs).
	if len(images) > 0 {
		draft.Images = images
	}
	if draft.WebsiteURL == "" {
		draft.WebsiteURL = strings.TrimSpace(req.URL)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "draft": draft})
}

// parseWeights turns "3,2,1" into [3,2,1]; falls back to [3,2,1] when empty.
func parseWeights(csv string) []int {
	parts := strings.Split(csv, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return []int{3, 2, 1}
	}
	return out
}

func (s *Server) locationEventID(id uuid.UUID) (uuid.UUID, uuid.UUID, bool) {
	var loc models.Location
	if err := s.DB.Select("event_id", "created_by").First(&loc, "id = ?", id).Error; err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return loc.EventID, loc.CreatedBy, true
}

type voterOut struct {
	UserID uuid.UUID `json:"userId"`
	Rank   int       `json:"rank"`
}

type locationOut struct {
	models.Location
	Score  int        `json:"score"`
	Voters []voterOut `json:"voters"`
}

func (s *Server) handleListLocations(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var event models.Event
	s.DB.First(&event, "id = ?", eventID)
	weights := parseWeights(event.VoteWeights)

	var locations []models.Location
	s.DB.Where("event_id = ?", eventID).Order("is_winner desc, created_at asc").Find(&locations)

	var votes []models.LocationVote
	s.DB.Where("event_id = ?", eventID).Find(&votes)

	// Weighted score per location + the caller's own ranked votes + voters list.
	score := map[uuid.UUID]int{}
	voters := map[uuid.UUID][]voterOut{}
	myVotes := map[string]uuid.UUID{}
	uid := userIDFrom(r)
	for _, v := range votes {
		if v.Rank >= 1 && v.Rank <= len(weights) {
			score[v.LocationID] += weights[v.Rank-1]
		}
		voters[v.LocationID] = append(voters[v.LocationID], voterOut{UserID: v.UserID, Rank: v.Rank})
		if v.UserID == uid {
			myVotes[strconv.Itoa(v.Rank)] = v.LocationID
		}
	}

	out := make([]locationOut, len(locations))
	for i, loc := range locations {
		out[i] = locationOut{Location: loc, Score: score[loc.ID], Voters: voters[loc.ID]}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"locations":   out,
		"myVotes":     myVotes,
		"voteWeights": weights,
	})
}

type locationReq struct {
	Title       string   `json:"title"`
	Address     string   `json:"address"`
	WebsiteURL  string   `json:"websiteUrl"`
	MapsURL     string   `json:"mapsUrl"`
	Beds        int      `json:"beds"`
	SingleBeds  int      `json:"singleBeds"`
	DoubleBeds  int      `json:"doubleBeds"`
	Toilets     int      `json:"toilets"`
	Price       float64  `json:"price"`
	Phone       string   `json:"phone"`
	UsefulInfo  string   `json:"usefulInfo"`
	Description string   `json:"description"`
	Observation string   `json:"observation"`
	Amenities   []string `json:"amenities"`
	Images      []string `json:"images"`
}

func (s *Server) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var req locationReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "intitulé requis")
		return
	}
	loc := models.Location{
		EventID: eventID, CreatedBy: userIDFrom(r),
		Title: req.Title, Address: req.Address, WebsiteURL: req.WebsiteURL, MapsURL: req.MapsURL,
		Beds: req.Beds, SingleBeds: req.SingleBeds, DoubleBeds: req.DoubleBeds, Toilets: req.Toilets,
		Price: req.Price, Phone: req.Phone, UsefulInfo: req.UsefulInfo, Description: req.Description,
		Observation: req.Observation, Amenities: req.Amenities, Images: req.Images,
	}
	if err := s.DB.Create(&loc).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "création impossible")
		return
	}
	writeJSON(w, http.StatusCreated, loc)
}

func (s *Server) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	_, owner, ok := s.locationEventID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "location introuvable")
		return
	}
	if owner != userIDFrom(r) && !isStaff(r) {
		writeError(w, http.StatusForbidden, "modification non autorisée")
		return
	}
	var req locationReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{
		"title": req.Title, "address": req.Address, "website_url": req.WebsiteURL, "maps_url": req.MapsURL,
		"beds": req.Beds, "single_beds": req.SingleBeds, "double_beds": req.DoubleBeds, "toilets": req.Toilets,
		"price": req.Price, "phone": req.Phone, "useful_info": req.UsefulInfo, "description": req.Description,
		"observation": req.Observation,
		"amenities": models.JSONStrings(req.Amenities), "images": models.JSONStrings(req.Images),
	}
	s.DB.Model(&models.Location{}).Where("id = ?", id).Updates(updates)
	var loc models.Location
	s.DB.First(&loc, "id = ?", id)
	writeJSON(w, http.StatusOK, loc)
}

func (s *Server) handleDeleteLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	_, owner, ok := s.locationEventID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "location introuvable")
		return
	}
	if owner != userIDFrom(r) && !isStaff(r) {
		writeError(w, http.StatusForbidden, "suppression non autorisée")
		return
	}
	s.DB.Where("location_id = ?", id).Delete(&models.LocationVote{})
	s.DB.Delete(&models.Location{}, "id = ?", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type voteItem struct {
	LocationID uuid.UUID `json:"locationId"`
	Rank       int       `json:"rank"`
}

type setVotesReq struct {
	Votes []voteItem `json:"votes"`
}

// handleSetVotes replaces the caller's ranked votes for the event.
func (s *Server) handleSetVotes(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if !s.canAccessEvent(r, eventID) {
		writeError(w, http.StatusForbidden, "accès refusé")
		return
	}
	var req setVotesReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	var event models.Event
	s.DB.First(&event, "id = ?", eventID)
	maxRank := len(parseWeights(event.VoteWeights))

	seenRank := map[int]bool{}
	seenLoc := map[uuid.UUID]bool{}
	uid := userIDFrom(r)
	rows := make([]models.LocationVote, 0, len(req.Votes))
	for _, v := range req.Votes {
		if v.Rank < 1 || v.Rank > maxRank || seenRank[v.Rank] || seenLoc[v.LocationID] || v.LocationID == uuid.Nil {
			continue
		}
		seenRank[v.Rank] = true
		seenLoc[v.LocationID] = true
		rows = append(rows, models.LocationVote{EventID: eventID, UserID: uid, Rank: v.Rank, LocationID: v.LocationID})
	}
	s.DB.Where("event_id = ? AND user_id = ?", eventID, uid).Delete(&models.LocationVote{})
	if len(rows) > 0 {
		s.DB.Create(&rows)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handlePromoteLocation (admin) marks the winning location.
func (s *Server) handlePromoteLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	eventID, _, ok := s.locationEventID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "location introuvable")
		return
	}
	// Toggle: if this location is already the winner, clear it (and the venue info)
	// so the choice can be undone.
	var current models.Location
	if s.DB.First(&current, "id = ?", id).Error == nil && current.IsWinner {
		s.DB.Model(&models.Location{}).Where("id = ?", id).Update("is_winner", false)
		s.DB.Model(&models.Event{}).Where("id = ?", eventID).Updates(map[string]any{
			"venue_address": "", "venue_maps_url": "", "venue_phone": "", "venue_info": "",
		})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	s.DB.Model(&models.Location{}).Where("event_id = ?", eventID).Update("is_winner", false)
	s.DB.Model(&models.Location{}).Where("id = ?", id).Update("is_winner", true)

	// Copy the winner's details into the event venue info (admin can refine).
	var loc models.Location
	if s.DB.First(&loc, "id = ?", id).Error == nil {
		mapsURL := loc.MapsURL
		if mapsURL == "" && loc.Address != "" {
			mapsURL = "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(loc.Address)
		}
		info := strings.TrimSpace(loc.Description + "\n" + loc.UsefulInfo)
		s.DB.Model(&models.Event{}).Where("id = ?", eventID).Updates(map[string]any{
			"venue_address":  loc.Address,
			"venue_maps_url": mapsURL,
			"venue_phone":    loc.Phone,
			"venue_info":     info,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
