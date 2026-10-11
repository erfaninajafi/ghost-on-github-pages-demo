package http

import (
	"errors"
	stdhttp "net/http"
	"strings"
	"time"

	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/usecase"
	"github.com/go-chi/chi/v5"
)

func (s *Server) uploadImage(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	r.Body = stdhttp.MaxBytesReader(w, r.Body, usecase.MaxImageBytes+(64<<10))
	if err := r.ParseMultipartForm(usecase.MaxImageBytes); err != nil {
		var tooLarge *stdhttp.MaxBytesError
		if errors.As(err, &tooLarge) {
			s.resultError(w, domain.ErrTooLarge)
		} else {
			s.resultError(w, domain.ErrInvalid)
		}
		return
	}
	defer r.MultipartForm.RemoveAll()
	if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["image"]) != 1 || len(r.MultipartForm.Value) != 0 {
		s.resultError(w, domain.ErrInvalid)
		return
	}
	file, err := r.MultipartForm.File["image"][0].Open()
	if err != nil {
		s.resultError(w, err)
		return
	}
	defer file.Close()
	image, err := s.Service.UploadImage(r.Context(), *current(r), file)
	if err != nil {
		s.resultError(w, err)
		return
	}
	write(w, 201, image)
}
func (s *Server) media(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	key, err := usecase.MediaKey(chi.URLParam(r, "owner"), chi.URLParam(r, "file"))
	if err != nil || s.Service.Media == nil {
		s.resultError(w, domain.ErrNotFound)
		return
	}
	file, err := s.Service.Media.Open(r.Context(), key)
	if err != nil {
		s.resultError(w, err)
		return
	}
	defer file.Close()
	kind := "image/jpeg"
	if strings.HasSuffix(key, ".png") {
		kind = "image/png"
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	stdhttp.ServeContent(w, r, key, time.Time{}, file)
}
