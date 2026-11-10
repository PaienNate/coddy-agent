package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// tavilySearchFunc is swapped in tests to avoid a live call.
var tavilySearchFunc func(ctx context.Context, q Query, s Settings) ([]Result, error)

// tavilyEndpoint is Tavily's REST search endpoint.
var tavilyEndpoint = "https://api.tavily.com/search"

var tavilyClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type tavilyResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

func runTavily(ctx context.Context, q Query, s Settings) ([]Result, error) {
	if q.Page > 1 {
		return nil, blocked("tavily does not support pagination; use page 1 or narrow the query")
	}
	if tavilySearchFunc != nil {
		return tavilySearchFunc(ctx, q, s)
	}
	count := q.MaxResults
	if count <= 0 || count > 20 {
		count = 8
	}
	body, err := json.Marshal(map[string]any{
		"query":             engineQuery(q),
		"search_depth":      "basic",
		"chunks_per_source": 3,
		"max_results":       count,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilyEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Coddy/1.0")
	req.Header.Set("X-Client-Name", "coddy")
	if key := strings.TrimSpace(s.TavilyAPIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	} else {
		req.Header.Set("X-Tavily-Access-Mode", "keyless")
	}
	resp, err := tavilyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, blocked("tavily api rejected the key (http %d)", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, blocked("tavily api rate limit (http 429)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := "http " + fmt.Sprint(resp.StatusCode)
		if buf, readErr := io.ReadAll(io.LimitReader(resp.Body, maxSERPBytes)); readErr == nil {
			var parsed struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if json.Unmarshal(buf, &parsed) == nil && strings.TrimSpace(parsed.Error.Message) != "" {
				msg = parsed.Error.Message
			}
		}
		return nil, blocked("tavily api: %s", msg)
	}
	answer, err := io.ReadAll(io.LimitReader(resp.Body, maxSERPBytes+1))
	if err != nil {
		return nil, err
	}
	if len(answer) > maxSERPBytes {
		return nil, blocked("tavily response exceeds the %d byte limit", maxSERPBytes)
	}
	var parsed tavilyResponse
	if err := json.Unmarshal(answer, &parsed); err != nil {
		return nil, blocked("tavily did not answer json (%v)", err)
	}
	out := make([]Result, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		u := strings.TrimSpace(r.URL)
		if u == "" {
			continue
		}
		out = append(out, Result{
			Title:   strings.TrimSpace(r.Title),
			URL:     u,
			Snippet: strings.TrimSpace(r.Content),
		})
		if count > 0 && len(out) >= count {
			break
		}
	}
	return out, nil
}
