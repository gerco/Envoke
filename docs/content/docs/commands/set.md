---
title: ee set
weight: 2
---

Store a secret in a backend namespace.

```shell
ee set [--backend <backend>] <namespace> <key> [value]
```

## Behavior

If `value` is omitted, it is read from stdin with echo suppressed so it never appears in shell history:

```shell
$ ee set myapp DB_PASSWORD
Value for myapp/DB_PASSWORD: ········
Stored myapp/DB_PASSWORD
```

After storing the secret, `ee set` adds the namespace to `.envoke` if that file already exists and the namespace is not yet declared. If no `.envoke` file exists, the secret is stored but nothing is written to disk.

## Flags

| Flag | Description |
|------|-------------|
| `--backend <name>` | Use a specific backend instead of looking up the namespace in `.envoke` |

Without `--backend`, the namespace must already be declared in `.envoke`, or the default keychain backend is used.

## Examples

```shell
# Prompt for value (hidden input)
ee set myapp DB_PASSWORD

# Pass value directly (appears in shell history — avoid for sensitive values)
echo "s3cr3t" | ee set myapp DB_PASSWORD

# Store into a specific backend, bypassing namespace lookup
ee set --backend aws myapp API_KEY
```

## `.envoke` update

When a namespace is added, `backend: keychain` is omitted if keychain is the backend, since it is the default:

```yaml
namespaces:
- name: myapp       # backend omitted — keychain is the default
```

If the namespace is already declared in `.envoke`, `ee set` leaves the file unchanged.
