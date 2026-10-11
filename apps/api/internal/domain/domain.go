// Package domain defines business contracts without infrastructure dependencies.
package domain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"
)

var (
	ErrInvalid      = errors.New("invalid input")
	ErrUnauthorized = errors.New("authentication required")
	ErrForbidden    = errors.New("permission denied")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("already exists or invalid relation")
	ErrTooLarge     = errors.New("image too large")
)

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	DisabledAt   *time.Time `json:"disabled_at"`
}
type AuthorPage struct {
	Items []User `json:"items"`
	Total int64  `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
type Taxon struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
type Post struct {
	ID                 string          `json:"id"`
	AuthorID           string          `json:"author_id"`
	AuthorName         string          `json:"author_name,omitempty"`
	CategoryID         *string         `json:"category_id"`
	Slug               string          `json:"slug"`
	Title              string          `json:"title"`
	ContentHTML        string          `json:"content_html"`
	ContentJSON        json.RawMessage `json:"content_json"`
	CoverImageURL      *string         `json:"cover_image_url"`
	Excerpt            string          `json:"excerpt"`
	ReadingTimeMinutes int32           `json:"reading_time_minutes"`
	ViewsCount         int64           `json:"views_count"`
	Status             string          `json:"status"`
	PublishedAt        *time.Time      `json:"published_at"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Tags               []Taxon         `json:"tags"`
}
type Comment struct {
	ID          string     `json:"id"`
	PostID      string     `json:"post_id"`
	UserID      *string    `json:"user_id,omitempty"`
	ParentID    *string    `json:"parent_id"`
	AuthorName  string     `json:"author_name"`
	AuthorEmail string     `json:"-"`
	Content     string     `json:"content"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Replies     []*Comment `json:"replies"`
}
type Filter struct {
	Page     int
	Limit    int
	Category string
	Tag      string
}
type PostPage struct {
	Items []Post `json:"items"`
	Total int64  `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
type Email struct {
	ID        string
	Recipient string
	Subject   string
	Body      string
}
type Repository interface {
	CreateUser(context.Context, User) (User, error)
	UserByEmail(context.Context, string) (User, error)
	UserByID(context.Context, string) (User, error)
	Authors(context.Context, int, int) (AuthorPage, error)
	UpdateAuthor(context.Context, User, bool) (User, error)
	Posts(context.Context, Filter) (PostPage, error)
	Popular(context.Context) ([]Post, error)
	Suggest(context.Context, string) ([]Post, error)
	PostBySlug(context.Context, string) (Post, error)
	PostByID(context.Context, string) (Post, error)
	IncrementViews(context.Context, string) (int64, error)
	OwnPosts(context.Context, User) ([]Post, error)
	DeletedPosts(context.Context) ([]Post, error)
	ChangePostStatus(context.Context, string, string) (Post, error)
	SavePost(context.Context, Post, []string) (Post, error)
	Taxonomy(context.Context) ([]Taxon, []Taxon, error)
	AddTaxon(context.Context, string, Taxon) (Taxon, error)
	Comments(context.Context, string) ([]Comment, error)
	CommentByID(context.Context, string) (Comment, error)
	CommentDepth(context.Context, string) (int32, error)
	AddComment(context.Context, Comment, Email) (Comment, error)
	Moderate(context.Context, string, string, *Email) (Comment, error)
	ModerationComments(context.Context, User, string) ([]Comment, error)
	ClaimEmail(context.Context) (Email, error)
	CompleteEmail(context.Context, string) error
}
type Mailer interface {
	Send(context.Context, Email) error
}
type Media struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
type MediaFile interface {
	io.Reader
	io.Seeker
	io.Closer
}
type MediaStore interface {
	Put(context.Context, string, []byte) error
	Open(context.Context, string) (MediaFile, error)
}

// SiteCache refreshes the presentation layer after durable article changes.
type SiteCache interface {
	Revalidate(context.Context, []string) error
}

// Cache is optional. PostgreSQL always remains the durable source of truth.
type Cache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
	Revision(context.Context) (string, error)
	Invalidate(context.Context) error
	Rank(context.Context, string, int64) error
}
