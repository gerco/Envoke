---
title: ee config
weight: 4
---

Manage the global configuration file that defines backends.

## Subcommands

### `ee config path`

Print the path to the global config file without opening it.

```shell
$ ee config path
/Users/alice/Library/Application Support/envoke/config.yaml
```

### `ee config edit`

Open the global config file in your default editor (`$EDITOR` on Unix, `vi` as fallback, `notepad.exe` on Windows). Creates the config directory if it doesn't exist yet.

```shell
$ ee config edit
```

In a non-interactive environment, prints the config path instead.

### `ee config show`

Display the current global config with sensitive values redacted.

```shell
$ ee config show
# Global config: /Users/alice/Library/Application Support/envoke/config.yaml
backends:
  prod-aws:
    type: aws
    region: us-east-1
```

If no config file exists, prints the expected location.
