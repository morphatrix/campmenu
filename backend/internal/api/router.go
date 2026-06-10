package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/morphatrix/campmenu/internal/settings"
)

// Router wires every route with its middleware stack.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(requestLogger)
	r.Use(cors.Handler(cors.Options{
		// Origins read live from the settings store so admins can add allowed
		// URLs without a restart.
		AllowOriginFunc: func(_ *http.Request, origin string) bool {
			for _, o := range s.Settings.AllowedOrigins() {
				if o == origin {
					return true
				}
			}
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(maxBody(10 << 20))                  // cap request bodies (~10 MiB)
		r.Use(httprate.LimitByIP(600, time.Minute)) // generous global abuse limit
		r.Use(s.originGuard)                       // CSRF: reject cross-origin unsafe requests
		r.Use(s.notifyOnChange)                    // broadcast SSE tick after successful writes

		// Public branding for the frontend.
		r.Get("/config", s.handlePublicConfig)

		// Auth (rate-limited).
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(10, time.Minute))
			r.Post("/auth/login", s.handleLogin)
			r.Post("/auth/register", s.handleRegister)
			r.Post("/auth/forgot-password", s.handleForgotPassword)
			r.Post("/auth/reset-password", s.handleResetPasswordPublic)
		})
		r.Get("/auth/confirm/{token}", s.handleConfirm)
		r.Get("/invite/{code}", s.handleGetInvite)
		r.Get("/images/{id}", s.handleGetImage) // public so <img> works cross-origin

		// Authenticated.
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)

			r.Post("/auth/logout", s.handleLogout)
			r.Post("/auth/stop-impersonate", s.handleStopImpersonate)
			r.Get("/me", s.handleMe)
			r.Patch("/me", s.handleUpdateMe)

			// Real-time updates (Server-Sent Events).
			r.Get("/stream", s.handleStream)

			// Events (read).
			r.Get("/events", s.handleListEvents)
			r.Get("/events/{eventID}", s.handleGetEvent)
			r.Get("/events/{eventID}/meals", s.handleListMeals)
			r.Get("/events/{eventID}/shopping", s.handleGetShoppingList)
			r.Patch("/events/{eventID}/shopping", s.handleUpdateShoppingLine)

			// Collaborative menu planning.
			r.Patch("/meals/{mealID}", s.handleUpdateMeal)
			r.Post("/meals/{mealID}/recipes", s.handleAddMealRecipe)
			r.Patch("/meal-recipes/{id}", s.handleUpdateMealRecipe)
			r.Delete("/meal-recipes/{id}", s.handleDeleteMealRecipe)
			r.Post("/meals/{mealID}/raw-items", s.handleAddRawItem)
			r.Delete("/meal-raw-items/{id}", s.handleDeleteRawItem)

			// Matrix tabs: read grid + set own consumption.
			r.Get("/tabs/{tabID}/consumption", s.handleGetConsumption)
			r.Put("/tabs/{tabID}/articles/{articleID}/consumption", s.handleSetConsumption)

			// Locations: propose, edit own, vote (podium).
			r.Get("/events/{eventID}/locations", s.handleListLocations)
			r.Post("/events/{eventID}/locations", s.handleCreateLocation)
			r.Put("/events/{eventID}/votes", s.handleSetVotes)
			r.Patch("/locations/{id}", s.handleUpdateLocation)
			r.Delete("/locations/{id}", s.handleDeleteLocation)

			// Recipes (library).
			r.Get("/recipes", s.handleListRecipes)
			r.Get("/recipes/{id}", s.handleGetRecipe)
			r.Post("/recipes", s.handleCreateRecipe)
			r.Patch("/recipes/{id}", s.handleUpdateRecipe)
			r.Delete("/recipes/{id}", s.handleDeleteRecipe)

			// Ingredients & units referential.
			r.Get("/ingredients", s.handleListIngredients)
			r.Get("/ingredients/suggest", s.handleSuggestIngredients)
			r.Post("/ingredients", s.handleCreateIngredient)
			r.Get("/units", s.handleListUnits)

			// Image upload (returns a /api/images/{id} URL).
			r.Post("/images", s.handleUploadImage)

			// Reusable product lists (sub-lists feeding matrix tabs).
			r.Get("/product-lists", s.handleListProductLists)
			r.Post("/product-lists", s.handleCreateProductList)
			r.Patch("/product-lists/{id}", s.handleUpdateProductList)
			r.Delete("/product-lists/{id}", s.handleDeleteProductList)
			r.Post("/product-lists/{id}/items", s.handleAddListItem)
			r.Patch("/product-list-items/{itemID}", s.handleUpdateListItem)
			r.Delete("/product-list-items/{itemID}", s.handleDeleteListItem)

			// Staff (admins + collaborators): full content management.
			r.Group(func(r chi.Router) {
				r.Use(s.requireStaff)

				r.Post("/invites", s.handleCreateInvite)
				r.Get("/invites", s.handleListInvites)
				r.Post("/invites/{id}/revoke", s.handleRevokeInvite)
				r.Get("/users", s.handleListUsers) // read: see everyone + their role
				r.Post("/users/{id}/promote-collaborator", s.handlePromoteCollaborator)

				r.Post("/events", s.handleCreateEvent)
				r.Patch("/events/{eventID}", s.handleUpdateEvent)
				r.Delete("/events/{eventID}", s.handleDeleteEvent)
				r.Post("/events/{eventID}/participants", s.handleAddParticipant)
				r.Patch("/events/{eventID}/participants/{userID}", s.handleUpdateParticipant)
				r.Delete("/events/{eventID}/participants/{userID}", s.handleRemoveParticipant)

				r.Post("/locations/{id}/promote", s.handlePromoteLocation)

				r.Post("/events/{eventID}/tabs", s.handleCreateTab)
				r.Put("/events/{eventID}/tabs/order", s.handleReorderTabs)
				r.Patch("/tabs/{tabID}", s.handleUpdateTab)
				r.Delete("/tabs/{tabID}", s.handleDeleteTab)
				r.Post("/tabs/{tabID}/articles", s.handleCreateArticle)
				r.Patch("/articles/{articleID}", s.handleUpdateArticle)
				r.Delete("/articles/{articleID}", s.handleDeleteArticle)

				r.Post("/tabs/{tabID}/recipes", s.handleAddTabRecipe)
				r.Patch("/tab-recipes/{id}", s.handleUpdateTabRecipe)
				r.Delete("/tab-recipes/{id}", s.handleDeleteTabRecipe)

				r.Get("/recipes-pending", s.handleListPendingRecipes)
				r.Post("/recipes/{id}/approve", s.handleApproveRecipe)
				r.Patch("/ingredients/{id}/rename", s.handleRenameIngredient)
			})

			// Admin-only: site settings + full user account management.
			r.Group(func(r chi.Router) {
				r.Use(s.requireAdmin)

				r.Post("/users/{id}/promote", s.handlePromoteUser) // to ADMIN
				r.Post("/users/{id}/impersonate", s.handleImpersonate)
				r.Patch("/users/{id}", s.handleAdminUpdateUser)
				r.Post("/users/{id}/reset-password", s.handleAdminResetPassword)
				r.Post("/users/{id}/resend-confirmation", s.handleResendConfirmation)
				r.Post("/users/{id}/confirm", s.handleAdminConfirmUser)
				r.Delete("/users/{id}", s.handleAdminDeleteUser)

				r.Get("/settings", s.handleGetSettings)
				r.Patch("/settings", s.handleUpdateSettings)
			})
		})
	})

	return r
}

func (s *Server) handlePublicConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"siteName":       s.Settings.Get(settings.KeySiteName),
		"logoUrl":        s.Settings.Get(settings.KeyLogoURL),
		"defaultTheme":   s.Settings.Get(settings.KeyDefaultTheme),
		"defaultPalette": s.Settings.Get(settings.KeyDefaultPalette),
	})
}
