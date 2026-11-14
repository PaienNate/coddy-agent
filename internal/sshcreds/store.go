// Package sshcreds keeps the SSH credentials an operator configured through
// the web UI. They live in one JSON file under CODDY_HOME, readable only by
// its owner, and never travel through the model: the ssh_run_command tool
// receives a host, not a secret, and the password a turn can never see.
//
// A host may carry a password, a credential-helper command whose trimmed
// stdout is the password (the same idea as providers[].api_key_command), a
// private key path, or none of them - then the tool falls back to the SSH
// agent and the keys under ~/.ssh as it always did.
package sshcreds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/EvilFreelancer/coddy-agent/internal/platform"
)

// FileName is the file a store keeps its entries in, under CODDY_HOME.
const FileName = "ssh-credentials.json"

// passwordCommandTimeout bounds a credential-helper command, the same idea as
// a provider's api_key_command: a helper that hangs must not hang a turn.
const passwordCommandTimeout = 15 * time.Second

// Entry is one configured host. Host, User and Port identify it; the rest are
// alternative ways to authenticate.
type Entry struct {
	Host string `json:"host"`
	User string `json:"user"`
	// Port is the SSH port; zero means 22.
	Port int `json:"port,omitempty"`
	// Password is the fallback credential, stored in the same file as the
	// rest: the file is 0600 and under the operator's home, the protection a
	// private key under ~/.ssh has.
	Password string `json:"password,omitempty"`
	// PasswordCommand is a credential helper run through the host shell when
	// Password is empty; its trimmed stdout is the password.
	PasswordCommand string `json:"password_command,omitempty"`
	// KeyPath is a private key chosen for this host, ahead of ~/.ssh.
	KeyPath string `json:"key_path,omitempty"`
}

// EffectivePort returns the port the entry connects to.
func (e Entry) EffectivePort() int {
	if e.Port > 0 {
		return e.Port
	}
	return 22
}

func (e Entry) key() string {
	return fmt.Sprintf("%s@%s:%d", strings.ToLower(strings.TrimSpace(e.User)),
		strings.ToLower(strings.TrimSpace(e.Host)), e.EffectivePort())
}

// Normalize trims the identifying fields and the paths.
func (e Entry) Normalize() Entry {
	e.Host = strings.TrimSpace(e.Host)
	e.User = strings.TrimSpace(e.User)
	e.KeyPath = strings.TrimSpace(e.KeyPath)
	e.PasswordCommand = strings.TrimSpace(e.PasswordCommand)
	if e.Port < 0 {
		e.Port = 0
	}
	return e
}

// Validate reports whether an entry can be saved: it needs a host, and a host
// without a user cannot authenticate.
func (e Entry) Validate() error {
	e = e.Normalize()
	if e.Host == "" {
		return errors.New("host is required")
	}
	if e.User == "" {
		return errors.New("user is required")
	}
	if e.Port < 0 || e.Port > 65535 {
		return fmt.Errorf("port %d is out of range", e.Port)
	}
	if strings.ContainsAny(e.Host, "/\\ \t") {
		// A host is a DNS name or an address; a path or a space here is a
		// mistake the operator should see now, not at connect time.
		return fmt.Errorf("host %q contains a path separator or whitespace", e.Host)
	}
	return nil
}

// Store is the on-disk set of entries. Every read and write takes the lock, so
// two surfaces editing the list at once cannot lose an entry.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore returns a store over one file. An empty path makes a store that
// keeps nothing, which is what a process with no home resolves to.
func NewStore(path string) *Store {
	return &Store{path: strings.TrimSpace(path)}
}

// StoreForHome returns the store of one CODDY_HOME. An empty home yields an
// in-memory-only store, so a caller with no home degrades to "no credentials"
// rather than writing to the working directory.
func StoreForHome(home string) *Store {
	home = strings.TrimSpace(home)
	if home == "" {
		return NewStore("")
	}
	return NewStore(filepath.Join(home, FileName))
}

// List returns the configured entries, oldest first. Passwords are included:
// the HTTP layer is what strips them before anything leaves the process.
func (s *Store) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return append([]Entry(nil), entries...), nil
}

// Get returns the entry naming host/user/port.
func (s *Store) Get(host, user string, port int) (Entry, bool, error) {
	want := Entry{Host: host, User: user, Port: port}.Normalize().key()
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.loadLocked()
	if err != nil {
		return Entry{}, false, err
	}
	for _, e := range entries {
		if e.Normalize().key() == want {
			return e, true, nil
		}
	}
	return Entry{}, false, nil
}

// Upsert stores an entry, replacing the one with the same host, user and port.
// An empty Password keeps the password already stored for that entry, so the
// settings form can round-trip a host without ever holding the secret; a
// non-empty PasswordCommand or KeyPath replaces the stored one, and an empty
// one clears it.
func (s *Store) Upsert(entry Entry) error {
	entry = entry.Normalize()
	if err := entry.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.loadLocked()
	if err != nil {
		return err
	}
	replaced := false
	for i := range entries {
		if entries[i].Normalize().key() == entry.key() {
			if entry.Password == "" {
				entry.Password = entries[i].Password
			}
			entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, entry)
	}
	return s.saveLocked(entries)
}

// Delete removes the entry naming host/user/port and reports whether one was
// there.
func (s *Store) Delete(host, user string, port int) (bool, error) {
	want := Entry{Host: host, User: user, Port: port}.Normalize().key()
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.loadLocked()
	if err != nil {
		return false, err
	}
	kept := entries[:0]
	found := false
	for _, e := range entries {
		if e.Normalize().key() == want {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		return false, nil
	}
	return true, s.saveLocked(kept)
}

// ResolvePassword returns the password for an entry: the credential helper's
// trimmed stdout when one is configured, else the stored password. An empty
// result with a nil error means the entry has no password to offer.
func (e Entry) ResolvePassword(ctx context.Context) (string, error) {
	if cmd := strings.TrimSpace(e.PasswordCommand); cmd != "" {
		return runPasswordCommand(ctx, cmd)
	}
	return e.Password, nil
}

func runPasswordCommand(parent context.Context, command string) (string, error) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, passwordCommandTimeout)
	defer cancel()
	shell := platform.CurrentShell()
	executable, args := shell.Command(command)
	cmd := exec.CommandContext(ctx, executable, args...)
	platform.AdaptCommand(cmd)
	out, err := cmd.Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("password_command: %w", ctxErr)
		}
		return "", fmt.Errorf("password_command failed: %w", err)
	}
	return strings.TrimSpace(platform.DecodeOutput(out)), nil
}

func (s *Store) loadLocked() ([]Entry, error) {
	if s.path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var doc struct {
		Hosts []Entry `json:"hosts"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", s.path, err)
	}
	return doc.Hosts, nil
}

// saveLocked writes the file through a temporary one and a rename, so a crash
// mid-write cannot leave half a list, and creates it 0600: the file holds
// passwords.
func (s *Store) saveLocked(entries []Entry) error {
	if s.path == "" {
		return errors.New("no CODDY_HOME: SSH credentials cannot be stored")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	doc := struct {
		Hosts []Entry `json:"hosts"`
	}{Hosts: entries}
	if doc.Hosts == nil {
		doc.Hosts = []Entry{}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".ssh-credentials-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}
