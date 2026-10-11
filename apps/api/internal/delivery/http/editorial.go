package http

import (
	stdhttp "net/http"

	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/usecase"
	"github.com/go-chi/chi/v5"
)

func (s *Server) authors(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	v, err := s.Service.Authors(r.Context(), *current(r), number(r, "page", 1), number(r, "limit", 20))
	if err != nil {
		s.resultError(w, err)
		return
	}
	write(w, 200, v)
}
func (s *Server) createAuthor(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in usecase.AuthorInput
	if err := decode(w, r, &in); err != nil {
		s.resultError(w, err)
		return
	}
	v, err := s.Service.CreateAuthor(r.Context(), *current(r), in)
	if err != nil {
		s.resultError(w, err)
		return
	}
	// Creating an author must not replace the administrator's session cookie.
	write(w, 201, v)
}
func (s *Server) updateAuthor(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in usecase.AuthorUpdate
	if err := decode(w, r, &in); err != nil {
		s.resultError(w, err)
		return
	}
	v, err := s.Service.UpdateAuthor(r.Context(), *current(r), chi.URLParam(r, "id"), in)
	if err != nil {
		s.resultError(w, err)
		return
	}
	write(w, 200, v)
}
func (s *Server) changePostStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if err := decode(w, r, &in); err != nil {
		s.resultError(w, err)
		return
	}
	// Deletion uses the explicit DELETE route.
	if in.Status != "archived" && in.Status != "draft" {
		s.resultError(w, domain.ErrInvalid)
		return
	}
	v, err := s.Service.ChangePostStatus(r.Context(), *current(r), chi.URLParam(r, "id"), in.Status)
	if err != nil {
		s.resultError(w, err)
		return
	}
	write(w, 200, v)
}
func (s *Server) deletePost(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	_, err := s.Service.ChangePostStatus(r.Context(), *current(r), chi.URLParam(r, "id"), "deleted")
	if err != nil {
		s.resultError(w, err)
		return
	}
	w.WriteHeader(204)
}
