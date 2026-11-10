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

// mcpClient never follows a redirect. Search endpoints are fixed, and a
// redirect is how an API key in a header or query string reaches a host the
// operator never chose.
var mcpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// mcpToolResult is the part of a JSON-RPC tools/call answer this package reads.
// Parallel puts its rows in structuredContent; the other providers put a JSON
// or text document in content[0].text.
type mcpToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent"`
}

type mcpEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// mcpToolCall sends one JSON-RPC tools/call and returns its result. It accepts
// either a JSON answer or an SSE answer whose data line holds the JSON.
func mcpToolCall(ctx context.Context, label, endpoint, tool string, args any, headers map[string]string) (mcpToolResult, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": args,
		},
	}
	body, _, err := mcpPost(ctx, label, endpoint, headers, payload)
	if err != nil {
		return mcpToolResult{}, err
	}
	env, err := decodeMCPEnvelope(body)
	if err != nil {
		return mcpToolResult{}, blocked("%s did not answer json-rpc (%v)", label, err)
	}
	if env.Error != nil {
		return mcpToolResult{}, blocked("%s api: %s", label, strings.TrimSpace(env.Error.Message))
	}
	if len(env.Result) == 0 || bytes.Equal(bytes.TrimSpace(env.Result), []byte("null")) {
		return mcpToolResult{}, blocked("%s answered without a result", label)
	}
	var result mcpToolResult
	if err := json.Unmarshal(env.Result, &result); err != nil {
		return mcpToolResult{}, blocked("%s result is not a tools/call answer (%v)", label, err)
	}
	if len(result.Content) == 0 && len(result.StructuredContent) == 0 {
		return mcpToolResult{}, blocked("%s answered without content", label)
	}
	return result, nil
}

// mcpPost sends one JSON-RPC message and returns the body and response
// headers. The message may be an initialize request, its initialized
// notification, or a tools/call.
func mcpPost(ctx context.Context, label, endpoint string, headers map[string]string, payload any) ([]byte, http.Header, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("User-Agent", "Coddy/1.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := mcpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, nil, blocked("%s api rejected the key (http %d)", label, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, nil, blocked("%s api rate limit (http 429)", label)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, blocked("%s api http %d", label, resp.StatusCode)
	}
	answer, err := io.ReadAll(io.LimitReader(resp.Body, maxSERPBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if len(answer) > maxSERPBytes {
		return nil, nil, blocked("%s response exceeds the %d byte limit", label, maxSERPBytes)
	}
	return answer, resp.Header, nil
}

func decodeMCPEnvelope(body []byte) (mcpEnvelope, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var env mcpEnvelope
		if err := json.Unmarshal(trimmed, &env); err == nil {
			return env, nil
		}
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data[0] != '{' {
			continue
		}
		var env mcpEnvelope
		if err := json.Unmarshal([]byte(data), &env); err != nil {
			continue
		}
		return env, nil
	}
	return mcpEnvelope{}, fmt.Errorf("no json-rpc object in the answer")
}

// mcpContentText returns the first text block of a tools/call answer.
func mcpContentText(result mcpToolResult) string {
	for _, item := range result.Content {
		if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
			return item.Text
		}
	}
	return ""
}
