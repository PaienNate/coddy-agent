package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// tinyFishSearchFunc is swapped in tests to avoid a live call.
var tinyFishSearchFunc func(ctx context.Context, q Query, s Settings) ([]Result, error)

// tinyFishEndpoint is TinyFish's MCP endpoint. It is a variable so a test can
// point it at a local server.
var tinyFishEndpoint = "https://agent.tinyfish.ai/mcp"

// tinyFishResponse is the JSON document TinyFish puts in content[0].text.
type tinyFishResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	} `json:"results"`
}

func runTinyFish(ctx context.Context, q Query, s Settings) ([]Result, error) {
	if q.Page > 1 {
		return nil, blocked("tinyfish does not support pagination; use page 1 or narrow the query")
	}
	if tinyFishSearchFunc != nil {
		return tinyFishSearchFunc(ctx, q, s)
	}
	key := strings.TrimSpace(s.TinyFishAPIKey)
	headers := map[string]string{}
	if key == "" {
		headers["X-TinyFish-Access-Mode"] = "keyless"
	} else {
		headers["X-API-Key"] = key
		session, err := tinyFishInitialize(ctx, headers)
		if err != nil {
			return nil, err
		}
		headers["Mcp-Session-Id"] = session
	}
	result, err := mcpToolCall(ctx, "tinyfish", tinyFishEndpoint, "search", map[string]any{
		"query": engineQuery(q),
	}, headers)
	if err != nil {
		return nil, err
	}
	text := mcpContentText(result)
	if strings.TrimSpace(text) == "" {
		return nil, blocked("tinyfish answered without a search document")
	}
	var parsed tinyFishResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, blocked("tinyfish did not answer the search json (%v)", err)
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
			Snippet: strings.TrimSpace(r.Snippet),
		})
		if q.MaxResults > 0 && len(out) >= q.MaxResults {
			break
		}
	}
	return out, nil
}

// tinyFishInitialize creates the MCP session a keyed TinyFish request needs.
// Keyless requests work without one, which is why we skip this handshake when
// no key is configured.
func tinyFishInitialize(ctx context.Context, headers map[string]string) (string, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "coddy",
				"version": "1.0",
			},
		},
	}
	_, responseHeader, err := mcpPost(ctx, "tinyfish", tinyFishEndpoint, headers, payload)
	if err != nil {
		return "", err
	}
	session := strings.TrimSpace(responseHeader.Get("Mcp-Session-Id"))
	if session == "" {
		return "", fmt.Errorf("tinyfish initialize returned no Mcp-Session-Id")
	}
	initialized := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notifyHeaders := map[string]string{"Mcp-Session-Id": session}
	for k, v := range headers {
		notifyHeaders[k] = v
	}
	if _, _, err := mcpPost(ctx, "tinyfish", tinyFishEndpoint, notifyHeaders, initialized); err != nil {
		return "", err
	}
	return session, nil
}
