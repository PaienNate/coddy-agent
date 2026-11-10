//go:build !windows

package serveui

import "context"

func platformAvailable() bool { return false }

// runPlatformTray is a no-op where Coddy has no notification area to put an
// icon in. Linux keeps its dedicated desktop tray in external/desktop; macOS
// and the BSDs have no serve tray yet.
func runPlatformTray(ctx context.Context, _ Options, serve func(context.Context) error) error {
	return serve(ctx)
}
