package sshcreds

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	home := t.TempDir()
	store := StoreForHome(home)

	if err := store.Upsert(Entry{Host: "nas.local", User: "admin", Password: "s3cret"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(Entry{Host: "web", User: "root", Port: 2222, KeyPath: "~/.ssh/id_ed25519"}); err != nil {
		t.Fatal(err)
	}

	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}

	got, ok, err := store.Get("NAS.LOCAL", "admin", 0)
	if err != nil || !ok {
		t.Fatalf("Get = %v, %v", ok, err)
	}
	if got.Password != "s3cret" {
		t.Errorf("password = %q", got.Password)
	}
	if got.EffectivePort() != 22 {
		t.Errorf("port = %d", got.EffectivePort())
	}

	// The port is part of the identity: the same host on another port is a
	// different entry.
	if _, ok, _ := store.Get("web", "root", 22); ok {
		t.Error("a port mismatch matched an entry")
	}
	if _, ok, _ := store.Get("web", "root", 2222); !ok {
		t.Error("the entry on its own port did not match")
	}
}

func TestStoreFileIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows reports permissions from ACLs, not from os.Chmod's bits;
		// the 0600 assertion belongs to the platforms that have those bits.
		t.Skip("POSIX file modes do not exist on Windows")
	}
	home := t.TempDir()
	store := StoreForHome(home)
	if err := store.Upsert(Entry{Host: "nas", User: "admin", Password: "s3cret"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(home, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("file mode = %o, want 0600", mode)
	}
	data, err := os.ReadFile(filepath.Join(home, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"password": "s3cret"`) {
		t.Errorf("the stored document lost the password: %s", data)
	}
}

func TestUpsertKeepsTheStoredPassword(t *testing.T) {
	store := StoreForHome(t.TempDir())
	if err := store.Upsert(Entry{Host: "nas", User: "admin", Password: "s3cret"}); err != nil {
		t.Fatal(err)
	}
	// The settings form round-trips a host it never saw the password of.
	if err := store.Upsert(Entry{Host: "nas", User: "admin", PasswordCommand: "pass show nas"}); err != nil {
		t.Fatal(err)
	}
	got, _, err := store.Get("nas", "admin", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "s3cret" {
		t.Errorf("password = %q, want the stored one", got.Password)
	}
	if got.PasswordCommand != "pass show nas" {
		t.Errorf("password_command = %q", got.PasswordCommand)
	}
	// An explicit non-empty password replaces it.
	if err := store.Upsert(Entry{Host: "nas", User: "admin", Password: "new"}); err != nil {
		t.Fatal(err)
	}
	got, _, _ = store.Get("nas", "admin", 0)
	if got.Password != "new" {
		t.Errorf("password = %q, want the new one", got.Password)
	}
}

func TestDelete(t *testing.T) {
	store := StoreForHome(t.TempDir())
	if err := store.Upsert(Entry{Host: "nas", User: "admin"}); err != nil {
		t.Fatal(err)
	}
	removed, err := store.Delete("nas", "admin", 0)
	if err != nil || !removed {
		t.Fatalf("Delete = %v, %v", removed, err)
	}
	if removed, _ := store.Delete("nas", "admin", 0); removed {
		t.Error("a second delete claimed to remove something")
	}
	entries, _ := store.List()
	if len(entries) != 0 {
		t.Errorf("entries = %d after delete", len(entries))
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		in   Entry
		ok   bool
	}{
		{"plain", Entry{Host: "nas", User: "admin"}, true},
		{"no host", Entry{User: "admin"}, false},
		{"no user", Entry{Host: "nas"}, false},
		{"path in host", Entry{Host: "../nas", User: "admin"}, false},
		{"port range", Entry{Host: "nas", User: "admin", Port: 70000}, false},
	}
	for _, c := range cases {
		if err := c.in.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: Validate = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestResolvePasswordCommand(t *testing.T) {
	entry := Entry{Host: "nas", User: "admin", PasswordCommand: "printf '  helper-secret  \\n'"}
	got, err := entry.ResolvePassword(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got != "helper-secret" {
		t.Errorf("password = %q", got)
	}
}

func TestStoreWithoutHomeKeepsNothing(t *testing.T) {
	store := StoreForHome("")
	if err := store.Upsert(Entry{Host: "nas", User: "admin"}); err == nil {
		t.Error("a store with no home accepted a write")
	}
	entries, err := store.List()
	if err != nil || len(entries) != 0 {
		t.Errorf("List = %v, %v", entries, err)
	}
}
