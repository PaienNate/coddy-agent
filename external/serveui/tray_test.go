package serveui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionURLPointsAtTheChat(t *testing.T) {
	for _, base := range []string{"http://127.0.0.1:12345", "http://127.0.0.1:12345/"} {
		if got, want := sessionURL(base, "ses_1"), "http://127.0.0.1:12345/#/s/ses_1"; got != want {
			t.Errorf("sessionURL(%q) = %q, want %q", base, got, want)
		}
	}
	if got := sessionURL("", "ses_1"); got != "" {
		t.Errorf("sessionURL with no base = %q", got)
	}
	if got := sessionURL("http://127.0.0.1:12345/", ""); got != "http://127.0.0.1:12345" {
		t.Errorf("sessionURL with no id = %q", got)
	}
}

func TestRecentFoldersKeepsAbsoluteUniqueRowsInOrder(t *testing.T) {
	root := t.TempDir()
	one := filepath.Join(root, "one")
	two := filepath.Join(root, "two")
	got := recentFolders([]string{" " + one + " ", "relative", two, one, ""})
	want := []string{one, two}
	if len(got) != len(want) {
		t.Fatalf("recentFolders = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("recentFolders = %v, want %v", got, want)
		}
	}
}

func TestRecentFoldersStopsAtTheMenuSize(t *testing.T) {
	root := t.TempDir()
	in := make([]string, 0, maxRecentFolders+3)
	for i := 0; i < maxRecentFolders+3; i++ {
		in = append(in, filepath.Join(root, fmt.Sprintf("p%d", i)))
	}
	if got := recentFolders(in); len(got) != maxRecentFolders {
		t.Fatalf("recentFolders kept %d rows, want %d", len(got), maxRecentFolders)
	}
}

func TestFolderLabelNamesTheParentAndTheFolder(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "project")
	label := folderLabel(folder)
	if !strings.HasSuffix(label, "project") {
		t.Fatalf("folderLabel(%q) = %q", folder, label)
	}
	if !strings.Contains(label, filepath.Base(root)+"/") && !strings.Contains(label, filepath.Base(root)+`\`) {
		t.Fatalf("folderLabel(%q) = %q, want the parent name", folder, label)
	}
}

func TestFolderLabelHasNoNewlines(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "a\nb")
	if strings.ContainsAny(folderLabel(folder), "\r\n") {
		t.Fatalf("folderLabel(%q) carries a line break", folder)
	}
}
