package repository_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/platform"
	"expertsacademy/blog/internal/repository"
	"expertsacademy/blog/internal/usecase"
	"github.com/redis/go-redis/v9"
)

func TestEditorialOwnershipAndLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := integrationPool(t)
	repo := repository.New(pool)
	options, err := redis.ParseURL(os.Getenv("TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cache := platform.Redis{Client: redis.NewClient(options)}
	defer cache.Client.Close()
	s := &usecase.Service{Repo: repo, Cache: cache, SiteURL: "http://localhost:3000", Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	admin, err := repo.CreateUser(ctx, domain.User{Name: "مدیر", Email: "admin@example.com", Role: "admin", PasswordHash: "unused"})
	if err != nil {
		t.Fatal(err)
	}
	author, err := s.CreateAuthor(ctx, admin, usecase.AuthorInput{Name: "نویسنده اول", Email: "first@example.com", Password: "integration-password-123"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.Signup(ctx, "نویسنده دوم", "second@example.com", "integration-password-123")
	if err != nil {
		t.Fatal(err)
	}
	authors, err := s.Authors(ctx, admin, 1, 20)
	if err != nil || authors.Total != 2 || len(authors.Items) != 2 {
		t.Fatal("author pagination", authors, err)
	}
	if _, err = s.UpdateAuthor(ctx, admin, admin.ID, usecase.AuthorUpdate{Name: "نام جدید", Email: "admin@example.com", Disabled: true}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("admin account was modified", err)
	}
	changed, err := s.UpdateAuthor(ctx, admin, author.ID, usecase.AuthorUpdate{Name: "نویسنده ویرایش‌شده", Email: "updated@example.com", Disabled: true})
	if err != nil || changed.DisabledAt == nil {
		t.Fatal("disable author", err)
	}
	if _, err = s.Login(ctx, "updated@example.com", "integration-password-123"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatal("disabled author logged in", err)
	}
	author, err = s.UpdateAuthor(ctx, admin, author.ID, usecase.AuthorUpdate{Name: changed.Name, Email: changed.Email})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Login(ctx, author.Email, "integration-password-123"); err != nil {
		t.Fatal("reactivation", err)
	}
	input := usecase.PostInput{Title: "مقاله نویسنده", Slug: "author-article", ContentFormat: "plain_text", ContentText: "محتوای کامل مقاله", Status: "published"}
	article, err := s.Save(ctx, author, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ID = article.ID
	if _, err = s.Save(ctx, other, input); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal("foreign article edited", err)
	}
	if _, err = s.Save(ctx, admin, input); err != nil {
		t.Fatal("admin editing another author", err)
	}
	stored, err := repo.PostByID(ctx, article.ID)
	if err != nil || stored.AuthorID != author.ID {
		t.Fatal("authorship changed", err)
	}
	input.ID = ""
	input.Slug = "admin-article"
	own, err := s.Save(ctx, admin, input)
	if err != nil || own.AuthorID != admin.ID {
		t.Fatal("admin own article", err)
	}
	for _, post := range []domain.Post{article, own} {
		for _, action := range []string{"archived", "deleted"} {
			if _, err = s.ChangePostStatus(ctx, author, post.ID, action); !errors.Is(err, domain.ErrForbidden) {
				t.Fatal("author performed admin operation", action, err)
			}
		}
		if _, err = s.ChangePostStatus(ctx, admin, post.ID, "archived"); err != nil {
			t.Fatal(err)
		}
		if _, err = s.Detail(ctx, post.Slug, false); !errors.Is(err, domain.ErrNotFound) {
			t.Fatal("archived article was public", err)
		}
		suggestions, err := s.Suggest(ctx, "مقاله")
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range suggestions {
			if p.ID == post.ID {
				t.Fatal("archive leaked into suggestions")
			}
		}
		authorPosts, err := s.StudioPosts(ctx, author, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range authorPosts {
			if p.ID == post.ID {
				t.Fatal("author can access archived article")
			}
		}
		adminPosts, err := s.StudioPosts(ctx, admin, false)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range adminPosts {
			if p.ID == post.ID && p.Status == "archived" {
				found = true
			}
		}
		if !found {
			t.Fatal("archive missing for admin")
		}
		if _, err = s.ChangePostStatus(ctx, admin, post.ID, "deleted"); err != nil {
			t.Fatal(err)
		}
		if _, err = s.Detail(ctx, post.Slug, false); !errors.Is(err, domain.ErrNotFound) {
			t.Fatal("deleted article was public", err)
		}
		trash, err := s.StudioPosts(ctx, admin, true)
		if err != nil || len(trash) != 1 || trash[0].ID != post.ID {
			t.Fatal("trash", trash, err)
		}
		if _, err = s.StudioPosts(ctx, author, true); !errors.Is(err, domain.ErrForbidden) {
			t.Fatal("author accessed trash", err)
		}
		if _, err = s.ChangePostStatus(ctx, admin, post.ID, "draft"); err != nil {
			t.Fatal(err)
		}
		restored, err := repo.PostByID(ctx, post.ID)
		if err != nil || restored.Status != "draft" || restored.AuthorID != post.AuthorID || restored.ContentHTML != post.ContentHTML {
			t.Fatal("restore lost data", err)
		}
		if _, err = s.Detail(ctx, post.Slug, false); !errors.Is(err, domain.ErrNotFound) {
			t.Fatal("restore published automatically", err)
		}
	}
}
