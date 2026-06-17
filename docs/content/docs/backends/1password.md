---
title: 1Password
weight: 3
---

The `1password` backend uses the [1Password Secrets Automation SDK](https://developer.1password.com/docs/sdks/) with a service account token. It is read-only.
This backend is not compiled into pre-built binaries yet. To test it, check out the source and compile it in.

## Global config

```yaml
backends:
  1password:
    token: ops_...    # service account token
```

Store the token in a secure location — do not commit it. You can also pass it via the `OP_SERVICE_ACCOUNT_TOKEN` environment variable.

## `.envoke` usage

```yaml
namespaces:
- name: stripe-test
  backend: 1password
```

The namespace name must match the vault name (or item title, depending on your configuration) in 1Password.

## Service account setup

1. In 1Password, go to **Integrations → Service Accounts**.
2. Create a service account with read access to the relevant vaults.
3. Copy the token into your global config or environment.
