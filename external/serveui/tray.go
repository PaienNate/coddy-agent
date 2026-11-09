// Package serveui is the notification-area control of `coddy serve`.
//
// It is deliberately small: the server keeps running whether or not a tray can
// be shown, so this package owns only the icon, the folder chooser and the
// three rows that open the page, open a project and stop the process. A
// platform without a tray, or a session without an interactive desktop, gets a
// no-op Start.
package serveui

import (
	"context"
	"log/slog"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// Options is what the tray needs from the process that serves.
type Options struct {
	// URL is the web UI's root, for example http://127.0.0.1:12345/.
	URL string
	// RecentFolders are the workspaces of this home's recent chats, newest
	// first. The tray lists the first few under Open project.
	RecentFolders []string
	// OpenURL hands an address to the machine's browser.
	OpenURL func(string) error
	// NewSession creates a chat in a folder and answers its session id. The
	// tray opens that session's page, which is the workspace the operator
	// picked.
	NewSession func(folder string) (string, error)
	// Quit ends the serve run. The tray calls it for Exit; the process stops
	// through its ordinary supervisor shutdown.
	Quit func()
	// Log receives tray failures; nil discards them.
	Log *slog.Logger
}

// maxRecentFolders is how many recent workspaces a tray menu offers. A tray
// menu is read at a glance; more rows are rows nobody reads.
const maxRecentFolders = 6

// sessionURL is the page of one chat under the web UI root.
func sessionURL(baseURL, sessionID string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	sessionID = strings.TrimSpace(sessionID)
	if baseURL == "" || sessionID == "" {
		return baseURL
	}
	return baseURL + "/#/s/" + url.PathEscape(sessionID)
}

// recentFolders filters the rows a tray can offer: absolute paths, each once,
// cleaned, newest first, bounded to what a menu can carry.
func recentFolders(folders []string) []string {
	out := make([]string, 0, maxRecentFolders)
	seen := make(map[string]bool, len(folders))
	for _, raw := range folders {
		folder := strings.TrimSpace(raw)
		if folder == "" || !filepath.IsAbs(folder) {
			continue
		}
		folder = filepath.Clean(folder)
		key := folder
		if runtime.GOOS == "windows" {
			// Windows paths compare case-insensitively; two spellings of one
			// folder are one row in the menu.
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, folder)
		if len(out) == maxRecentFolders {
			break
		}
	}
	return out
}

// folderLabel names a row: the folder and the one above it, which is what
// tells two checkouts of one repository apart. The whole path stays in the
// click target.
func folderLabel(folder string) string {
	folder = filepath.Clean(folder)
	parent, name := filepath.Split(folder)
	if name == "" {
		name = folder
	}
	above := filepath.Base(filepath.Clean(parent))
	if above != "" && above != "." && above != string(filepath.Separator) {
		name = above + "/" + name
	}
	return strings.ReplaceAll(strings.ReplaceAll(name, "\n", " "), "\r", " ")
}

// Available reports whether this build has a serve tray. The caller checks it
// before reading the session store for recent workspaces, so a Linux server
// does not pay for a menu it can never draw.
func Available() bool { return platformAvailable() }

// Start shows the tray and answers a function that takes it down. Platforms
// without a tray, and a tray that cannot be shown, still answer a harmless
// stop function: the server is the product, the icon is a convenience.
func Start(ctx context.Context, opts Options) func() { return startPlatformTray(ctx, opts) }
