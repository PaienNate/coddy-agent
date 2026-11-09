// Package opencodeimport imports OpenCode v2 session transcripts through its
// HTTP API and writes them as Coddy session bundles.
//
// The package deliberately has no SQLite dependency: the source is the API the
// OpenCode v2 server exposes. A migration therefore needs a reachable OpenCode
// server, and nothing is read from or written to its database.
package opencodeimport

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/llm"
	"github.com/EvilFreelancer/coddy-agent/internal/session"
)

// Config is the saved OpenCode v2 connection. It lives outside config.yaml so
// a one-time migration credential does not become part of the agent's runtime
// configuration.
type Config struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

// ConfigPath returns the connection file under a Coddy home.
func ConfigPath(home string) string {
	return filepath.Join(home, "opencode.json")
}

// LoadConfig reads the saved connection, returning an empty Config when the
// file does not exist.
func LoadConfig(home string) (Config, error) {
	path := ConfigPath(home)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	cfg.URL = strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if cfg.Username == "" {
		cfg.Username = "opencode"
	}
	return cfg, nil
}

// SaveConfig writes the connection with owner-only permissions.
func SaveConfig(home string, cfg Config) error {
	cfg.URL = strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if cfg.Username == "" {
		cfg.Username = "opencode"
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return errors.New("OpenCode URL is required")
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(ConfigPath(home)), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(ConfigPath(home)), ".opencode-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, ConfigPath(home))
}

// Project is the part of OpenCode's project row the importer needs.
type Project struct {
	ID        string `json:"id"`
	Canonical string `json:"canonical"`
	// Worktree is accepted from older/adapter payloads; the v2 API names the
	// project root canonical.
	Worktree string `json:"worktree"`
	Name     string `json:"name"`
	Archived bool   `json:"-"`
	Time     struct {
		Archived *int64 `json:"archived,omitempty"`
	} `json:"time"`
}

// Root is the project directory the importer falls back to when a session
// carries no usable location.
func (p Project) Root() string {
	if strings.TrimSpace(p.Canonical) != "" {
		return p.Canonical
	}
	return p.Worktree
}

// ArchivedAt reports whether the project is archived and when.
func (p Project) ArchivedAt() *int64 {
	if p.Time.Archived != nil {
		return p.Time.Archived
	}
	return nil
}

// Session is the part of OpenCode's session row the importer needs.
type Session struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectID"`
	ParentID  string `json:"parentID"`
	Title     string `json:"title"`
	Location  struct {
		Directory string `json:"directory"`
	} `json:"location"`
	Time struct {
		Created  int64  `json:"created"`
		Updated  int64  `json:"updated"`
		Archived *int64 `json:"archived,omitempty"`
	} `json:"time"`
}

// Transfer is the subset of one session export this importer consumes.
type Transfer struct {
	Data struct {
		Info     Session   `json:"info"`
		Messages []Message `json:"messages"`
	} `json:"data"`
}

// Message is one OpenCode transcript message. Only fields the importer maps are
// declared; unknown provider-owned data is ignored.
type Message struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Text    string           `json:"text"`
	Summary string           `json:"summary"`
	Files   []FileAttachment `json:"files"`
	Content []Part           `json:"content"`
	Time    struct {
		Created int64 `json:"created"`
	} `json:"time"`
}

// FileAttachment is one prompt file exported by OpenCode.
type FileAttachment struct {
	Data string `json:"data"`
	Mime string `json:"mime"`
	Name string `json:"name"`
}

// Part is one assistant content part.
type Part struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Text  string          `json:"text"`
	State json.RawMessage `json:"state"`
}

// ToolState is the subset of a tool part state the importer reads.
type ToolState struct {
	Status   string          `json:"status"`
	Input    json.RawMessage `json:"input"`
	Content  []Part          `json:"content"`
	Metadata struct {
		Output string `json:"output"`
	} `json:"metadata"`
	Error json.RawMessage `json:"error"`
}

// Client talks to an OpenCode v2 server.
type Client struct {
	BaseURL  string
	Username string
	Password string
	HTTP     *http.Client
}

// NewClient builds a client from a saved Config.
func NewClient(cfg Config) *Client {
	base := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	user := strings.TrimSpace(cfg.Username)
	if user == "" {
		user = "opencode"
	}
	return &Client{
		BaseURL:  base,
		Username: user,
		Password: cfg.Password,
		HTTP: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (c *Client) do(ctx context.Context, method, p string, out any) error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("OpenCode URL is empty")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("OpenCode URL: %w", err)
	}
	ref, err := url.Parse(p)
	if err != nil {
		return fmt.Errorf("OpenCode path %q: %w", p, err)
	}
	u.Path = path.Join(strings.TrimRight(u.Path, "/"), strings.TrimLeft(ref.Path, "/"))
	u.RawQuery = ref.RawQuery
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.Password != "" {
		token := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
		req.Header.Set("Authorization", "Basic "+token)
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("OpenCode %s %s: HTTP %d: %s", method, p, resp.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("OpenCode %s %s: decode response: %w", method, p, err)
	}
	return nil
}

// Health asks the server for its version. OpenCode v2 exposes this as
// /api/info; /global/health belongs to older servers and is not a JSON route on
// v2 (the SPA fallback answers it with HTML).
func (c *Client) Health(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/info", &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Version) == "" {
		return "", errors.New("OpenCode server did not report a version")
	}
	return out.Version, nil
}

// Projects lists every project, archived ones included.
func (c *Client) Projects(ctx context.Context) ([]Project, error) {
	var out []Project
	if err := c.do(ctx, http.MethodGet, "/api/project?archived=true", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Sessions lists sessions of one project. OpenCode defaults to the newest 50,
// so ask for a page large enough for a migration and follow the cursor if the
// server still has more.
func (c *Client) Sessions(ctx context.Context, projectID string, includeArchived bool) ([]Session, error) {
	var all []Session
	cursor := ""
	for {
		q := url.Values{}
		q.Set("project", projectID)
		q.Set("limit", "1000")
		q.Set("order", "asc")
		if !includeArchived {
			q.Set("archived", "false")
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var out struct {
			Data   []Session `json:"data"`
			Cursor struct {
				Next json.RawMessage `json:"next"`
			} `json:"cursor"`
		}
		if err := c.do(ctx, http.MethodGet, "/api/session?"+q.Encode(), &out); err != nil {
			return nil, err
		}
		all = append(all, out.Data...)
		if len(out.Cursor.Next) == 0 || string(out.Cursor.Next) == "null" || len(out.Data) == 0 {
			break
		}
		var next string
		if err := json.Unmarshal(out.Cursor.Next, &next); err != nil || strings.TrimSpace(next) == "" {
			break
		}
		cursor = next
	}
	return all, nil
}

// Export fetches one complete session transcript.
func (c *Client) Export(ctx context.Context, sessionID string) (*Transfer, error) {
	var out Transfer
	if err := c.do(ctx, http.MethodGet, "/api/experimental/session/"+url.PathEscape(sessionID)+"/export", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ImportOptions controls a migration.
type ImportOptions struct {
	Projects        []Project
	IncludeArchived bool
	Concurrency     int
	DryRun          bool
	Replace         bool
	SessionsRoot    string
	FallbackCWD     string
	Progress        func(done, total int, sessionID string)
}

// ImportResult is the report of one migration.
type ImportResult struct {
	Projects int      `json:"projects"`
	Sessions int      `json:"sessions"`
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	ReadOnly int      `json:"read_only"`
	Errors   []string `json:"errors,omitempty"`
}

// ImportAll exports every session of every project and writes Coddy bundles.
func ImportAll(ctx context.Context, client *Client, opts ImportOptions) (*ImportResult, error) {
	if client == nil {
		return nil, errors.New("OpenCode client is nil")
	}
	if strings.TrimSpace(opts.SessionsRoot) == "" {
		return nil, errors.New("sessions root is empty")
	}
	if opts.Concurrency < 1 {
		opts.Concurrency = 4
	}
	res := &ImportResult{}
	type job struct {
		project Project
		session Session
	}
	var jobs []job
	for _, p := range opts.Projects {
		sessions, err := client.Sessions(ctx, p.ID, opts.IncludeArchived)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("project %s: %v", p.ID, err))
			res.Failed++
			continue
		}
		for _, s := range sessions {
			if !opts.IncludeArchived && s.Time.Archived != nil {
				continue
			}
			jobs = append(jobs, job{project: p, session: s})
		}
	}
	res.Projects = len(opts.Projects)
	res.Sessions = len(jobs)
	// Parents must be written before children: EnsureChildLayout resolves the
	// parent bundle, and a child of a parent that is not on disk is imported
	// as an orphan at the root instead.
	byID := make(map[string]Session, len(jobs))
	for _, j := range jobs {
		byID[j.session.ID] = j.session
	}
	depthOf := func(s Session) int {
		seen := map[string]bool{}
		depth := 0
		for strings.TrimSpace(s.ParentID) != "" && depth < 64 {
			parent, ok := byID[strings.TrimSpace(s.ParentID)]
			if !ok || seen[s.ID] {
				break
			}
			seen[s.ID] = true
			depth++
			s = parent
		}
		return depth
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		return depthOf(jobs[i].session) < depthOf(jobs[j].session)
	})
	if opts.DryRun {
		return res, nil
	}

	store := &session.FileStore{Root: opts.SessionsRoot}
	work := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range work {
				if err := importOne(ctx, client, store, j.project, j.session, opts); errors.Is(err, errSkip) {
					mu.Lock()
					res.Skipped++
					mu.Unlock()
				} else if err != nil {
					mu.Lock()
					res.Failed++
					if len(res.Errors) < 50 {
						res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", j.session.ID, err))
					}
					mu.Unlock()
				} else {
					mu.Lock()
					res.Imported++
					mu.Unlock()
				}
				mu.Lock()
				done++
				d := done
				mu.Unlock()
				if opts.Progress != nil {
					opts.Progress(d, res.Sessions, j.session.ID)
				}
			}
		}()
	}
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			close(work)
			wg.Wait()
			return res, ctx.Err()
		case work <- j:
		}
	}
	close(work)
	wg.Wait()
	return res, nil
}

func importOne(ctx context.Context, client *Client, store *session.FileStore, project Project, info Session, opts ImportOptions) error {
	if err := session.ValidateFolderSessionID(info.ID); err != nil {
		return err
	}
	if !opts.Replace && store.HasPersistedSnapshot(info.ID) {
		return errSkip
	}
	tr, err := client.Export(ctx, info.ID)
	if err != nil {
		return err
	}
	msgs, readOnly := convertMessages(tr.Data.Messages)
	cwd := strings.TrimSpace(tr.Data.Info.Location.Directory)
	if cwd == "" {
		cwd = strings.TrimSpace(project.Root())
	}
	if cwd == "" {
		cwd = strings.TrimSpace(opts.FallbackCWD)
	}
	if cwd != "" {
		if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
			readOnly = true
		}
	}
	parentID := strings.TrimSpace(tr.Data.Info.ParentID)
	orphan := false
	if parentID != "" && !store.HasPersistedSnapshot(parentID) {
		parentID = ""
		orphan = true
	}
	var dir string
	if parentID != "" {
		dir, err = store.EnsureChildLayout(parentID, info.ID)
	} else {
		dir = filepath.Join(opts.SessionsRoot, info.ID)
		err = os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		return err
	}
	for i := range msgs {
		if err := session.SavePartsToAssets(msgs[i].ImageParts, dir); err != nil {
			return err
		}
		keep := msgs[i].ImageParts[:0]
		for _, part := range msgs[i].ImageParts {
			if part.FilePath != "" && !strings.HasPrefix(part.MIMEType, "image/") {
				note := "\n\n[attached file: " + part.Name + " at " + part.FilePath + "]"
				msgs[i].Content += note
				continue
			}
			keep = append(keep, part)
		}
		msgs[i].ImageParts = keep
	}
	st := &session.State{
		ID:         info.ID,
		CWD:        cwd,
		Mode:       session.ModeAgent,
		SessionDir: dir,
		Messages:   msgs,
	}
	if parentID != "" {
		st.SetSubagentMeta(session.SubagentMeta{
			Name:            "opencode-import",
			ParentSessionID: parentID,
			Depth:           1,
		})
	}
	st.SetTitlePinned(strings.TrimSpace(info.Title))
	tags := []string{"opencode"}
	if readOnly {
		tags = append(tags, "opencode-missing-workspace")
	}
	if orphan {
		tags = append(tags, "opencode-orphan-child")
	}
	st.SetTags(tags)
	return store.Save(st)
}

var errSkip = errors.New("already imported")

// openCodeToolAliases maps OpenCode's tool vocabulary onto the Coddy tools
// that do the same job. An imported transcript is replayed as conversation
// history, so keeping OpenCode's names would teach a resumed session to call
// tools Coddy does not have - most visibly "bash", which is OpenCode's shell
// tool but is "run_command" here.
var openCodeToolAliases = map[string]string{
	"bash":      "run_command",
	"shell":     "run_command",
	"read":      "read",
	"write":     "write",
	"edit":      "edit",
	"glob":      "glob",
	"grep":      "grep",
	"list":      "print_tree",
	"webfetch":  "webfetch",
	"websearch": "websearch",
	"task":      "spawn_agent",
	"subagent":  "spawn_agent",
	"skill":     "load_skill",
	"question":  "question",
	"todowrite": "coddy_todo_plan_replace",
	"todoread":  "coddy_todo_plan_read",
	"patch":     "apply_patch",
}

// normalizeToolCall translates one OpenCode tool part. The boolean is false
// for a tool Coddy cannot offer: dropping both the call and its result is
// preferable to preserving a name the model would imitate and then fail on.
func normalizeToolCall(name, input string) (string, string, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	target, ok := openCodeToolAliases[key]
	if !ok {
		return "", "", false
	}
	return target, normalizeToolInput(key, input), true
}

// normalizeToolInput rewrites the argument names whose OpenCode spelling
// differs from Coddy's schema. The values stay raw JSON so numbers, booleans
// and nested objects survive unchanged.
func normalizeToolInput(tool, input string) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(input), &fields); err != nil {
		return input
	}
	move := func(from, to string) {
		v, ok := fields[from]
		if !ok {
			return
		}
		if _, exists := fields[to]; !exists {
			fields[to] = v
		}
		delete(fields, from)
	}
	drop := func(name string) { delete(fields, name) }

	switch tool {
	case "bash", "shell":
		move("workdir", "cwd")
		// OpenCode measures a shell timeout in milliseconds; Coddy's
		// run_command measures it in seconds.
		moveTimeoutMillis("timeout", "timeout_seconds", fields)
		move("description", "permission_rationale")
	case "read", "write", "edit":
		move("filePath", "path")
	case "grep":
		move("include", "glob")
	case "webfetch":
		move("timeout", "timeout_seconds")
		drop("format")
	case "task", "subagent":
		move("subagent_type", "agent")
		drop("command")
		if _, ok := fields["agent"]; !ok {
			fields["agent"] = json.RawMessage(`"general"`)
		}
	case "patch":
		move("filePath", "path")
		move("patchText", "patch")
		move("diff", "patch")
	case "todowrite":
		if markdown := todosAsMarkdown(fields["todos"]); markdown != "" {
			b, _ := json.Marshal(map[string]string{"markdown": markdown})
			return string(b)
		}
	case "todoread":
		return "{}"
	}

	if len(fields) == 0 {
		return "{}"
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return input
	}
	return string(b)
}

// moveTimeoutMillis converts an OpenCode millisecond timeout to Coddy's
// second-based timeout. OpenCode's schema says milliseconds, but some models
// send the value as a JSON string; both shapes are accepted.
func moveTimeoutMillis(from, to string, fields map[string]json.RawMessage) {
	raw, ok := fields[from]
	if !ok {
		return
	}
	var ms int64
	if err := json.Unmarshal(raw, &ms); err != nil {
		var f float64
		if json.Unmarshal(raw, &f) == nil {
			ms = int64(f)
		} else {
			var text string
			if json.Unmarshal(raw, &text) != nil {
				delete(fields, from)
				return
			}
			ms, _ = strconv.ParseInt(text, 10, 64)
		}
	}
	if ms < 0 {
		ms = 0
	}
	seconds := (ms + 999) / 1000
	if _, exists := fields[to]; !exists {
		b, _ := json.Marshal(seconds)
		fields[to] = b
	}
	delete(fields, from)
}

// todosAsMarkdown renders OpenCode's structured todo list as the checklist
// coddy_todo_plan_replace expects.
func todosAsMarkdown(raw json.RawMessage) string {
	var todos []struct {
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(raw, &todos); err != nil {
		return ""
	}
	var b strings.Builder
	for _, todo := range todos {
		content := strings.TrimSpace(todo.Content)
		if content == "" {
			continue
		}
		mark := " "
		if strings.EqualFold(todo.Status, "completed") {
			mark = "x"
		}
		b.WriteString("- [")
		b.WriteString(mark)
		b.WriteString("] ")
		b.WriteString(content)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func convertMessages(in []Message) ([]llm.Message, bool) {
	var out []llm.Message
	readOnly := false
	for _, m := range in {
		switch m.Type {
		case "user":
			text := strings.TrimSpace(m.Text)
			if text == "" {
				continue
			}
			parts := make([]llm.ImagePart, 0, len(m.Files))
			for _, f := range m.Files {
				if strings.TrimSpace(f.Data) == "" || strings.TrimSpace(f.Mime) == "" {
					continue
				}
				name := strings.TrimSpace(f.Name)
				if name == "" {
					name = "attachment"
				}
				parts = append(parts, llm.ImagePart{
					DataURL:  "data:" + f.Mime + ";base64," + f.Data,
					Name:     name,
					MIMEType: f.Mime,
				})
			}
			out = append(out, llm.Message{
				Role:       llm.RoleUser,
				Content:    m.Text,
				ImageParts: parts,
				CreatedAt:  millis(m.Time.Created),
			})
		case "assistant":
			var content, reasoning strings.Builder
			var calls []llm.ToolCall
			var results []llm.Message
			for _, p := range m.Content {
				switch p.Type {
				case "text":
					content.WriteString(p.Text)
				case "reasoning":
					reasoning.WriteString(p.Text)
				case "tool":
					var state ToolState
					_ = json.Unmarshal(p.State, &state)
					input := strings.TrimSpace(string(state.Input))
					if input == "" || input == "null" {
						input = "{}"
					}
					name, normalizedInput, keep := normalizeToolCall(p.Name, input)
					if !keep {
						continue
					}
					calls = append(calls, llm.ToolCall{ID: p.ID, Name: name, InputJSON: normalizedInput})
					result := toolResult(state, p)
					results = append(results, llm.Message{
						Role:       llm.RoleTool,
						Content:    result,
						ToolCallID: p.ID,
						CreatedAt:  millis(m.Time.Created),
					})
				}
			}
			if content.Len() == 0 && reasoning.Len() == 0 && len(calls) == 0 {
				continue
			}
			out = append(out, llm.Message{
				Role:      llm.RoleAssistant,
				Content:   content.String(),
				Reasoning: reasoning.String(),
				ToolCalls: calls,
				CreatedAt: millis(m.Time.Created),
			})
			out = append(out, results...)
		case "compaction":
			summary := strings.TrimSpace(m.Summary)
			if summary == "" {
				summary = strings.TrimSpace(m.Text)
			}
			if summary != "" {
				out = append(out, llm.Message{
					Role:              llm.RoleAssistant,
					Content:           summary,
					CompactionSummary: true,
					CreatedAt:         millis(m.Time.Created),
				})
			}
		case "system", "synthetic", "idle", "model-switched", "agent-switched", "location-switched", "skill", "shell":
			// Host/runtime events are not conversation turns for Coddy. Their
			// effect is either already in the transcript or not portable.
		}
	}
	return out, readOnly
}

func toolResult(state ToolState, p Part) string {
	var parts []string
	for _, c := range state.Content {
		if strings.TrimSpace(c.Text) != "" {
			parts = append(parts, c.Text)
		}
	}
	if len(parts) == 0 && strings.TrimSpace(state.Metadata.Output) != "" {
		parts = append(parts, state.Metadata.Output)
	}
	out := strings.Join(parts, "\n")
	if strings.TrimSpace(out) == "" {
		if state.Status == "error" && len(state.Error) > 0 {
			out = string(state.Error)
		} else {
			out = "(no output)"
		}
	}
	return out
}

func millis(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339Nano)
}

// ReadPassword reads one line from r, for --password-stdin.
func ReadPassword(r io.Reader) (string, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", errors.New("no password on stdin")
	}
	return strings.TrimRight(sc.Text(), "\r\n"), nil
}

// NormalizeBaseURL is exported for the UI and CLI to share one spelling.
func NormalizeBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}
