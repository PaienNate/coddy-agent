package main

import (
	"log/slog"
	"sort"
	"strings"

	"github.com/EvilFreelancer/coddy-agent/internal/serve"
)

// recentWorkspaces are the folders this home's chats were last had in, most
// recent first. The desktop application menu and the Windows serve tray both
// offer them as rows that open a new chat in that folder.
//
// They are read here rather than in either launcher because the session store
// is the agent's, and a launcher knows nothing of it. A store that cannot be
// read is a menu with no folders in it, which is what both menus looked like
// before there were any.
func recentWorkspaces(rt *serve.Runtime, log *slog.Logger) []string {
	if rt == nil || rt.Mgr == nil {
		return nil
	}
	store := rt.Mgr.FileStore()
	if store == nil {
		return nil
	}
	rows, err := store.ListSnapshots("", false)
	if err != nil {
		if log != nil {
			log.Debug("the recent workspaces could not be read", "error", err)
		}
		return nil
	}
	// Newest first, by the time the session was last written.
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.CWD) != "" {
			out = append(out, row.CWD)
		}
	}
	return out
}
