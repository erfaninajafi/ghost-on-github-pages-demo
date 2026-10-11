package usecase

import (
	"encoding/json"
	"expertsacademy/blog/internal/domain"
	"html"
	"strings"
)

// editorDocument preserves the original plain text and its format for later edits.
// Public pages always render the server-generated, safe content_html.
type editorDocument struct {
	Version int    `json:"version"`
	Format  string `json:"format"`
	Content string `json:"content"`
}

func prepareContent(in PostInput) (string, json.RawMessage, error) {
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
