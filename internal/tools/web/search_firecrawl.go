package web

import (
	"context"
	"encoding/json"
	"strings"
)

// firecrawlSearchFunc is swapped in tests to avoid a live call.
var firecrawlSearchFunc func(ctx context.Context, q Query, s Settings) ([]Result, error)

// firecrawlEndpoint is Firecrawl's hosted MCP endpoint.
var firecrawlEndpoint = "https://mcp.firecrawl.dev/v2/mcp"

type firecrawlResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Web []struct {
			URL         string `json:"url"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"web"`
	} `json:"data"`
}

func runFirecrawl(ctx context.Context, q Query, s Settings) ([]Result, error) {
	if q.Page > 1 {
		return nil, blocked("firecrawl does not support pagination; use page 1 or narrow the query")
	}
	if firecrawlSearchFunc != nil {
		return firecrawlSearchFunc(ctx, q, s)
	}
	count := q.MaxResults
	if count <= 0 || count > 20 {
		count = 8
	}
	headers := map[string]string{}
	if key := strings.TrimSpace(s.FirecrawlAPIKey); key != "" {
		headers["Authorization"] = "Bearer " + key
	}
	result, err := mcpToolCall(ctx, "firecrawl", firecrawlEndpoint, "firecrawl_search", map[string]any{
		"query": engineQuery(q),
		"limit": count,
	}, headers)
	if err != nil {
		return nil, err
	}
	text := mcpContentText(result)
	if strings.TrimSpace(text) == "" {
		return nil, blocked("firecrawl answered without a search document")
	}
	var parsed firecrawlResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, blocked("firecrawl did not answer the search json (%v)", err)
	}
	if !parsed.Success {
		return nil, blocked("firecrawl reported an unsuccessful search")
	}
	out := make([]Result, 0, len(parsed.Data.Web))
	for _, r := range parsed.Data.Web {
		u := strings.TrimSpace(r.URL)
		if u == "" {
			continue
		}
		out = append(out, Result{
			Title:   strings.TrimSpace(r.Title),
			URL:     u,
			Snippet: strings.TrimSpace(r.Description),
		})
		if count > 0 && len(out) >= count {
			break
		}
	}
	return out, nil
}
