---
title: "ee [namespace] -- <command>"
weight: 1
---

Run a command with secrets injected into its environment.

```shell
ee [<namespace>] [--] <command> [args...]
```

## Forms

**With a `.envoke` file — `--` is optional:**

```shell
ee -- make dev
ee make dev        # equivalent
```

Reads `.envoke` from the current directory (walking up the tree to find it), fetches secrets from all configured namespaces, and injects them into the subprocess environment.

**With an explicit namespace — `--` is required:**

```shell
ee myapp -- psql -h localhost -U myuser mydb
```

Bypasses `.envoke` entirely. Fetches secrets from the named keychain namespace and injects them into the subprocess. This is useful before a `.envoke` file exists, or when you want to run a one-off command against a specific namespace without affecting the project config.

The `--` separator is required here to distinguish the namespace name from the start of the command.

## Behavior

1. If a namespace is given, loads that keychain namespace only. Otherwise reads `.envoke`.
2. Fetches secrets from each applicable backend.
3. Spawns `<command>` as a subprocess with those secrets merged into the environment.
4. Exits with the same exit code as the subprocess.

Secrets exist only for the lifetime of the subprocess. They are never written to disk, exported to the parent shell, or visible to sibling processes.

## Environment variable resolution

Variables from all namespaces are merged into the subprocess environment. Backend values take precedence over variables already present in the environment.

## Examples

```shell
ee -- make dev
ee -- npm start
ee -- pytest -x

ee myapp -- psql -h $DB_HOST -U $DB_USER
ee staging -- kubectl get pods
```
