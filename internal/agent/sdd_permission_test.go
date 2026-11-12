package agent

import (
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/llm"
	"github.com/EvilFreelancer/coddy-agent/internal/tooling"
	"github.com/EvilFreelancer/coddy-agent/internal/tools"
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

// The decisions safety check must see the command an sdd done call runs: it is
// the same shell path as run_command, so it must not be a way around the check.
func TestSDDDoneIsJudgedLikeARunCommand(t *testing.T) {
	env := &tooling.Env{CWD: "/repo"}
	subject, ok := commandSafetySubject(tools.SDDToolName,
		`{"action":"done","plan":"p","task":1,"base":"a","command":"rm -rf /"}`, env)
	if !ok || subject.Command != "rm -rf /" {
		t.Fatalf("subject = %+v, ok = %v", subject, ok)
	}
	if _, ok := commandSafetySubject(tools.SDDToolName, `{"action":"start","plan":"p","task":1}`, env); ok {
		t.Fatal("an sdd start call was judged as a command")
	}
}

func TestSDDAllowlistCoversTheDecisionsCheck(t *testing.T) {
	env := &tooling.Env{CWD: "/repo", CommandAllowlist: []string{"go"}}
	tc := llm.ToolCall{Name: tools.SDDToolName, InputJSON: `{"action":"done","plan":"p","task":1,"base":"a","command":"go test ./..."}`}
	if !commandAllowlisted(tc, env, nil) {
		t.Fatal("an allowlisted command was sent to the decisions check")
	}
}
