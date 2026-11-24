//go:build http

package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/config"
)

// terminalCrossOrigin asks for the chat's terminal the way a page another
// origin served asks for it: a browser labels such a request "cross-site" and
// names the page it came from.
func terminalCrossOrigin(t *testing.T, srv *Server, sid, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/coddy/sessions/"+sid+"/terminal", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	return rec
}

// The web UI read from a phone, or through another address, is a page another
// origin serves. The operator names it in terminal.allowed_origins, and the
// route answers it; an origin nobody named is still refused.
func TestTerminalAllowsAConfiguredOrigin(t *testing.T) {
	srv, sid, _ := newTerminalEnvWith(t, config.TerminalConfig{
		Enable:         true,
		AllowedOrigins: []string{"http://tablet:9000"},
	})

	if rec := terminalCrossOrigin(t, srv, sid, "http://tablet:9000"); rec.Code != http.StatusOK {
		t.Fatalf("listed origin = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	for _, origin := range []string{"http://tablet:9001", "http://other:9000", ""} {
		if rec := terminalCrossOrigin(t, srv, sid, origin); rec.Code != http.StatusForbidden {
			t.Fatalf("origin %q = %d, want 403", origin, rec.Code)
		}
	}
}

// The scheme is not compared, as it is not for this server's own origin: a
// proxy in front may terminate TLS, so https from the browser reaches a server
// that speaks http.
func TestTerminalOriginIgnoresTheScheme(t *testing.T) {
	srv, sid, _ := newTerminalEnvWith(t, config.TerminalConfig{
		Enable:         true,
		AllowedOrigins: []string{"tablet:9000"},
	})
	if rec := terminalCrossOrigin(t, srv, sid, "https://tablet:9000"); rec.Code != http.StatusOK {
		t.Fatalf("scheme-less entry = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

// A shell is granted to a page, never to the web.
func TestTerminalRefusesAWildcardOrigin(t *testing.T) {
	srv, sid, _ := newTerminalEnvWith(t, config.TerminalConfig{
		Enable:         true,
		AllowedOrigins: []string{"*"},
	})
	if rec := terminalCrossOrigin(t, srv, sid, "http://tablet:9000"); rec.Code != http.StatusForbidden {
		t.Fatalf("wildcard entry = %d, want 403", rec.Code)
	}
}

// The allowance is only granted to the listed page: a request from another
// origin still gets no CORS answer on the terminal route.
func TestTerminalCorsAnswersOnlyTheListedOrigin(t *testing.T) {
	srv, sid, _ := newTerminalEnvWith(t, config.TerminalConfig{
		Enable:         true,
		AllowedOrigins: []string{"http://tablet:9000"},
	})
	// The CORS layer wraps the mux rather than sitting inside it, as it does
	// for the server the test stands up.
	handler := srv.corsMiddleware(srv.mux)

	listed := httptest.NewRequest(http.MethodOptions, "/coddy/sessions/"+sid+"/terminal", nil)
	listed.Header.Set("Origin", "http://tablet:9000")
	listed.Header.Set("Access-Control-Request-Method", "GET")
	listedRec := httptest.NewRecorder()
	handler.ServeHTTP(listedRec, listed)
	if got := listedRec.Header().Get("Access-Control-Allow-Origin"); got != "http://tablet:9000" {
		t.Fatalf("listed origin ACAO = %q, want the origin echoed", got)
	}

	other := httptest.NewRequest(http.MethodOptions, "/coddy/sessions/"+sid+"/terminal", nil)
	other.Header.Set("Origin", "http://tablet:9001")
	other.Header.Set("Access-Control-Request-Method", "GET")
	otherRec := httptest.NewRecorder()
	handler.ServeHTTP(otherRec, other)
	if got := otherRec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unlisted origin ACAO = %q, want none", got)
	}
}
