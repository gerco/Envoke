# Agent Guidelines — Envoke

## Project

Envoke (`ee`) is a Go CLI tool that injects secrets per-command from pluggable backends. Single static binary. Cross-compiles to Windows, macOS, Linux.

## Philosophy

Unix philosophy applies throughout:

- **Do one thing well.** `ee` injects secrets into a subprocess environment. It is not a secret manager, a shell, or a config framework.
- **No output if nothing is wrong.** Commands succeed silently. Errors go to stderr with a non-zero exit code.
- **Composable.** `ee` works in pipes, scripts, and CI without special modes or flags.
- **Degrade gracefully.** TUI commands (`ee status`, `ee edit`, `ee run`) detect non-interactive environments and fall back to plain text output.

## Language and Safety

- **Go only.** No other languages, no CGO unless unavoidable (document it explicitly when used).
- **Compile-time safety preferred.** Prefer strong types, typed errors, and exhaustive switches over runtime assertions and `interface{}`. Use `iota` enums, typed constants, and structs with unexported fields where appropriate.
- **No panics in library code.** Panics are only acceptable in `main()` during startup validation.
- **Errors are values.** Wrap errors with `fmt.Errorf("context: %w", err)`. Never discard errors silently.

## Dependencies

- **Prefer the standard library.** Before adding a dependency, ask whether `encoding/toml` (Go 1.21+), `os/exec`, `crypto`, or another stdlib package covers the need. The dependency must earn its place.
- **Approved dependencies** (already in scope per DESIGN.md):
  - `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss` — TUI only
  - `99designs/keyring` — OS keychain abstraction
  - `spf13/cobra` — CLI structure
  - Keeper Secrets Manager Go SDK
  - JumpCloud Go SDK / REST client
- **Do not add** test frameworks (use `testing` + `testify` only if it already exists in `go.mod`), logging libraries, or utility belts (`lo`, `samber`, etc.).

## Repository and Issues

- This project is hosted on **Gitea**, not GitHub. Use the `tea` CLI or the Gitea API for issue, milestone, and release operations.
- Remote: `ssh://git@git.dries.info/gerco/Envoke`
- Do not use `gh` (GitHub CLI).

## Code Style

- `gofmt` and `go vet` must pass with no warnings.
- Keep packages small and purposeful. `internal/backend` holds the interface and registry. Each backend lives in its own sub-package (`internal/backend/keychain`, etc.).
- No global state outside of `main`. Pass configuration and dependencies explicitly.
- Table-driven tests for parsers and config logic.

## Milestones

- **1.0** — core runner, config/dotfile parsing, pluggable backend interface, OS keychain + Keeper + JumpCloud backends, TUI commands, cross-platform release builds.
- **2.0** — SSH agent feature (issue #1), SOPS backend, HashiCorp Vault backend.

When creating issues, assign them to the correct milestone from the start.
