package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func withEndpoint(t *testing.T, target *string, url string) {
	t.Helper()
	old := *target
	*target = url
	t.Cleanup(func() { *target = old })
}

func TestExaParsesTheMCPBlockFormat(t *testing.T) {
	var gotQuery string
	var gotNum float64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Params struct {
				Arguments struct {
					Query      string  `json:"query"`
					NumResults float64 `json:"numResults"`
				} `json:"arguments"`
			} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotQuery = body.Params.Arguments.Query
		gotNum = body.Params.Arguments.NumResults
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, `data: {"result":{"content":[{"type":"text","text":"Title: Context\nURL: https://go.dev/ref/spec#Context\nPublished: 2026-10-20T00:00:00.000Z\nHighlights:\nA Context carries deadlines.\n\n---\n\nTitle: Second\nURL: https://example.com/second\nHighlights:\nSecond body."}]}}`)
	}))
	defer srv.Close()
	withEndpoint(t, &exaEndpoint, srv.URL)

	rows, err := runExa(context.Background(), Query{Text: "go context", MaxResults: 5}, Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "go context" || gotNum != 5 {
		t.Fatalf("request query=%q num=%v", gotQuery, gotNum)
	}
	if len(rows) != 2 || rows[0].URL != "https://go.dev/ref/spec#Context" || !strings.Contains(rows[0].Snippet, "deadlines") {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestFirecrawlParsesTheMCPJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer fc-key" {
			t.Errorf("Authorization = %q", got)
		}
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"{\"success\":true,\"data\":{\"web\":[{\"url\":\"https://example.com/a\",\"title\":\"A\",\"description\":\"Description A\"}]}}"}]}}`)
	}))
	defer srv.Close()
	withEndpoint(t, &firecrawlEndpoint, srv.URL)

	rows, err := runFirecrawl(context.Background(), Query{Text: "a", MaxResults: 3}, Settings{FirecrawlAPIKey: "fc-key"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].URL != "https://example.com/a" || rows[0].Snippet != "Description A" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestParallelReadsStructuredContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","result":{"structuredContent":{"results":[{"url":"https://example.com/p","title":"P","excerpts":["one","two"]}]}}}`)
	}))
	defer srv.Close()
	withEndpoint(t, &parallelEndpoint, srv.URL)

	rows, err := runParallel(context.Background(), Query{Text: "parallel", MaxResults: 4}, Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Snippet != "one\n\ntwo" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestTinyFishKeylessAndKeyedSessions(t *testing.T) {
	t.Run("keyless", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("X-TinyFish-Access-Mode"); got != "keyless" {
				t.Errorf("access mode = %q", got)
			}
			if got := r.Header.Get("X-API-Key"); got != "" {
				t.Errorf("keyless request carried an API key")
			}
			_, _ = io.WriteString(w, `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"{\"results\":[{\"url\":\"https://example.com/t\",\"title\":\"T\",\"snippet\":\"tiny\"}]}"}]}}`)
		}))
		defer srv.Close()
		withEndpoint(t, &tinyFishEndpoint, srv.URL)

		rows, err := runTinyFish(context.Background(), Query{Text: "tiny"}, Settings{})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].URL != "https://example.com/t" {
			t.Fatalf("rows = %+v", rows)
		}
	})

	t.Run("keyed initializes a session", func(t *testing.T) {
		var sawInitialize, sawNotification, sawCall bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("X-API-Key"); got != "tf-key" {
				t.Errorf("X-API-Key = %q", got)
			}
			var body struct {
				Method string `json:"method"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			switch body.Method {
			case "initialize":
				sawInitialize = true
				w.Header().Set("Mcp-Session-Id", "session-1")
				_, _ = io.WriteString(w, `{"jsonrpc":"2.0","result":{}}`)
			case "notifications/initialized":
				sawNotification = true
				if got := r.Header.Get("Mcp-Session-Id"); got != "session-1" {
					t.Errorf("notification session = %q", got)
				}
				w.WriteHeader(http.StatusAccepted)
			case "tools/call":
				sawCall = true
				if got := r.Header.Get("Mcp-Session-Id"); got != "session-1" {
					t.Errorf("call session = %q", got)
				}
				_, _ = io.WriteString(w, `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"{\"results\":[]}"}]}}`)
			}
		}))
		defer srv.Close()
		withEndpoint(t, &tinyFishEndpoint, srv.URL)

		_, err := runTinyFish(context.Background(), Query{Text: "tiny"}, Settings{TinyFishAPIKey: "tf-key"})
		if err != nil {
			t.Fatal(err)
		}
		if !sawInitialize || !sawNotification || !sawCall {
			t.Fatalf("initialize=%v notification=%v call=%v", sawInitialize, sawNotification, sawCall)
		}
	})
}

func TestTavilyPostsTheSearchBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Tavily-Access-Mode"); got != "keyless" {
			t.Errorf("access mode = %q", got)
		}
		var body struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Query != "tavily" || body.MaxResults != 4 {
			t.Errorf("body = %+v", body)
		}
		_, _ = io.WriteString(w, `{"results":[{"title":"T","url":"https://example.com/v","content":"snippet"}]}`)
	}))
	defer srv.Close()
	withEndpoint(t, &tavilyEndpoint, srv.URL)

	rows, err := runTavily(context.Background(), Query{Text: "tavily", MaxResults: 4}, Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].URL != "https://example.com/v" {
		t.Fatalf("rows = %+v", rows)
	}
}
