package ssh

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/sshcreds"
	gossh "golang.org/x/crypto/ssh"
)

// testDialTimeout bounds the settings page's "Test connection": a host that
// does not answer must answer the operator, not leave the button spinning.
const testDialTimeout = 15 * time.Second

// TestConnection dials a configured host with the same authentication chain
// ssh_run_command uses and closes the client without opening a session. It is
// what the settings page's Test button reports on: a yes or no, and the
// reason, without running anything on the remote machine.
//
// permissionMode follows the tool: "bypass" skips host-key verification, and
// every other mode accepts a new key on first sight (TOFU) and verifies it
// afterwards.
func TestConnection(ctx context.Context, entry sshcreds.Entry, permissionMode string) error {
	entry = entry.Normalize()
	if err := entry.Validate(); err != nil {
		return err
	}
	sshDir := resolveSSHDir()
	dirKnownHosts := ""
	if sshDir != "" {
		dirKnownHosts = filepath.Join(sshDir, "known_hosts")
	}
	hkCB, err := hostKeyCallback(knownHostsModeFrom(permissionMode), dirKnownHosts)
	if err != nil {
		return err
	}
	methods, agentCloser, passwordErr := buildAuthMethods(ctx, sshDir, entry, true)
	if agentCloser != nil {
		defer func() { _ = agentCloser.Close() }()
	}
	if len(methods) == 0 {
		if passwordErr != nil {
			return fmt.Errorf("no authentication methods available: password_command failed: %w", passwordErr)
		}
		return fmt.Errorf("no authentication methods available: no key and no password configured for %s@%s:%d",
			entry.User, entry.Host, entry.EffectivePort())
	}

	addr := fmt.Sprintf("%s:%d", entry.Host, entry.EffectivePort())
	cfg := &gossh.ClientConfig{
		User:            entry.User,
		Auth:            methods,
		HostKeyCallback: hkCB,
		Timeout:         testDialTimeout,
	}
	dialCtx, cancel := context.WithTimeout(ctx, testDialTimeout)
	defer cancel()
	client, err := dialContext(dialCtx, addr, cfg)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}
	return client.Close()
}
