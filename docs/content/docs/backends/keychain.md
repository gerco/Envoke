---
title: OS Keychain
weight: 1
---

The `keychain` backend stores secrets in the operating system's native credential store via [`99designs/keyring`](https://github.com/99designs/keyring):

| OS | Store |
|----|-------|
| macOS | Keychain |
| Windows | Credential Manager |
| Linux | Secret Service (GNOME Keyring / KWallet) |

## Configuration

No global config entry is required for most systems. If you need to override the defaults — most commonly on Linux, where multiple applications share the `login` collection — you can add an explicit backend entry:

```yaml
backends:
  keychain:
    type: keychain
    service_name: my-app   # keyring collection/service to use
    key_prefix: myapp/     # prefix applied to all stored keys
```

| Option | Default | Description |
|--------|---------|-------------|
| `service_name` | `login` (Linux), `envoke` (macOS/Windows) | The keyring collection or service name |
| `key_prefix` | `envoke/` (Linux), empty (macOS/Windows) | Prefix added to every stored key |

On macOS and Windows the service name already scopes keys to Envoke, so these options are rarely needed.

## `.envoke` usage

`keychain` is the default backend, so `backend:` can be omitted:

```yaml
namespaces:
- name: db-local
```

It can also be stated explicitly:

```yaml
namespaces:
- name: db-local
  backend: keychain
```

## Security notes

Access is gated by the OS unlock mechanism — Touch ID, Face ID, or Windows Hello. Envoke inherits this for free. Secrets stored in the keychain are not accessible to other users or applications on the system.
