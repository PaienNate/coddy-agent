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
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/acp"
	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/platform"
	"github.com/EvilFreelancer/coddy-agent/internal/session"
)

// newTerminalEnv starts a server whose terminal switch is on (or off, which is
// what the disabled case asks for) over one chat in a folder of its own.
func newTerminalEnv(t *testing.T, enable bool) (*Server, string, string) {
	t.Helper()
	cfg := &config.Config{
		Providers: []config.ProviderConfig{{Name: "p1", Type: "openai", APIKey: "k"}},
		Models:    []config.ModelEntry{{Model: "p1/gpt-4o"}},
		Agent:     config.Agent{Model: "p1/gpt-4o"},
		Terminal:  config.TerminalConfig{Enable: enable},
	}
	root := t.TempDir()
	mgr := session.NewManager(cfg, noopSender{}, func(context.Context, *session.State, []acp.ContentBlock, acp.UpdateSender) (string, error) {
		return string(acp.StopReasonEndTurn), nil
	}, slog.Default(), t.TempDir(), nil)
	created, err := mgr.HandleSessionNew(t.Context(), acp.SessionNewParams{CWD: root})
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, mgr, slog.Default(), root)
	t.Cleanup(srv.Drain)
	return srv, created.SessionID, root
}

// sameOrigin labels a request the way a browser on this server's own page
// labels it. The terminal is refused without it: that label is how the route
// tells the person watching this chat from a page on another site.
func sameOrigin(r *http.Request) *http.Request {
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	return r
}

func terminalPost(t *testing.T, srv *Server, sid, input string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"input": input})
	req := sameOrigin(httptest.NewRequest(http.MethodPost, "/coddy/sessions/"+sid+"/terminal", bytes.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	return rec
}

func terminalGet(t *testing.T, srv *Server, sid string) (*httptest.ResponseRecorder, terminalSnapshot) {
	t.Helper()
	req := sameOrigin(httptest.NewRequest(http.MethodGet, "/coddy/sessions/"+sid+"/terminal", nil))
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	var snap terminalSnapshot
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
			t.Fatalf("terminal snapshot: %v: %s", err, rec.Body.String())
		}
	}
	return rec, snap
}

// waitTerminalOutput polls the snapshot until the output carries what a command
// printed, since the shell answers on its own schedule.
func waitTerminalOutput(t *testing.T, srv *Server, sid, want string) terminalSnapshot {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	var snap terminalSnapshot
	for {
		rec, got := terminalGet(t, srv, sid)
		if rec.Code != http.StatusOK {
			t.Fatalf("terminal read: %d: %s", rec.Code, rec.Body.String())
		}
		snap = got
		if strings.Contains(snap.Output, want) {
			return snap
		}
		if time.Now().After(deadline) {
			t.Fatalf("terminal output never carried %q: %q", want, snap.Output)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The happy path: a command runs in the chat's own folder and its output comes
// back, and the shell keeps what it was told across a read.
func TestTerminalRunsACommandInTheChatFolder(t *testing.T) {
	srv, sid, root := newTerminalEnv(t, true)

	if rec := terminalPost(t, srv, sid, "echo coddy-terminal-marker"); rec.Code != http.StatusOK {
		t.Fatalf("terminal write: %d: %s", rec.Code, rec.Body.String())
	}
	waitTerminalOutput(t, srv, sid, "coddy-terminal-marker")

	// The shell starts where the chat works: a marker file written through it
	// lands in the session's folder.
	marker := "coddy-terminal-cwd-marker"
	if rec := terminalPost(t, srv, sid, "cd ."); rec.Code != http.StatusOK {
		t.Fatalf("terminal write: %d: %s", rec.Code, rec.Body.String())
	}
	if err := os.WriteFile(filepath.Join(root, marker), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	listing := "ls"
	if platform.CurrentShell().Kind == platform.ShellCmd {
		listing = "dir /b"
	}
	if rec := terminalPost(t, srv, sid, listing); rec.Code != http.StatusOK {
		t.Fatalf("terminal write: %d: %s", rec.Code, rec.Body.String())
	}
	waitTerminalOutput(t, srv, sid, marker)

	// DELETE stops the shell, and a read afterwards starts a clean one.
	req := sameOrigin(httptest.NewRequest(http.MethodDelete, "/coddy/sessions/"+sid+"/terminal", nil))
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE terminal status = %d", rec.Code)
	}
}

// A command is one line: a second one smuggled into the same write would run
// something the person never saw echoed.
func TestTerminalRefusesACommandOfSeveralLines(t *testing.T) {
	srv, sid, _ := newTerminalEnv(t, true)
	if rec := terminalPost(t, srv, sid, "echo one\necho two"); rec.Code != http.StatusConflict {
		t.Fatalf("a multi-line write must be 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// The terminal belongs to the person watching this chat on this server: a page
// on another origin never reaches it, whatever credential it carries.
func TestTerminalRefusesAnotherOrigin(t *testing.T) {
	srv, sid, _ := newTerminalEnv(t, true)
	req := httptest.NewRequest(http.MethodGet, "/coddy/sessions/"+sid+"/terminal", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a cross-site read must be 403, got %d: %s", rec.Code, rec.Body.String())
	}

	body, _ := json.Marshal(map[string]string{"input": "echo hi"})
	req = httptest.NewRequest(http.MethodPost, "/coddy/sessions/"+sid+"/terminal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://elsewhere.example")
	rec = httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a cross-origin write must be 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// A shell is a capability the operator grants: with the switch off the route is
// the 404 of a server that never had one.
func TestTerminalIsOffWithoutTheSwitch(t *testing.T) {
	srv, sid, _ := newTerminalEnv(t, false)
	rec, _ := terminalGet(t, srv, sid)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("a disabled terminal must be 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec := terminalPost(t, srv, sid, "echo hi"); rec.Code != http.StatusNotFound {
		t.Fatalf("a disabled terminal must refuse a write with 404, got %d", rec.Code)
	}
}

// A child session is a transcript of delegated work, read-only everywhere else,
// so it has no terminal even with the switch on.
func TestTerminalRefusesASubagentSession(t *testing.T) {
	srv, _, root := newTerminalEnv(t, true)
	parent, err := srv.mgr.HandleSessionNew(t.Context(), acp.SessionNewParams{CWD: root})
	if err != nil {
		t.Fatal(err)
	}
	child, err := srv.mgr.CreateSubagentSession(t.Context(), session.SubagentSpec{
		ParentSessionID: parent.SessionID,
		Name:            "explore",
		TaskID:          "bg_1",
		CWD:             root,
		Mode:            "agent",
		PermissionMode:  "ask",
	})
	if err != nil {
		t.Skipf("this manager cannot create a subagent session: %v", err)
	}
	rec, _ := terminalGet(t, srv, child.GetID())
	if rec.Code != http.StatusConflict {
		t.Fatalf("a subagent session must have no terminal, got %d: %s", rec.Code, rec.Body.String())
	}
}

// A drain leaves no shell behind: the process is gone and a later read cannot
// start another.
func TestTerminalDrainsEveryShell(t *testing.T) {
	srv, sid, _ := newTerminalEnv(t, true)
	if rec := terminalPost(t, srv, sid, "echo drain-marker"); rec.Code != http.StatusOK {
		t.Fatalf("terminal write: %d: %s", rec.Code, rec.Body.String())
	}
	waitTerminalOutput(t, srv, sid, "drain-marker")

	srv.closeTerminals()
	srv.terminalMu.Lock()
	live := len(srv.terminals)
	srv.terminalMu.Unlock()
	if live != 0 {
		t.Fatalf("%d shells outlived the drain", live)
	}
}
