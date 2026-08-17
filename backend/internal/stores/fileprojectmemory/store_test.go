package fileprojectmemory

import (
	"context"
	"path/filepath"
	"testing"

	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

func TestStoreRoundTrip(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := store.Get(context.Background(), serviceproject.ID("p1"))
	if err != nil {
		t.Fatalf("Get empty: %v", err)
	}
	if got.Content != "" || got.Enabled {
		t.Fatalf("empty memory = %+v", got)
	}

	content := "Use Go 1.25. Prefer table-driven tests. Never touch prod."
	saved, err := store.Set(context.Background(), serviceproject.ID("p1"), content, true)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if saved.Content != content || !saved.Enabled || saved.UpdatedAt == 0 {
		t.Fatalf("saved = %+v", saved)
	}

	got, err = store.Get(context.Background(), serviceproject.ID("p1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != content || !got.Enabled {
		t.Fatalf("got = %+v", got)
	}
}

func TestStoreIsPerProject(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := store.Set(context.Background(), serviceproject.ID("p1"), "one", true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Set(context.Background(), serviceproject.ID("p2"), "two", false); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), serviceproject.ID("p1"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "one" || !got.Enabled {
		t.Fatalf("p1 = %+v", got)
	}
	got, err = store.Get(context.Background(), serviceproject.ID("p2"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "two" || got.Enabled {
		t.Fatalf("p2 = %+v", got)
	}
}

func TestStorePersistsOnDisk(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := store.Set(context.Background(), serviceproject.ID("p1"), "persisted", true); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := reopened.Get(context.Background(), serviceproject.ID("p1"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "persisted" {
		t.Fatalf("reopened = %+v", got)
	}
	if _, err := New(filepath.Join(dir, "nested", "deep")); err != nil {
		t.Fatalf("New nested: %v", err)
	}
}
