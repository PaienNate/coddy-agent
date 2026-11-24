//go:build http

package httpserver

import (
	"net/http"
	"strings"
)

// isTerminalPath reports whether the path is one of the chat terminal's
// routes, which a page named by terminal.allowed_origins may call from its own
// origin.
func isTerminalPath(path string) bool {
	return strings.HasPrefix(path, "/coddy/sessions/") && strings.HasSuffix(path, "/terminal")
}

// corsMiddleware answers CORS preflight requests and adds Access-Control-* headers for allowed
// origins. It wraps the auth gate so preflight OPTIONS (which carry no Authorization header) are
// handled before authentication. It is a transparent pass-through when CORS is disabled.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A canvas frame runs in an opaque origin, so its own module scripts
		// and fetches are cross-origin requests from "null": the frame's files
		// answer any origin, without credentials, whatever httpserver.cors
		// says, including the preflight of a fetch with a header of its own.
		if isCanvasFilePath(r.URL.Path) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", "*")
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", "GET, HEAD")
				if req := r.Header.Get("Access-Control-Request-Headers"); req != "" {
					h.Set("Access-Control-Allow-Headers", req)
				}
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" {
			if c := s.activeCfg(); c != nil {
				allow, ok := c.HTTPServer.CORSAllowOrigin(origin)
				// A page named by terminal.allowed_origins may drive the chat
				// terminal from its own origin, which the API's own CORS list
				// never names: the browser has to read the answer there, and a
				// page signed in by cookie needs the credential flag, so the
				// origin is echoed rather than widened to "*".
				terminal := false
				if !ok && isTerminalPath(r.URL.Path) && c.Terminal.AllowsOrigin(origin) {
					allow, ok, terminal = origin, true, true
				}
				if ok {
					h := w.Header()
					h.Set("Access-Control-Allow-Origin", allow)
					h.Add("Vary", "Origin")
					h.Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Coddy-Session-ID, Range, If-None-Match")
					h.Set("Access-Control-Expose-Headers", "Content-Range, Accept-Ranges, ETag, Content-Disposition")
					h.Set("Access-Control-Max-Age", "600")
					if terminal {
						h.Set("Access-Control-Allow-Credentials", "true")
					}
				}
			}
		}
		// Preflight: short-circuit before auth. A disallowed origin simply lacks the ACAO header.
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
