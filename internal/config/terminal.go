package config

import "strings"

// TerminalConfig turns on the terminal of a chat: one host shell per chat,
// served by GET/POST/DELETE /coddy/sessions/{id}/terminal and shown in a window
// of the web UI. It is off unless the operator asks for it, because a shell
// running the host's commands from a browser is a capability to grant on
// purpose, never one a default hands out.
//
// Enabling it opens nothing beyond this process's own origin, unless
// AllowedOrigins names the pages that may use it: a server read from a phone, or
// through another address, serves the UI from an origin of its own, and that
// page is refused until it is named here. The route refuses a session a subagent
// run owns and stays closed while the server is shutting down. It does not
// depend on httpserver.allow_insecure, on the web UI being built in, or on what
// the model may call: no model, tool or suggestion can write to this terminal.
type TerminalConfig struct {
	Enable bool `yaml:"enable" json:"enable"`
	// AllowedOrigins names the origins, besides this server's own, whose pages
	// may drive the terminal: "http://192.168.1.20:12345", or the bare
	// "192.168.1.20:12345". An entry is the page's origin, never a path, and
	// "*" is refused: a shell is granted to a page, not to the web.
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`
}

// Serves reports whether this configuration opens the terminal routes.
func (t *TerminalConfig) Serves() bool {
	return t != nil && t.Enable
}

// AllowsOrigin reports whether a page of this origin may drive the terminal.
//
// The scheme is ignored on both sides, as it is for the same-origin check: a
// proxy in front may terminate TLS, so the browser's "https://box" reaches a
// server that speaks plain http. Host and port are compared, so "http://box:1"
// never grants "http://box:2". "*" is never honoured.
func (t *TerminalConfig) AllowsOrigin(origin string) bool {
	if t == nil {
		return false
	}
	want := hostOfOrigin(origin)
	if want == "" {
		return false
	}
	for _, allowed := range t.AllowedOrigins {
		if host := hostOfOrigin(allowed); host != "" && strings.EqualFold(host, want) {
			return true
		}
	}
	return false
}

// hostOfOrigin drops the scheme and the trailing slash of an origin, leaving
// "host:port" to compare. Anything that is not an origin (a "*", a path, an
// opaque "null") answers empty, which no entry matches.
func hostOfOrigin(origin string) string {
	o := strings.TrimSpace(origin)
	if i := strings.Index(o, "://"); i >= 0 {
		o = o[i+3:]
	}
	o = strings.TrimSuffix(o, "/")
	if o == "" || o == "*" || strings.EqualFold(o, "null") || strings.ContainsAny(o, "/ ") {
		return ""
	}
	return o
}
