package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/sshcreds"
	"github.com/EvilFreelancer/coddy-agent/internal/tooling"
	gossh "golang.org/x/crypto/ssh"
)

// startTestSSHServer serves one password-protected account that answers every
// exec request with a fixed line and exit status 0. It is a real SSH server
// over a loopback socket, so the test exercises the same handshake, auth chain
// and session the tool does in production.
func startTestSSHServer(t *testing.T, user, password string) (addr string, port int) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostKey, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &gossh.ServerConfig{
		PasswordCallback: func(meta gossh.ConnMetadata, pw []byte) (*gossh.Permissions, error) {
			if meta.User() == user && string(pw) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		},
	}
	cfg.AddHostKey(hostKey)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serveTestConn(conn, cfg)
		}
	}()

	tcpAddr := listener.Addr().(*net.TCPAddr)
	return listener.Addr().String(), tcpAddr.Port
}

func serveTestConn(conn net.Conn, cfg *gossh.ServerConfig) {
	sc, chans, reqs, err := gossh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	defer func() { _ = sc.Close() }()
	go gossh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(gossh.UnknownChannelType, "only sessions")
			continue
		}
		ch, requests, err := newCh.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer func() { _ = ch.Close() }()
			for req := range requests {
				switch req.Type {
				case "exec":
					_, _ = ch.Write([]byte("test-ok"))
					_, _ = ch.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{0}))
					if req.WantReply {
						_ = req.Reply(true, nil)
					}
					return
				default:
					if req.WantReply {
						_ = req.Reply(false, nil)
					}
				}
			}
		}()
	}
}

// testEnv returns a tool environment whose home is the store's, with the agent
// disabled (no SSH_AUTH_SOCK) and host keys unverified, so the test never
// writes to the real ~/.ssh.
func testEnv(t *testing.T, home string) *tooling.Env {
	t.Helper()
	orig, had := os.LookupEnv("SSH_AUTH_SOCK")
	if err := os.Unsetenv("SSH_AUTH_SOCK"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("SSH_AUTH_SOCK", orig)
		}
	})
	// An empty home keeps discovery out of the machine's real ~/.ssh.
	t.Setenv("HOME", t.TempDir())
	return &tooling.Env{
		ConfigHome:        home,
		PermissionMode:    "bypass",
		SSHConnectTimeout: 10,
	}
}

func TestSSHRunCommandWithStoredPassword(t *testing.T) {
	_, port := startTestSSHServer(t, "admin", "s3cret")
	home := t.TempDir()
	if err := sshcreds.StoreForHome(home).Upsert(sshcreds.Entry{
		Host: "127.0.0.1", User: "admin", Port: port, Password: "s3cret",
	}); err != nil {
		t.Fatal(err)
	}

	args, err := json.Marshal(map[string]interface{}{
		"host": "admin@127.0.0.1", "port": port, "command": "echo hi", "timeout_seconds": 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := SSHRunCommandTool().Execute(t.Context(), string(args), testEnv(t, home))
	if err != nil {
		t.Fatal(err)
	}
	if out != "test-ok" {
		t.Fatalf("output = %q, want the server's line", out)
	}
}

func TestSSHRunCommandWrongPassword(t *testing.T) {
	_, port := startTestSSHServer(t, "admin", "s3cret")
	home := t.TempDir()
	if err := sshcreds.StoreForHome(home).Upsert(sshcreds.Entry{
		Host: "127.0.0.1", User: "admin", Port: port, Password: "wrong",
	}); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]interface{}{
		"host": "admin@127.0.0.1", "port": port, "command": "echo hi", "timeout_seconds": 10,
	})
	if _, err := SSHRunCommandTool().Execute(t.Context(), string(args), testEnv(t, home)); err == nil {
		t.Fatal("a wrong password was accepted")
	}
}

func TestSSHRunCommandWithoutCredentialsSaysWhere(t *testing.T) {
	home := t.TempDir()
	args, _ := json.Marshal(map[string]interface{}{
		"host": "nobody@127.0.0.1", "port": 1, "command": "echo hi",
	})
	_, err := SSHRunCommandTool().Execute(t.Context(), string(args), testEnv(t, home))
	if err == nil {
		t.Fatal("want an error with no credentials")
	}
	if got := err.Error(); !strings.Contains(got, "settings page") {
		t.Errorf("error does not say where credentials go: %v", err)
	}
}

// prepareIsolatedHome points home and the agent at nothing the machine has, so
// key discovery and the agent cannot make a test's outcome host-dependent.
func prepareIsolatedHome(t *testing.T) {
	t.Helper()
	_ = os.Unsetenv("SSH_AUTH_SOCK")
	t.Setenv("HOME", t.TempDir())
}

func TestTestConnection(t *testing.T) {
	_, port := startTestSSHServer(t, "admin", "s3cret")
	prepareIsolatedHome(t)

	entry := sshcreds.Entry{Host: "127.0.0.1", User: "admin", Port: port, Password: "s3cret"}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if err := TestConnection(ctx, entry, "bypass"); err != nil {
		t.Fatalf("TestConnection = %v", err)
	}
	entry.Password = "wrong"
	entry.PasswordCommand = ""
	if err := TestConnection(ctx, entry, "bypass"); err == nil {
		t.Fatal("TestConnection accepted a wrong password")
	}
}

func TestTestConnectionReportsNoCredentials(t *testing.T) {
	prepareIsolatedHome(t)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err := TestConnection(ctx, sshcreds.Entry{Host: "127.0.0.1", User: "admin", Port: 1}, "bypass")
	if err == nil {
		t.Fatal("want an error with no credentials")
	}
}
