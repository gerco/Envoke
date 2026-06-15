---
title: Commands
weight: 4
---

All Envoke functionality is accessed through the `ee` command.

## Global flags

| Flag | Description |
|------|-------------|
| `-v`, `--verbose` | Print backend activity to stderr |
| `-vv` | Print individual key fetches and shell commands |
| `--config <path>` | Override global config file path |

## Commands

| Command | Description |
|---------|-------------|
| `ee [--] <cmd>` | Run a command with all `.envoke` namespaces injected |
| `ee <namespace> -- <cmd>` | Run a command with a single keychain namespace injected |
| `ee set <namespace> <key> [value]` | Store a secret in a backend namespace |
| `ee list [namespace]` | List secret key names in a namespace or all namespaces |
| `ee status` | Show authentication status for all namespaces |
| `ee run` | Fuzzy-pick a profile and run interactively |
| `ee config path` | Print the global config file path |
| `ee config edit` | Open the global config in your editor |
| `ee config show` | Display the global config (redacted) |
