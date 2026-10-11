package usecase

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"expertsacademy/blog/internal/domain"
	"github.com/google/uuid"
)

type memoryMedia map[string][]byte
type mediaReader struct{ *bytes.Reader }

func (mediaReader) Close() error                                           { return nil }
func (m memoryMedia) Put(_ context.Context, key string, data []byte) error { m[key] = data; return nil }
func (m memoryMedia) Open(_ context.Context, key string) (domain.MediaFile, error) {
	data, ok := m[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return mediaReader{bytes.NewReader(data)}, nil
}
func TestImageUploadValidationAndOwnership(t *testing.T) {
	ctx := context.Background()
	s := testService(&fakeRepo{})
	s.Media = memoryMedia{}
	author := domain.User{ID: uuid.NewString(), Role: "author"}
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{255, 100, 0, 255})
	for _, kind := range []string{"png", "jpeg"} {
		var data bytes.Buffer
		if kind == "png" {
			if err := png.Encode(&data, img); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := jpeg.Encode(&data, img, nil); err != nil {
				t.Fatal(err)
			}
		}
		data.WriteString("injected payload must be removed")
		uploaded, err := s.UploadImage(ctx, author, &data)
		if err != nil || uploaded.Width != 2 || uploaded.Height != 3 {
			t.Fatal("image upload", err)
		}
		stored := s.Media.(memoryMedia)[strings.TrimPrefix(uploaded.URL, "/api/v1/media/")]
		if bytes.Contains(stored, []byte("injected payload")) {
			t.Fatal("payload persisted")
		}
		if err = s.validateCover(ctx, author, author.ID, uploaded.URL); err != nil {
			t.Fatal(err)
		}
		if err = s.validateCover(ctx, domain.User{ID: uuid.NewString(), Role: "author"}, uuid.NewString(), uploaded.URL); !errors.Is(err, domain.ErrForbidden) {
			t.Fatal("foreign cover accepted", err)
		}
		if err = s.validateCover(ctx, domain.User{ID: uuid.NewString(), Role: "admin"}, author.ID, uploaded.URL); err != nil {
			t.Fatal("admin cannot reuse cover", err)
		}
	}
	for _, invalid := range []string{"<svg onload='bad()'></svg>", "not an image", ""} {
		if _, err := s.UploadImage(ctx, author, strings.NewReader(invalid)); !errors.Is(err, domain.ErrInvalid) {
			t.Fatal("invalid image accepted", err)
		}
	}
	if _, err := s.UploadImage(ctx, author, strings.NewReader(strings.Repeat("x", int(MaxImageBytes+1)))); !errors.Is(err, domain.ErrTooLarge) {
		t.Fatal("oversized upload", err)
	}
	var oversized bytes.Buffer
	if err := png.Encode(&oversized, image.NewRGBA(image.Rect(0, 0, 8001, 1))); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UploadImage(ctx, author, &oversized); !errors.Is(err, domain.ErrTooLarge) {
		t.Fatal("oversized dimensions", err)
	}
	if _, err := s.UploadImage(ctx, domain.User{Role: "reader"}, strings.NewReader("")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal("reader uploaded", err)
	}
	for _, url := range []string{"https://evil.example/a.png", "/api/v1/media/../../file.png", "/api/v1/media/" + author.ID + "/" + uuid.NewString() + ".svg"} {
		if err := s.validateCover(ctx, author, author.ID, url); !errors.Is(err, domain.ErrInvalid) {
			t.Fatal("invalid media path accepted", url, err)
		}
	}
}

type siteRecorder struct{ slugs []string }

func (r *siteRecorder) Revalidate(_ context.Context, slugs []string) error {
	r.slugs = slugs
	return nil
}
func TestSavingPostRevalidatesOldAndNewSlugs(t *testing.T) {
	id := uuid.NewString()
	repo := &fakeRepo{post: domain.Post{AuthorID: id, Slug: "قدیمی", Status: "published"}}
	s := testService(repo)
	site := &siteRecorder{}
	s.SiteCache = site
	_, err := s.Save(context.Background(), domain.User{ID: id, Role: "author"}, PostInput{ID: uuid.NewString(), Slug: "جدید", Title: "عنوان", ContentHTML: "<p>محتوا</p>", Status: "published"})
	if err != nil || len(site.slugs) != 2 || site.slugs[0] != "جدید" || site.slugs[1] != "قدیمی" {
		t.Fatal("revalidation omitted slug", site.slugs, err)
	}
}
