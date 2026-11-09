package opencodeimport_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/opencodeimport"
	"github.com/EvilFreelancer/coddy-agent/internal/session"
)

func TestClientListsAndExports(t *testing.T) {
	const user, password = "opencode", "secret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
		if r.Header.Get("Authorization") != want {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/api/info":
			_, _ = w.Write([]byte(`{"version":"2.1.0"}`))
		case r.URL.Path == "/api/project":
			_, _ = w.Write([]byte(`[{"id":"p1","worktree":"/tmp/project","name":"demo"}]`))
		case r.URL.Path == "/api/session":
			if r.URL.Query().Get("project") != "p1" {
				t.Errorf("project query = %q", r.URL.Query().Get("project"))
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"ses_1","projectID":"p1","title":"hello","location":{"directory":"/tmp/project"},"time":{"created":1000,"updated":2000}}],"cursor":{}}`))
		case r.URL.Path == "/api/experimental/session/ses_1/export":
			_, _ = w.Write([]byte(`{"data":{"info":{"id":"ses_1","projectID":"p1","title":"hello","location":{"directory":"/tmp/project"}},"messages":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := opencodeimport.NewClient(opencodeimport.Config{URL: srv.URL, Username: user, Password: password})
	version, err := client.Health(context.Background())
	if err != nil || version != "2.1.0" {
		t.Fatalf("Health = %q, %v", version, err)
	}
	projects, err := client.Projects(context.Background())
	if err != nil || len(projects) != 1 || projects[0].ID != "p1" {
		t.Fatalf("Projects = %+v, %v", projects, err)
	}
	sessions, err := client.Sessions(context.Background(), "p1", true)
	if err != nil || len(sessions) != 1 || sessions[0].ID != "ses_1" || sessions[0].Location.Directory != "/tmp/project" {
		t.Fatalf("Sessions = %+v, %v", sessions, err)
	}
	tr, err := client.Export(context.Background(), "ses_1")
	if err != nil || tr.Data.Info.ID != "ses_1" {
		t.Fatalf("Export = %+v, %v", tr, err)
	}
}

func TestImportAllWritesCoddyBundle(t *testing.T) {
	workspace := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/session":
			_, _ = w.Write([]byte(`{"data":[{"id":"ses_import","projectID":"p1","title":"测试会话","location":{"directory":"` + workspace + `"},"time":{"created":1000,"updated":2000}}],"cursor":{}}`))
		case "/api/experimental/session/ses_import/export":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"info": map[string]any{"id": "ses_import", "projectID": "p1", "title": "测试会话", "location": map[string]any{"directory": workspace}},
				"messages": []any{
					map[string]any{"id": "msg_1", "type": "user", "text": "你好", "time": map[string]any{"created": 1000}},
					map[string]any{"id": "msg_2", "type": "assistant", "time": map[string]any{"created": 2000}, "content": []any{
						map[string]any{"type": "text", "text": "你好，有什么可以帮你？"},
						map[string]any{"type": "tool", "id": "tool_1", "name": "bash", "state": map[string]any{
							"status": "completed", "input": map[string]any{"command": "echo hi"},
							"content": []any{map[string]any{"type": "text", "text": "hi"}},
						}},
					}},
				},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	root := filepath.Join(t.TempDir(), "sessions")
	client := opencodeimport.NewClient(opencodeimport.Config{URL: srv.URL})
	res, err := opencodeimport.ImportAll(context.Background(), client, opencodeimport.ImportOptions{
		Projects:        []opencodeimport.Project{{ID: "p1", Worktree: workspace}},
		SessionsRoot:    root,
		Concurrency:     1,
		IncludeArchived: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 1 || res.Sessions != 1 {
		t.Fatalf("result = %+v", res)
	}
	store := &session.FileStore{Root: root}
	snap, err := store.ReadSnapshot("ses_import")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Meta.Title != "测试会话" {
		t.Fatalf("title = %q", snap.Meta.Title)
	}
	if len(snap.Meta.Tags) == 0 || snap.Meta.Tags[0] != "opencode" {
		t.Fatalf("tags = %v", snap.Meta.Tags)
	}
	if len(snap.Messages) != 3 {
		t.Fatalf("messages = %+v", snap.Messages)
	}
	if snap.Messages[0].Content != "你好" || snap.Messages[1].Content == "" {
		t.Fatalf("messages = %+v", snap.Messages)
	}
	if len(snap.Messages[1].ToolCalls) != 1 || snap.Messages[1].ToolCalls[0].Name != "run_command" {
		t.Fatalf("tool calls = %+v", snap.Messages[1].ToolCalls)
	}
	if snap.Messages[1].ToolCalls[0].InputJSON != `{"command":"echo hi"}` {
		t.Fatalf("tool input = %s", snap.Messages[1].ToolCalls[0].InputJSON)
	}
	if snap.Messages[2].Role != "tool" || snap.Messages[2].Content != "hi" {
		t.Fatalf("tool result = %+v", snap.Messages[2])
	}
	// A second run must skip the already imported bundle.
	res, err = opencodeimport.ImportAll(context.Background(), client, opencodeimport.ImportOptions{
		Projects:     []opencodeimport.Project{{ID: "p1", Worktree: workspace}},
		SessionsRoot: root,
		Concurrency:  1,
	})
	if err != nil || res.Skipped != 1 || res.Imported != 0 {
		t.Fatalf("second run = %+v, %v", res, err)
	}
}

func TestImportAllUsesFallbackForMissingWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/session":
			_, _ = w.Write([]byte(`{"data":[{"id":"ses_missing","projectID":"p1","title":"missing","location":{"directory":"/does/not/exist"},"time":{"created":1000}}],"cursor":{}}`))
		case "/api/experimental/session/ses_missing/export":
			_, _ = w.Write([]byte(`{"data":{"info":{"id":"ses_missing","projectID":"p1","title":"missing","location":{"directory":"/does/not/exist"}},"messages":[{"type":"user","text":"x","time":{"created":1}}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := opencodeimport.NewClient(opencodeimport.Config{URL: srv.URL})
	if _, err := opencodeimport.ImportAll(context.Background(), client, opencodeimport.ImportOptions{
		Projects:     []opencodeimport.Project{{ID: "p1"}},
		SessionsRoot: root,
		Concurrency:  1,
	}); err != nil {
		t.Fatal(err)
	}
	store := &session.FileStore{Root: root}
	snap, err := store.ReadSnapshot("ses_missing")
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Meta.Tags) < 2 || snap.Meta.Tags[1] != "opencode-missing-workspace" {
		t.Fatalf("tags = %v", snap.Meta.Tags)
	}
}

func TestImportAllDryRunWritesNothing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/session" {
			_, _ = w.Write([]byte(`{"data":[{"id":"ses_dry","projectID":"p1","title":"dry","location":{"directory":"/tmp"},"time":{"created":1}}],"cursor":{}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	client := opencodeimport.NewClient(opencodeimport.Config{URL: srv.URL})
	res, err := opencodeimport.ImportAll(context.Background(), client, opencodeimport.ImportOptions{
		Projects:     []opencodeimport.Project{{ID: "p1"}},
		SessionsRoot: root,
		DryRun:       true,
	})
	if err != nil || res.Sessions != 1 {
		t.Fatalf("result = %+v, %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(root, "ses_dry")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote a bundle: %v", err)
	}
}

func TestImportAllNormalizesOpenCodeToolNames(t *testing.T) {
	workspace := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/session":
			_, _ = w.Write([]byte(`{"data":[{"id":"ses_tools","projectID":"p1","title":"tools","location":{"directory":"` + workspace + `"},"time":{"created":1}}],"cursor":{}}`))
		case "/api/experimental/session/ses_tools/export":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"info": map[string]any{"id": "ses_tools", "projectID": "p1", "title": "tools", "location": map[string]any{"directory": workspace}},
				"messages": []any{
					map[string]any{"type": "user", "text": "run", "time": map[string]any{"created": 1}},
					map[string]any{"type": "assistant", "time": map[string]any{"created": 2}, "content": []any{
						map[string]any{"type": "text", "text": "working"},
						toolPart("t_bash", "bash", map[string]any{"command": "ls", "workdir": "/tmp/project", "timeout": 12500, "description": "list"}),
						toolPart("t_read", "read", map[string]any{"filePath": "a.go", "offset": 2, "limit": 3}),
						toolPart("t_edit", "edit", map[string]any{"filePath": "a.go", "oldString": "a", "newString": "b", "replaceAll": true}),
						toolPart("t_grep", "grep", map[string]any{"pattern": "x", "path": "/tmp", "include": "*.go"}),
						toolPart("t_webfetch", "webfetch", map[string]any{"url": "https://example.com", "format": "text", "timeout": 9}),
						toolPart("t_task", "task", map[string]any{"command": "run", "description": "explore", "prompt": "look", "subagent_type": "explore"}),
						toolPart("t_todo", "todowrite", map[string]any{"todos": []any{
							map[string]any{"content": "one", "status": "completed"},
							map[string]any{"content": "two", "status": "in_progress"},
						}}),
						toolPart("t_skill", "skill", map[string]any{"name": "brainstorming"}),
						toolPart("t_unknown", "lsp", map[string]any{"foo": "bar"}),
					}},
				},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	root := filepath.Join(t.TempDir(), "sessions")
	client := opencodeimport.NewClient(opencodeimport.Config{URL: srv.URL})
	res, err := opencodeimport.ImportAll(context.Background(), client, opencodeimport.ImportOptions{
		Projects:     []opencodeimport.Project{{ID: "p1", Worktree: workspace}},
		SessionsRoot: root,
		Concurrency:  1,
	})
	if err != nil || res.Imported != 1 {
		t.Fatalf("result = %+v, %v", res, err)
	}
	store := &session.FileStore{Root: root}
	snap, err := store.ReadSnapshot("ses_tools")
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Messages) != 10 {
		t.Fatalf("messages = %d, want 10", len(snap.Messages))
	}
	calls := snap.Messages[1].ToolCalls
	want := map[string]struct {
		name  string
		input string
	}{
		"t_bash":     {"run_command", `{"command":"ls","cwd":"/tmp/project","timeout_seconds":13,"permission_rationale":"list"}`},
		"t_read":     {"read", `{"path":"a.go","offset":2,"limit":3}`},
		"t_edit":     {"edit", `{"path":"a.go","oldString":"a","newString":"b","replaceAll":true}`},
		"t_grep":     {"grep", `{"pattern":"x","path":"/tmp","glob":"*.go"}`},
		"t_webfetch": {"webfetch", `{"url":"https://example.com","timeout_seconds":9}`},
		"t_task":     {"spawn_agent", `{"description":"explore","prompt":"look","agent":"explore"}`},
		"t_todo":     {"coddy_todo_plan_replace", `{"markdown":"- [x] one\n- [ ] two"}`},
		"t_skill":    {"load_skill", `{"name":"brainstorming"}`},
	}
	if len(calls) != len(want) {
		t.Fatalf("tool calls = %+v", calls)
	}
	for _, call := range calls {
		expected, ok := want[call.ID]
		if !ok {
			t.Fatalf("unexpected tool call %s (%s)", call.ID, call.Name)
		}
		if call.Name != expected.name {
			t.Errorf("%s name = %q, want %q", call.ID, call.Name, expected.name)
		}
		var got, wantInput any
		if err := json.Unmarshal([]byte(call.InputJSON), &got); err != nil {
			t.Errorf("%s input is not JSON: %v", call.ID, err)
			continue
		}
		if err := json.Unmarshal([]byte(expected.input), &wantInput); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, wantInput) {
			t.Errorf("%s input = %s, want %s", call.ID, call.InputJSON, expected.input)
		}
	}
	for _, msg := range snap.Messages[2:] {
		if msg.ToolCallID == "t_unknown" {
			t.Fatalf("unknown tool result was preserved: %+v", msg)
		}
	}
}

func toolPart(id, name string, input map[string]any) map[string]any {
	return map[string]any{"type": "tool", "id": id, "name": name, "state": map[string]any{
		"status": "completed", "input": input,
		"content": []any{map[string]any{"type": "text", "text": "ok"}},
	}}
}

func TestImportAllPreservesChildSessionsAndAttachments(t *testing.T) {
	workspace := t.TempDir()
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/session":
			_, _ = w.Write([]byte(`{"data":[
				{"id":"ses_parent","projectID":"p1","title":"parent","location":{"directory":"` + workspace + `"},"time":{"created":1}},
				{"id":"ses_child","projectID":"p1","parentID":"ses_parent","title":"child","location":{"directory":"` + workspace + `"},"time":{"created":2}}
			],"cursor":{}}`))
		case "/api/experimental/session/ses_parent/export":
			_, _ = w.Write([]byte(`{"data":{"info":{"id":"ses_parent","projectID":"p1","title":"parent","location":{"directory":"` + workspace + `"}},"messages":[{"type":"user","text":"parent","time":{"created":1}}]}}`))
		case "/api/experimental/session/ses_child/export":
			_, _ = w.Write([]byte(`{"data":{"info":{"id":"ses_child","projectID":"p1","parentID":"ses_parent","title":"child","location":{"directory":"` + workspace + `"}},"messages":[{"type":"user","text":"child","time":{"created":2},"files":[{"data":"` + png + `","mime":"image/png","name":"pixel.png"}]}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	root := filepath.Join(t.TempDir(), "sessions")
	client := opencodeimport.NewClient(opencodeimport.Config{URL: srv.URL})
	res, err := opencodeimport.ImportAll(context.Background(), client, opencodeimport.ImportOptions{
		Projects:     []opencodeimport.Project{{ID: "p1", Worktree: workspace}},
		SessionsRoot: root,
		Concurrency:  1,
	})
	if err != nil || res.Imported != 2 {
		t.Fatalf("result = %+v, %v", res, err)
	}
	store := &session.FileStore{Root: root}
	child, err := store.ReadSnapshot("ses_child")
	if err != nil {
		t.Fatal(err)
	}
	if !child.Meta.SubagentRun || child.Meta.ParentSessionID != "ses_parent" {
		t.Fatalf("child meta = %+v", child.Meta)
	}
	if !strings.Contains(child.Dir, string(filepath.Separator)+session.ChildSessionsDirName+string(filepath.Separator)) {
		t.Fatalf("child dir = %q", child.Dir)
	}
	if len(child.Messages) != 1 || len(child.Messages[0].ImageParts) != 1 {
		t.Fatalf("child messages = %+v", child.Messages)
	}
	if _, err := os.Stat(child.Messages[0].ImageParts[0].FilePath); err != nil {
		t.Fatalf("attachment missing: %v", err)
	}
	rows, err := store.ListSnapshotsWith(session.ListOptions{IncludeImportedSubagents: true})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if row.SessionID == "ses_child" {
			found = true
		}
	}
	if !found {
		t.Fatalf("imported child missing from the list: %+v", rows)
	}
}
