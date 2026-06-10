package seed

import (
	"log/slog"
	"os"
	"strings"

	"github.com/morphatrix/campmenu/internal/auth"
	"github.com/morphatrix/campmenu/internal/config"
	"github.com/morphatrix/campmenu/internal/models"
	"gorm.io/gorm"
)

// Run bootstraps the first admin and seeds reference data. Idempotent.
func Run(db *gorm.DB, cfg *config.Config) {
	bootstrapAdmin(db, cfg)
	seedUnits(db)
	seedIngredients(db)
	seedRecipeLibrary(db)
	seedProductCatalogs(db)
}

// seedProductCatalogs creates the reusable non-voted lists (Apéro, Indispensables)
// and enforces their non-voted mode + sections (corrects rows seeded before the
// GORM default-false fix).
func seedProductCatalogs(db *gorm.DB) {
	for _, c := range seedCatalogs {
		var existing models.ProductList
		if err := db.Where("LOWER(name) = LOWER(?)", c.Name).First(&existing).Error; err == nil {
			db.Model(&existing).Updates(map[string]any{"voted": false, "sections": models.JSONStrings(c.Sections)})
			continue
		}
		list := models.ProductList{Name: c.Name, Voted: false, Sections: c.Sections}
		if err := db.Create(&list).Error; err != nil {
			slog.Error("seed catalog failed", "name", c.Name, "error", err)
			continue
		}
		db.Model(&list).Update("voted", false) // GORM omits false with a default
		for i, it := range c.Items {
			db.Create(&models.ProductListItem{
				ListID: list.ID, Name: it.Name, Unit: it.Unit, Section: it.Section,
				Quantity: it.Quantity, Position: i,
			})
		}
	}
	slog.Info("product catalogs seeded")
}

// bootstrapAdmin creates the first ADMIN from env when no admin exists yet.
func bootstrapAdmin(db *gorm.DB, cfg *config.Config) {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")))
	pass := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if email == "" || pass == "" {
		return
	}
	var count int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	if count > 0 {
		return
	}
	hash, err := auth.HashPassword(pass, cfg.BcryptCost)
	if err != nil {
		slog.Error("bootstrap admin: hash failed", "error", err)
		return
	}
	admin := models.User{
		Email: email, PasswordHash: hash, Role: models.RoleAdmin,
		EmailConfirmed: true, FirstName: "Admin",
		Theme: cfg.DefaultTheme, ColorPalette: cfg.DefaultPalette, Language: "fr",
	}
	if err := db.Create(&admin).Error; err != nil {
		slog.Error("bootstrap admin: create failed", "error", err)
		return
	}
	slog.Info("bootstrap admin created", "email", email)
}

func seedUnits(db *gorm.DB) {
	for _, name := range canonicalUnits {
		db.Where(models.Unit{Name: name}).FirstOrCreate(&models.Unit{Name: name})
	}
}

func seedIngredients(db *gorm.DB) {
	for _, ing := range canonicalIngredients {
		db.Where("LOWER(canonical_name) = LOWER(?)", ing.Name).
			FirstOrCreate(&models.Ingredient{}, models.Ingredient{CanonicalName: ing.Name, DefaultUnit: ing.Unit})
	}
}

func seedRecipeLibrary(db *gorm.DB) {
	for _, sr := range seedRecipes {
		// Idempotent per recipe: add any missing one (e.g. new cocktails) without
		// duplicating those already seeded.
		var existing models.Recipe
		if err := db.Where("LOWER(name) = LOWER(?)", sr.Name).First(&existing).Error; err == nil {
			// Backfill a photo on already-seeded recipes that don't have one.
			if existing.PhotoURL == "" && recipePhotos[sr.Name] != "" {
				db.Model(&existing).Update("photo_url", recipePhotos[sr.Name])
			}
			continue
		}
		recipe := models.Recipe{
			Name: sr.Name, BasePersons: sr.Base, Coefficient: 1,
			Kind: sr.Kind, Tags: models.JSONStrings{sr.Kind}, Instructions: sr.Instr,
			PhotoURL: recipePhotos[sr.Name], Approved: true,
		}
		if err := db.Create(&recipe).Error; err != nil {
			slog.Error("seed recipe failed", "recipe", sr.Name, "error", err)
			continue
		}
		for _, si := range sr.Ingredients {
			var ing models.Ingredient
			if err := db.Where("LOWER(canonical_name) = LOWER(?)", si.Name).First(&ing).Error; err != nil {
				ing = models.Ingredient{CanonicalName: si.Name, DefaultUnit: si.Unit}
				db.Create(&ing)
			}
			db.Create(&models.RecipeIngredient{
				RecipeID: recipe.ID, IngredientID: ing.ID, Quantity: si.Qty, Unit: si.Unit,
			})
		}
	}
	slog.Info("recipe library seeded", "count", len(seedRecipes))
}
