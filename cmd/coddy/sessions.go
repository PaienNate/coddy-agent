package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/gitws"
	"github.com/EvilFreelancer/coddy-agent/internal/opencodeimport"
	"github.com/EvilFreelancer/coddy-agent/internal/session"
)

// sessionsExportUsage is the usage line of `coddy sessions export`.
func sessionsExportUsage() string {
	return fmt.Sprintf("usage: %s sessions export <session-id> [--format md|html|json|jsonl] [--out <path>] [--no-tools] [--no-thinking] [--sessions-dir <path>]", os.Args[0])
}

// parseAroundPositional parses flags that may sit before or after the single
// positional argument (`export <id> --format json` and `export --format json
// <id>` both work) and returns that argument.
func parseAroundPositional(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return "", nil
	}
	positional := rest[0]
	if len(rest) > 1 {
		if err := fs.Parse(rest[1:]); err != nil {
			return "", err
		}
		if fs.NArg() > 0 {
			return "", fmt.Errorf("unexpected argument %q", fs.Arg(0))
		}
	}
	return positional, nil
}

// sessionsExport implements `coddy sessions export`, the shell twin of the
// built-in /export command. It reads a stored bundle (an exact id or a unique
// prefix), builds the same export document, and writes it under cwd or
// wherever --out points. A nil store is opened from --sessions-dir or the
// configuration.
func sessionsExport(w io.Writer, store *session.FileStore, cfg *config.Config, cwd string, args []string) error {
	fs := flag.NewFlagSet("sessions export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	format := fs.String("format", "", "export format: md, html, json, jsonl (default: from the --out extension, else md)")
	out := fs.String("out", "", "output file or directory (default: the current directory)")
	noTools := fs.Bool("no-tools", false, "leave out tool calls and their results")
	noThinking := fs.Bool("no-thinking", false, "leave out the model's reasoning")
	rootFlag := fs.String("sessions-dir", "", "sessions root (empty uses config sessions.dir or ~/.coddy/sessions)")

	id, err := parseAroundPositional(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(w, sessionsExportUsage())
			fs.SetOutput(w)
			fs.PrintDefaults()
			return nil
		}
		return fmt.Errorf("%w\n%s", err, sessionsExportUsage())
	}
	if strings.TrimSpace(id) == "" {
		return errors.New(sessionsExportUsage())
	}
	if store == nil {
		store, err = openSessionStore(*rootFlag, cfg)
		if err != nil {
			return err
		}
	}
	resolved, err := store.ResolveSessionID(id)
	if err != nil {
		return err
	}
	snap, err := store.ReadSnapshot(resolved)
	if err != nil {
		return err
	}
	root, target, err := session.PrepareExportOutput(cwd, *out)
	if err != nil {
		return err
	}

	// A detached State reuses the title and model rules of a live session.
	st := &session.State{
		ID:              resolved,
		CWD:             snap.Meta.CWD,
		Mode:            session.Mode(snap.Meta.Mode),
		SelectedModelID: snap.Meta.SelectedModelID,
		Messages:        snap.Messages,
	}
	st.SetTitlePinnedWithoutPersist(snap.Meta.TitlePinned)
	title := strings.TrimSpace(snap.Meta.Title)
	if title == "" {
		title = st.ConversationTitle()
	}
	in := session.ExportInput{
		SessionID:  resolved,
		Title:      title,
		CWD:        snap.Meta.CWD,
		Model:      st.EffectiveModelID(cfg),
		Messages:   snap.Messages,
		ExportedAt: time.Now().UTC(),
		OutputRoot: root,
		SessionDir: snap.Dir,
	}
	if ws := strings.TrimSpace(snap.Meta.CWD); ws != "" {
		if fi, statErr := os.Stat(ws); statErr == nil && fi.IsDir() {
			in.GitBranch = gitws.Describe(ws).Branch
		}
	}
	if stats, statsErr := session.ReadSessionStats(snap.Dir); statsErr == nil {
		in.Stats = stats
	}
	res, err := session.ExportSession(in, session.ExportRequest{
		Format:  *format,
		Target:  target,
		Options: session.ExportOptions{NoTools: *noTools, NoThinking: *noThinking},
	})
	if err != nil {
		return err
	}
	path := res.Target.Path
	if abs, absErr := filepath.Abs(path); absErr == nil {
		path = abs
	}
	_, _ = fmt.Fprintf(w, "Session exported to %s: %s (%d transcript entries)\n", res.Format.DisplayName(), path, res.Entries)
	return nil
}

// sessionsImportUsage is the usage line of `coddy sessions import`.
func sessionsImportUsage() string {
	return fmt.Sprintf("usage: %s sessions import --from opencode [--url <url>] [--username <name>] [--password <password> | --password-stdin] [--all-projects] [--project <id>] [--skip-archived] [--dry-run] [--concurrency N] [--replace] [--save] [--sessions-dir <path>]", os.Args[0])
}

// sessionsImport implements `coddy sessions import`, the one-shot OpenCode v2
// migration. It reads the OpenCode API and writes Coddy bundles; it never opens
// the OpenCode database.
func sessionsImport(w io.Writer, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("sessions import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	from := fs.String("from", "opencode", "source agent (only opencode is supported)")
	rawURL := fs.String("url", "", "OpenCode v2 API base URL, e.g. https://opencode.example.com:4096")
	username := fs.String("username", "", "OpenCode HTTP basic-auth username (default opencode)")
	password := fs.String("password", "", "OpenCode HTTP basic-auth password")
	passwordStdin := fs.Bool("password-stdin", false, "read the OpenCode password from stdin")
	allProjects := fs.Bool("all-projects", true, "import every project the server lists")
	projectID := fs.String("project", "", "import only this OpenCode project id")
	skipArchived := fs.Bool("skip-archived", false, "skip sessions and projects marked archived")
	dryRun := fs.Bool("dry-run", false, "list what would be imported without writing anything")
	concurrency := fs.Int("concurrency", 4, "parallel session exports")
	replace := fs.Bool("replace", false, "replace an existing session with the same id")
	save := fs.Bool("save", false, "save the URL and credentials under CODDY_HOME/opencode.json")
	rootFlag := fs.String("sessions-dir", "", "sessions root (empty uses config sessions.dir or ~/.coddy/sessions)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(w, sessionsImportUsage())
			return nil
		}
		return fmt.Errorf("%w\n%s", err, sessionsImportUsage())
	}
	if strings.TrimSpace(*from) != "opencode" {
		return fmt.Errorf("unsupported import source %q (only opencode)", *from)
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q\n%s", fs.Arg(0), sessionsImportUsage())
	}
	if *allProjects && strings.TrimSpace(*projectID) != "" {
		return errors.New("--all-projects and --project cannot be used together")
	}

	saved, err := opencodeimport.LoadConfig(cfg.Paths.Home)
	if err != nil {
		return err
	}
	conn := saved
	if strings.TrimSpace(*rawURL) != "" {
		conn.URL = opencodeimport.NormalizeBaseURL(*rawURL)
	}
	if strings.TrimSpace(*username) != "" {
		conn.Username = strings.TrimSpace(*username)
	}
	if *passwordStdin {
		pw, err := opencodeimport.ReadPassword(os.Stdin)
		if err != nil {
			return err
		}
		conn.Password = pw
	} else if *password != "" {
		conn.Password = *password
	}
	if conn.Username == "" {
		conn.Username = "opencode"
	}
	if strings.TrimSpace(conn.URL) == "" {
		return errors.New("OpenCode URL is required: pass --url or configure it in Settings -> OpenCode import")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := opencodeimport.NewClient(conn)
	version, err := client.Health(ctx)
	if err != nil {
		return fmt.Errorf("connect to OpenCode: %w", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(version), "2.") {
		return fmt.Errorf("unsupported OpenCode version %q; this importer requires OpenCode v2", version)
	}

	projects, err := client.Projects(ctx)
	if err != nil {
		return fmt.Errorf("list OpenCode projects: %w", err)
	}
	if strings.TrimSpace(*projectID) != "" {
		var one []opencodeimport.Project
		for _, p := range projects {
			if p.ID == strings.TrimSpace(*projectID) {
				one = append(one, p)
				break
			}
		}
		if len(one) == 0 {
			return fmt.Errorf("OpenCode project %q not found", *projectID)
		}
		projects = one
	}
	if !*allProjects && strings.TrimSpace(*projectID) == "" {
		return errors.New("--all-projects is required unless --project names one project")
	}
	if *skipArchived {
		active := projects[:0]
		for _, p := range projects {
			if p.ArchivedAt() == nil {
				active = append(active, p)
			}
		}
		projects = active
	}
	if len(projects) == 0 {
		return errors.New("OpenCode returned no projects")
	}

	root := strings.TrimSpace(*rootFlag)
	if root == "" {
		root = cfg.ResolvedSessionsRoot()
	}
	cwd, _ := os.Getwd()
	opts := opencodeimport.ImportOptions{
		Projects:        projects,
		IncludeArchived: !*skipArchived,
		Concurrency:     *concurrency,
		DryRun:          *dryRun,
		Replace:         *replace,
		SessionsRoot:    root,
		FallbackCWD:     cwd,
	}
	if !*dryRun {
		opts.Progress = func(done, total int, id string) {
			if done == total || done%10 == 0 {
				_, _ = fmt.Fprintf(os.Stderr, "imported %d/%d sessions (%s)\n", done, total, id)
			}
		}
	}
	res, err := opencodeimport.ImportAll(ctx, client, opts)
	if err != nil {
		return err
	}
	if *save {
		if err := opencodeimport.SaveConfig(cfg.Paths.Home, conn); err != nil {
			return fmt.Errorf("save OpenCode connection: %w", err)
		}
	}
	_, _ = fmt.Fprintf(w, "OpenCode %s: %d project(s), %d session(s)\n", version, res.Projects, res.Sessions)
	if *dryRun {
		_, _ = fmt.Fprintln(w, "dry run: no Coddy session was written")
		return nil
	}
	_, _ = fmt.Fprintf(w, "imported %d, skipped %d, failed %d\n", res.Imported, res.Skipped, res.Failed)
	for _, e := range res.Errors {
		_, _ = fmt.Fprintf(w, "warning: %s\n", e)
	}
	if res.Failed > 0 {
		return fmt.Errorf("%d session(s) failed to import", res.Failed)
	}
	return nil
}
