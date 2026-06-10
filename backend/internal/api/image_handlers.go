package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
)

const maxImageBytes = 5 << 20 // 5 MiB

// handleUploadImage accepts a multipart "file" field, stores it in the DB and
// returns a relative URL (/api/images/{id}) for use as a recipe/profile photo.
func (s *Server) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+1024)
	if err := r.ParseMultipartForm(maxImageBytes + 1024); err != nil {
		writeError(w, http.StatusBadRequest, "fichier trop volumineux (max 5 Mo)")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "champ 'file' manquant")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		writeError(w, http.StatusBadRequest, "le fichier doit être une image")
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "lecture du fichier impossible")
		return
	}
	if len(data) > maxImageBytes {
		writeError(w, http.StatusBadRequest, "fichier trop volumineux (max 5 Mo)")
		return
	}

	img := models.Image{ContentType: contentType, Data: data}
	if err := s.DB.Create(&img).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "enregistrement impossible")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"url": "/api/images/" + img.ID.String()})
}

// handleGetImage serves a stored image (public so <img> tags work everywhere).
func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var img models.Image
	if err := s.DB.First(&img, "id = ?", id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", img.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(img.Data)
}
