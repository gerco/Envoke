//go:build darwin

package keychain

import "github.com/99designs/keyring"

func allowedBackends() []keyring.BackendType {
	return []keyring.BackendType{keyring.KeychainBackend}
}
