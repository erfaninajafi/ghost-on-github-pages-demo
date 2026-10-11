package usecase

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"expertsacademy/blog/internal/domain"
	"github.com/google/uuid"
)

const MaxImageBytes int64 = 5 << 20
const maxImagePixels = 12_000_000

var imageSlots = make(chan struct{}, 2)

// MediaKey accepts only server-generated identifiers, never client filenames.
func MediaKey(owner, filename string) (string, error) {
	base, ext, ok := strings.Cut(filename, ".")
	if !ok || !ValidID(owner) || !ValidID(base) || (ext != "jpg" && ext != "png") {
		return "", domain.ErrInvalid
	}
	return owner + "/" + filename, nil
}
func (s *Service) UploadImage(ctx context.Context, u domain.User, r io.Reader) (domain.Media, error) {
	if (u.Role != "author" && u.Role != "admin") || u.DisabledAt != nil {
		return domain.Media{}, domain.ErrForbidden
	}
	if s.Media == nil || !ValidID(u.ID) {
		return domain.Media{}, domain.ErrInvalid
	}
	select {
	case imageSlots <- struct{}{}:
		defer func() { <-imageSlots }()
	case <-ctx.Done():
		return domain.Media{}, ctx.Err()
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxImageBytes+1))
	if err != nil {
		return domain.Media{}, err
	}
	if int64(len(data)) > MaxImageBytes {
		return domain.Media{}, domain.ErrTooLarge
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 {
		return domain.Media{}, domain.ErrInvalid
	}
	if cfg.Width > 8000 || cfg.Height > 8000 || int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return domain.Media{}, domain.ErrTooLarge
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return domain.Media{}, domain.ErrInvalid
	}
	// Re-encoding strips metadata, scripts and arbitrary appended payloads.
	var output bytes.Buffer
	ext := "jpg"
	if format == "png" {
		ext = "png"
		err = png.Encode(&output, decoded)
	} else {
		err = jpeg.Encode(&output, decoded, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		return domain.Media{}, err
	}
	if int64(output.Len()) > MaxImageBytes {
		return domain.Media{}, domain.ErrTooLarge
	}
	key, err := MediaKey(u.ID, uuid.NewString()+"."+ext)
	if err != nil {
		return domain.Media{}, err
	}
	if err = s.Media.Put(ctx, key, output.Bytes()); err != nil {
		return domain.Media{}, err
	}
	return domain.Media{URL: "/api/v1/media/" + key, Width: cfg.Width, Height: cfg.Height}, nil
}
func (s *Service) validateCover(ctx context.Context, u domain.User, author, url string) error {
	raw, ok := strings.CutPrefix(url, "/api/v1/media/")
	if !ok {
		return domain.ErrInvalid
	}
	owner, filename, ok := strings.Cut(raw, "/")
	if !ok {
		return domain.ErrInvalid
	}
	key, err := MediaKey(owner, filename)
	if err != nil {
		return err
	}
	if u.Role != "admin" && owner != u.ID && owner != author {
		return domain.ErrForbidden
	}
	if s.Media == nil {
		return domain.ErrInvalid
	}
	file, err := s.Media.Open(ctx, key)
	if err != nil {
		return err
	}
	return file.Close()
}
