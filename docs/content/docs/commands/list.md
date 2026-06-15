---
title: ee list
weight: 3
---

List secret key names in a namespace or across all project namespaces.

```shell
ee list [--backend <backend>] [namespace]
```

Only key names are printed — never values.

## Forms

**List all namespaces in the project:**

```shell
$ ee list
myapp        API_KEY
myapp        DB_PASSWORD
myapp-aws    STRIPE_SECRET
```

Reads `.envoke` and lists keys from every declared namespace, grouped in a two-column table.

**List a single namespace:**

```shell
$ ee list myapp
API_KEY
DB_PASSWORD
```

If the namespace is not declared in `.envoke`, the keychain backend is used.

## Flags

| Flag | Description |
|------|-------------|
| `--backend <name>` | Use a specific backend instead of namespace lookup |

`--backend` requires a namespace argument:

```shell
ee list --backend aws myapp-aws
```
