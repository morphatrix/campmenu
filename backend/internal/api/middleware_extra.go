package api

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// requestLogger emits one structured (JSON) log line per request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", r.RemoteAddr,
		)
	})
}

// isLocalhostOrigin reports whether origin is a localhost address (any port).
// A remote page cannot forge Origin: localhost, so allowing it is safe even in
// production while keeping local development friction-free.
func isLocalhostOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	h := u.Hostname()
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// originGuard is a lightweight CSRF defense: unsafe cross-origin requests must
// carry an Origin header that matches the configured allow-list. Browsers set
// Origin automatically and cannot forge it, while same-origin and non-browser
// clients are unaffected.
func (s *Server) originGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin == "" || isLocalhostOrigin(origin) {
			next.ServeHTTP(w, r)
			return
		}
		// Allow-list read live from the settings store (editable in admin UI).
		allowed := map[string]bool{}
		for _, o := range s.Settings.AllowedOrigins() {
			allowed[o] = true
		}
		if allowed[origin] {
			next.ServeHTTP(w, r)
			return
		}
		// Same-origin requests are safe regardless of the configured allow-list:
		// accept when the Origin host matches the request Host (works behind the
		// ingress and via a direct port-forward, since the proxy forwards Host).
		if u, err := url.Parse(origin); err == nil && u.Host == r.Host {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusForbidden, "origine non autorisée")
	})
}
