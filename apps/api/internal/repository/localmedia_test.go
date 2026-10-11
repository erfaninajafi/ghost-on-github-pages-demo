package repository

import (
	"context"
	"errors"
	"io"
	"testing"

	"expertsacademy/blog/internal/domain"
)

func TestLocalMediaDurableStorageAndTraversal(t *testing.T) {
	root := t.TempDir()
	store := LocalMedia{Root: root}
	ctx := context.Background()
	if err := store.Put(ctx, "owner/image.png", []byte("test")); err != nil {
		t.Fatal(err)
	}
	reopened := LocalMedia{Root: root}
	file, err := reopened.Open(ctx, "owner/image.png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil || string(data) != "test" {
		t.Fatal("file did not persist", err)
	}
	for _, key := range []string{"../private", "/etc/passwd", "owner/../../private"} {
		if err := store.Put(ctx, key, nil); !errors.Is(err, domain.ErrInvalid) {
			t.Fatal("path escaped", key, err)
		}
	}
	if _, err := store.Open(ctx, "owner/missing.png"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}
