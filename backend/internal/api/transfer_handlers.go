package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

var internalImageURLRe = regexp.MustCompile(`^/api/images/([0-9a-fA-F-]{36})$`)

const transferBundleVersion = 1

// TransferBundle is the full export/import payload. All cross-references use
// natural keys (recipe name, user email) instead of raw UUIDs so the file is
// portable between databases (e.g. campmenu-dev <-> prod).
type TransferBundle struct {
	Version    int            `json:"version"`
	ExportedAt time.Time      `json:"exportedAt"`
	Recipes    []RecipeExport `json:"recipes"`
	Events     []EventExport  `json:"events"`
	Users      []UserExport   `json:"users"`
}

type RecipeExport struct {
	Name        string  `json:"name"`
	BasePersons int     `json:"basePersons"`
	Coefficient float64 `json:"coefficient"`
	PhotoURL    string  `json:"photoUrl"`
	// PhotoData/PhotoContentType embed the actual image bytes (base64) when
	// PhotoURL is an internal /api/images/{id} reference — otherwise that
	// reference wouldn't resolve to anything in the target database.
	PhotoData        string                   `json:"photoData,omitempty"`
	PhotoContentType string                   `json:"photoContentType,omitempty"`
	SourceURL        string                   `json:"sourceUrl"`
	Instructions     string                   `json:"instructions"`
	Kind             string                   `json:"kind"`
	Tags             []string                 `json:"tags"`
	Approved         bool                     `json:"approved"`
	Ingredients      []RecipeIngredientExport `json:"ingredients"`
}

type RecipeIngredientExport struct {
	IngredientName string  `json:"ingredientName"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
}

type UserExport struct {
	Email            string     `json:"email"`
	PasswordHash     string     `json:"passwordHash"`
	EmailConfirmed   bool       `json:"emailConfirmed"`
	Role             string     `json:"role"`
	FirstName        string     `json:"firstName"`
	LastName         string     `json:"lastName"`
	BirthDate        *time.Time `json:"birthDate"`
	ShoeSize         *float64   `json:"shoeSize"`
	Weight           *float64   `json:"weight"`
	PhotoURL         string     `json:"photoUrl"`
	PhotoData        string     `json:"photoData,omitempty"`
	PhotoContentType string     `json:"photoContentType,omitempty"`
	Theme            string     `json:"theme"`
	ColorPalette     string     `json:"colorPalette"`
	Nickname         string     `json:"nickname"`
	IBAN             string     `json:"iban"`
	IBANVisibility   string     `json:"ibanVisibility"`
	ColorblindMode   bool       `json:"colorblindMode"`
	Language         string     `json:"language"`
}

type EventExport struct {
	Name                string                `json:"name"`
	StartDate           time.Time             `json:"startDate"`
	EndDate             time.Time             `json:"endDate"`
	InitialParticipants int                   `json:"initialParticipants"`
	PhotoURL            string                `json:"photoUrl"`
	VoteWeights         string                `json:"voteWeights"`
	VenueAddress        string                `json:"venueAddress"`
	VenueMapsURL        string                `json:"venueMapsUrl"`
	VenuePhone          string                `json:"venuePhone"`
	VenueInfo           string                `json:"venueInfo"`
	Participants        []ParticipantExport   `json:"participants"`
	Tabs                []TabExport           `json:"tabs"`
	Meals               []MealExport          `json:"meals"`
	Locations           []LocationExport      `json:"locations"`
	LocationVotes       []LocationVoteExport  `json:"locationVotes"`
	Shopping            []ShoppingEntryExport `json:"shopping"`
}

type ParticipantExport struct {
	UserEmail string `json:"userEmail"`
	Counted   bool   `json:"counted"`
}

type TabExport struct {
	Kind              string                 `json:"kind"`
	Name              string                 `json:"name"`
	Icon              string                 `json:"icon"`
	Position          int                    `json:"position"`
	Removable         bool                   `json:"removable"`
	WithRecipes       bool                   `json:"withRecipes"`
	Voted             bool                   `json:"voted"`
	Adhoc             bool                   `json:"adhoc"`
	Sections          []string               `json:"sections"`
	ConsumptionLabels map[string]string      `json:"consumptionLabels"`
	Articles          []TabArticleExport     `json:"articles"`
	Recipes           []TabRecipeExport      `json:"recipes"`
	Consumptions      []TabConsumptionExport `json:"consumptions"`
}

type TabArticleExport struct {
	IngredientName string             `json:"ingredientName"`
	Name           string             `json:"name"`
	Unit           string             `json:"unit"`
	Section        string             `json:"section"`
	QtyPerLevel    map[string]float64 `json:"qtyPerLevel"`
	Quantity       float64            `json:"quantity"`
	Position       int                `json:"position"`
}

type TabRecipeExport struct {
	RecipeName       string `json:"recipeName"`
	Section          string `json:"section"`
	ParticipantCount int    `json:"participantCount"`
	Position         int    `json:"position"`
}

type TabConsumptionExport struct {
	ArticleIndex int    `json:"articleIndex"`
	UserEmail    string `json:"userEmail"`
	Level        int    `json:"level"`
}

type MealExport struct {
	DayIndex         int                 `json:"dayIndex"`
	Type             string              `json:"type"`
	Variant          string              `json:"variant"`
	ParticipantCount *int                `json:"participantCount"`
	Recipes          []MealRecipeExport  `json:"recipes"`
	RawItems         []MealRawItemExport `json:"rawItems"`
}

type MealRecipeExport struct {
	RecipeName       string `json:"recipeName"`
	ParticipantCount int    `json:"participantCount"`
	Position         int    `json:"position"`
}

type MealRawItemExport struct {
	IngredientName string  `json:"ingredientName"`
	Name           string  `json:"name"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
}

type LocationExport struct {
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
	IsWinner    bool     `json:"isWinner"`
}

type LocationVoteExport struct {
	UserEmail     string `json:"userEmail"`
	Rank          int    `json:"rank"`
	LocationIndex int    `json:"locationIndex"`
}

type ShoppingEntryExport struct {
	IngredientName string  `json:"ingredientName"`
	Section        string  `json:"section"`
	Name           string  `json:"name"`
	Unit           string  `json:"unit"`
	Source         string  `json:"source"`
	Observation    string  `json:"observation"`
	Bought         bool    `json:"bought"`
	BoughtQuantity float64 `json:"boughtQuantity"`
	BroughtByEmail string  `json:"broughtByEmail"`
}

// ---- export ----

type exportReq struct {
	RecipeIDs []string `json:"recipeIds"`
	EventIDs  []string `json:"eventIds"`
	UserIDs   []string `json:"userIds"`
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var req exportReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	bundle := TransferBundle{Version: transferBundleVersion, ExportedAt: time.Now()}

	if len(req.RecipeIDs) > 0 {
		var recipes []models.Recipe
		s.DB.Preload("Ingredients.Ingredient").Where("id IN (?)", req.RecipeIDs).Find(&recipes)
		for _, rc := range recipes {
			bundle.Recipes = append(bundle.Recipes, s.exportRecipe(rc))
		}
	}

	if len(req.UserIDs) > 0 {
		var users []models.User
		s.DB.Where("id IN (?)", req.UserIDs).Find(&users)
		for _, u := range users {
			bundle.Users = append(bundle.Users, s.exportUser(u))
		}
	}

	for _, id := range req.EventIDs {
		ev, err := s.exportEvent(id)
		if err == nil {
			bundle.Events = append(bundle.Events, ev)
		}
	}

	// Never serialize nil slices as JSON null — the frontend always expects
	// arrays it can .filter()/.map() directly, even when a category is empty.
	if bundle.Recipes == nil {
		bundle.Recipes = []RecipeExport{}
	}
	if bundle.Events == nil {
		bundle.Events = []EventExport{}
	}
	if bundle.Users == nil {
		bundle.Users = []UserExport{}
	}

	writeJSON(w, http.StatusOK, bundle)
}

// embedPhoto resolves an internal /api/images/{id} reference to its stored
// bytes so it can be carried inside the export bundle — that reference would
// otherwise be meaningless in a different target database. External URLs
// (or an empty photoURL) pass through with no embedded data.
func (s *Server) embedPhoto(photoURL string) (data, contentType string) {
	m := internalImageURLRe.FindStringSubmatch(photoURL)
	if m == nil {
		return "", ""
	}
	var img models.Image
	if err := s.DB.First(&img, "id = ?", m[1]).Error; err != nil {
		return "", ""
	}
	return base64.StdEncoding.EncodeToString(img.Data), img.ContentType
}

func (s *Server) exportRecipe(rc models.Recipe) RecipeExport {
	data, contentType := s.embedPhoto(rc.PhotoURL)
	out := RecipeExport{
		Name: rc.Name, BasePersons: rc.BasePersons, Coefficient: rc.Coefficient,
		PhotoURL: rc.PhotoURL, PhotoData: data, PhotoContentType: contentType,
		SourceURL: rc.SourceURL, Instructions: rc.Instructions, Kind: rc.Kind,
		Tags: rc.Tags, Approved: rc.Approved,
	}
	for _, ri := range rc.Ingredients {
		name := ""
		if ri.Ingredient != nil {
			name = ri.Ingredient.CanonicalName
		}
		out.Ingredients = append(out.Ingredients, RecipeIngredientExport{
			IngredientName: name, Quantity: ri.Quantity, Unit: ri.Unit,
		})
	}
	return out
}

func (s *Server) exportUser(u models.User) UserExport {
	data, contentType := s.embedPhoto(u.PhotoURL)
	return UserExport{
		Email: u.Email, PasswordHash: u.PasswordHash, EmailConfirmed: u.EmailConfirmed,
		Role: string(u.Role), FirstName: u.FirstName, LastName: u.LastName,
		BirthDate: u.BirthDate, ShoeSize: u.ShoeSize, Weight: u.Weight,
		PhotoURL: u.PhotoURL, PhotoData: data, PhotoContentType: contentType,
		Theme: u.Theme, ColorPalette: u.ColorPalette, Nickname: u.Nickname, IBAN: u.IBAN,
		IBANVisibility: u.IBANVisibility, ColorblindMode: u.ColorblindMode, Language: u.Language,
	}
}

// emailByID is a tiny best-effort resolver used across the export; empty
// string when the user no longer exists.
func (s *Server) emailByID(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	var u models.User
	if err := s.DB.Select("email").First(&u, "id = ?", id).Error; err != nil {
		return ""
	}
	return u.Email
}

func (s *Server) ingredientNameByID(id *uuid.UUID) string {
	if id == nil || *id == uuid.Nil {
		return ""
	}
	var ing models.Ingredient
	if err := s.DB.Select("canonical_name").First(&ing, "id = ?", *id).Error; err != nil {
		return ""
	}
	return ing.CanonicalName
}

func (s *Server) exportEvent(idStr string) (EventExport, error) {
	var ev models.Event
	if err := s.DB.First(&ev, "id = ?", idStr).Error; err != nil {
		return EventExport{}, err
	}
	out := EventExport{
		Name: ev.Name, StartDate: ev.StartDate, EndDate: ev.EndDate,
		InitialParticipants: ev.InitialParticipants, PhotoURL: ev.PhotoURL,
		VoteWeights: ev.VoteWeights, VenueAddress: ev.VenueAddress, VenueMapsURL: ev.VenueMapsURL,
		VenuePhone: ev.VenuePhone, VenueInfo: ev.VenueInfo,
	}

	var participants []models.EventParticipant
	s.DB.Where("event_id = ?", ev.ID).Find(&participants)
	for _, p := range participants {
		out.Participants = append(out.Participants, ParticipantExport{
			UserEmail: s.emailByID(p.UserID), Counted: p.Counted,
		})
	}

	var tabs []models.EventTab
	s.DB.Preload("Articles").Preload("Recipes.Recipe").Where("event_id = ?", ev.ID).Order("position asc").Find(&tabs)
	for _, tab := range tabs {
		te := TabExport{
			Kind: string(tab.Kind), Name: tab.Name, Icon: tab.Icon, Position: tab.Position,
			Removable: tab.Removable, WithRecipes: tab.WithRecipes, Voted: tab.Voted, Adhoc: tab.Adhoc,
			Sections: tab.Sections, ConsumptionLabels: tab.ConsumptionLabels,
		}
		articleIndexByID := map[uuid.UUID]int{}
		for i, a := range tab.Articles {
			articleIndexByID[a.ID] = i
			te.Articles = append(te.Articles, TabArticleExport{
				IngredientName: s.ingredientNameByID(a.IngredientID), Name: a.Name, Unit: a.Unit,
				Section: a.Section, QtyPerLevel: a.QtyPerLevel, Quantity: a.Quantity, Position: a.Position,
			})
		}
		for _, tr := range tab.Recipes {
			name := ""
			if tr.Recipe != nil {
				name = tr.Recipe.Name
			}
			te.Recipes = append(te.Recipes, TabRecipeExport{
				RecipeName: name, Section: tr.Section, ParticipantCount: tr.ParticipantCount, Position: tr.Position,
			})
		}
		var cons []models.TabConsumption
		s.DB.Where("tab_id = ?", tab.ID).Find(&cons)
		for _, c := range cons {
			idx, ok := articleIndexByID[c.ArticleID]
			if !ok {
				continue
			}
			te.Consumptions = append(te.Consumptions, TabConsumptionExport{
				ArticleIndex: idx, UserEmail: s.emailByID(c.UserID), Level: c.Level,
			})
		}
		if te.Articles == nil {
			te.Articles = []TabArticleExport{}
		}
		if te.Recipes == nil {
			te.Recipes = []TabRecipeExport{}
		}
		if te.Consumptions == nil {
			te.Consumptions = []TabConsumptionExport{}
		}
		out.Tabs = append(out.Tabs, te)
	}

	var meals []models.Meal
	s.DB.Preload("Recipes.Recipe").Preload("RawItems").Where("event_id = ?", ev.ID).Order("day_index asc").Find(&meals)
	for _, m := range meals {
		me := MealExport{DayIndex: m.DayIndex, Type: string(m.Type), Variant: m.Variant, ParticipantCount: m.ParticipantCount}
		for _, mr := range m.Recipes {
			name := ""
			if mr.Recipe != nil {
				name = mr.Recipe.Name
			}
			me.Recipes = append(me.Recipes, MealRecipeExport{RecipeName: name, ParticipantCount: mr.ParticipantCount, Position: mr.Position})
		}
		for _, ri := range m.RawItems {
			me.RawItems = append(me.RawItems, MealRawItemExport{
				IngredientName: s.ingredientNameByID(ri.IngredientID), Name: ri.Name, Quantity: ri.Quantity, Unit: ri.Unit,
			})
		}
		if me.Recipes == nil {
			me.Recipes = []MealRecipeExport{}
		}
		if me.RawItems == nil {
			me.RawItems = []MealRawItemExport{}
		}
		out.Meals = append(out.Meals, me)
	}

	var locations []models.Location
	s.DB.Where("event_id = ?", ev.ID).Order("created_at asc").Find(&locations)
	locationIndexByID := map[uuid.UUID]int{}
	for i, loc := range locations {
		locationIndexByID[loc.ID] = i
		out.Locations = append(out.Locations, LocationExport{
			Title: loc.Title, Address: loc.Address, WebsiteURL: loc.WebsiteURL, MapsURL: loc.MapsURL,
			Beds: loc.Beds, SingleBeds: loc.SingleBeds, DoubleBeds: loc.DoubleBeds, Toilets: loc.Toilets,
			Price: loc.Price, Phone: loc.Phone, UsefulInfo: loc.UsefulInfo, Description: loc.Description,
			Observation: loc.Observation, Amenities: loc.Amenities, Images: loc.Images, IsWinner: loc.IsWinner,
		})
	}

	var votes []models.LocationVote
	s.DB.Where("event_id = ?", ev.ID).Find(&votes)
	for _, v := range votes {
		idx, ok := locationIndexByID[v.LocationID]
		if !ok {
			continue
		}
		out.LocationVotes = append(out.LocationVotes, LocationVoteExport{
			UserEmail: s.emailByID(v.UserID), Rank: v.Rank, LocationIndex: idx,
		})
	}

	var entries []models.ShoppingEntry
	s.DB.Where("event_id = ?", ev.ID).Find(&entries)
	for _, se := range entries {
		broughtBy := ""
		if se.BroughtBy != nil {
			broughtBy = s.emailByID(*se.BroughtBy)
		}
		out.Shopping = append(out.Shopping, ShoppingEntryExport{
			IngredientName: s.ingredientNameByID(se.IngredientID), Section: se.Section, Name: se.Name, Unit: se.Unit,
			Source: se.Source, Observation: se.Observation, Bought: se.Bought, BoughtQuantity: se.BoughtQuantity,
			BroughtByEmail: broughtBy,
		})
	}

	if out.Participants == nil {
		out.Participants = []ParticipantExport{}
	}
	if out.Tabs == nil {
		out.Tabs = []TabExport{}
	}
	if out.Meals == nil {
		out.Meals = []MealExport{}
	}
	if out.Locations == nil {
		out.Locations = []LocationExport{}
	}
	if out.LocationVotes == nil {
		out.LocationVotes = []LocationVoteExport{}
	}
	if out.Shopping == nil {
		out.Shopping = []ShoppingEntryExport{}
	}

	return out, nil
}

// ---- import preview ----

type previewItem struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Exists bool   `json:"exists"`
}

type previewResp struct {
	Recipes []previewItem `json:"recipes"`
	Events  []previewItem `json:"events"`
	Users   []previewItem `json:"users"`
}

func eventNaturalKey(name string, start time.Time) string {
	return name + "|" + start.Format(time.RFC3339)
}

func (s *Server) handleImportPreview(w http.ResponseWriter, r *http.Request) {
	var bundle TransferBundle
	if err := decode(r, &bundle); err != nil {
		writeError(w, http.StatusBadRequest, "fichier invalide")
		return
	}
	resp := previewResp{Recipes: []previewItem{}, Events: []previewItem{}, Users: []previewItem{}}
	for _, rc := range bundle.Recipes {
		var count int64
		s.DB.Model(&models.Recipe{}).Where("LOWER(name) = LOWER(?)", rc.Name).Count(&count)
		resp.Recipes = append(resp.Recipes, previewItem{Key: rc.Name, Label: rc.Name, Exists: count > 0})
	}
	for _, u := range bundle.Users {
		var count int64
		s.DB.Model(&models.User{}).Where("LOWER(email) = LOWER(?)", u.Email).Count(&count)
		resp.Users = append(resp.Users, previewItem{
			Key: u.Email, Label: fmt.Sprintf("%s %s <%s>", u.FirstName, u.LastName, u.Email), Exists: count > 0,
		})
	}
	for _, ev := range bundle.Events {
		var count int64
		s.DB.Model(&models.Event{}).Where("LOWER(name) = LOWER(?) AND start_date = ?", ev.Name, ev.StartDate).Count(&count)
		resp.Events = append(resp.Events, previewItem{
			Key: eventNaturalKey(ev.Name, ev.StartDate), Label: fmt.Sprintf("%s (%s)", ev.Name, ev.StartDate.Format("2006-01-02")), Exists: count > 0,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---- import commit ----

type commitReq struct {
	Bundle     TransferBundle `json:"bundle"`
	Selections struct {
		Recipes []string `json:"recipes"`
		Events  []string `json:"events"`
		Users   []string `json:"users"`
	} `json:"selections"`
}

type commitResp struct {
	ImportedUsers   int      `json:"importedUsers"`
	ImportedRecipes int      `json:"importedRecipes"`
	ImportedEvents  int      `json:"importedEvents"`
	Skipped         []string `json:"skipped"`
}

func toSet(items []string) map[string]bool {
	set := map[string]bool{}
	for _, it := range items {
		set[strings.ToLower(it)] = true
	}
	return set
}

func (s *Server) handleImportCommit(w http.ResponseWriter, r *http.Request) {
	var req commitReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	adminID := userIDFrom(r)
	selRecipes := toSet(req.Selections.Recipes)
	selEvents := toSet(req.Selections.Events)
	selUsers := toSet(req.Selections.Users)

	resp := commitResp{Skipped: []string{}}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, u := range req.Bundle.Users {
			if !selUsers[strings.ToLower(u.Email)] {
				continue
			}
			if err := upsertUser(tx, u); err != nil {
				return err
			}
			resp.ImportedUsers++
		}
		for _, rc := range req.Bundle.Recipes {
			if !selRecipes[strings.ToLower(rc.Name)] {
				continue
			}
			if err := upsertRecipe(tx, rc, adminID); err != nil {
				return err
			}
			resp.ImportedRecipes++
		}
		for _, ev := range req.Bundle.Events {
			if !selEvents[strings.ToLower(eventNaturalKey(ev.Name, ev.StartDate))] {
				continue
			}
			skipped, err := upsertEvent(tx, ev, adminID)
			if err != nil {
				return err
			}
			resp.Skipped = append(resp.Skipped, skipped...)
			resp.ImportedEvents++
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "import impossible")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func firstOrCreateIngredient(tx *gorm.DB, name, unit string) (models.Ingredient, bool) {
	if strings.TrimSpace(name) == "" {
		return models.Ingredient{}, false
	}
	var ing models.Ingredient
	if err := tx.Where("LOWER(canonical_name) = LOWER(?)", name).First(&ing).Error; err == nil {
		return ing, true
	}
	ing = models.Ingredient{CanonicalName: name, DefaultUnit: unit}
	if err := tx.Create(&ing).Error; err != nil {
		return models.Ingredient{}, false
	}
	return ing, true
}

func findUserByEmail(tx *gorm.DB, email string) (models.User, bool) {
	if strings.TrimSpace(email) == "" {
		return models.User{}, false
	}
	var u models.User
	if err := tx.Where("LOWER(email) = LOWER(?)", email).First(&u).Error; err != nil {
		return models.User{}, false
	}
	return u, true
}

func findRecipeByName(tx *gorm.DB, name string) (models.Recipe, bool) {
	if strings.TrimSpace(name) == "" {
		return models.Recipe{}, false
	}
	var rc models.Recipe
	if err := tx.Where("LOWER(name) = LOWER(?)", name).First(&rc).Error; err != nil {
		return models.Recipe{}, false
	}
	return rc, true
}

// createImageFromBase64 recreates an Image row from embedded export data,
// returning a fresh /api/images/{id} reference valid in this database. No-op
// (ok=false) when there's no embedded data, e.g. the source had an external
// photo URL or none at all.
func createImageFromBase64(tx *gorm.DB, data, contentType string) (string, bool) {
	if data == "" {
		return "", false
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", false
	}
	img := models.Image{ContentType: contentType, Data: raw}
	if err := tx.Create(&img).Error; err != nil {
		return "", false
	}
	return "/api/images/" + img.ID.String(), true
}

func upsertUser(tx *gorm.DB, u UserExport) error {
	photoURL := u.PhotoURL
	if url, ok := createImageFromBase64(tx, u.PhotoData, u.PhotoContentType); ok {
		photoURL = url
	}
	rec := models.User{
		Email: u.Email, PasswordHash: u.PasswordHash, EmailConfirmed: u.EmailConfirmed,
		Role: models.Role(u.Role), FirstName: u.FirstName, LastName: u.LastName, BirthDate: u.BirthDate,
		ShoeSize: u.ShoeSize, Weight: u.Weight, PhotoURL: photoURL, Theme: u.Theme, ColorPalette: u.ColorPalette,
		Nickname: u.Nickname, IBAN: u.IBAN, IBANVisibility: u.IBANVisibility, ColorblindMode: u.ColorblindMode,
		Language: u.Language,
	}
	var existing models.User
	if err := tx.Where("LOWER(email) = LOWER(?)", u.Email).First(&existing).Error; err == nil {
		rec.Base = existing.Base
		return tx.Model(&existing).Select("*").Updates(rec).Error
	}
	return tx.Create(&rec).Error
}

func upsertRecipe(tx *gorm.DB, rc RecipeExport, adminID uuid.UUID) error {
	photoURL := rc.PhotoURL
	if url, ok := createImageFromBase64(tx, rc.PhotoData, rc.PhotoContentType); ok {
		photoURL = url
	}
	recipe := models.Recipe{
		Name: rc.Name, BasePersons: rc.BasePersons, Coefficient: rc.Coefficient, PhotoURL: photoURL,
		SourceURL: rc.SourceURL, Instructions: rc.Instructions, Kind: rc.Kind, Tags: rc.Tags, Approved: rc.Approved, CreatedBy: adminID,
	}
	var existing models.Recipe
	if err := tx.Where("LOWER(name) = LOWER(?)", rc.Name).First(&existing).Error; err == nil {
		recipe.Base = existing.Base
		if err := tx.Where("recipe_id = ?", existing.ID).Delete(&models.RecipeIngredient{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&existing).Select("*").Updates(recipe).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Create(&recipe).Error; err != nil {
			return err
		}
	}
	for _, ri := range rc.Ingredients {
		ing, ok := firstOrCreateIngredient(tx, ri.IngredientName, ri.Unit)
		if !ok {
			continue
		}
		if err := tx.Create(&models.RecipeIngredient{RecipeID: recipe.ID, IngredientID: ing.ID, Quantity: ri.Quantity, Unit: ri.Unit}).Error; err != nil {
			return err
		}
	}
	return nil
}

// upsertEvent creates or fully replaces one event's graph. Unresolvable
// cross-references (a recipe/user not found in the target) are skipped
// individually and reported back rather than aborting the whole import.
func upsertEvent(tx *gorm.DB, ev EventExport, adminID uuid.UUID) ([]string, error) {
	var skipped []string
	event := models.Event{
		Name: ev.Name, StartDate: ev.StartDate, EndDate: ev.EndDate, InitialParticipants: ev.InitialParticipants,
		PhotoURL: ev.PhotoURL, VoteWeights: ev.VoteWeights, VenueAddress: ev.VenueAddress, VenueMapsURL: ev.VenueMapsURL,
		VenuePhone: ev.VenuePhone, VenueInfo: ev.VenueInfo, CreatedBy: adminID,
	}
	var existing models.Event
	if err := tx.Where("LOWER(name) = LOWER(?) AND start_date = ?", ev.Name, ev.StartDate).First(&existing).Error; err == nil {
		event.Base = existing.Base
		var tabIDs []uuid.UUID
		tx.Model(&models.EventTab{}).Where("event_id = ?", existing.ID).Pluck("id", &tabIDs)
		if len(tabIDs) > 0 {
			if err := tx.Where("tab_id IN (?)", tabIDs).Delete(&models.TabConsumption{}).Error; err != nil {
				return nil, err
			}
		}
		if err := tx.Where("event_id = ?", existing.ID).Delete(&models.Location{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("event_id = ?", existing.ID).Delete(&models.LocationVote{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("event_id = ?", existing.ID).Delete(&models.ShoppingEntry{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("event_id = ?", existing.ID).Delete(&models.Meal{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("event_id = ?", existing.ID).Delete(&models.EventParticipant{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("event_id = ?", existing.ID).Delete(&models.EventTab{}).Error; err != nil {
			return nil, err
		}
		if err := tx.Model(&existing).Select("*").Updates(event).Error; err != nil {
			return nil, err
		}
	} else {
		if err := tx.Create(&event).Error; err != nil {
			return nil, err
		}
	}

	for _, p := range ev.Participants {
		u, ok := findUserByEmail(tx, p.UserEmail)
		if !ok {
			skipped = append(skipped, fmt.Sprintf("événement %s : participant %s introuvable", ev.Name, p.UserEmail))
			continue
		}
		if err := tx.Create(&models.EventParticipant{EventID: event.ID, UserID: u.ID, Counted: p.Counted}).Error; err != nil {
			return nil, err
		}
	}

	for _, t := range ev.Tabs {
		tab := models.EventTab{
			EventID: event.ID, Kind: models.TabKind(t.Kind), Name: t.Name, Icon: t.Icon, Position: t.Position,
			Removable: t.Removable, WithRecipes: t.WithRecipes, Voted: t.Voted, Adhoc: t.Adhoc,
			Sections: t.Sections, ConsumptionLabels: t.ConsumptionLabels,
		}
		if err := tx.Create(&tab).Error; err != nil {
			return nil, err
		}
		articleIDByIndex := map[int]uuid.UUID{}
		for i, a := range t.Articles {
			var ingID *uuid.UUID
			if ing, ok := firstOrCreateIngredient(tx, a.IngredientName, a.Unit); ok {
				ingID = &ing.ID
			}
			article := models.TabArticle{
				TabID: tab.ID, IngredientID: ingID, Name: a.Name, Unit: a.Unit, Section: a.Section,
				QtyPerLevel: a.QtyPerLevel, Quantity: a.Quantity, Position: a.Position,
			}
			if err := tx.Create(&article).Error; err != nil {
				return nil, err
			}
			articleIDByIndex[i] = article.ID
		}
		for _, tr := range t.Recipes {
			rc, ok := findRecipeByName(tx, tr.RecipeName)
			if !ok {
				skipped = append(skipped, fmt.Sprintf("événement %s : recette %s introuvable pour le tab %s", ev.Name, tr.RecipeName, t.Name))
				continue
			}
			if err := tx.Create(&models.TabRecipe{TabID: tab.ID, RecipeID: rc.ID, Section: tr.Section, ParticipantCount: tr.ParticipantCount, Position: tr.Position}).Error; err != nil {
				return nil, err
			}
		}
		for _, c := range t.Consumptions {
			articleID, ok := articleIDByIndex[c.ArticleIndex]
			if !ok {
				continue
			}
			u, ok := findUserByEmail(tx, c.UserEmail)
			if !ok {
				skipped = append(skipped, fmt.Sprintf("événement %s : vote de %s introuvable (utilisateur absent)", ev.Name, c.UserEmail))
				continue
			}
			if err := tx.Create(&models.TabConsumption{TabID: tab.ID, ArticleID: articleID, UserID: u.ID, Level: c.Level}).Error; err != nil {
				return nil, err
			}
		}
	}

	for _, m := range ev.Meals {
		meal := models.Meal{EventID: event.ID, DayIndex: m.DayIndex, Type: models.MealType(m.Type), Variant: m.Variant, ParticipantCount: m.ParticipantCount}
		if err := tx.Create(&meal).Error; err != nil {
			return nil, err
		}
		for _, mr := range m.Recipes {
			rc, ok := findRecipeByName(tx, mr.RecipeName)
			if !ok {
				skipped = append(skipped, fmt.Sprintf("événement %s : recette %s introuvable pour un repas", ev.Name, mr.RecipeName))
				continue
			}
			if err := tx.Create(&models.MealRecipe{MealID: meal.ID, RecipeID: rc.ID, ParticipantCount: mr.ParticipantCount, Position: mr.Position}).Error; err != nil {
				return nil, err
			}
		}
		for _, ri := range m.RawItems {
			var ingID *uuid.UUID
			if ing, ok := firstOrCreateIngredient(tx, ri.IngredientName, ri.Unit); ok {
				ingID = &ing.ID
			}
			if err := tx.Create(&models.MealRawItem{MealID: meal.ID, IngredientID: ingID, Name: ri.Name, Quantity: ri.Quantity, Unit: ri.Unit}).Error; err != nil {
				return nil, err
			}
		}
	}

	locationIDByIndex := map[int]uuid.UUID{}
	for i, l := range ev.Locations {
		loc := models.Location{
			EventID: event.ID, CreatedBy: adminID, Title: l.Title, Address: l.Address, WebsiteURL: l.WebsiteURL,
			MapsURL: l.MapsURL, Beds: l.Beds, SingleBeds: l.SingleBeds, DoubleBeds: l.DoubleBeds, Toilets: l.Toilets,
			Price: l.Price, Phone: l.Phone, UsefulInfo: l.UsefulInfo, Description: l.Description, Observation: l.Observation,
			Amenities: l.Amenities, Images: l.Images, IsWinner: l.IsWinner,
		}
		if err := tx.Create(&loc).Error; err != nil {
			return nil, err
		}
		locationIDByIndex[i] = loc.ID
	}
	for _, v := range ev.LocationVotes {
		locID, ok := locationIDByIndex[v.LocationIndex]
		if !ok {
			continue
		}
		u, ok := findUserByEmail(tx, v.UserEmail)
		if !ok {
			skipped = append(skipped, fmt.Sprintf("événement %s : vote logement de %s introuvable (utilisateur absent)", ev.Name, v.UserEmail))
			continue
		}
		if err := tx.Create(&models.LocationVote{EventID: event.ID, UserID: u.ID, Rank: v.Rank, LocationID: locID}).Error; err != nil {
			return nil, err
		}
	}

	for _, se := range ev.Shopping {
		var ingID *uuid.UUID
		if ing, ok := firstOrCreateIngredient(tx, se.IngredientName, se.Unit); ok {
			ingID = &ing.ID
		}
		var broughtBy *uuid.UUID
		if u, ok := findUserByEmail(tx, se.BroughtByEmail); ok {
			broughtBy = &u.ID
		}
		entry := models.ShoppingEntry{
			EventID: event.ID, IngredientID: ingID, Section: se.Section, Name: se.Name, Unit: se.Unit,
			Source: se.Source, Observation: se.Observation, Bought: se.Bought, BoughtQuantity: se.BoughtQuantity,
			BroughtBy: broughtBy,
		}
		if err := tx.Create(&entry).Error; err != nil {
			return nil, err
		}
	}

	return skipped, nil
}
