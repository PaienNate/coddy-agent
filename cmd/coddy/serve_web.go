package main

import (
	"net"
	"strings"

	"github.com/EvilFreelancer/coddy-agent/internal/serve"
)

// serveWebURL is the address the tray and its Open page row hand to a browser.
//
// The listener may bind 0.0.0.0 (or ::) because the server is meant to be
// reached from another machine, but a tray on this machine must open a
// loopback address. A named host is kept as the operator wrote it: it is the
// address they chose, and a machine with several interfaces may mean one of
// them.
func serveWebURL(httpAddr string, enabled []serve.Subsystem) (string, bool) {
	hasHTTP := false
	for _, sub := range enabled {
		if sub.Kind == serve.KindHTTP {
			hasHTTP = true
			break
		}
	}
	if !hasHTTP {
		return "", false
	}
	host, port, err := net.SplitHostPort(strings.TrimSpace(httpAddr))
	if err != nil || strings.TrimSpace(port) == "" {
		return "", false
	}
	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/", true
}
