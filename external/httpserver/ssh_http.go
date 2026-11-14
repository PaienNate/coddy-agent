//go:build http

package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/sshcreds"
	toolsssh "github.com/EvilFreelancer/coddy-agent/internal/tools/ssh"
)

// sshBodyLimit bounds the credential document a client may send: a host list
// is small, and a password is shorter than this by orders of magnitude.
const sshBodyLimit = 64 << 10

// registerSSHRoutes serves the SSH credentials of the settings page and the
// Test button beside them. The routes live behind the API's gate like every
// other /coddy route; the password never leaves the process - neither GET nor
// the test answer carries it.
func (s *Server) registerSSHRoutes() {
	s.mux.HandleFunc("GET /coddy/ssh/hosts", s.coddySSHHostsList)
	s.mux.HandleFunc("PUT /coddy/ssh/hosts", s.coddySSHHostsPut)
	s.mux.HandleFunc("DELETE /coddy/ssh/hosts", s.coddySSHHostsDelete)
	s.mux.HandleFunc("POST /coddy/ssh/test", s.coddySSHTest)
}

func (s *Server) sshStore() *sshcreds.Store {
	home := ""
	if cfg := s.activeCfg(); cfg != nil {
		home = cfg.Paths.Home
	}
	return sshcreds.StoreForHome(home)
}

// sshHostJSON is one configured host as the API shows it: everything but the
// password, which is reported only as present or absent.
type sshHostJSON struct {
	Host string `json:"host"`
	User string `json:"user"`
	// Port is omitted when it is the default 22.
	Port int `json:"port,omitempty"`
	// HasPassword says a password (or a credential helper) is configured for
	// this host. The value itself is never returned.
	HasPassword     bool   `json:"has_password"`
	PasswordCommand string `json:"password_command,omitempty"`
	KeyPath         string `json:"key_path,omitempty"`
}

func sshHostView(e sshcreds.Entry) sshHostJSON {
	return sshHostJSON{
		Host:            e.Host,
		User:            e.User,
		Port:            e.Port,
		HasPassword:     e.Password != "" || e.PasswordCommand != "",
		PasswordCommand: e.PasswordCommand,
		KeyPath:         e.KeyPath,
	}
}

func (s *Server) coddySSHHostsList(w http.ResponseWriter, r *http.Request) {
	entries, err := s.sshStore().List()
	if err != nil {
		s.queueError(w, http.StatusInternalServerError, "ssh_credentials_unreadable", err)
		return
	}
	hosts := make([]sshHostJSON, 0, len(entries))
	for _, e := range entries {
		hosts = append(hosts, sshHostView(e))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"object": "coddy.ssh_hosts",
		"hosts":  hosts,
	})
}

// sshHostRequest is one host as a client writes it. An empty password keeps
// the stored one, so the form can save a host without ever holding the
// secret; password_command and key_path replace what is stored, and an empty
// value clears them.
type sshHostRequest struct {
	Host            string `json:"host"`
	User            string `json:"user"`
	Port            int    `json:"port"`
	Password        string `json:"password"`
	PasswordCommand string `json:"password_command"`
	KeyPath         string `json:"key_path"`
}

func (s *Server) coddySSHHostsPut(w http.ResponseWriter, r *http.Request) {
	var req sshHostRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, sshBodyLimit)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		s.queueError(w, http.StatusBadRequest, "invalid_body", err)
		return
	}
	entry := sshcreds.Entry{
		Host:            req.Host,
		User:            req.User,
		Port:            req.Port,
		Password:        req.Password,
		PasswordCommand: req.PasswordCommand,
		KeyPath:         req.KeyPath,
	}
	store := s.sshStore()
	if err := store.Upsert(entry); err != nil {
		s.queueError(w, http.StatusBadRequest, "invalid_ssh_host", err)
		return
	}
	saved, _, err := store.Get(entry.Host, entry.User, entry.Port)
	if err != nil {
		s.queueError(w, http.StatusInternalServerError, "ssh_credentials_unreadable", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"object": "coddy.ssh_host",
		"host":   sshHostView(saved),
	})
}

func (s *Server) coddySSHHostsDelete(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	host := strings.TrimSpace(q.Get("host"))
	user := strings.TrimSpace(q.Get("user"))
	port := 0
	if raw := strings.TrimSpace(q.Get("port")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			s.queueError(w, http.StatusBadRequest, "invalid_port", err)
			return
		}
		port = n
	}
	if host == "" || user == "" {
		s.queueError(w, http.StatusBadRequest, "missing_host_user", errors.New("host and user are required"))
		return
	}
	removed, err := s.sshStore().Delete(host, user, port)
	if err != nil {
		s.queueError(w, http.StatusInternalServerError, "ssh_credentials_unwritable", err)
		return
	}
	if !removed {
		s.queueError(w, http.StatusNotFound, "ssh_host_not_found", errors.New("no such SSH host"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"object":  "coddy.ssh_host",
		"deleted": true,
	})
}

// coddySSHTest dials a configured host with the tool's authentication chain
// and closes the connection without running anything. The answer is 200 with
// ok=false when the remote side refused or failed: a failing test is a result
// of the test, not an API error, and the page shows the reason as written.
func (s *Server) coddySSHTest(w http.ResponseWriter, r *http.Request) {
	var req sshHostRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, sshBodyLimit)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		s.queueError(w, http.StatusBadRequest, "invalid_body", err)
		return
	}
	store := s.sshStore()
	entry, found, err := store.Get(req.Host, req.User, req.Port)
	if err != nil {
		s.queueError(w, http.StatusInternalServerError, "ssh_credentials_unreadable", err)
		return
	}
	if !found {
		// A host the form has not saved yet is still worth testing: the
		// request carries the credentials it typed.
		entry = sshcreds.Entry{
			Host:            req.Host,
			User:            req.User,
			Port:            req.Port,
			Password:        req.Password,
			PasswordCommand: req.PasswordCommand,
			KeyPath:         req.KeyPath,
		}
	}
	start := time.Now()
	testErr := toolsssh.TestConnection(r.Context(), entry, "")
	elapsed := time.Since(start).Milliseconds()
	payload := map[string]interface{}{
		"object":     "coddy.ssh_test",
		"ok":         testErr == nil,
		"latency_ms": elapsed,
	}
	if testErr != nil {
		payload["message"] = testErr.Error()
	}
	writeJSON(w, http.StatusOK, payload)
}
