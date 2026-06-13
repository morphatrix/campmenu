package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base is embedded in every entity: UUID primary key + timestamps.
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BeforeCreate generates a UUID when one was not supplied.
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type Role string

const (
	RoleAdmin        Role = "ADMIN"
	RoleCollaborator Role = "COLLABORATOR"
	RoleUser         Role = "USER"
)

// IsStaff reports whether the role has management rights (admin or collaborator).
func (r Role) IsStaff() bool {
	return r == RoleAdmin || r == RoleCollaborator
}

// User is a participant/account.
type User struct {
	Base
	Email             string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash      string     `gorm:"not null" json:"-"`
	EmailConfirmed    bool       `gorm:"not null;default:false" json:"emailConfirmed"`
	ConfirmationToken string     `gorm:"index" json:"-"`
	Role              Role       `gorm:"type:text;not null;default:'USER'" json:"role"`
	FirstName         string     `json:"firstName"`
	LastName          string     `json:"lastName"`
	BirthDate         *time.Time `json:"birthDate"`
	ShoeSize          *float64   `json:"shoeSize"`
	Weight            *float64   `json:"weight"`
	PhotoURL          string     `json:"photoUrl"`
	Theme             string     `gorm:"default:'auto'" json:"theme"`          // light | dark | auto
	ColorPalette      string     `gorm:"default:'default'" json:"colorPalette"` // default | palette2..4
	Nickname          string     `json:"nickname"`
	IBAN              string     `json:"iban"`
	ColorblindMode    bool       `gorm:"default:false" json:"colorblindMode"`
	Language          string     `gorm:"default:'fr'" json:"language"`
	ResetToken        string     `gorm:"index" json:"-"`
	ResetTokenExpiry  *time.Time `json:"-"`
	// Impersonating is transient (not stored): set on /me when an admin is
	// currently impersonating this user.
	Impersonating bool `gorm:"-" json:"impersonating"`
}

// Image stores an uploaded picture (recipe/profile) directly in PostgreSQL,
// so no external object storage is required for the homelab.
type Image struct {
	Base
	ContentType string `gorm:"not null" json:"contentType"`
	Data        []byte `gorm:"type:bytea" json:"-"`
}

// Invite is a registration link. It can be reused: valid while not expired and
// (MaxUses == 0 [unlimited]) or (UseCount < MaxUses).
type Invite struct {
	Base
	Code      string     `gorm:"uniqueIndex;not null" json:"code"`
	Email     string     `json:"email"`
	Role      Role       `gorm:"type:text;default:'USER'" json:"role"`
	CreatedBy uuid.UUID  `gorm:"type:uuid" json:"createdBy"`
	MaxUses   int        `gorm:"default:0" json:"maxUses"` // 0 = unlimited
	UseCount  int        `gorm:"default:0" json:"useCount"`
	Revoked   bool       `gorm:"default:false" json:"revoked"`
	UsedAt    *time.Time `json:"usedAt"` // first use, kept for reference
	ExpiresAt *time.Time `json:"expiresAt"`
}

// Exhausted reports whether the invite can no longer be used.
func (i *Invite) Exhausted() bool {
	if i.Revoked {
		return true
	}
	if i.MaxUses > 0 && i.UseCount >= i.MaxUses {
		return true
	}
	if i.ExpiresAt != nil && i.ExpiresAt.Before(time.Now()) {
		return true
	}
	return false
}

// Event is a trip with a date range and a roster of participants.
type Event struct {
	Base
	Name                string             `gorm:"not null" json:"name"`
	StartDate           time.Time          `gorm:"not null" json:"startDate"`
	EndDate             time.Time          `gorm:"not null" json:"endDate"`
	InitialParticipants int                `json:"initialParticipants"`
	PhotoURL            string             `json:"photoUrl"`
	VoteWeights         string             `gorm:"default:'3,2,1'" json:"voteWeights"` // CSV weights for location podium votes
	VenueAddress        string             `json:"venueAddress"`
	VenueMapsURL        string             `json:"venueMapsUrl"`
	VenuePhone          string             `json:"venuePhone"`
	VenueInfo           string             `json:"venueInfo"`
	CreatedBy           uuid.UUID          `gorm:"type:uuid" json:"createdBy"`
	Participants        []EventParticipant `gorm:"constraint:OnDelete:CASCADE" json:"participants,omitempty"`
	Tabs                []EventTab         `gorm:"constraint:OnDelete:CASCADE" json:"tabs,omitempty"`
}

// EventParticipant links a user to an event (the invited roster).
type EventParticipant struct {
	Base
	EventID uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_event_user" json:"eventId"`
	UserID  uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_event_user" json:"userId"`
	User    *User     `json:"user,omitempty"`
	Counted bool      `gorm:"default:true" json:"counted"` // counted in quantity computations
}

type TabKind string

const (
	TabMenus     TabKind = "MENUS"     // day x meal grid
	TabShopping  TabKind = "SHOPPING"  // mandatory, non-removable
	TabMatrix    TabKind = "MATRIX"    // breakfast / slopes / custom: participant x article grid
	TabLocations TabKind = "LOCATIONS" // candidate lodgings + weighted podium votes
)

// EventTab is a modular tab attached to an event.
type EventTab struct {
	Base
	EventID           uuid.UUID    `gorm:"type:uuid;index" json:"eventId"`
	Kind              TabKind      `gorm:"type:varchar(16);not null" json:"kind"`
	Name              string       `gorm:"not null" json:"name"`
	Icon              string       `json:"icon"`
	Position          int          `gorm:"not null;default:0" json:"position"`
	Removable         bool         `gorm:"default:true" json:"removable"`
	WithRecipes       bool         `gorm:"default:false" json:"withRecipes"` // custom tab: free list vs recipe-backed
	Voted             bool         `gorm:"default:true" json:"voted"`        // true: participant consumption; false: organizer-set totals
	ListID            *uuid.UUID   `gorm:"type:uuid" json:"listId"`          // source ProductList, if any
	Sections          JSONStrings  `gorm:"type:jsonb" json:"sections"`       // ordered section names for grouping
	ConsumptionLabels JSONMap      `gorm:"type:jsonb" json:"consumptionLabels"`
	Articles          []TabArticle `gorm:"foreignKey:TabID;constraint:OnDelete:CASCADE" json:"articles,omitempty"`
	Recipes           []TabRecipe  `gorm:"foreignKey:TabID;constraint:OnDelete:CASCADE" json:"recipes,omitempty"`
}

// TabArticle is one row of a matrix tab (e.g. "croissants", "café").
type TabArticle struct {
	Base
	TabID        uuid.UUID  `gorm:"type:uuid;index" json:"tabId"`
	IngredientID *uuid.UUID `gorm:"type:uuid" json:"ingredientId"`
	Name         string     `gorm:"not null" json:"name"`
	Unit         string     `json:"unit"`
	Section      string     `json:"section"` // grouping within the tab (e.g. Cuisine, Hygiène)
	// Quantity per person per day for each consumption level, e.g. {"1":1,"2":2,"3":3} (voted tabs).
	QtyPerLevel JSONNum `gorm:"type:jsonb" json:"qtyPerLevel"`
	// Quantity is the organizer-set total for the whole event (non-voted tabs).
	Quantity float64 `json:"quantity"`
	Position int     `json:"position"`
}

// TabRecipe attaches a recipe (e.g. a cocktail) to a tab section with a serving count.
type TabRecipe struct {
	Base
	TabID            uuid.UUID `gorm:"type:uuid;index" json:"tabId"`
	RecipeID         uuid.UUID `gorm:"type:uuid;index" json:"recipeId"`
	Recipe           *Recipe   `json:"recipe,omitempty"`
	Section          string    `json:"section"`
	ParticipantCount int       `json:"participantCount"` // servings to prepare
	Position         int       `json:"position"`
}

// TabConsumption is a participant's chosen level (0-3) for an article.
type TabConsumption struct {
	Base
	TabID     uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_tab_article_user" json:"tabId"`
	ArticleID uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_tab_article_user" json:"articleId"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_tab_article_user" json:"userId"`
	Level     int       `gorm:"default:0" json:"level"` // 0..3
}

// Location is a candidate lodging proposed by a participant for an event,
// later promoted to the final venue.
type Location struct {
	Base
	EventID     uuid.UUID   `gorm:"type:uuid;index" json:"eventId"`
	CreatedBy   uuid.UUID   `gorm:"type:uuid" json:"createdBy"`
	Title       string      `gorm:"not null" json:"title"`
	Address     string      `json:"address"`
	WebsiteURL  string      `json:"websiteUrl"`
	MapsURL     string      `json:"mapsUrl"`
	Beds        int         `json:"beds"`
	SingleBeds  int         `json:"singleBeds"`
	DoubleBeds  int         `json:"doubleBeds"`
	Toilets     int         `json:"toilets"`
	Price       float64     `json:"price"` // total price for the stay
	Phone       string      `json:"phone"`
	UsefulInfo  string      `json:"usefulInfo"`
	Description string      `json:"description"`
	Amenities   JSONStrings `gorm:"type:jsonb" json:"amenities"`
	Images      JSONStrings `gorm:"type:jsonb" json:"images"`
	IsWinner    bool        `json:"isWinner"`
}

// LocationVote is one ranked vote (1=best) of a participant for a location.
// The weight comes from the event's VoteWeights config.
type LocationVote struct {
	Base
	EventID    uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_event_user_rank" json:"eventId"`
	UserID     uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_event_user_rank" json:"userId"`
	Rank       int       `gorm:"uniqueIndex:idx_event_user_rank" json:"rank"`
	LocationID uuid.UUID `gorm:"type:uuid;index" json:"locationId"`
}

// ProductList is a catalog of articles (e.g. "Sur les pistes") from which matrix
// tabs are populated. When EventID is nil the list belongs to the shared global
// catalog (visible on the Lists page); when set it is private to one event and
// hidden from the catalog until an organizer "saves" it (clears EventID).
type ProductList struct {
	Base
	Name     string            `gorm:"not null;index" json:"name"`
	EventID  *uuid.UUID        `gorm:"type:uuid;index" json:"eventId"` // nil = global catalog
	Voted    bool              `gorm:"default:true" json:"voted"`      // tabs created from it default to this mode
	Sections JSONStrings       `gorm:"type:jsonb" json:"sections"`
	Items    []ProductListItem `gorm:"foreignKey:ListID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// ProductListItem is one catalog entry of a ProductList.
type ProductListItem struct {
	Base
	ListID      uuid.UUID `gorm:"type:uuid;index" json:"listId"`
	Name        string    `gorm:"not null" json:"name"`
	Unit        string    `json:"unit"`
	Section     string    `json:"section"`
	QtyPerLevel JSONNum   `gorm:"type:jsonb" json:"qtyPerLevel"`
	Quantity    float64   `json:"quantity"` // default total for non-voted lists
	Position    int       `json:"position"`
}

// Unit is the canonical units referential (the Excel "data" sheet).
type Unit struct {
	Base
	Name string `gorm:"uniqueIndex;not null" json:"name"`
}

// Ingredient is the centralized ingredient referential (global rename source).
type Ingredient struct {
	Base
	CanonicalName string `gorm:"uniqueIndex;not null" json:"canonicalName"`
	DefaultUnit   string `json:"defaultUnit"`
}

// Recipe is a shared library recipe (approval-gated).
type Recipe struct {
	Base
	Name         string             `gorm:"not null" json:"name"`
	BasePersons  int                `gorm:"default:1" json:"basePersons"`
	Coefficient  float64            `gorm:"default:1" json:"coefficient"`
	PhotoURL     string             `json:"photoUrl"`
	Instructions string             `json:"instructions"`
	Kind         string             `json:"kind"`                    // legacy single category (kept in sync with tags)
	Tags         JSONStrings        `gorm:"type:jsonb" json:"tags"`  // apéro | entrée | plat | dessert | cocktail | …
	Approved     bool               `gorm:"default:false" json:"approved"`
	CreatedBy    uuid.UUID          `gorm:"type:uuid" json:"createdBy"`
	Ingredients  []RecipeIngredient `gorm:"constraint:OnDelete:CASCADE" json:"ingredients,omitempty"`
}

// RecipeIngredient is a line of a recipe.
type RecipeIngredient struct {
	Base
	RecipeID     uuid.UUID   `gorm:"type:uuid;index" json:"recipeId"`
	IngredientID uuid.UUID   `gorm:"type:uuid;index" json:"ingredientId"`
	Ingredient   *Ingredient `json:"ingredient,omitempty"`
	Quantity     float64     `json:"quantity"`
	Unit         string      `json:"unit"`
}

type MealType string

const (
	MealBreakfast MealType = "BREAKFAST"
	MealLunch     MealType = "LUNCH"
	MealDinner    MealType = "DINNER"
	MealAperitif  MealType = "APERITIF"
	MealDessert   MealType = "DESSERT"
)

// Meal is a single slot in the day x meal-type grid.
type Meal struct {
	Base
	EventID          uuid.UUID     `gorm:"type:uuid;index" json:"eventId"`
	DayIndex         int           `gorm:"not null" json:"dayIndex"` // 0-based from StartDate
	Type             MealType      `gorm:"type:varchar(16);not null" json:"type"`
	Variant          string        `json:"variant"`          // e.g. "principal" / "reloud"
	ParticipantCount *int          `json:"participantCount"` // override; nil => event default
	Recipes          []MealRecipe  `gorm:"constraint:OnDelete:CASCADE" json:"recipes,omitempty"`
	RawItems         []MealRawItem `gorm:"constraint:OnDelete:CASCADE" json:"rawItems,omitempty"`
}

// MealRecipe links a recipe to a meal slot with a per-recipe participant weighting.
type MealRecipe struct {
	Base
	MealID           uuid.UUID `gorm:"type:uuid;index" json:"mealId"`
	RecipeID         uuid.UUID `gorm:"type:uuid;index" json:"recipeId"`
	Recipe           *Recipe   `json:"recipe,omitempty"`
	ParticipantCount int       `json:"participantCount"` // how many people this recipe feeds
	Position         int       `json:"position"`
}

// MealRawItem is an ad-hoc ingredient line on a meal with no associated recipe.
type MealRawItem struct {
	Base
	MealID       uuid.UUID  `gorm:"type:uuid;index" json:"mealId"`
	IngredientID *uuid.UUID `gorm:"type:uuid" json:"ingredientId"`
	Name         string     `json:"name"`
	Quantity     float64    `json:"quantity"`
	Unit         string     `json:"unit"`
}

// ShoppingEntry stores the manual overrides for a consolidated shopping line.
// Quantities are computed on the fly; this row only persists editable metadata,
// keyed by (event, ingredient/name, unit).
type ShoppingEntry struct {
	Base
	EventID      uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_event_ing_unit" json:"eventId"`
	IngredientID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_event_ing_unit" json:"ingredientId"`
	Section      string     `gorm:"uniqueIndex:idx_event_ing_unit" json:"section"`
	Name         string     `gorm:"uniqueIndex:idx_event_ing_unit" json:"name"`
	Unit         string     `gorm:"uniqueIndex:idx_event_ing_unit" json:"unit"`
	Source       string     `json:"source"`      // Drive | Station | Ramené par X | ...
	Observation  string     `json:"observation"` // free text
	Bought       bool       `gorm:"default:false" json:"bought"`
	BroughtBy    *uuid.UUID `gorm:"type:uuid" json:"broughtBy"`
}

// AppSetting is a runtime-editable configuration entry (key/value), overriding
// the env defaults for non-bootstrap settings (SMTP, branding, origins…).
type AppSetting struct {
	Key       string    `gorm:"primaryKey;column:key" json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AllModels returns every model for AutoMigrate.
func AllModels() []any {
	return []any{
		&AppSetting{},
		&Image{},
		&User{},
		&Invite{},
		&Event{},
		&EventParticipant{},
		&EventTab{},
		&TabArticle{},
		&TabRecipe{},
		&TabConsumption{},
		&Unit{},
		&ProductList{},
		&ProductListItem{},
		&Ingredient{},
		&Recipe{},
		&RecipeIngredient{},
		&Meal{},
		&MealRecipe{},
		&MealRawItem{},
		&ShoppingEntry{},
		&Location{},
		&LocationVote{},
	}
}
