package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"expertsacademy/blog/internal/domain"
)

type LocalMedia struct{ Root string }

func (s LocalMedia) Put(ctx context.Context, key string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".upload-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
func (s LocalMedia) Open(ctx context.Context, key string) (domain.MediaFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, domain.ErrNotFound
	}
	return f, err
}
func (s LocalMedia) path(key string) (string, error) {
	// Defence in depth: storage callers cannot escape the configured root.
	clean := filepath.Clean(filepath.FromSlash(key))
	if s.Root == "" || clean == "." || filepath.IsAbs(clean) || !filepath.IsLocal(clean) {
		return "", domain.ErrInvalid
	}
	return filepath.Join(s.Root, clean), nil
}
