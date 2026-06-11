//go:build linux

package keychain

import "github.com/99designs/keyring"

func allowedBackends() []keyring.BackendType {
	return []keyring.BackendType{keyring.SecretServiceBackend, keyring.KWalletBackend}
}

func serviceName() string { return "login" }

// keyPrefix returns the prefix to add to keys on this platform.
// On Linux, we use the "login" collection but prefix keys with "envoke/" for consistency.
func keyPrefix() string { return "envoke/" }
