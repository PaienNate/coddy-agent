//go:build windows

package serveui

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fy0/systray"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
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
	// readyCh is closed once the notification icon and its menu exist. The
	// caller uses it to tell a tray that started from one that could not.
	readyCh   chan struct{}
	readyOnce sync.Once
}

// runPlatformTray starts the Windows notification-area icon on the calling
// goroutine. The systray message loop owns this OS thread: running it in a
// goroutine makes Shell_NotifyIcon fail silently on Windows, which is why the
// old Start implementation left the server up without an icon. The server runs
// beside the tray, and a tray that cannot be shown leaves it running.
func runPlatformTray(ctx context.Context, opts Options, serve func(context.Context) error) error {
	if strings.TrimSpace(opts.URL) == "" {
		return serve(ctx)
	}
	t := &serveTray{
		opts:    opts,
		log:     logger(opts.Log),
		readyCh: make(chan struct{}),
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- serve(runCtx)
		select {
		case <-t.readyCh:
			systray.Quit()
		case <-runCtx.Done():
		}
	}()
	go func() {
		<-runCtx.Done()
		systray.Quit()
	}()

	systray.Run(t.readyTray, func() {})

	// A tray that never became ready failed to initialise. The server is
	// already running: leave it alone until its own context ends instead of
	// cancelling it just because the icon could not be shown.
	select {
	case <-t.readyCh:
		cancel()
	default:
	}
	return <-done
}

func (t *serveTray) readyTray() {
	t.ready()
	t.readyOnce.Do(func() { close(t.readyCh) })
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
	contextMenu := systray.AddMenuItemCheckbox(
		"注册右键菜单",
		"在资源管理器中显示“用 Coddy 打开”",
		contextMenuEnabled(t.opts.Home),
	)
	systray.AddSeparator()
	quit := systray.AddMenuItem("退出", "停止 Coddy 后台服务")

	go func() {
		for range openPage.ClickedCh {
			t.open(t.opts.URL)
		}
	}()
	go t.loopProject(chooseFolder, "")
	go func() {
		for range contextMenu.ClickedCh {
			enable := !contextMenu.Checked()
			if err := setContextMenu(enable, t.opts.Home); err != nil {
				t.fail("资源管理器右键菜单设置失败", err)
				continue
			}
			if enable {
				contextMenu.Check()
			} else {
				contextMenu.Uncheck()
			}
		}
	}()
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

// contextMenuEntry is one Explorer verb: a folder, the background of a folder,
// or a drive root. The argument is the one Explorer substitutes for the chosen
// path in each place.
type contextMenuEntry struct {
	key string
	arg string
}

var contextMenuEntries = []contextMenuEntry{
	{key: `Software\Classes\Directory\shell\Coddy`, arg: "%1"},
	{key: `Software\Classes\Directory\Background\shell\Coddy`, arg: "%V"},
	{key: `Software\Classes\Drive\shell\Coddy`, arg: "%1"},
}

const contextMenuLabel = "用 Coddy 打开"

// contextMenuCommand is what Explorer runs: this binary, the open verb, the
// agent home, and the path Explorer substitutes.
func contextMenuCommand(home, arg string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	exe = filepath.Clean(exe)
	command := quoteWindowsArg(exe) + " open"
	if home = strings.TrimSpace(home); home != "" {
		command += " --home " + quoteWindowsArg(filepath.Clean(home))
	}
	return command + " " + quoteWindowsArg(arg), nil
}

func quoteWindowsArg(s string) string {
	if s == "%1" || s == "%V" {
		// Explorer substitutes a path that may contain spaces; the placeholder
		// itself has none, so it is quoted explicitly.
		return `"` + s + `"`
	}
	return windows.EscapeArg(s)
}

// contextMenuEnabled reports whether all three verbs are registered for this
// binary and home. A stale entry from a moved binary reads as off, which is
// what the checkbox should say.
func contextMenuEnabled(home string) bool {
	for _, entry := range contextMenuEntries {
		want, err := contextMenuCommand(home, entry.arg)
		if err != nil {
			return false
		}
		key, err := registry.OpenKey(registry.CURRENT_USER, entry.key+`\command`, registry.QUERY_VALUE)
		if err != nil {
			return false
		}
		got, _, err := key.GetStringValue("")
		_ = key.Close()
		if err != nil || !strings.EqualFold(got, want) {
			return false
		}
	}
	return true
}

// setContextMenu writes or removes the per-user Explorer verbs. HKCU needs no
// administrator rights, and Windows 11 shows the classic verbs under Show more
// options.
func setContextMenu(enable bool, home string) error {
	for _, entry := range contextMenuEntries {
		if !enable {
			_ = registry.DeleteKey(registry.CURRENT_USER, entry.key+`\command`)
			if err := registry.DeleteKey(registry.CURRENT_USER, entry.key); err != nil && !errors.Is(err, registry.ErrNotExist) {
				return err
			}
			continue
		}
		command, err := contextMenuCommand(home, entry.arg)
		if err != nil {
			return err
		}
		exe, _ := os.Executable()
		key, _, err := registry.CreateKey(registry.CURRENT_USER, entry.key, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("create %s: %w", entry.key, err)
		}
		if err := key.SetStringValue("", contextMenuLabel); err != nil {
			_ = key.Close()
			return err
		}
		if exe != "" {
			if err := key.SetStringValue("Icon", exe); err != nil {
				_ = key.Close()
				return err
			}
		}
		_ = key.Close()

		commandKey, _, err := registry.CreateKey(registry.CURRENT_USER, entry.key+`\command`, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("create %s\\command: %w", entry.key, err)
		}
		err = commandKey.SetStringValue("", command)
		_ = commandKey.Close()
		if err != nil {
			return err
		}
	}
	return nil
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
