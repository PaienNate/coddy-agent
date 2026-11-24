package config

// TerminalConfig turns on the terminal of a chat: one host shell per chat,
// served by GET/POST/DELETE /coddy/sessions/{id}/terminal and shown in a window
// of the web UI. It is off unless the operator asks for it, because a shell
// running the host's commands from a browser is a capability to grant on
// purpose, never one a default hands out.
//
// Enabling it opens nothing beyond this process's own origin: the route refuses
// a request another origin made and a session a subagent run owns, and it stays
// closed while the server is shutting down. It does not depend on
// httpserver.allow_insecure, on the web UI being built in, or on what the model
// may call: no model, tool or suggestion can write to this terminal.
type TerminalConfig struct {
	Enable bool `yaml:"enable" json:"enable"`
}

// Serves reports whether this configuration opens the terminal routes.
func (t *TerminalConfig) Serves() bool {
	return t != nil && t.Enable
}
