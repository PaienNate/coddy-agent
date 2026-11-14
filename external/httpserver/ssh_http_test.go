//go:build http

package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/acp"
	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/session"
	"github.com/EvilFreelancer/coddy-agent/internal/sshcreds"
)

func newSSHTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	runner := func(context.Context, *session.State, []acp.ContentBlock, acp.UpdateSender) (string, error) {
		return "", nil
	}
	home := t.TempDir()
	cfg := &config.Config{
		Agent:  config.Agent{Model: "openai/gpt-4o"},
		Models: []config.ModelEntry{{Model: "openai/gpt-4o", MaxTokens: 100}},
		Paths:  config.Paths{Home: home},
	}
	mgr := session.NewManager(cfg, noopSender{}, runner, slog.Default(), home, nil)
	srv := New(cfg, mgr, slog.Default(), home)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, home
}

// TestSSHHostsCRUD covers the settings page's credential list: a host is
// saved, listed without its password, updated without losing the password it
// never received, and deleted.
func TestSSHHostsCRUD(t *testing.T) {
	ts, home := newSSHTestServer(t)

	put := func(body map[string]interface{}) *http.Response {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/coddy/ssh/hosts", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = res.Body.Close() })
		return res
	}

	res := put(map[string]interface{}{"host": "nas.local", "user": "admin", "password": "s3cret"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d", res.StatusCode)
	}
	var saved struct {
		Host sshHostJSON `json:"host"`
	}
	if err := json.NewDecoder(res.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if !saved.Host.HasPassword {
		t.Error("a saved password is not reported as present")
	}

	// The list never carries the secret itself.
	listRes, err := http.Get(ts.URL + "/coddy/ssh/hosts")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listRes.Body.Close() }()
	var listRaw bytes.Buffer
	if _, err := listRaw.ReadFrom(listRes.Body); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(listRaw.String(), "s3cret") {
		t.Fatalf("the list leaked the password: %s", listRaw.String())
	}
	var list struct {
		Object string        `json:"object"`
		Hosts  []sshHostJSON `json:"hosts"`
	}
	if err := json.Unmarshal(listRaw.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Object != "coddy.ssh_hosts" || len(list.Hosts) != 1 {
		t.Fatalf("list = %#v", list)
	}

	// Saving the same host without a password keeps the stored one.
	if res := put(map[string]interface{}{"host": "nas.local", "user": "admin", "key_path": "~/.ssh/id_ed25519"}); res.StatusCode != http.StatusOK {
		t.Fatalf("second PUT status = %d", res.StatusCode)
	}
	stored, ok, err := sshcreds.StoreForHome(home).Get("nas.local", "admin", 0)
	if err != nil || !ok {
		t.Fatalf("stored lookup = %v, %v", ok, err)
	}
	if stored.Password != "s3cret" {
		t.Errorf("the empty password erased the stored one: %q", stored.Password)
	}
	if stored.KeyPath != "~/.ssh/id_ed25519" {
		t.Errorf("key_path = %q", stored.KeyPath)
	}

	// A deletion reports itself, and a second one says it found nothing.
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/coddy/ssh/hosts?host=nas.local&user=admin", nil)
	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = delRes.Body.Close()
	if delRes.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d", delRes.StatusCode)
	}
	delReq2, _ := http.NewRequest(http.MethodDelete, ts.URL+"/coddy/ssh/hosts?host=nas.local&user=admin", nil)
	delRes2, err := http.DefaultClient.Do(delReq2)
	if err != nil {
		t.Fatal(err)
	}
	_ = delRes2.Body.Close()
	if delRes2.StatusCode != http.StatusNotFound {
		t.Fatalf("second DELETE status = %d, want 404", delRes2.StatusCode)
	}
}

// TestSSHHostsFileMode checks the file the API writes is private: no other
// account on the machine may read a password out of it.
func TestSSHHostsFileMode(t *testing.T) {
	ts, home := newSSHTestServer(t)
	raw, _ := json.Marshal(map[string]interface{}{"host": "nas.local", "user": "admin", "password": "s3cret"})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/coddy/ssh/hosts", bytes.NewReader(raw))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	info, err := os.Stat(filepath.Join(home, sshcreds.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("credentials file mode = %o, want 0600", mode)
	}
}

// TestSSHTestEndpointReportsFailure covers the Test button's contract: a
// failing test is a 200 with ok=false and the reason, so the page can print
// it, not an API error it would have to translate.
func TestSSHTestEndpointReportsFailure(t *testing.T) {
	ts, _ := newSSHTestServer(t)
	raw, _ := json.Marshal(map[string]interface{}{"host": "127.0.0.1", "user": "admin", "port": 1})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/coddy/ssh/test", bytes.NewReader(raw))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var body struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.OK {
		t.Fatal("an unreachable host answered ok")
	}
	if body.Message == "" {
		t.Fatal("a failed test carried no reason")
	}
}
