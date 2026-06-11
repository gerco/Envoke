//go:build windows

package keychain

import "github.com/99designs/keyring"

func allowedBackends() []keyring.BackendType {
	return []keyring.BackendType{keyring.WinCredBackend}
}

func serviceName() string { return "envoke" }
