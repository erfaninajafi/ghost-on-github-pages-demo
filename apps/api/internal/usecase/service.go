package usecase

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"expertsacademy/blog/internal/domain"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Repo    domain.Repository
	Cache   domain.Cache
	Invite  string
	SiteURL string
	Log     *slog.Logger
}

var sanitizer = bluemonday.UGCPolicy()

func validLength(s string, min, max int) bool {
	n := utf8.RuneCountInString(s)
	return n >= min && n <= max
}
func validEmail(s string) bool {
	a, e := mail.ParseAddress(s)
	return e == nil && a.Address == s && len(s) <= 254 && !strings.ContainsAny(s, "\r\n")
}
func ValidID(id string) bool { _, e := uuid.Parse(id); return e == nil }
func ValidSlug(s string) bool {
	if !validLength(s, 1, 200) {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return false
		}
	}
	return true
}
func Normalize(s string) string {
	return strings.NewReplacer("ي", "ی", "ك", "ک").Replace(strings.TrimSpace(s))
}
func (s *Service) Signup(ctx context.Context, name, email, password, invite string) (domain.User, error) {
	if s.Invite == "" || subtle.ConstantTimeCompare([]byte(invite), []byte(s.Invite)) != 1 {
		return domain.User{}, domain.ErrForbidden
	}
	name = Normalize(name)
	email = strings.TrimSpace(email)
	if !validLength(name, 2, 100) || !validEmail(email) || len(password) < 12 || len(password) > 72 {
		return domain.User{}, domain.ErrInvalid
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(password), 12)
	if e != nil {
		return domain.User{}, e
	}
	return s.Repo.CreateUser(ctx, domain.User{Name: name, Email: email, PasswordHash: string(hash), Role: "author"})
}

// A fixed valid hash equalizes the expensive password check for unknown accounts.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-user-password"), 12)

func (s *Service) Login(ctx context.Context, email, password string) (domain.User, error) {
	u, e := s.Repo.UserByEmail(ctx, strings.TrimSpace(email))
	if e != nil {
		if !errors.Is(e, domain.ErrNotFound) {
			return domain.User{}, e
		}
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return domain.User{}, domain.ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	return u, nil
}
func (s *Service) Posts(ctx context.Context, f domain.Filter) (domain.PostPage, error) {
	if f.Page < 1 || f.Page > 10000 || f.Limit < 1 || f.Limit > 50 {
		return domain.PostPage{}, domain.ErrInvalid
	}
	return s.Repo.Posts(ctx, f)
}
func summaries(posts []domain.Post) []domain.Post {
	for i := range posts {
		posts[i].ContentHTML = ""
		posts[i].ContentJSON = json.RawMessage("{}")
	}
	return posts
}
func (s *Service) cached(ctx context.Context, key string, fetch func(context.Context) ([]domain.Post, error)) ([]domain.Post, error) {
	revision, e := s.Cache.Revision(ctx)
	usable := e == nil
	key = "blog:" + revision + ":" + key
	if usable {
		if b, e := s.Cache.Get(ctx, key); e == nil {
			var v []domain.Post
			if json.Unmarshal(b, &v) == nil {
				return v, nil
			}
		}
	}
	v, e := fetch(ctx)
	if e != nil {
		return nil, e
	}
	v = summaries(v)
	if usable {
		if b, e := json.Marshal(v); e == nil {
			if e = s.Cache.Set(ctx, key, b, 20*time.Second); e != nil {
				s.Log.Warn("cache write failed", "error", e)
			}
		}
	}
	return v, nil
}
func (s *Service) Popular(ctx context.Context) ([]domain.Post, error) {
	return s.cached(ctx, "popular", s.Repo.Popular)
}
func (s *Service) Suggest(ctx context.Context, q string) ([]domain.Post, error) {
	q = Normalize(q)
	if !validLength(q, 2, 100) {
		return []domain.Post{}, nil
	}
	return s.cached(ctx, "suggest:"+q, func(ctx context.Context) ([]domain.Post, error) { return s.Repo.Suggest(ctx, q) })
}
func (s *Service) Detail(ctx context.Context, slug string, count bool) (domain.Post, error) {
	p, e := s.Repo.PostBySlug(ctx, slug)
	if e != nil {
		return p, e
	}
	if count {
		p.ViewsCount, e = s.Repo.IncrementViews(ctx, p.ID)
		if e != nil {
			return p, e
		}
		if e = s.Cache.Rank(ctx, p.ID, p.ViewsCount); e != nil {
			s.Log.Warn("ranking unavailable", "error", e)
		}
	}
	return p, nil
}

type PostInput struct {
	ID          string          `json:"id"`
	CategoryID  *string         `json:"category_id"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	ContentHTML string          `json:"content_html"`
	ContentJSON json.RawMessage `json:"content_json"`
	Excerpt     string          `json:"excerpt"`
	Status      string          `json:"status"`
	TagIDs      []string        `json:"tag_ids"`
}

func (s *Service) Save(ctx context.Context, u domain.User, in PostInput) (domain.Post, error) {
	if u.Role != "author" && u.Role != "admin" {
		return domain.Post{}, domain.ErrForbidden
	}
	in.Title = Normalize(in.Title)
	in.Slug = Normalize(in.Slug)
	in.Excerpt = Normalize(in.Excerpt)
	if !ValidSlug(in.Slug) || !validLength(in.Title, 1, 240) || !validLength(in.Excerpt, 0, 500) || len(in.ContentHTML) > 500000 || (in.Status != "draft" && in.Status != "published") || len(in.TagIDs) > 20 {
		return domain.Post{}, domain.ErrInvalid
	}
	if in.CategoryID != nil && !ValidID(*in.CategoryID) {
		return domain.Post{}, domain.ErrInvalid
	}
	seen := map[string]bool{}
	for _, id := range in.TagIDs {
		if !ValidID(id) || seen[id] {
			return domain.Post{}, domain.ErrInvalid
		}
		seen[id] = true
	}
	if len(in.ContentJSON) == 0 {
		in.ContentJSON = json.RawMessage("{}")
	}
	if !json.Valid(in.ContentJSON) {
		return domain.Post{}, domain.ErrInvalid
	}
	author := u.ID
	if in.ID == "" {
		in.ID = uuid.NewString()
	} else {
		if !ValidID(in.ID) {
			return domain.Post{}, domain.ErrInvalid
		}
		existing, e := s.Repo.PostByID(ctx, in.ID)
		if e != nil {
			return domain.Post{}, e
		}
		if existing.AuthorID != u.ID && u.Role != "admin" {
			return domain.Post{}, domain.ErrForbidden
		}
		author = existing.AuthorID
	}
	html := sanitizer.Sanitize(in.ContentHTML)
	plain := bluemonday.StrictPolicy().Sanitize(html)
	words := len(strings.Fields(plain))
	if in.Status == "published" && words == 0 {
		return domain.Post{}, domain.ErrInvalid
	}
	minutes := int32((words + 199) / 200)
	if minutes < 1 {
		minutes = 1
	}
	p, e := s.Repo.SavePost(ctx, domain.Post{ID: in.ID, AuthorID: author, CategoryID: in.CategoryID, Slug: in.Slug, Title: in.Title, ContentHTML: html, ContentJSON: in.ContentJSON, Excerpt: in.Excerpt, Status: in.Status, ReadingTimeMinutes: minutes}, in.TagIDs)
	if e == nil {
		if err := s.Cache.Invalidate(ctx); err != nil {
			s.Log.Warn("cache invalidation failed; TTL bounds staleness", "error", err)
		}
	}
	return p, e
}
func (s *Service) publicPost(ctx context.Context, id string) (domain.Post, error) {
	if !ValidID(id) {
		return domain.Post{}, domain.ErrInvalid
	}
	p, e := s.Repo.PostByID(ctx, id)
	if e != nil {
		return p, e
	}
	if p.Status != "published" || p.PublishedAt == nil || p.PublishedAt.After(time.Now()) {
		return domain.Post{}, domain.ErrNotFound
	}
	return p, nil
}

// Tree is O(n). A child is visible only if its full ancestor chain is approved.
func Tree(rows []domain.Comment) []*domain.Comment {
	nodes := make(map[string]*domain.Comment, len(rows))
	roots := []*domain.Comment{}
	for i := range rows {
		rows[i].Replies = []*domain.Comment{}
		nodes[rows[i].ID] = &rows[i]
	}
	for i := range rows {
		c := &rows[i]
		if c.ParentID == nil {
			roots = append(roots, c)
		} else if p, ok := nodes[*c.ParentID]; ok {
			p.Replies = append(p.Replies, c)
		}
	}
	return roots
}
func (s *Service) Comments(ctx context.Context, id string) ([]*domain.Comment, error) {
	if _, e := s.publicPost(ctx, id); e != nil {
		return nil, e
	}
	rows, e := s.Repo.Comments(ctx, id)
	return Tree(rows), e
}

type CommentInput struct {
	ParentID    *string `json:"parent_id"`
	AuthorName  string  `json:"author_name"`
	AuthorEmail string  `json:"author_email"`
	Content     string  `json:"content"`
}

func (s *Service) AddComment(ctx context.Context, id string, u *domain.User, in CommentInput) (domain.Comment, error) {
	p, e := s.publicPost(ctx, id)
	if e != nil {
		return domain.Comment{}, e
	}
	in.AuthorName = Normalize(in.AuthorName)
	in.Content = strings.TrimSpace(in.Content)
	in.AuthorEmail = strings.TrimSpace(in.AuthorEmail)
	var userID *string
	if u != nil {
		userID = &u.ID
		in.AuthorName = u.Name
		in.AuthorEmail = u.Email
	}
	if !validLength(in.AuthorName, 2, 100) || !validEmail(in.AuthorEmail) || !validLength(in.Content, 2, 5000) {
		return domain.Comment{}, domain.ErrInvalid
	}
	if in.ParentID != nil {
		if !ValidID(*in.ParentID) {
			return domain.Comment{}, domain.ErrInvalid
		}
		parent, e := s.Repo.CommentByID(ctx, *in.ParentID)
		if e != nil {
			return domain.Comment{}, e
		}
		if parent.PostID != id || parent.Status != "approved" {
			return domain.Comment{}, domain.ErrInvalid
		}
		depth, e := s.Repo.CommentDepth(ctx, parent.ID)
		if e != nil {
			return domain.Comment{}, e
		}
		if depth >= 8 {
			return domain.Comment{}, domain.ErrInvalid
		}
	}
	author, e := s.Repo.UserByID(ctx, p.AuthorID)
	if e != nil {
		return domain.Comment{}, e
	}
	m := domain.Email{Recipient: author.Email, Subject: "نظر جدید در ExpertsAcademy", Body: fmt.Sprintf("نظر جدید برای «%s» در انتظار بررسی است.\n%s/studio", p.Title, s.SiteURL)}
	return s.Repo.AddComment(ctx, domain.Comment{PostID: id, UserID: userID, ParentID: in.ParentID, AuthorName: in.AuthorName, AuthorEmail: in.AuthorEmail, Content: in.Content}, m)
}
func (s *Service) Moderate(ctx context.Context, u domain.User, id, status string) (domain.Comment, error) {
	if !ValidID(id) || (status != "approved" && status != "spam" && status != "pending") {
		return domain.Comment{}, domain.ErrInvalid
	}
	c, e := s.Repo.CommentByID(ctx, id)
	if e != nil {
		return c, e
	}
	p, e := s.Repo.PostByID(ctx, c.PostID)
	if e != nil {
		return c, e
	}
	if u.Role != "admin" && (u.Role != "author" || u.ID != p.AuthorID) {
		return c, domain.ErrForbidden
	}
	var m *domain.Email
	if status == "approved" && c.ParentID != nil {
		parent, e := s.Repo.CommentByID(ctx, *c.ParentID)
		if e != nil {
			return c, e
		}
		if parent.Status != "approved" {
			return c, domain.ErrInvalid
		}
		m = &domain.Email{Recipient: parent.AuthorEmail, Subject: "پاسخ تازه به نظر شما", Body: fmt.Sprintf("پاسخی به نظر شما در «%s» منتشر شد.\n%s/blog/%s#comments", p.Title, s.SiteURL, url.PathEscape(p.Slug))}
	}
	return s.Repo.Moderate(ctx, id, status, m)
}
func (s *Service) AddTaxon(ctx context.Context, u domain.User, kind string, t domain.Taxon) (domain.Taxon, error) {
	if u.Role != "admin" {
		return t, domain.ErrForbidden
	}
	t.Name = Normalize(t.Name)
	t.Slug = Normalize(t.Slug)
	if (kind != "categories" && kind != "tags") || !validLength(t.Name, 1, 100) || !ValidSlug(t.Slug) {
		return t, domain.ErrInvalid
	}
	return s.Repo.AddTaxon(ctx, kind, t)
}
func (s *Service) DeliverEmails(ctx context.Context, m domain.Mailer) {
	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for i := 0; i < 20; i++ {
				message, e := s.Repo.ClaimEmail(ctx)
				if errors.Is(e, domain.ErrNotFound) {
					break
				}
				if e != nil {
					s.Log.Error("outbox claim failed", "error", e)
					break
				}
				sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
				e = m.Send(sendCtx, message)
				cancel()
				if e != nil {
					s.Log.Error("email delivery failed", "job_id", message.ID, "error", e)
					continue
				}
				if e = s.Repo.CompleteEmail(ctx, message.ID); e != nil {
					s.Log.Error("email completion failed", "job_id", message.ID, "error", e)
				}
			}
		}
	}
}
