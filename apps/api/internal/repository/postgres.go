package repository

import (
	"context"
	"errors"
	"strings"

	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/repository/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool *pgxpool.Pool
	q    *db.Queries
}

func New(pool *pgxpool.Pool) *Postgres { return &Postgres{pool, db.New(pool)} }
func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505", "23503":
			return domain.ErrConflict
		case "23514", "22P02":
			return domain.ErrInvalid
		}
	}
	return err
}
func user(v db.User) domain.User {
	return domain.User{ID: v.ID, Email: v.Email, Name: v.Name, PasswordHash: v.PasswordHash, Role: v.Role}
}
func post(v db.Post) domain.Post {
	return domain.Post{ID: v.ID, AuthorID: v.AuthorID, CategoryID: v.CategoryID, Slug: v.Slug, Title: v.Title, ContentHTML: v.ContentHtml, ContentJSON: v.ContentJson, Excerpt: v.Excerpt, ReadingTimeMinutes: v.ReadingTimeMinutes, ViewsCount: v.ViewsCount, Status: v.Status, PublishedAt: v.PublishedAt, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, Tags: []domain.Taxon{}}
}
func posts(v []db.Post) []domain.Post {
	out := make([]domain.Post, 0, len(v))
	for _, p := range v {
		out = append(out, post(p))
	}
	return out
}
func comment(v db.Comment) domain.Comment {
	return domain.Comment{ID: v.ID, PostID: v.PostID, UserID: v.UserID, ParentID: v.ParentID, AuthorName: v.AuthorName, AuthorEmail: v.AuthorEmail, Content: v.Content, Status: v.Status, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, Replies: []*domain.Comment{}}
}
func (r *Postgres) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	v, e := r.q.CreateUser(ctx, db.CreateUserParams{Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: u.Role})
	return user(v), translate(e)
}
func (r *Postgres) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	v, e := r.q.UserByEmail(ctx, email)
	return user(v), translate(e)
}
func (r *Postgres) UserByID(ctx context.Context, id string) (domain.User, error) {
	v, e := r.q.UserByID(ctx, id)
	return user(v), translate(e)
}
func (r *Postgres) Posts(ctx context.Context, f domain.Filter) (domain.PostPage, error) {
	v, e := r.q.ListPosts(ctx, db.ListPostsParams{Category: f.Category, Tag: f.Tag, PageLimit: int32(f.Limit), PageOffset: int32((f.Page - 1) * f.Limit)})
	if e != nil {
		return domain.PostPage{}, translate(e)
	}
	total, e := r.q.CountPosts(ctx, db.CountPostsParams{Category: f.Category, Tag: f.Tag})
	return domain.PostPage{Items: posts(v), Total: total, Page: f.Page, Limit: f.Limit}, translate(e)
}
func (r *Postgres) Popular(ctx context.Context) ([]domain.Post, error) {
	v, e := r.q.PopularPosts(ctx)
	return posts(v), translate(e)
}
func (r *Postgres) Suggest(ctx context.Context, s string) ([]domain.Post, error) {
	s = strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(s)
	v, e := r.q.SuggestPosts(ctx, "%"+s+"%")
	return posts(v), translate(e)
}
func (r *Postgres) enrich(ctx context.Context, p db.Post, e error) (domain.Post, error) {
	if e != nil {
		return domain.Post{}, translate(e)
	}
	v := post(p)
	tags, e := r.q.PostTags(ctx, p.ID)
	for _, t := range tags {
		v.Tags = append(v.Tags, domain.Taxon{ID: t.ID, Name: t.Name, Slug: t.Slug})
	}
	return v, translate(e)
}
func (r *Postgres) PostBySlug(ctx context.Context, s string) (domain.Post, error) {
	v, e := r.q.PostBySlug(ctx, s)
	return r.enrich(ctx, v, e)
}
func (r *Postgres) PostByID(ctx context.Context, id string) (domain.Post, error) {
	v, e := r.q.PostByID(ctx, id)
	return r.enrich(ctx, v, e)
}
func (r *Postgres) IncrementViews(ctx context.Context, id string) (int64, error) {
	v, e := r.q.IncrementViews(ctx, id)
	return v, translate(e)
}
func (r *Postgres) OwnPosts(ctx context.Context, u domain.User) ([]domain.Post, error) {
	v, e := r.q.OwnPosts(ctx, db.OwnPostsParams{IsAdmin: u.Role == "admin", AuthorID: u.ID})
	if e != nil {
		return nil, translate(e)
	}
	out := make([]domain.Post, 0, len(v))
	for _, p := range v {
		item, err := r.enrich(ctx, p, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
func (r *Postgres) SavePost(ctx context.Context, p domain.Post, tags []string) (domain.Post, error) {
	tx, e := r.Pool.Begin(ctx)
	if e != nil {
		return domain.Post{}, e
	}
	defer tx.Rollback(ctx)
	q := r.q.WithTx(tx)
	v, e := q.SavePost(ctx, db.SavePostParams{ID: p.ID, AuthorID: p.AuthorID, CategoryID: p.CategoryID, Slug: p.Slug, Title: p.Title, ContentHtml: p.ContentHTML, ContentJson: p.ContentJSON, Excerpt: p.Excerpt, ReadingTimeMinutes: p.ReadingTimeMinutes, Status: p.Status})
	if e != nil {
		return domain.Post{}, translate(e)
	}
	if e = q.ClearPostTags(ctx, p.ID); e != nil {
		return domain.Post{}, e
	}
	for _, id := range tags {
		if e = q.AddPostTag(ctx, db.AddPostTagParams{PostID: p.ID, TagID: id}); e != nil {
			return domain.Post{}, translate(e)
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return domain.Post{}, e
	}
	return r.enrich(ctx, v, nil)
}
func (r *Postgres) Taxonomy(ctx context.Context) ([]domain.Taxon, []domain.Taxon, error) {
	cats := []domain.Taxon{}
	tags := []domain.Taxon{}
	c, e := r.q.Categories(ctx)
	if e != nil {
		return nil, nil, e
	}
	t, e := r.q.Tags(ctx)
	if e != nil {
		return nil, nil, e
	}
	for _, v := range c {
		cats = append(cats, domain.Taxon{ID: v.ID, Name: v.Name, Slug: v.Slug})
	}
	for _, v := range t {
		tags = append(tags, domain.Taxon{ID: v.ID, Name: v.Name, Slug: v.Slug})
	}
	return cats, tags, nil
}
func (r *Postgres) AddTaxon(ctx context.Context, kind string, t domain.Taxon) (domain.Taxon, error) {
	if kind == "categories" {
		v, e := r.q.AddCategory(ctx, db.AddCategoryParams{Name: t.Name, Slug: t.Slug})
		return domain.Taxon{ID: v.ID, Name: v.Name, Slug: v.Slug}, translate(e)
	}
	v, e := r.q.AddTag(ctx, db.AddTagParams{Name: t.Name, Slug: t.Slug})
	return domain.Taxon{ID: v.ID, Name: v.Name, Slug: v.Slug}, translate(e)
}
func (r *Postgres) Comments(ctx context.Context, id string) ([]domain.Comment, error) {
	v, e := r.q.CommentTree(ctx, id)
	out := make([]domain.Comment, 0, len(v))
	for _, c := range v {
		out = append(out, comment(db.Comment(c)))
	}
	return out, translate(e)
}
func (r *Postgres) CommentByID(ctx context.Context, id string) (domain.Comment, error) {
	v, e := r.q.CommentByID(ctx, id)
	return comment(v), translate(e)
}
func (r *Postgres) CommentDepth(ctx context.Context, id string) (int32, error) {
	v, e := r.q.CommentDepth(ctx, id)
	return v, translate(e)
}
func enqueue(ctx context.Context, q *db.Queries, m domain.Email) error {
	return q.EnqueueEmail(ctx, db.EnqueueEmailParams{Recipient: m.Recipient, Subject: m.Subject, Body: m.Body})
}
func (r *Postgres) AddComment(ctx context.Context, c domain.Comment, m domain.Email) (domain.Comment, error) {
	tx, e := r.Pool.Begin(ctx)
	if e != nil {
		return domain.Comment{}, e
	}
	defer tx.Rollback(ctx)
	q := r.q.WithTx(tx)
	v, e := q.AddComment(ctx, db.AddCommentParams{PostID: c.PostID, UserID: c.UserID, ParentID: c.ParentID, AuthorName: c.AuthorName, AuthorEmail: c.AuthorEmail, Content: c.Content})
	if e != nil {
		return domain.Comment{}, translate(e)
	}
	if e = enqueue(ctx, q, m); e != nil {
		return domain.Comment{}, e
	}
	e = tx.Commit(ctx)
	return comment(v), e
}
func (r *Postgres) Moderate(ctx context.Context, id, status string, m *domain.Email) (domain.Comment, error) {
	tx, e := r.Pool.Begin(ctx)
	if e != nil {
		return domain.Comment{}, e
	}
	defer tx.Rollback(ctx)
	q := r.q.WithTx(tx)
	// Serialize moderators so retries cannot create duplicate notification jobs.
	var previous string
	if e = tx.QueryRow(ctx, "SELECT status FROM comments WHERE id=$1 FOR UPDATE", id).Scan(&previous); e != nil {
		return domain.Comment{}, translate(e)
	}
	v, e := q.ModerateComment(ctx, db.ModerateCommentParams{ID: id, Status: status})
	if e != nil {
		return domain.Comment{}, translate(e)
	}
	if m != nil && previous != "approved" {
		if e = enqueue(ctx, q, *m); e != nil {
			return domain.Comment{}, e
		}
	}
	e = tx.Commit(ctx)
	return comment(v), e
}
func (r *Postgres) ModerationComments(ctx context.Context, u domain.User, status string) ([]domain.Comment, error) {
	v, e := r.q.ModerationComments(ctx, db.ModerationCommentsParams{IsAdmin: u.Role == "admin", AuthorID: u.ID, Status: status})
	out := make([]domain.Comment, 0, len(v))
	for _, c := range v {
		out = append(out, comment(c))
	}
	return out, translate(e)
}
func (r *Postgres) ClaimEmail(ctx context.Context) (domain.Email, error) {
	v, e := r.q.ClaimEmail(ctx)
	return domain.Email{ID: v.ID, Recipient: v.Recipient, Subject: v.Subject, Body: v.Body}, translate(e)
}
func (r *Postgres) CompleteEmail(ctx context.Context, id string) error {
	return r.q.CompleteEmail(ctx, id)
}
