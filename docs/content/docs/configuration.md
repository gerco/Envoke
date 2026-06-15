---
title: Configuration
weight: 2
---

Envoke uses three config layers, each with a distinct responsibility:

```
~/.config/envoke/config.yaml      ← global: your backends, credentials   (gitignored)
<project>/.envoke                 ← project: what this project needs      (committed)
<project>/.envoke.local           ← local: personal overrides             (gitignored)
```

## Global config

Located at the OS-appropriate path:

| OS | Path |
|----|------|
| Linux | `$XDG_CONFIG_HOME/envoke/config.yaml` or `~/.config/envoke/config.yaml` |
| macOS | `~/Library/Application Support/envoke/config.yaml` |
| Windows | `%APPDATA%\envoke\config.yaml` |

The global config describes *how* to reach backends — credentials, endpoints, tokens. It is never committed.

## Project dotfile (`.envoke`)

The dotfile lists namespaces the project requires and which backend holds each one. It contains no secrets and is committed to git.

```yaml
namespaces:
- name: aws-dev
  backend: aws
- name: db-local        # backend: keychain is the default and can be omitted
- name: stripe-test
  backend: 1password
```

Multiple developers share the same `.envoke`. Their global configs differ; the dotfile stays the same.

## Local overrides (`.envoke.local`)

Overrides namespace configuration for a single developer without touching the shared dotfile. Gitignored by convention — add it to your global `.gitignore`:

```shell
echo '.envoke.local' >> ~/.gitignore
```
