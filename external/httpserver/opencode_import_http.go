//go:build http

package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/opencodeimport"
	"github.com/EvilFreelancer/coddy-agent/internal/platform"
)

// opencodeImportJob is the progress of one background migration. The latest
// snapshot is also written under ${CODDY_HOME}/opencode-import/jobs so a
// restart can still answer for it.
type opencodeImportJob struct {
	ID      string                       `json:"id"`
	Started string                       `json:"started"`
	Done    bool                         `json:"done"`
	Error   string                       `json:"error,omitempty"`
	DoneN   int                          `json:"done_sessions"`
	TotalN  int                          `json:"total_sessions"`
	Current string                       `json:"current_session,omitempty"`
	Result  *opencodeimport.ImportResult `json:"result,omitempty"`
	home    string                       `json:"-"`
}

type opencodeConfigDTO struct {
	URL                string `json:"url"`
	Username           string `json:"username"`
	PasswordConfigured bool   `json:"password_configured"`
}

type opencodeTestDTO struct {
	Version  string `json:"version"`
	Projects int    `json:"projects"`
	Sessions int    `json:"sessions"`
}

type opencodeImportRequest struct {
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DryRun       bool   `json:"dry_run"`
	SkipArchived bool   `json:"skip_archived"`
	ProjectID    string `json:"project"`
	Concurrency  int    `json:"concurrency"`
	Replace      bool   `json:"replace"`
}

func (s *Server) registerOpenCodeImportRoutes() {
	s.mux.HandleFunc("GET /coddy/opencode", s.opencodeConfigGet)
	s.mux.HandleFunc("PUT /coddy/opencode", s.opencodeConfigPut)
	s.mux.HandleFunc("POST /coddy/opencode/test", s.opencodeTest)
	s.mux.HandleFunc("POST /coddy/opencode/import", s.opencodeImportStart)
	s.mux.HandleFunc("GET /coddy/opencode/import/{id}", s.opencodeImportStatus)
}

func (s *Server) opencodeConfigGet(w http.ResponseWriter, _ *http.Request) {
	cfg := s.activeCfg()
	if cfg == nil || strings.TrimSpace(cfg.Paths.Home) == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Coddy home is not configured"})
		return
	}
	conn, err := opencodeimport.LoadConfig(cfg.Paths.Home)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, opencodeConfigDTO{
		URL:                conn.URL,
		Username:           conn.Username,
		PasswordConfigured: strings.TrimSpace(conn.Password) != "",
	})
}

func (s *Server) opencodeConfigPut(w http.ResponseWriter, r *http.Request) {
	cfg := s.activeCfg()
	if cfg == nil || strings.TrimSpace(cfg.Paths.Home) == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Coddy home is not configured"})
		return
	}
	var body struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	conn, err := opencodeimport.LoadConfig(cfg.Paths.Home)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.URL) != "" {
		conn.URL = opencodeimport.NormalizeBaseURL(body.URL)
	}
	if strings.TrimSpace(body.Username) != "" {
		conn.Username = strings.TrimSpace(body.Username)
	}
	if body.Password != "" {
		conn.Password = body.Password
	}
	u, err := url.Parse(conn.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "URL must be an http(s) address"})
		return
	}
	if conn.Username == "" {
		conn.Username = "opencode"
	}
	if err := opencodeimport.SaveConfig(cfg.Paths.Home, conn); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, opencodeConfigDTO{
		URL:                conn.URL,
		Username:           conn.Username,
		PasswordConfigured: strings.TrimSpace(conn.Password) != "",
	})
}

func (s *Server) opencodeTest(w http.ResponseWriter, r *http.Request) {
	cfg := s.activeCfg()
	if cfg == nil || strings.TrimSpace(cfg.Paths.Home) == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Coddy home is not configured"})
		return
	}
	conn, err := opencodeimport.LoadConfig(cfg.Paths.Home)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	var body struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body)
	if strings.TrimSpace(body.URL) != "" {
		conn.URL = opencodeimport.NormalizeBaseURL(body.URL)
	}
	if strings.TrimSpace(body.Username) != "" {
		conn.Username = strings.TrimSpace(body.Username)
	}
	if body.Password != "" {
		conn.Password = body.Password
	}
	if strings.TrimSpace(conn.URL) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "OpenCode URL is not configured"})
		return
	}
	client := opencodeimport.NewClient(conn)
	version, err := client.Health(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	projects, err := client.Projects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	sessions := 0
	for _, p := range projects {
		found, err := client.Sessions(r.Context(), p.ID, true)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		sessions += len(found)
	}
	writeJSON(w, http.StatusOK, opencodeTestDTO{Version: version, Projects: len(projects), Sessions: sessions})
}

func (s *Server) opencodeImportStart(w http.ResponseWriter, r *http.Request) {
	cfg := s.activeCfg()
	if cfg == nil || strings.TrimSpace(cfg.Paths.Home) == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Coddy home is not configured"})
		return
	}
	var body opencodeImportRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	conn, err := opencodeimport.LoadConfig(cfg.Paths.Home)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.URL) != "" {
		conn.URL = opencodeimport.NormalizeBaseURL(body.URL)
	}
	if strings.TrimSpace(body.Username) != "" {
		conn.Username = strings.TrimSpace(body.Username)
	}
	if body.Password != "" {
		conn.Password = body.Password
	}
	if strings.TrimSpace(conn.URL) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "OpenCode URL is not configured"})
		return
	}
	client := opencodeimport.NewClient(conn)
	version, err := client.Health(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if !strings.HasPrefix(strings.TrimSpace(version), "2.") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("unsupported OpenCode version %q; v2 is required", version)})
		return
	}
	projects, err := client.Projects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.ProjectID) != "" {
		var one []opencodeimport.Project
		for _, p := range projects {
			if p.ID == strings.TrimSpace(body.ProjectID) {
				one = append(one, p)
				break
			}
		}
		if len(one) == 0 {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "OpenCode project not found"})
			return
		}
		projects = one
	}
	if body.SkipArchived {
		active := projects[:0]
		for _, p := range projects {
			if p.ArchivedAt() == nil {
				active = append(active, p)
			}
		}
		projects = active
	}
	if len(projects) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "OpenCode returned no projects"})
		return
	}

	id := newOpenCodeJobID()
	job := &opencodeImportJob{ID: id, Started: time.Now().UTC().Format(time.RFC3339), Done: body.DryRun, home: cfg.Paths.Home}
	s.opencodeMu.Lock()
	s.opencodeJobs[id] = job
	s.opencodeMu.Unlock()
	s.persistOpenCodeJob(job)

	opts := opencodeimport.ImportOptions{
		Projects:        projects,
		IncludeArchived: !body.SkipArchived,
		Concurrency:     body.Concurrency,
		DryRun:          body.DryRun,
		Replace:         body.Replace,
		SessionsRoot:    cfg.ResolvedSessionsRoot(),
		FallbackCWD:     s.defaultCWD,
		Progress: func(done, total int, sessionID string) {
			s.opencodeMu.Lock()
			job.DoneN, job.TotalN, job.Current = done, total, sessionID
			s.opencodeMu.Unlock()
			s.persistOpenCodeJob(job)
		},
	}
	ctx := r.Context()
	if !body.DryRun {
		ctx = context.Background()
	}
	run := func() {
		res, err := opencodeimport.ImportAll(ctx, client, opts)
		s.opencodeMu.Lock()
		defer s.opencodeMu.Unlock()
		job.Done = true
		job.Result = res
		if err != nil {
			job.Error = err.Error()
		}
	}
	if body.DryRun {
		run()
		s.persistOpenCodeJob(job)
		writeJSON(w, http.StatusOK, s.snapshotOpenCodeJob(job))
		return
	}
	go func() {
		run()
		s.persistOpenCodeJob(job)
	}()
	writeJSON(w, http.StatusAccepted, s.snapshotOpenCodeJob(job))
}

func (s *Server) opencodeImportStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if !strings.HasPrefix(id, "opencode-") || strings.ContainsAny(id, "/\\") {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "import job not found"})
		return
	}
	s.opencodeMu.Lock()
	job := s.opencodeJobs[id]
	s.opencodeMu.Unlock()
	if job == nil {
		job = s.loadOpenCodeJob(id)
		if job == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "import job not found"})
			return
		}
		// A job that was running when the process stopped did not finish; the
		// bundles it wrote are still on disk, and the report says so honestly.
		if !job.Done {
			job.Done = true
			job.Error = "import interrupted by a server restart"
			s.persistOpenCodeJob(job)
		}
	}
	writeJSON(w, http.StatusOK, s.snapshotOpenCodeJob(job))
}

func (s *Server) snapshotOpenCodeJob(job *opencodeImportJob) opencodeImportJob {
	s.opencodeMu.Lock()
	defer s.opencodeMu.Unlock()
	return *job
}

func opencodeJobPath(home, id string) string {
	return filepath.Join(home, "opencode-import", "jobs", id+".json")
}

func (s *Server) persistOpenCodeJob(job *opencodeImportJob) {
	if job == nil || strings.TrimSpace(job.home) == "" {
		return
	}
	s.opencodeMu.Lock()
	copy := *job
	s.opencodeMu.Unlock()
	data, err := json.MarshalIndent(copy, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')
	path := opencodeJobPath(job.home, job.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	_, _ = platform.WriteFileReplacing(path, data, 0o600)
}

func (s *Server) loadOpenCodeJob(id string) *opencodeImportJob {
	cfg := s.activeCfg()
	if cfg == nil || strings.TrimSpace(cfg.Paths.Home) == "" {
		return nil
	}
	data, err := os.ReadFile(opencodeJobPath(cfg.Paths.Home, id))
	if err != nil {
		return nil
	}
	var job opencodeImportJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil
	}
	job.home = cfg.Paths.Home
	s.opencodeMu.Lock()
	s.opencodeJobs[id] = &job
	s.opencodeMu.Unlock()
	return &job
}

func newOpenCodeJobID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("opencode-%d", time.Now().UnixNano())
	}
	return "opencode-" + hex.EncodeToString(b[:])
}
