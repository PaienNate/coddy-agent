package main

import (
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/serve"
)

func TestServeWebURLUsesLoopbackForWildcardListeners(t *testing.T) {
	http := []serve.Subsystem{{Kind: serve.KindHTTP}}
	for _, tc := range []struct {
		addr string
		want string
	}{
		{"127.0.0.1:12345", "http://127.0.0.1:12345/"},
		{"0.0.0.0:12345", "http://127.0.0.1:12345/"},
		{":12345", "http://127.0.0.1:12345/"},
		{"[::]:12345", "http://[::1]:12345/"},
		{"192.0.2.10:12345", "http://192.0.2.10:12345/"},
	} {
		got, ok := serveWebURL(tc.addr, http)
		if !ok || got != tc.want {
			t.Errorf("serveWebURL(%q) = %q, %v; want %q, true", tc.addr, got, ok, tc.want)
		}
	}
}

func TestServeWebURLNeedsAnHTTPSubsystem(t *testing.T) {
	if got, ok := serveWebURL("127.0.0.1:12345", []serve.Subsystem{{Kind: serve.KindScheduler}}); ok || got != "" {
		t.Fatalf("serveWebURL without HTTP = %q, %v", got, ok)
	}
	if got, ok := serveWebURL("not-an-address", []serve.Subsystem{{Kind: serve.KindHTTP}}); ok || got != "" {
		t.Fatalf("serveWebURL with a bad address = %q, %v", got, ok)
	}
}
