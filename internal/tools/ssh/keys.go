// Package ssh provides an SSH remote-execution tool for the agent.
package ssh

import (
	"os"
	"path/filepath"

	gossh "golang.org/x/crypto/ssh"
)

// defaultKeyNames is the ordered list of standard private key filenames tried during discovery.
var defaultKeyNames = []string{
	"id_ed25519",
	"id_rsa",
	"id_ecdsa",
	"id_dsa",
}

// LoadSigners returns all usable SSH signers found in dir by scanning defaultKeyNames.
// Non-existent dir, missing key files, and keys with passphrases are silently skipped.
// Returns nil when dir is empty.
func LoadSigners(dir string) []gossh.Signer {
	if dir == "" {
		return nil
	}
	var signers []gossh.Signer
	for _, name := range defaultKeyNames {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		signer, err := gossh.ParsePrivateKey(data)
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}
	return signers
}

// loadSignerFile parses one private key the operator named for a host. Unlike
// the discovery above it reports why a key did not load, so the settings page
// and the tool can say "the key at that path is not a private key" instead of
// silently falling back to a password.
func loadSignerFile(path string) (gossh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := gossh.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}
	return signer, nil
}
