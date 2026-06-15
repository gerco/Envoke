---
title: Getting Started
weight: 1
---

## Installation

**macOS and Linux** — install via Homebrew:

```shell
brew install gerco/tap/envoke
```

**Windows** — download the latest binary from the [releases page](https://github.com/gerco/Envoke/releases).

**From source** — requires Go 1.21+:

```shell
go install github.com/gerco/Envoke/cmd/ee@latest
```

## Quick start

The fastest way to get started is with the OS keychain backend — no extra accounts or configuration needed.

**1. Store a secret**

Use `ee set` to write a value into the keychain. If you omit the value, it is read from stdin with echo suppressed so it never appears in shell history:

```shell
$ ee set myapp DB_PASSWORD
Value for myapp/DB_PASSWORD: ········
Stored myapp/DB_PASSWORD
Added namespace "myapp" (backend: keychain) to .envoke
```

`ee set` automatically creates an `.envoke` file in the current directory and adds the namespace if it isn't already there.

**2. Run a command with the secret injected**

```shell
$ ee myapp -- psql -h localhost -U myuser mydb
```

Secrets from the `myapp` namespace are fetched from the keychain and injected into the subprocess environment. They exist only for the lifetime of that process — nothing is exported to your shell.

**3. Commit `.envoke`**

The `.envoke` file records which namespaces this project needs and which backend holds them. It contains no secrets and is safe to commit:

```yaml
namespaces:
- name: myapp-local      # backend: keychain is the default
- name: myapp-aws
  backend: aws
- name: myapp-secrets
  backend: 1password
```

```shell
$ git add .envoke && git commit -m "add envoke config"
```

The committed dotfile is the contract: it tells everyone exactly which namespaces the project needs and where each one lives.

## Next steps

- [Configuration reference](configuration) — dotfile format, layering, and global config
- [Backends](backends) — AWS, 1Password, shell, and others
- [Commands](commands) — full CLI reference
