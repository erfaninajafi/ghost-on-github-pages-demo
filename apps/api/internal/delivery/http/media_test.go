package http

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/repository"
	"expertsacademy/blog/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMultipartImageUploadAndPublicDelivery(t *testing.T) {
	s := serverForTest()
	s.Service = &usecase.Service{Media: repository.LocalMedia{Root: t.TempDir()}}
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("image", "../../unsafe.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(imageData.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err = form.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/v1/studio/media", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	r = r.WithContext(context.WithValue(r.Context(), userKey{}, &domain.User{ID: uuid.NewString(), Role: "author"}))
	w := httptest.NewRecorder()
	s.uploadImage(w, r)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var uploaded domain.Media
	if err = json.Unmarshal(w.Body.Bytes(), &uploaded); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(uploaded.URL, "unsafe") || uploaded.Width != 3 {
		t.Fatal("unsafe filename retained")
	}
	router := chi.NewRouter()
	router.Get("/api/v1/media/{owner}/{file}", s.media)
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest("GET", uploaded.URL, nil))
	if out.Code != 200 || out.Header().Get("Content-Type") != "image/png" || !strings.Contains(out.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("image delivery", out.Code, out.Header())
	}
	if _, err = png.Decode(bytes.NewReader(out.Body.Bytes())); err != nil {
		t.Fatal("stored image corrupted", err)
	}
}

type slugRepo struct{ domain.Repository }

func (slugRepo) PostBySlug(_ context.Context, slug string) (domain.Post, error) {
	if slug != "اولین-تست" && slug != "first-test" {
		return domain.Post{}, domain.ErrNotFound
	}
	return domain.Post{Slug: slug}, nil
}
func TestPersianSlugHTTPDecodesBeforeLookup(t *testing.T) {
	s := serverForTest()
	s.Service = &usecase.Service{Repo: slugRepo{}}
	router := chi.NewRouter()
	router.Get("/posts/{slug}", s.detail)
	for _, path := range []string{url.PathEscape("اولین-تست"), strings.ToLower(url.PathEscape("اولین-تست")), "first-test"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/posts/"+path+"?count_view=false", nil))
		if w.Code != 200 {
			t.Fatal("encoded slug returned 404", path, w.Code)
		}
	}
}
