package main

import "runtime/debug"

// version is set at release time via -ldflags "-X main.version=...".
// When empty, it falls back to the module version recorded by the Go
// toolchain, which is populated for "go install module@version" builds.
var version string

func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

func init() {
	rootCmd.Version = resolveVersion()
}
