//go:build windows

package serveui

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"

	"github.com/fy0/systray"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// The mark of the serve tray: blue ground, white 派恩.
//
//go:embed assets/pine-tray.ico
var trayIcon []byte

const (
	bifReturnOnlyFSDirs = 0x00000001
	bifEditBox          = 0x00000010
	bifNewDialogStyle   = 0x00000040
)

func platformAvailable() bool { return true }

type serveTray struct {
	opts Options
	log  *slog.Logger
	// opening serialises the folder chooser and the session it creates. Two
	// clicks on a recent row must not create two chats while one dialog is
	// still open.
	opening sync.Mutex
}

// startPlatformTray starts the Windows notification-area icon in its own
// locked OS thread. The server keeps running when this package cannot show an
// icon; every failure is logged rather than returned.
func startPlatformTray(ctx context.Context, opts Options) func() {
	if strings.TrimSpace(opts.URL) == "" {
		return func() {}
	}
	t := &serveTray{opts: opts, log: logger(opts.Log)}

	go func() {
		runtime.LockOSThread()
		systray.Run(t.ready, func() {})
	}()
	go func() {
		<-ctx.Done()
		systray.Quit()
	}()
	return systray.Quit
}

func logger(l *slog.Logger) *slog.Logger {
	if l != nil {
		return l
	}
	return slog.New(slog.DiscardHandler)
}

func (t *serveTray) ready() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip("Coddy 后台服务")

	openPage := systray.AddMenuItem("打开页面", "在默认浏览器中打开 Coddy")
	openProject := systray.AddMenuItem("打开项目", "选择一个项目目录并打开工作区")
	chooseFolder := openProject.AddSubMenuItem("选择文件夹…", "浏览本机目录")
	recent := recentFolders(t.opts.RecentFolders)
	for _, folder := range recent {
		item := openProject.AddSubMenuItem(folderLabel(folder), folder)
		go t.loopProject(item, folder)
	}
	systray.AddSeparator()
	quit := systray.AddMenuItem("退出", "停止 Coddy 后台服务")

	go func() {
		for range openPage.ClickedCh {
			t.open(t.opts.URL)
		}
	}()
	go t.loopProject(chooseFolder, "")
	go func() {
		for range quit.ClickedCh {
			t.quit()
			return
		}
	}()
	t.log.Info("serve tray ready", "recent_workspaces", len(recent))
}

func (t *serveTray) loopProject(item *systray.MenuItem, folder string) {
	for range item.ClickedCh {
		t.openProject(folder)
	}
}

func (t *serveTray) openProject(folder string) {
	t.opening.Lock()
	defer t.opening.Unlock()

	folder = strings.TrimSpace(folder)
	if folder == "" {
		var err error
		folder, err = pickFolder()
		if err != nil {
			t.fail("选择项目目录失败", err)
			return
		}
		if folder == "" {
			return // the operator cancelled
		}
	}
	if t.opts.NewSession == nil {
		t.open(t.opts.URL)
		return
	}
	id, err := t.opts.NewSession(folder)
	if err != nil {
		t.fail("创建工作区失败", err)
		return
	}
	t.open(sessionURL(t.opts.URL, id))
}

func (t *serveTray) open(target string) {
	if t.opts.OpenURL == nil {
		t.log.Warn("the server has no browser opener")
		return
	}
	if err := t.opts.OpenURL(target); err != nil {
		t.fail("无法打开页面", err)
	}
}

func (t *serveTray) quit() {
	t.log.Info("serve tray requested shutdown")
	if t.opts.Quit != nil {
		t.opts.Quit()
	}
	systray.Quit()
}

func (t *serveTray) fail(title string, err error) {
	t.log.Error(title, "error", err)
	text, textErr := windows.UTF16PtrFromString(title + "\n\n" + err.Error())
	caption, captionErr := windows.UTF16PtrFromString("Coddy")
	if textErr != nil || captionErr != nil {
		return
	}
	win.MessageBox(0, text, caption, win.MB_OK|win.MB_ICONERROR)
}

// pickFolder is the system folder chooser. SHBrowseForFolder is the dialog
// Windows has always had for "choose a directory"; it can type a path, browse
// drives and create a folder without Coddy drawing a dialog of its own.
func pickFolder() (string, error) {
	hr := win.OleInitialize()
	defer win.OleUninitialize()
	if hr != 0 && hr != 1 { // S_OK or S_FALSE: already initialised on this thread
		return "", fmt.Errorf("OleInitialize: %#x", uint32(hr))
	}

	display := make([]uint16, win.MAX_PATH)
	title, err := windows.UTF16PtrFromString("选择项目目录")
	if err != nil {
		return "", err
	}
	info := win.BROWSEINFO{
		PszDisplayName: &display[0],
		LpszTitle:      title,
		UlFlags:        bifReturnOnlyFSDirs | bifEditBox | bifNewDialogStyle,
	}
	pidl := win.SHBrowseForFolder(&info)
	if pidl == 0 {
		return "", nil // cancelled
	}
	defer win.CoTaskMemFree(pidl)

	path := make([]uint16, win.MAX_PATH)
	if !win.SHGetPathFromIDList(pidl, &path[0]) {
		return "", errors.New("the folder chooser returned no path")
	}
	return strings.TrimSpace(win.UTF16PtrToString(&path[0])), nil
}
