//go:build http

package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/acp"
)

// TestPromptSuggestionRoutes covers the composer's two doors to the
// suggestion side call: the prediction itself and the outcome report. The
// call's own behaviour - the fork, the gates, the back-off - is covered by
// the session package; this holds the wiring.
func TestPromptSuggestionRoutes(t *testing.T) {
	mgr, srv, root := testHTTPServerPersist(t)
	created, err := mgr.HandleSessionNew(context.Background(), acp.SessionNewParams{CWD: root})
	if err != nil {
		t.Fatal(err)
	}
	sid := created.SessionID
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	get := func(path string) (*http.Response, map[string]string) {
		t.Helper()
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		if strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
			_ = json.Unmarshal(body, &out)
		}
		return res, out
	}

	// A session with nothing predicted yet: 200 with an empty text, never a 500.
	res, body := get("/coddy/sessions/" + sid + "/prompt-suggestion")
	if res.StatusCode != http.StatusOK || body["object"] != "coddy.prompt_suggestion" || body["text"] != "" {
		t.Fatalf("status=%d body=%v", res.StatusCode, body)
	}

	// The switch off: the same empty answer, whatever the session holds.
	off := false
	cfg := srv.activeCfg()
	cfg.Agent.PromptSuggestions = &off
	res, body = get("/coddy/sessions/" + sid + "/prompt-suggestion")
	if res.StatusCode != http.StatusOK || body["text"] != "" {
		t.Fatalf("disabled: status=%d body=%v", res.StatusCode, body)
	}
	on := true
	cfg.Agent.PromptSuggestions = &on

	// An unknown session is a 404 while the feature is on.
	if res, _ = get("/coddy/sessions/sess_missing/prompt-suggestion"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown session status=%d", res.StatusCode)
	}
	// A malformed id never reaches the manager.
	if res, _ = get("/coddy/sessions/..%2Fescape/prompt-suggestion"); res.StatusCode == http.StatusOK {
		t.Fatal("malformed id accepted")
	}

	// The outcome report: accepted resets the streak, and a bad body is refused.
	post := func(payload string) int {
		t.Helper()
		res, err := http.Post(ts.URL+"/coddy/sessions/"+sid+"/prompt-suggestion/outcome", "application/json", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		return res.StatusCode
	}
	if code := post(`{"accepted":true}`); code != http.StatusOK {
		t.Fatalf("outcome status=%d", code)
	}
	if code := post(`{`); code != http.StatusBadRequest {
		t.Fatalf("bad outcome status=%d", code)
	}
}

// TestPromptSuggestionRouteWritesNothingToTheTranscript holds the one thing a
// suppressed turn must not do: leave a trace in the conversation.
func TestPromptSuggestionRouteWritesNothingToTheTranscript(t *testing.T) {
	mgr, srv, root := testHTTPServerPersist(t)
	created, err := mgr.HandleSessionNew(context.Background(), acp.SessionNewParams{CWD: root})
	if err != nil {
		t.Fatal(err)
	}
	st := mgr.SessionByID(created.SessionID)
	if st == nil {
		t.Fatal("session missing")
	}
	before, rev := st.MessagesWithRev()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/coddy/sessions/" + created.SessionID + "/prompt-suggestion")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	after, afterRev := st.MessagesWithRev()
	if len(after) != len(before) || afterRev != rev {
		t.Fatalf("transcript moved: %d -> %d messages, rev %d -> %d", len(before), len(after), rev, afterRev)
	}
	// The fork is what a suggestion needs, and a turn that never ran one
	// leaves none.
	if _, _, ok := st.PromptSuggestionFork(); ok {
		t.Fatal("a fork appeared without a turn")
	}
}
