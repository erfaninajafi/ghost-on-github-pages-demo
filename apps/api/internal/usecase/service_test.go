package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"expertsacademy/blog/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type fakeRepo struct {
	domain.Repository
	saved        domain.Post
	post         domain.Post
	parent       domain.Comment
	depth        int32
	user         domain.User
	comment      domain.Comment
	notification *domain.Email
}

func (f *fakeRepo) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	f.user = u
	return u, nil
}
func (f *fakeRepo) PostByID(context.Context, string) (domain.Post, error) { return f.post, nil }
func (f *fakeRepo) SavePost(_ context.Context, p domain.Post, _ []string) (domain.Post, error) {
	f.saved = p
	return p, nil
}
func (f *fakeRepo) CommentByID(context.Context, string) (domain.Comment, error) { return f.parent, nil }
func (f *fakeRepo) CommentDepth(context.Context, string) (int32, error)         { return f.depth, nil }
func (f *fakeRepo) UserByID(context.Context, string) (domain.User, error)       { return f.user, nil }
func (f *fakeRepo) AddComment(_ context.Context, c domain.Comment, m domain.Email) (domain.Comment, error) {
	f.comment = c
	f.notification = &m
	return c, nil
}

type fakeCache struct {
	domain.Cache
	invalidated bool
}

func (f *fakeCache) Invalidate(context.Context) error { f.invalidated = true; return nil }
func testService(repo *fakeRepo) *Service {
	return &Service{Repo: repo, Cache: &fakeCache{}, SiteURL: "https://blog.expertsacademy.ir", Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}
func TestSignupDoesNotAllowPrivilegeEscalation(t *testing.T) {
	r := &fakeRepo{}
	s := testService(r)
	u, e := s.Signup(context.Background(), "", "writer@example.com", "correct-password-123")
	if e != nil {
		t.Fatal(e)
	}
	if u.Name != "نویسنده" || u.Role != "author" || u.PasswordHash == "correct-password-123" || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("correct-password-123")) != nil {
		t.Fatal("password/role invariant failed")
	}
	b, _ := json.Marshal(u)
	if strings.Contains(string(b), "password") {
		t.Fatal("hash leaked")
	}
}
func TestPublicationSanitizationAndOwnership(t *testing.T) {
	r := &fakeRepo{}
	s := testService(r)
	u := domain.User{ID: uuid.NewString(), Role: "author"}
	in := PostInput{Title: "عنوان", Slug: "تست", ContentHTML: `<p onclick="bad()">متن</p><script>alert(1)</script><a href="javascript:bad()">پیوند</a>`, Status: "published"}
	p, e := s.Save(context.Background(), u, in)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(p.ContentHTML, "script") || strings.Contains(p.ContentHTML, "onclick") || p.ReadingTimeMinutes != 1 {
		t.Fatalf("unsafe result %s", p.ContentHTML)
	}
	if !s.Cache.(*fakeCache).invalidated {
		t.Fatal("cache not invalidated")
	}
	in.ID = uuid.NewString()
	r.post = domain.Post{AuthorID: uuid.NewString()}
	if _, e = s.Save(context.Background(), u, in); !errors.Is(e, domain.ErrForbidden) {
		t.Fatal("author edited another writer", e)
	}
	u.Role = "reader"
	in.ID = ""
	if _, e = s.Save(context.Background(), u, in); !errors.Is(e, domain.ErrForbidden) {
		t.Fatal("reader published", e)
	}
	u.Role = "author"
	in.ContentHTML = "<script>evil()</script>"
	if _, e = s.Save(context.Background(), u, in); !errors.Is(e, domain.ErrInvalid) {
		t.Fatal("empty sanitized publication", e)
	}
}
func TestReplyValidationAndPrivateEmail(t *testing.T) {
	now := time.Now().Add(-time.Hour)
	postID, parentID := uuid.NewString(), uuid.NewString()
	r := &fakeRepo{post: domain.Post{ID: postID, Status: "published", PublishedAt: &now}, parent: domain.Comment{ID: parentID, PostID: uuid.NewString(), Status: "approved"}, user: domain.User{Email: "author@example.com"}}
	s := testService(r)
	in := CommentInput{ParentID: &parentID, AuthorName: "مهمان", AuthorEmail: "guest@example.com", Content: "یک نظر مفید"}
	if _, e := s.AddComment(context.Background(), postID, nil, in); !errors.Is(e, domain.ErrInvalid) {
		t.Fatal("cross-post reply accepted", e)
	}
	r.parent.PostID = postID
	r.depth = 8
	if _, e := s.AddComment(context.Background(), postID, nil, in); !errors.Is(e, domain.ErrInvalid) {
		t.Fatal("deep reply accepted", e)
	}
	r.depth = 2
	c, e := s.AddComment(context.Background(), postID, nil, in)
	if e != nil {
		t.Fatal(e)
	}
	if r.notification.Recipient != "author@example.com" {
		t.Fatal("wrong pending notification recipient")
	}
	b, _ := json.Marshal(c)
	if strings.Contains(string(b), "guest@example.com") {
		t.Fatal("email leaked publicly")
	}
}
func TestTreeHandlesOutOfOrderChildren(t *testing.T) {
	root, child := "root", "child"
	rows := []domain.Comment{{ID: child, ParentID: &root}, {ID: root}, {ID: "grandchild", ParentID: &child}}
	tree := Tree(rows)
	if len(tree) != 1 || len(tree[0].Replies) != 1 || len(tree[0].Replies[0].Replies) != 1 {
		t.Fatalf("bad tree: %+v", tree)
	}
	if Tree(nil) == nil {
		t.Fatal("empty tree must serialize to []")
	}
}
func TestSlugAndPersianNormalization(t *testing.T) {
	for _, bad := range []string{"", "../admin", "hello world", "x?y", "<script>"} {
		if ValidSlug(bad) {
			t.Fatal(bad)
		}
	}
	if !ValidSlug("آموزش-گو-۱۲") || Normalize(" كتاب يكي ") != "کتاب یکی" {
		t.Fatal("Persian normalization failed")
	}
}

func TestPublicSignupValidation(t *testing.T) {
	s := testService(&fakeRepo{})
	for _, input := range []struct{ email, password string }{{"invalid", "correct-password-123"}, {"writer@example.com", "short"}, {"writer@example.com", strings.Repeat("x", 73)}} {
		if _, err := s.Signup(context.Background(), "", input.email, input.password); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("invalid signup accepted: %v", err)
		}
	}
}

func TestPlainTextPublicationAndEditorRoundTrip(t *testing.T) {
	r := &fakeRepo{}
	s := testService(r)
	u := domain.User{ID: uuid.NewString(), Role: "author"}
	source := "سلام <script>alert(1)</script> & متن\r\nخط دوم\r\n\r\nپاراگراف بعد"
	in := PostInput{Title: "عنوان", Slug: "متن-ساده", Status: "published", ContentFormat: "plain_text", ContentText: source, ContentHTML: "<script>must be ignored</script>", ContentJSON: json.RawMessage(`{"format":"html","content":"untrusted metadata"}`)}
	p, err := s.Save(context.Background(), u, in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.ContentHTML, "<script>") || !strings.Contains(p.ContentHTML, "&lt;script&gt;") || !strings.Contains(p.ContentHTML, "<br>خط دوم</p>\n<p>پاراگراف بعد</p>") {
		t.Fatal("unsafe or broken plain text conversion", p.ContentHTML)
	}
	var document editorDocument
	if err = json.Unmarshal(p.ContentJSON, &document); err != nil {
		t.Fatal(err)
	}
	if document.Format != "plain_text" || document.Content != strings.ReplaceAll(source, "\r\n", "\n") {
		t.Fatal("plain text source was lost", document)
	}
	in.ID = p.ID
	r.post = p
	in.ContentText = document.Content
	again, err := s.Save(context.Background(), u, in)
	if err != nil || again.ContentHTML != p.ContentHTML {
		t.Fatal("resaving changed the article", err)
	}
	in.ContentText = " \n\n "
	if _, err = s.Save(context.Background(), u, in); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("empty plain text published", err)
	}
	in.ContentFormat = "markdown"
	if _, err = s.Save(context.Background(), u, in); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("unknown format accepted", err)
	}
}

func TestExplicitHTMLStoresSanitizedEditorSource(t *testing.T) {
	rendered, metadata, err := prepareContent(PostInput{ContentFormat: "html", ContentHTML: `<h2>عنوان</h2><script>bad()</script>`})
	if err != nil || rendered != "<h2>عنوان</h2>" {
		t.Fatal(rendered, err)
	}
	var document editorDocument
	if err = json.Unmarshal(metadata, &document); err != nil {
		t.Fatal(err)
	}
	if document.Format != "html" || document.Content != rendered {
		t.Fatal("HTML editor source not sanitized")
	}
}
