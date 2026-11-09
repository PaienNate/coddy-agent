package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenControlCreatesASession(t *testing.T) {
	home := t.TempDir()
	folder := t.TempDir()
	var gotFolder string
	stop, err := startOpenControl(context.Background(), home, "http://127.0.0.1:12345/", func(f string) (string, error) {
		gotFolder = f
		return "ses_opened", nil
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	state, err := readOpenControl(home)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(openControlPath(home)); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("state mode = %o, want 600", info.Mode().Perm())
	}

	body, _ := json.Marshal(map[string]string{"cwd": folder})
	req, err := http.NewRequest(http.MethodPost, state.Control, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+state.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %s, body = %s", res.Status, readTestBody(res.Body))
	}
	var out struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.SessionID != "ses_opened" {
		t.Fatalf("sessionId = %q", out.SessionID)
	}
	if gotFolder != folder {
		t.Fatalf("newSession folder = %q, want %q", gotFolder, folder)
	}
}

func TestOpenControlRejectsTheWrongToken(t *testing.T) {
	home := t.TempDir()
	stop, err := startOpenControl(context.Background(), home, "http://127.0.0.1:12345/", func(string) (string, error) {
		return "ses_opened", nil
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	state, err := readOpenControl(home)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"cwd": t.TempDir()})
	req, _ := http.NewRequest(http.MethodPost, state.Control, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer wrong")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %s", res.Status)
	}
}

func TestOpenControlRemovesItsState(t *testing.T) {
	home := t.TempDir()
	stop, err := startOpenControl(context.Background(), home, "http://127.0.0.1:12345/", func(string) (string, error) {
		return "ses_opened", nil
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if _, err := os.Stat(openControlPath(home)); !os.IsNotExist(err) {
		t.Fatalf("state file survived stop: %v", err)
	}
}

func TestOpenControlRejectsMissingFoldersAndFiles(t *testing.T) {
	root := t.TempDir()
	if _, err := validOpenFolder(filepath.Join(root, "missing")); err == nil {
		t.Fatal("a missing folder was accepted")
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validOpenFolder(file); err == nil {
		t.Fatal("a file was accepted as a folder")
	}
}

func readTestBody(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 4096))
	return string(b)
}
