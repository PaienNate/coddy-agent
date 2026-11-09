package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/EvilFreelancer/coddy-agent/external/serveui"
	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/platform"
)

// runOpen is the verb an Explorer context-menu row runs. It is deliberately a
// separate process: Explorer does not share the serve process's memory, and
// this is the only command that needs to reach it.
func runOpen(args []string) error {
	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	homeDir := fs.String("home", "", "agent state directory (CODDY_HOME, default ~/.coddy)")
	cwd := fs.String("cwd", "", "folder to open as a workspace")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: coddy open [--home DIR] <folder> (open a folder in the running coddy serve)\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("coddy open takes one folder, got %d", fs.NArg())
	}
	folder := strings.TrimSpace(*cwd)
	if folder == "" && fs.NArg() == 1 {
		folder = fs.Arg(0)
	}
	if folder == "" {
		wd, err := os.Getwd()
		if err != nil {
			showOpenError(err)
			return err
		}
		folder = wd
	}
	folder, err := validOpenFolder(folder)
	if err != nil {
		showOpenError(err)
		return err
	}
	paths, err := config.Resolve(config.CLIPaths{Home: strings.TrimSpace(*homeDir)})
	if err != nil {
		showOpenError(err)
		return err
	}
	state, err := readOpenControl(paths.Home)
	if err != nil {
		err = errors.New("Coddy serve is not running; start `coddy serve` and open the folder from its tray icon")
		showOpenError(err)
		return err
	}
	target, err := requestOpenProject(state, folder)
	if err != nil {
		showOpenError(err)
		return err
	}
	if err := platform.OpenBrowser(target); err != nil {
		showOpenError(err)
		return err
	}
	fmt.Println(target)
	return nil
}

func requestOpenProject(state openControlState, folder string) (string, error) {
	body, err := json.Marshal(map[string]string{"cwd": folder})
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, state.Control, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+state.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("Coddy serve did not answer: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return "", fmt.Errorf("Coddy serve answered %s: %s", res.Status, strings.TrimSpace(string(msg)))
	}
	var out struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.SessionID) == "" {
		return "", errors.New("Coddy serve returned no session id")
	}
	return serveui.SessionURL(state.URL, out.SessionID), nil
}
