package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The History metadata cache is keyed by session.json's file identity. A
// bundle rewritten by another process must be read again on the next listing,
// not served from the parsed copy the previous page left behind.
func TestListSnapshotsRefreshesCachedMetaFromDisk(t *testing.T) {
	fs, st := savedState(t, "sess_cached", 2)

	first, err := fs.ListSnapshotsWith(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].Title == "" {
		t.Fatalf("first listing = %+v", first)
	}

	metaPath := filepath.Join(st.SessionDir, sessionMetaFile)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	var meta SessionMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	meta.Title = "a title written by another process"
	next, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, next, 0o600); err != nil {
		t.Fatal(err)
	}

	second, err := fs.ListSnapshotsWith(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Title != meta.Title {
		t.Fatalf("second listing = %+v, want title %q", second, meta.Title)
	}
}

func TestListSnapshotsSeesASaveAfterWarmingTheCache(t *testing.T) {
	fs, st := savedState(t, "sess_saved", 1)
	if _, err := fs.ListSnapshotsWith(ListOptions{}); err != nil {
		t.Fatal(err)
	}
	st.SetTitlePinned("renamed in this process")
	if err := fs.Save(st); err != nil {
		t.Fatal(err)
	}
	rows, err := fs.ListSnapshotsWith(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Title != "renamed in this process" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestListSnapshotsDropsADeletedBundleFromTheCache(t *testing.T) {
	fs, st := savedState(t, "sess_deleted", 1)
	if _, err := fs.ListSnapshotsWith(ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(st.SessionDir); err != nil {
		t.Fatal(err)
	}
	rows, err := fs.ListSnapshotsWith(ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %+v, want none", rows)
	}
}
