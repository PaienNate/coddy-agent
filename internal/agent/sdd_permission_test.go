package agent

import (
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/tooling"
)

func TestSDDPermissionOnlyForDone(t *testing.T) {
	env := &tooling.Env{CWD: "/repo"}
	start := `{"action":"start","plan":"docs/plans/x.md","task":1}`
	if sddNeedsPermission(config.PermModeAsk, env, nil, start) {
		t.Fatal("sdd start asked for approval")
	}
	for _, action := range []string{"workspace", "brief", "review"} {
		if sddNeedsPermission(config.PermModeAsk, env, nil, `{"action":"`+action+`","plan":"p","base":"a","head":"b"}`) {
			t.Fatalf("sdd %s asked for approval", action)
		}
	}
	done := `{"action":"done","plan":"p","task":1,"base":"a","command":"go test ./..."}`
	if !sddNeedsPermission(config.PermModeAsk, env, nil, done) {
		t.Fatal("sdd done did not ask for approval")
	}
}

func TestSDDPermissionHonoursGrants(t *testing.T) {
	env := &tooling.Env{CWD: "/repo"}
	done := `{"action":"done","plan":"p","task":1,"base":"a","command":"go test ./..."}`

	if sddNeedsPermission(config.PermModeAsk, env, []string{"go test ./..."}, done) {
		t.Fatal("an allowed command still asked")
	}
	if !sddNeedsPermission(config.PermModeAsk, env, []string{"go test ./internal/agent"}, done) {
		t.Fatal("a different grant approved this command")
	}
	allowlisted := &tooling.Env{CWD: "/repo", CommandAllowlist: []string{"go"}}
	if sddNeedsPermission(config.PermModeAsk, allowlisted, nil, done) {
		t.Fatal("tools.command_allowlist did not approve the command")
	}
}

func TestSDDPermissionBypassNeverAsks(t *testing.T) {
	env := &tooling.Env{CWD: "/repo"}
	done := `{"action":"done","plan":"p","task":1,"base":"a","command":"go test ./..."}`
	if sddNeedsPermission(config.PermModeBypass, env, nil, done) {
		t.Fatal("bypass mode asked")
	}
}

func TestSDDPermissionIgnoresAMalformedCall(t *testing.T) {
	env := &tooling.Env{CWD: "/repo"}
	if sddNeedsPermission(config.PermModeAsk, env, nil, `{`) {
		t.Fatal("a call the tool will reject asked for approval")
	}
}
