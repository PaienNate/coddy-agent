//go:build !windows

package serveui

import "context"

func platformAvailable() bool { return false }

// Start is a no-op where Coddy has no notification area to put an icon in.
// Linux keeps its dedicated desktop tray in external/desktop; macOS and the
// BSDs have no serve tray yet.
func startPlatformTray(context.Context, Options) func() { return func() {} }
