package usecase

import (
	"context"
	"strings"

	"expertsacademy/blog/internal/domain"
)

// Administrative operations never accept a client-supplied role or author ID.
func requireAdmin(u domain.User) error {
	if u.Role != "admin" || u.DisabledAt != nil {
		return domain.ErrForbidden
	}
	return nil
}

type AuthorInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type AuthorUpdate struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`
}

func (s *Service) Authors(ctx context.Context, u domain.User, page, limit int) (domain.AuthorPage, error) {
	if err := requireAdmin(u); err != nil {
		return domain.AuthorPage{}, err
	}
	if page < 1 || page > 10000 || limit < 1 || limit > 50 {
		return domain.AuthorPage{}, domain.ErrInvalid
	}
	return s.Repo.Authors(ctx, page, limit)
}
func (s *Service) CreateAuthor(ctx context.Context, u domain.User, in AuthorInput) (domain.User, error) {
	if err := requireAdmin(u); err != nil {
		return domain.User{}, err
	}
	return s.Signup(ctx, in.Name, in.Email, in.Password)
}
func (s *Service) UpdateAuthor(ctx context.Context, u domain.User, id string, in AuthorUpdate) (domain.User, error) {
	if err := requireAdmin(u); err != nil {
		return domain.User{}, err
	}
	in.Name = Normalize(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	if !ValidID(id) || !validLength(in.Name, 2, 100) || !validEmail(in.Email) {
		return domain.User{}, domain.ErrInvalid
	}
	// The repository also constrains this update to role=author, protecting admin accounts.
	return s.Repo.UpdateAuthor(ctx, domain.User{ID: id, Name: in.Name, Email: in.Email}, in.Disabled)
}
func (s *Service) StudioPosts(ctx context.Context, u domain.User, trash bool) ([]domain.Post, error) {
	if (u.Role != "author" && u.Role != "admin") || u.DisabledAt != nil {
		return nil, domain.ErrForbidden
	}
	if trash {
		if err := requireAdmin(u); err != nil {
			return nil, err
		}
		return s.Repo.DeletedPosts(ctx)
	}
	return s.Repo.OwnPosts(ctx, u)
}

// Status changes retain content, tags, comments and authorship. Restoring always
// returns to a private draft; publication requires a separate explicit save.
func (s *Service) ChangePostStatus(ctx context.Context, u domain.User, id, status string) (domain.Post, error) {
	if err := requireAdmin(u); err != nil {
		return domain.Post{}, err
	}
	if !ValidID(id) || (status != "archived" && status != "draft" && status != "deleted") {
		return domain.Post{}, domain.ErrInvalid
	}
	p, err := s.Repo.ChangePostStatus(ctx, id, status)
	if err == nil {
		if e := s.Cache.Invalidate(ctx); e != nil {
			s.Log.Warn("cache invalidation failed; TTL bounds staleness", "error", e)
		}
		s.revalidateSite(ctx, p.Slug)
	}
	return p, err
}
func (s *Service) revalidateSite(ctx context.Context, slugs ...string) {
	if s.SiteCache == nil {
		return
	}
	unique := []string{}
	seen := map[string]bool{}
	for _, slug := range slugs {
		if slug != "" && !seen[slug] {
			unique = append(unique, slug)
			seen[slug] = true
		}
	}
	if err := s.SiteCache.Revalidate(ctx, unique); err != nil {
		s.Log.Warn("site cache revalidation failed; ISR bounds staleness", "error", err)
	}
}
