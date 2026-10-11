package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"expertsacademy/blog/internal/domain"
	"github.com/google/uuid"
)

func TestArticleBlocksRoundTripBothFormats(t *testing.T) {
	for _, format := range []string{"plain_text", "html"} {
		t.Run(format, func(t *testing.T) {
			u := domain.User{ID: uuid.NewString(), Role: "author"}
			repo := &fakeRepo{}
			s := testService(repo)
			first := u.ID + "/" + uuid.NewString() + ".png"
			second := u.ID + "/" + uuid.NewString() + ".jpg"
			s.Media = memoryMedia{first: {1}, second: {2}}
			in := PostInput{Title: "مقاله", Slug: "مقاله", Status: "published", ContentFormat: format, ContentBlocks: []ContentBlock{
				{Type: "text", Text: "قبل <script>alert(1)</script>\r\n\r\n"},
				{Type: "image", Src: "/api/v1/media/" + first, Alt: `عکس "اول"`, Width: 320},
				{Type: "text", Text: "بعد "},
				{Type: "link", Href: "/blog/%D9%85%D9%82%D8%A7%D9%84%D9%87", Text: "مقاله دیگر"},
				{Type: "link", Href: "https://example.com/path?x=1&y=2", Text: "<b>خارجی</b>"},
				{Type: "image", Src: "/api/v1/media/" + second, Width: 960},
				{Type: "text"},
			}}
			p, err := s.Save(context.Background(), u, in)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(p.ContentHTML, "<img ") != 2 || !strings.Contains(p.ContentHTML, `width="320"`) || !strings.Contains(p.ContentHTML, `width="960"`) || strings.Contains(p.ContentHTML, "<script>") || !strings.Contains(p.ContentHTML, "https://example.com/path?x=1&amp;y=2") {
				t.Fatal("unsafe or missing article media", p.ContentHTML)
			}
			if format == "plain_text" && !strings.Contains(p.ContentHTML, "&lt;script&gt;") {
				t.Fatal("plain text interpreted as HTML")
			}
			var doc editorDocument
			if err = json.Unmarshal(p.ContentJSON, &doc); err != nil {
				t.Fatal(err)
			}
			if doc.Version != 2 || doc.Format != format || len(doc.Blocks) != len(in.ContentBlocks) || doc.Blocks[1].Width != 320 || strings.Contains(doc.Blocks[0].Text, "\r") {
				t.Fatal("editor metadata lost", doc)
			}
			// Empty text and alt fields must survive JSON for the browser's block reader.
			if !strings.Contains(string(p.ContentJSON), `"text":""`) || !strings.Contains(string(p.ContentJSON), `"alt":""`) {
				t.Fatal("empty editor fields omitted")
			}
			in.ID = p.ID
			repo.post = p
			in.ContentBlocks = doc.Blocks
			again, err := s.Save(context.Background(), u, in)
			if err != nil || again.ContentHTML != p.ContentHTML {
				t.Fatal("resaving changed article", err)
			}
			in.ContentBlocks[1].Width = 160
			resized, err := s.Save(context.Background(), u, in)
			if err != nil || !strings.Contains(resized.ContentHTML, `width="160"`) {
				t.Fatal("resized image lost", err)
			}
		})
	}
}

func TestArticleBlockValidation(t *testing.T) {
	u := domain.User{ID: uuid.NewString(), Role: "author"}
	s := testService(&fakeRepo{})
	foreign := uuid.NewString() + "/" + uuid.NewString() + ".png"
	s.Media = memoryMedia{foreign: {1}}
	in := PostInput{Title: "مقاله", Slug: "تست", Status: "draft", ContentFormat: "plain_text"}
	for _, href := range []string{"javascript:alert(1)", "data:text/html,evil", "//evil.example", "/\\evil.example", "https://user:pass@example.com", "https://example.com\n", ""} {
		in.ContentBlocks = []ContentBlock{{Type: "link", Href: href, Text: "لینک"}}
		if _, err := s.Save(context.Background(), u, in); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("unsafe URL accepted: %q, %v", href, err)
		}
	}
	for _, href := range []string{"https://example.com", "http://example.com/path?q=1#here", "/blog/آموزش"} {
		if !validArticleLink(href) {
			t.Fatal("valid link rejected", href)
		}
	}
	in.ContentBlocks = []ContentBlock{{Type: "image", Src: "/api/v1/media/" + foreign, Width: 640}}
	if _, err := s.Save(context.Background(), u, in); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal("foreign image accepted", err)
	}
	in.ContentBlocks[0].Src = "https://example.com/image.png"
	if _, err := s.Save(context.Background(), u, in); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("external image accepted", err)
	}
	for _, width := range []int{-1, 0, 79, 2001} {
		in.ContentBlocks[0].Width = width
		if _, err := s.Save(context.Background(), u, in); !errors.Is(err, domain.ErrInvalid) {
			t.Fatal("invalid image width accepted", width, err)
		}
	}
	for _, blocks := range [][]ContentBlock{{}, {{Type: "script"}}, {{Type: "text", Text: strings.Repeat("x", 500001)}}, make([]ContentBlock, 201)} {
		in.ContentBlocks = blocks
		if _, err := s.Save(context.Background(), u, in); !errors.Is(err, domain.ErrInvalid) {
			t.Fatal("invalid blocks accepted", err)
		}
	}
}
