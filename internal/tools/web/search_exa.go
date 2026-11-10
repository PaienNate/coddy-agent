package web

import (
	"context"
	"net/url"
	"strings"
)

// exaSearchFunc is swapped in tests to avoid a live call.
var exaSearchFunc func(ctx context.Context, q Query, s Settings) ([]Result, error)

// exaEndpoint is Exa's hosted MCP endpoint. It is a variable so a test can
// point it at a local server.
var exaEndpoint = "https://mcp.exa.ai/mcp"

func runExa(ctx context.Context, q Query, s Settings) ([]Result, error) {
	if q.Page > 1 {
		return nil, blocked("exa does not support pagination; use page 1 or narrow the query")
	}
	if exaSearchFunc != nil {
		return exaSearchFunc(ctx, q, s)
	}
	endpoint := exaEndpoint
	if key := strings.TrimSpace(s.ExaAPIKey); key != "" {
		endpoint += "?exaApiKey=" + url.QueryEscape(key)
	}
	count := q.MaxResults
	if count <= 0 || count > 20 {
		count = 8
	}
	result, err := mcpToolCall(ctx, "exa", endpoint, "web_search_exa", map[string]any{
		"query":      engineQuery(q),
		"numResults": count,
	}, nil)
	if err != nil {
		return nil, err
	}
	text := mcpContentText(result)
	if strings.TrimSpace(text) == "" {
		return nil, blocked("exa answered without a search document")
	}
	return parseExaResults(text), nil
}

// parseExaResults reads the block format Exa's MCP tool returns:
//
//	Title: ...
//	URL: ...
//	Published: ...
//	Highlights:
//	...
//
// Blocks are separated by a blank "---" line. Fields Exa does not know are
// ignored rather than making the whole answer unreadable.
func parseExaResults(text string) []Result {
	var out []Result
	for _, block := range strings.Split(text, "\n\n---\n\n") {
		var title, rawURL, content string
		lines := strings.Split(block, "\n")
		body := false
		var bodyLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			switch {
			case !body && strings.HasPrefix(trimmed, "URL:"):
				rawURL = strings.TrimSpace(strings.TrimPrefix(trimmed, "URL:"))
			case !body && strings.HasPrefix(trimmed, "Title:"):
				title = strings.TrimSpace(strings.TrimPrefix(trimmed, "Title:"))
			case !body && (trimmed == "Highlights:" || trimmed == "Text:"):
				body = true
			case body:
				bodyLines = append(bodyLines, line)
			}
		}
		content = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		if rawURL == "" || (!strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://")) {
			continue
		}
		out = append(out, Result{Title: title, URL: rawURL, Snippet: content})
	}
	return out
}
