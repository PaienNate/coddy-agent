package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// openControlState is the loopback endpoint the Explorer verb talks to. It is
// kept under the agent home with owner-only permissions, because the token in
// it is a capability to create a chat; it is not the web UI's own credential.
type openControlState struct {
	URL     string `json:"url"`
	Control string `json:"control"`
	Token   string `json:"token"`
}

func openControlPath(home string) string {
	return filepath.Join(home, "serve", "open.json")
}

func readOpenControl(home string) (openControlState, error) {
	var st openControlState
	b, err := os.ReadFile(openControlPath(home))
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, fmt.Errorf("read %s: %w", openControlPath(home), err)
	}
	if strings.TrimSpace(st.URL) == "" || strings.TrimSpace(st.Control) == "" || strings.TrimSpace(st.Token) == "" {
		return st, fmt.Errorf("%s is missing a field", openControlPath(home))
	}
	return st, nil
}

func writeOpenControl(home string, st openControlState) error {
	path := openControlPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "open-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func newOpenToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// validOpenFolder is the one thing the Explorer verb may ask a running server
// to open: an absolute directory that exists. A relative path is resolved
// against the invoking process, which is the folder Explorer was launched in.
func validOpenFolder(raw string) (string, error) {
	folder := strings.TrimSpace(raw)
	if folder == "" {
		return "", errors.New("a folder is required")
	}
	if !filepath.IsAbs(folder) {
		abs, err := filepath.Abs(folder)
		if err != nil {
			return "", err
		}
		folder = abs
	}
	folder = filepath.Clean(folder)
	info, err := os.Stat(folder)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", folder)
	}
	return folder, nil
}

// startOpenControl opens a loopback HTTP endpoint for the Explorer verb. The
// endpoint is separate from the web UI's listener on purpose: it works with
// whatever authentication the UI carries, and the only caller that knows its
// token is this user's own `coddy open` process.
func startOpenControl(
	ctx context.Context,
	home, webURL string,
	newSession func(folder string) (string, error),
	log *slog.Logger,
) (func(), error) {
	if newSession == nil {
		return nil, errors.New("no session manager for the Explorer verb")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	token, err := newOpenToken()
	if err != nil {
		_ = ln.Close()
		return nil, err
	}
	state := openControlState{
		URL:     webURL,
		Control: "http://" + ln.Addr().String() + "/open",
		Token:   token,
	}
	if err := writeOpenControl(home, state); err != nil {
		_ = ln.Close()
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /open", func(w http.ResponseWriter, r *http.Request) {
		want := "Bearer " + token
		got := r.Header.Get("Authorization")
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var body struct {
			CWD string `json:"cwd"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		folder, err := validOpenFolder(body.CWD)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, err := newSession(folder)
		if err != nil {
			if log != nil {
				log.Warn("the Explorer verb could not create a session", "folder", folder, "error", err)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"sessionId": id})
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) && log != nil {
			log.Debug("the Explorer control listener stopped", "error", err)
		}
	}()

	stop := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		// Only remove the state this server wrote; a newer serve may already
		// have replaced it.
		if current, err := readOpenControl(home); err == nil && current.Token == token {
			_ = os.Remove(openControlPath(home))
		}
	}
	go func() {
		<-ctx.Done()
		stop()
	}()
	return stop, nil
}
