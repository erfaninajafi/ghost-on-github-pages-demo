package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"expertsacademy/blog/internal/domain"
	"github.com/google/uuid"
)

func (f *fakeRepo) UserByEmail(context.Context, string) (domain.User, error) { return f.user, nil }
func (f *fakeRepo) ChangePostStatus(_ context.Context, id, status string) (domain.Post, error) {
	f.post.ID = id
	f.post.Status = status
	return f.post, nil
}
func TestAdministrativeOperationsDenyNonAdmins(t *testing.T) {
	ctx := context.Background()
	s := testService(&fakeRepo{})
	id := uuid.NewString()
	now := time.Now()
	for _, u := range []domain.User{{ID: id, Role: "author"}, {ID: id, Role: "reader"}, {Role: "admin", DisabledAt: &now}} {
		if _, e := s.Authors(ctx, u, 1, 20); !errors.Is(e, domain.ErrForbidden) {
			t.Fatal("authors", u, e)
		}
		if _, e := s.CreateAuthor(ctx, u, AuthorInput{}); !errors.Is(e, domain.ErrForbidden) {
			t.Fatal("create author", u, e)
		}
		if _, e := s.UpdateAuthor(ctx, u, id, AuthorUpdate{}); !errors.Is(e, domain.ErrForbidden) {
			t.Fatal("update author", u, e)
		}
		for _, status := range []string{"archived", "deleted", "draft"} {
			if _, e := s.ChangePostStatus(ctx, u, id, status); !errors.Is(e, domain.ErrForbidden) {
				t.Fatal("post lifecycle", u, status, e)
			}
		}
		if _, e := s.StudioPosts(ctx, u, true); !errors.Is(e, domain.ErrForbidden) {
			t.Fatal("trash", u, e)
		}
		if _, e := s.Moderate(ctx, u, id, "approved"); !errors.Is(e, domain.ErrForbidden) {
			t.Fatal("moderation", u, e)
		}
	}
}
func TestAdminCreatesAuthorAndKeepsPostOwnership(t *testing.T) {
	ctx := context.Background()
	r := &fakeRepo{}
	s := testService(r)
	admin := domain.User{ID: uuid.NewString(), Role: "admin"}
	author, err := s.CreateAuthor(ctx, admin, AuthorInput{Name: "نویسنده تازه", Email: "new@example.com", Password: "correct-password-123"})
	if err != nil || author.Role != "author" {
		t.Fatal("author creation", author.Role, err)
	}
	input := PostInput{Title: "عنوان", Slug: "admin-post", Status: "published", ContentHTML: "<p>محتوا</p>"}
	p, err := s.Save(ctx, admin, input)
	if err != nil || p.AuthorID != admin.ID {
		t.Fatal("admin's own article", err)
	}
	otherID := uuid.NewString()
	r.post = domain.Post{AuthorID: otherID, Status: "published"}
	input.ID = uuid.NewString()
	p, err = s.Save(ctx, admin, input)
	if err != nil || p.AuthorID != otherID {
		t.Fatal("admin reassigned authorship", err)
	}
	for _, status := range []string{"archived", "deleted", "draft"} {
		p, err = s.ChangePostStatus(ctx, admin, input.ID, status)
		if err != nil || p.Status != status || !s.Cache.(*fakeCache).invalidated {
			t.Fatal("status change", status, err)
		}
	}
	if _, err = s.ChangePostStatus(ctx, admin, input.ID, "published"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("implicit publication accepted", err)
	}
}
func TestArchivedAndDeletedPostsCannotBeRepublishedBySave(t *testing.T) {
	ctx := context.Background()
	id := uuid.NewString()
	r := &fakeRepo{post: domain.Post{AuthorID: id}}
	s := testService(r)
	for _, role := range []string{"author", "admin"} {
		for _, status := range []string{"archived", "deleted"} {
			r.post.Status = status
			_, err := s.Save(ctx, domain.User{ID: id, Role: role}, PostInput{ID: uuid.NewString(), Title: "عنوان", Slug: "locked", Status: "published", ContentHTML: "<p>محتوا</p>"})
			if !errors.Is(err, domain.ErrConflict) {
				t.Fatal(role, status, err)
			}
		}
	}
}
func TestDisabledAuthorCannotLoginOrPublish(t *testing.T) {
	ctx := context.Background()
	r := &fakeRepo{}
	s := testService(r)
	u, err := s.Signup(ctx, "", "disabled@example.com", "correct-password-123")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	u.DisabledAt = &now
	r.user = u
	if _, err = s.Login(ctx, u.Email, "correct-password-123"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatal("disabled login", err)
	}
	if _, err = s.Save(ctx, u, PostInput{}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal("disabled publication", err)
	}
}
