package web

import (
	"context"
	"encoding/json"
	"strings"
)

// parallelSearchFunc is swapped in tests to avoid a live call.
var parallelSearchFunc func(ctx context.Context, q Query, s Settings) ([]Result, error)

// parallelEndpoint is Parallel's hosted MCP search endpoint.
var parallelEndpoint = "https://search.parallel.ai/mcp"

type parallelResponse struct {
	Results []struct {
		URL         string   `json:"url"`
		Title       string   `json:"title"`
		PublishDate string   `json:"publish_date"`
		Excerpts    []string `json:"excerpts"`
	} `json:"results"`
}

func runParallel(ctx context.Context, q Query, s Settings) ([]Result, error) {
	if q.Page > 1 {
		return nil, blocked("parallel does not support pagination; use page 1 or narrow the query")
	}
	if parallelSearchFunc != nil {
		return parallelSearchFunc(ctx, q, s)
	}
	headers := map[string]string{}
	if key := strings.TrimSpace(s.ParallelAPIKey); key != "" {
		headers["Authorization"] = "Bearer " + key
	}
	result, err := mcpToolCall(ctx, "parallel", parallelEndpoint, "web_search", map[string]any{
		"objective":      strings.TrimSpace(q.Text),
		"search_queries": []string{engineQuery(q)},
	}, headers)
	if err != nil {
		return nil, err
	}
	raw := result.StructuredContent
	if len(raw) == 0 {
		raw = []byte(mcpContentText(result))
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, blocked("parallel answered without a search document")
	}
	var parsed parallelResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, blocked("parallel did not answer the search json (%v)", err)
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
			Snippet: strings.TrimSpace(strings.Join(r.Excerpts, "\n\n")),
		})
		if q.MaxResults > 0 && len(out) >= q.MaxResults {
			break
		}
	}
	return out, nil
}
