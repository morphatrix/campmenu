package api

import (
	"context"
	"net/http"

	"github.com/morphatrix/campmenu/internal/auth"
	"github.com/morphatrix/campmenu/internal/models"
)

const authCookieName = "campmenu_token"

// requireAuth validates the JWT (cookie first, then Authorization: Bearer)
// and injects the user id + role into the request context.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, err := r.Cookie(authCookieName); err == nil {
			token = c.Value
		} else if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
			token = h[7:]
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "authentification requise")
			return
		}
		claims, err := auth.ParseJWT(s.Cfg.JWTSecret, token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "session invalide")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxRole, claims.Role)
		ctx = context.WithValue(ctx, ctxImp, claims.Imp)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAdmin must wrap a route already behind requireAuth.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if roleFrom(r) != models.RoleAdmin {
			writeError(w, http.StatusForbidden, "réservé aux administrateurs")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireStaff allows admins and collaborators (content management).
func (s *Server) requireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !roleFrom(r).IsStaff() {
			writeError(w, http.StatusForbidden, "réservé aux administrateurs et collaborateurs")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isStaff is a per-request helper for handlers that branch on management rights.
func isStaff(r *http.Request) bool {
	return roleFrom(r).IsStaff()
}
