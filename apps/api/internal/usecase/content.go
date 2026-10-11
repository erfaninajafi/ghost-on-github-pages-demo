package usecase

import (
	"context"
	"encoding/json"
	"expertsacademy/blog/internal/domain"
	"html"
	"net/url"
	"strconv"
	"strings"
)

// Blocks preserve image placement and editable links without interpreting plain text as HTML.
type ContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Src   string `json:"src,omitempty"`
	Alt   string `json:"alt"`
	Width int    `json:"width,omitempty"`
	Href  string `json:"href,omitempty"`
}

func validArticleLink(value string) bool {
	if value == "" || len(value) > 2048 || strings.ContainsAny(value, "\\\r\n\t") || strings.TrimSpace(value) != value {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || u.User != nil {
		return false
	}
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//") && u.Host == "" && u.Scheme == ""
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

func (s *Service) validateContentBlocks(ctx context.Context, u domain.User, author string, in PostInput) error {
	if in.ContentBlocks == nil {
		return nil
	}
	if len(in.ContentBlocks) == 0 || len(in.ContentBlocks) > 200 || (in.ContentFormat != "plain_text" && in.ContentFormat != "html") {
		return domain.ErrInvalid
	}
	size := 0
	for _, block := range in.ContentBlocks {
		size += len(block.Text) + len(block.Src) + len(block.Alt) + len(block.Href)
		if size > 500000 {
			return domain.ErrInvalid
		}
		switch block.Type {
		case "text":
			if block.Src != "" || block.Href != "" || block.Alt != "" || block.Width != 0 {
				return domain.ErrInvalid
			}
		case "image":
			if block.Width < 80 || block.Width > 2000 || !validLength(block.Alt, 0, 300) || len(block.Src) > 500 || block.Text != "" || block.Href != "" {
				return domain.ErrInvalid
			}
			if err := s.validateCover(ctx, u, author, block.Src); err != nil {
				return err
			}
		case "link":
			if !validArticleLink(block.Href) || !validLength(strings.TrimSpace(block.Text), 1, 500) || block.Src != "" || block.Alt != "" || block.Width != 0 {
				return domain.ErrInvalid
			}
		default:
			return domain.ErrInvalid
		}
	}
	return nil
}

// editorDocument preserves the original plain text and its format for later edits.
// Public pages always render the server-generated, safe content_html.
type editorDocument struct {
	Version int            `json:"version"`
	Format  string         `json:"format"`
	Content string         `json:"content"`
	Blocks  []ContentBlock `json:"blocks,omitempty"`
}

func prepareContent(in PostInput) (string, json.RawMessage, error) {
	if in.ContentBlocks != nil {
		return prepareBlocks(in)
	}
	format := in.ContentFormat
	if format == "" {
		// Existing API clients and articles remain compatible with HTML input.
		return sanitizer.Sanitize(in.ContentHTML), in.ContentJSON, nil
	}
	var source, rendered string
	switch format {
	case "plain_text":
		source = strings.ReplaceAll(strings.ReplaceAll(in.ContentText, "\r\n", "\n"), "\r", "\n")
		paragraphs := strings.Split(source, "\n\n")
		for i, paragraph := range paragraphs {
			paragraphs[i] = "<p>" + strings.ReplaceAll(html.EscapeString(paragraph), "\n", "<br>") + "</p>"
		}
		rendered = strings.Join(paragraphs, "\n")
	case "html":
		rendered = sanitizer.Sanitize(in.ContentHTML)
		source = rendered
	default:
		return "", nil, domain.ErrInvalid
	}
	document, err := json.Marshal(editorDocument{Version: 1, Format: format, Content: source})
	return rendered, document, err
}

func prepareBlocks(in PostInput) (string, json.RawMessage, error) {
	blocks := append([]ContentBlock(nil), in.ContentBlocks...)
	var source strings.Builder
	for i, block := range blocks {
		switch block.Type {
		case "text":
			block.Text = strings.ReplaceAll(strings.ReplaceAll(block.Text, "\r\n", "\n"), "\r", "\n")
			blocks[i] = block
			if in.ContentFormat == "plain_text" {
				text := strings.ReplaceAll(html.EscapeString(block.Text), "\n\n", "</p><p>")
				source.WriteString(strings.ReplaceAll(text, "\n", "<br>"))
			} else {
				source.WriteString(block.Text)
			}
		case "image":
			source.WriteString(`<img src="` + html.EscapeString(block.Src) + `" alt="` + html.EscapeString(block.Alt) + `" width="` + strconv.Itoa(block.Width) + `">`)
		case "link":
			source.WriteString(`<a href="` + html.EscapeString(block.Href) + `">` + html.EscapeString(block.Text) + `</a>`)
		default:
			return "", nil, domain.ErrInvalid
		}
	}
	rendered := source.String()
	if in.ContentFormat == "plain_text" {
		rendered = "<p>" + rendered + "</p>"
	}
	rendered = sanitizer.Sanitize(rendered)
	document, err := json.Marshal(editorDocument{Version: 2, Format: in.ContentFormat, Blocks: blocks})
	return rendered, document, err
}
