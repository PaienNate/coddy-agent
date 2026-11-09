//go:build http

package httpserver

import (
	"os"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/config"
)

func TestOpenCodeJobSnapshotSurvivesARestart(t *testing.T) {
	home := t.TempDir()
	cfg := &config.Config{Paths: config.Paths{Home: home}}
	s := &Server{opencodeJobs: make(map[string]*opencodeImportJob)}
	s.cfgAt.Store(cfg)
	job := &opencodeImportJob{
		ID:      "opencode-test",
		Started: "2026-11-09T00:00:00Z",
		Done:    true,
		DoneN:   3,
		TotalN:  3,
		home:    home,
	}
	s.persistOpenCodeJob(job)
	path := opencodeJobPath(home, job.ID)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("job snapshot: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("snapshot permissions = %v, %v", info, err)
	}

	// A fresh Server has an empty memory map; the status load must find the
	// snapshot on disk and hand it to the caller.
	fresh := &Server{opencodeJobs: make(map[string]*opencodeImportJob)}
	fresh.cfgAt.Store(cfg)
	loaded := fresh.loadOpenCodeJob(job.ID)
	if loaded == nil || !loaded.Done || loaded.DoneN != 3 || loaded.home != home {
		t.Fatalf("loaded = %+v", loaded)
	}
}
