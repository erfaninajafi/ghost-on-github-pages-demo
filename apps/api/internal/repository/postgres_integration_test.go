package repository_test

import (
	"context"
	"errors"
	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/platform"
	"expertsacademy/blog/internal/repository"
	"expertsacademy/blog/internal/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; isolated PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		_, err := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		if err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	sql, e := os.ReadFile("../../db/schema.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, string(sql)); e != nil {
		t.Fatal(e)
	}
	return pool
}
func TestPostgresPublicationCommentsAndOutbox(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	repo := repository.New(pool)
	options, e := redis.ParseURL(os.Getenv("TEST_REDIS_URL"))
	if e != nil {
		t.Fatal("TEST_REDIS_URL required", e)
	}
	cache := platform.Redis{Client: redis.NewClient(options)}
	defer cache.Client.Close()
	s := &usecase.Service{Repo: repo, Cache: cache, Invite: "integration-invite", SiteURL: "https://blog.expertsacademy.ir", Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	author, e := s.Signup(ctx, "نویسنده", "author@example.com", "integration-password-123", "integration-invite")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Signup(ctx, "نویسنده", "AUTHOR@example.com", "integration-password-123", "integration-invite"); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("citext uniqueness", e)
	}
	logged, e := s.Login(ctx, "AUTHOR@example.com", "integration-password-123")
	if e != nil || logged.ID != author.ID {
		t.Fatal("login", e)
	}
	if _, e = s.Login(ctx, "author@example.com", "wrong-password"); !errors.Is(e, domain.ErrUnauthorized) {
		t.Fatal(e)
	}
	admin := author
	admin.Role = "admin"
	category, e := s.AddTaxon(ctx, admin, "categories", domain.Taxon{Name: "مهندسی", Slug: "engineering"})
	if e != nil {
		t.Fatal(e)
	}
	tag, e := s.AddTaxon(ctx, admin, "tags", domain.Taxon{Name: "گو", Slug: "go"})
	if e != nil {
		t.Fatal(e)
	}
	in := usecase.PostInput{Title: "آموزش مهندسی نرم‌افزار", Slug: "go-clean", ContentHTML: "<h2>شروع</h2><p>محتوای آموزشی</p><script>bad()</script>", Excerpt: "خلاصه", Status: "draft", CategoryID: &category.ID, TagIDs: []string{tag.ID}}
	p, e := s.Save(ctx, author, in)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Detail(ctx, p.Slug, false); !errors.Is(e, domain.ErrNotFound) {
		t.Fatal("draft was public", e)
	}
	in.ID = p.ID
	in.Status = "published"
	p, e = s.Save(ctx, author, in)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Tags) != 1 || p.PublishedAt == nil || strings.Contains(p.ContentHTML, "script") {
		t.Fatal("publication", p)
	}
	page, e := s.Posts(ctx, domain.Filter{Page: 1, Limit: 12, Category: "engineering", Tag: "go"})
	if e != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatal("filters", page, e)
	}
	suggestions, e := repo.Suggest(ctx, "مهندسی")
	if e != nil || len(suggestions) != 1 {
		t.Fatal("Persian search", e)
	}
	suggestions, e = repo.Suggest(ctx, "%")
	if e != nil || len(suggestions) != 0 {
		t.Fatal("wildcard escaped", e)
	}
	original := p.Title
	p.Title = "must rollback"
	if _, e = repo.SavePost(ctx, p, []string{uuid.NewString()}); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("invalid tag", e)
	}
	p, e = repo.PostByID(ctx, p.ID)
	if e != nil || p.Title != original {
		t.Fatal("post/tag transaction did not rollback", e)
	}
	root, e := s.AddComment(ctx, p.ID, nil, usecase.CommentInput{AuthorName: "مهمان", AuthorEmail: "guest@example.com", Content: "نظر نخست"})
	if e != nil {
		t.Fatal(e)
	}
	tree, e := s.Comments(ctx, p.ID)
	if e != nil || len(tree) != 0 {
		t.Fatal("pending comment was public", e)
	}
	if _, e = s.Moderate(ctx, domain.User{ID: uuid.NewString(), Role: "author"}, root.ID, "approved"); !errors.Is(e, domain.ErrForbidden) {
		t.Fatal("foreign moderator", e)
	}
	if _, e = s.Moderate(ctx, author, root.ID, "approved"); e != nil {
		t.Fatal(e)
	}
	reply, e := s.AddComment(ctx, p.ID, nil, usecase.CommentInput{ParentID: &root.ID, AuthorName: "پاسخ‌دهنده", AuthorEmail: "reply@example.com", Content: "پاسخ آموزشی"})
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 40)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Moderate(ctx, author, reply.ID, "approved")
			if err != nil {
				failures <- err
			}
		}()
	}
	wg.Wait()
	tree, e = s.Comments(ctx, p.ID)
	if e != nil || len(tree) != 1 || len(tree[0].Replies) != 1 {
		t.Fatal("recursive tree", e)
	}
	approved, err := repo.ModerationComments(ctx, author, "approved")
	if err != nil || len(approved) != 2 {
		t.Fatal("moderation status filter", len(approved), err)
	}
	var jobs int
	if e = pool.QueryRow(ctx, "SELECT count(*) FROM email_outbox").Scan(&jobs); e != nil || jobs != 3 {
		t.Fatal("transactional/idempotent outbox", jobs, e)
	}
	first, e := repo.ClaimEmail(ctx)
	if e != nil {
		t.Fatal(e)
	}
	second, e := repo.ClaimEmail(ctx)
	if e != nil || first.ID == second.ID {
		t.Fatal("outbox lease", e)
	}
	if e = repo.CompleteEmail(ctx, first.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Moderate(ctx, author, root.ID, "spam"); e != nil {
		t.Fatal(e)
	}
	tree, e = s.Comments(ctx, p.ID)
	if e != nil || len(tree) != 0 {
		t.Fatal("hidden ancestor leaked", e)
	}
	in.ID = ""
	in.Slug = "second-post"
	p2, e := s.Save(ctx, author, in)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = repo.AddComment(ctx, domain.Comment{PostID: p2.ID, ParentID: &root.ID, AuthorName: "test", AuthorEmail: "test@example.com", Content: "bad relation"}, domain.Email{}); !errors.Is(e, domain.ErrConflict) {
		t.Fatal("database cross-post invariant", e)
	}
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Detail(ctx, p.Slug, true)
			if err != nil {
				failures <- err
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	p, e = repo.PostByID(ctx, p.ID)
	if e != nil || p.ViewsCount != 30 {
		t.Fatal("lost concurrent views", p.ViewsCount, e)
	}
	score, e := cache.Client.ZScore(ctx, "blog:ranking", p.ID).Result()
	if e != nil || score != 30 {
		t.Fatal("monotonic redis ranking", score, e)
	}
	defer cache.Client.ZRem(ctx, "blog:ranking", p.ID, p2.ID)
	popular, e := s.Popular(ctx)
	if e != nil || len(popular) == 0 || popular[0].ID != p.ID {
		t.Fatal("popular", e)
	}
	if e = cache.Invalidate(ctx); e != nil {
		t.Fatal(e)
	}
}
func TestRedisRateLimitAtomic(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set")
	}
	options, e := redis.ParseURL(url)
	if e != nil {
		t.Fatal(e)
	}
	r := platform.Redis{Client: redis.NewClient(options)}
	defer r.Client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := uuid.NewString()
	defer r.Client.Del(ctx, "limit:"+key)
	for i := 0; i < 4; i++ {
		ok, e := r.Allow(ctx, key, 3)
		if e != nil || ok != (i < 3) {
			t.Fatal("limiter", i, ok, e)
		}
	}
	ttl, e := r.Client.TTL(ctx, "limit:"+key).Result()
	if e != nil || ttl <= 0 || ttl > time.Minute {
		t.Fatal("expiry", ttl, e)
	}
}

func TestMigrationRoundTrip(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	down, err := os.ReadFile("../../db/migrations/000001_init.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../db/migrations/000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_type='BASE TABLE'").Scan(&count); err != nil || count != 7 {
		t.Fatal("migration round trip", count, err)
	}
}
