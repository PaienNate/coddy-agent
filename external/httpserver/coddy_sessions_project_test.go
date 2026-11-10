//go:build http

package httpserver

import (
	"path/filepath"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/session"
)

func TestSessionsInProjectKeepsTheProjectAndLeavesOthers(t *testing.T) {
	project := t.TempDir()
	other := t.TempDir()
	rows := []session.SessionListEntry{
		{SessionID: "inside", CWD: project},
		{SessionID: "outside", CWD: other},
	}
	got := sessionsInProject(rows, filepath.Clean(project))
	if len(got) != 1 || got[0].SessionID != "inside" {
		t.Fatalf("sessionsInProject = %+v", got)
	}
}

func TestSessionsOfParentKeepsOnlyThatParentsChildren(t *testing.T) {
	rows := []session.SessionListEntry{
		{SessionID: "child-a", SubagentRun: true, ParentSessionID: "parent-a"},
		{SessionID: "parent-a"},
		{SessionID: "child-b", SubagentRun: true, ParentSessionID: "parent-b"},
	}
	got := sessionsOfParent(rows, "parent-a")
	if len(got) != 1 || got[0].SessionID != "child-a" {
		t.Fatalf("sessionsOfParent = %+v", got)
	}
}
