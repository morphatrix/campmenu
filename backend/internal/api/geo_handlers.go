package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Map data (address search, elevation) comes from public OSM-based services.
// The browser can't call them directly — the frontend's CSP allows tile images
// from anywhere but restricts connect-src to 'self' — so they're proxied here,
// which also lets us send a proper User-Agent and respect Nominatim's rate limit.
const geoUserAgent = "CampMenu/1.0 (self-hosted trip planner)"

var geoClient = &http.Client{Timeout: 12 * time.Second}

// nominatimGate enforces Nominatim's usage policy of at most one request per
// second across all users of this instance.
var nominatimGate struct {
	mu   sync.Mutex
	last time.Time
}

func awaitNominatimSlot() {
	nominatimGate.mu.Lock()
	defer nominatimGate.mu.Unlock()
	if wait := time.Second - time.Since(nominatimGate.last); wait > 0 {
		time.Sleep(wait)
	}
	nominatimGate.last = time.Now()
}

type geoResult struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

// handleGeoSearch turns a pasted address into candidate coordinates so the user
// can jump the map there before dropping the marker on the exact roof.
func (s *Server) handleGeoSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, []geoResult{})
		return
	}
	awaitNominatimSlot()
	endpoint := "https://nominatim.openstreetmap.org/search?format=jsonv2&limit=6&addressdetails=0&q=" + url.QueryEscape(q)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "recherche impossible")
		return
	}
	req.Header.Set("User-Agent", geoUserAgent)
	req.Header.Set("Accept-Language", "fr")
	resp, err := geoClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "service de recherche indisponible")
		return
	}
	defer resp.Body.Close()
	var raw []struct {
		DisplayName string `json:"display_name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&raw); err != nil {
		writeError(w, http.StatusBadGateway, "réponse de recherche illisible")
		return
	}
	out := make([]geoResult, 0, len(raw))
	for _, it := range raw {
		lat, err1 := strconv.ParseFloat(it.Lat, 64)
		lon, err2 := strconv.ParseFloat(it.Lon, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		out = append(out, geoResult{Label: it.DisplayName, Lat: lat, Lon: lon})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGeoElevation resolves the ground altitude of a point, so a chalet can be
// compared to the resort's altitude at a glance.
func (s *Server) handleGeoElevation(w http.ResponseWriter, r *http.Request) {
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, err2 := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		writeError(w, http.StatusBadRequest, "coordonnées invalides")
		return
	}
	endpoint := fmt.Sprintf("https://api.open-meteo.com/v1/elevation?latitude=%f&longitude=%f", lat, lon)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "altitude indisponible")
		return
	}
	req.Header.Set("User-Agent", geoUserAgent)
	resp, err := geoClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "service d'altitude indisponible")
		return
	}
	defer resp.Body.Close()
	var raw struct {
		Elevation []float64 `json:"elevation"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&raw); err != nil || len(raw.Elevation) == 0 {
		writeError(w, http.StatusBadGateway, "altitude introuvable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"elevation": raw.Elevation[0]})
}
