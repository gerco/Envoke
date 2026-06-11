//go:build darwin

package keychain

import "github.com/99designs/keyring"

func allowedBackends() []keyring.BackendType {
	return []keyring.BackendType{keyring.KeychainBackend}
}

func serviceName() string { return "envoke" }

// keyPrefix returns the prefix to add to keys on this platform.
// On macOS, the ServiceName is already part of the key, so no additional prefix needed.
func keyPrefix() string { return "" }
