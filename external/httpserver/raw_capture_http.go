//go:build http

package httpserver

import (
	"errors"
	"net/http"

	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/llm"
)

// registerRawCaptureRoutes serves the raw request/response viewer: a debug aid
// that shows what actually went over the wire to a provider. The routes exist
// only while httpserver.raw_capture is enabled; when it is off the GET answers
// {"enabled":false} and the DELETE is a 404, so the SPA shows no panel. What
// the recorder keeps is bound and in memory only.
func (s *Server) registerRawCaptureRoutes() {
	s.mux.HandleFunc("GET /coddy/raw-capture", s.coddyRawCaptureGet)
	s.mux.HandleFunc("DELETE /coddy/raw-capture", s.coddyRawCaptureDelete)
}

// syncRawCapture brings the process-wide recorder in line with the served
// configuration: install one when httpserver.raw_capture is on, remove it when
// off. Called at New and again on every ReplaceConfig.
func (s *Server) syncRawCapture(cfg *config.Config) {
	if cfg != nil && cfg.HTTPServer.RawCapture {
		if s.rawCapture == nil {
			s.rawCapture = llm.NewRawCaptureRecorder(100)
		}
		llm.SetRawCaptureRecorder(s.rawCapture)
		return
	}
	// Capture is off: the transport passes every request through untouched.
	// Keep the server's own reference so a later reload that turns it back on
	// reuses the ring rather than starting empty.
	llm.SetRawCaptureRecorder(nil)
}

func (s *Server) coddyRawCaptureGet(w http.ResponseWriter, r *http.Request) {
	if s.rawCapture == nil {
		writeJSON(w, http.StatusOK, rawCapturePayload(false, nil))
		return
	}
	writeJSON(w, http.StatusOK, rawCapturePayload(true, s.rawCapture.Entries()))
}

func (s *Server) coddyRawCaptureDelete(w http.ResponseWriter, r *http.Request) {
	if s.rawCapture == nil {
		s.queueError(w, http.StatusNotFound, "not_enabled", errors.New("raw capture is not enabled"))
		return
	}
	s.rawCapture.Clear()
	writeJSON(w, http.StatusOK, rawCapturePayload(true, nil))
}

func rawCapturePayload(enabled bool, entries []llm.RawCaptureEntry) map[string]interface{} {
	if entries == nil {
		entries = []llm.RawCaptureEntry{}
	}
	return map[string]interface{}{
		"object":  "coddy.raw_capture",
		"enabled": enabled,
		"entries": entries,
	}
}
