---
title: Shell
weight: 4
---

The `shell` backend runs an arbitrary shell command to obtain a secret value. It is useful for integrating with tools that have their own CLI but no native Envoke backend.

The backend is read-only — `ee set` is not supported for shell namespaces.

## `.envoke` usage

Each entry under `vars` maps an environment variable name to the shell command that produces its value. The command's stdout (trimmed) becomes the variable's value.

```yaml
namespaces:
- name: infisical-dev
  backend: shell
  vars:
    DB_PASSWORD: infisical secrets get DB_PASSWORD --plain
    API_KEY: infisical secrets get API_KEY --plain
```

## Default shell

Commands are executed with `sh -c` on Unix and `powershell -Command` on Windows.

## Custom shell

To use a different interpreter, add an explicit backend entry in the global config with a `shell` option:

```yaml
backends:
  my-bash:
    type: shell
    shell: /bin/bash -c
```

Then reference it by name in `.envoke`:

```yaml
namespaces:
- name: infisical-dev
  backend: my-bash
  vars:
    DB_PASSWORD: infisical secrets get DB_PASSWORD --plain
    API_KEY: infisical secrets get API_KEY --plain
```

The `shell` value is split on whitespace to form the command prefix — the var's command string is appended as the final argument.

## Security notes

The shell command runs with the same privileges as the `ee` process. Avoid commands that write secrets to disk or shell history. Command output is trimmed of trailing whitespace before being injected into the environment.
