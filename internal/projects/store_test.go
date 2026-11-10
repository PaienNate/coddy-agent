package projects

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/session"
)

func TestStoreCreatePersistsAndDefaultsTheName(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(t.TempDir(), "my-project")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := New(home)
	row, err := store.Create(dir, "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if row.ID == "" || row.Name != "my-project" {
		t.Fatalf("row = %+v", row)
	}
	if row.Path != session.CanonicalWorkspacePath(dir) {
		t.Fatalf("path = %q", row.Path)
	}
	if row.CreatedAt == "" || row.UpdatedAt == "" {
		t.Fatalf("timestamps missing: %+v", row)
	}

	// A second Store over the same home reads what the first wrote.
	again, err := New(home).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(again) != 1 || again[0].ID != row.ID || again[0].Path != row.Path {
		t.Fatalf("persisted rows = %+v", again)
	}
}

func TestStoreCreateRejectsBadPathsAndDuplicates(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	store := New(home)
	if _, err := store.Create("", "", ""); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("empty path error = %v", err)
	}
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(file, "", ""); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("file path error = %v", err)
	}
	if _, err := store.Create(dir, "one", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create(dir, "two", ""); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate error = %v", err)
	}
	// Another spelling of the same folder is still the same project.
	if _, err := store.Create(dir+string(filepath.Separator), "", ""); !errors.Is(err, ErrExists) {
		t.Fatalf("trailing separator error = %v", err)
	}
}

func TestStoreUpdateAndDelete(t *testing.T) {
	home := t.TempDir()
	one := t.TempDir()
	two := t.TempDir()
	store := New(home)
	a, err := store.Create(one, "one", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(two, "two", "")
	if err != nil {
		t.Fatal(err)
	}

	name := "renamed"
	pinned := true
	order := 3
	updated, err := store.Update(a.ID, Patch{
		Name:   &name,
		Pinned: &pinned,
		Order:  &order,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != name || !updated.Pinned || updated.Order != order {
		t.Fatalf("updated = %+v", updated)
	}
	if _, err := store.Update("missing", Patch{Name: &name}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing update error = %v", err)
	}
	if _, err := store.Update(b.ID, Patch{Path: &one}); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate path error = %v", err)
	}

	rows, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != a.ID || rows[1].ID != b.ID {
		t.Fatalf("list order = %+v", rows)
	}

	if err := store.Delete(a.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := store.Delete(a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete error = %v", err)
	}
	rows, err = New(home).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != b.ID {
		t.Fatalf("after delete = %+v", rows)
	}
}

func TestStoreReadsEmptyOrMissingFileAsAnEmptyList(t *testing.T) {
	home := t.TempDir()
	rows, err := New(home).List()
	if err != nil || len(rows) != 0 {
		t.Fatalf("missing file: %v %v", rows, err)
	}
	if err := os.WriteFile(filepath.Join(home, FileName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	rows, err = New(home).List()
	if err != nil || len(rows) != 0 {
		t.Fatalf("empty file: %v %v", rows, err)
	}
}
