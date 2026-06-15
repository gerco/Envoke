---
title: ee status
weight: 3
---

Display authentication and reachability status for all namespaces in the current project.

```shell
ee status
```

## Output

```
Namespace       Backend     Status
──────────────────────────────────
aws-dev         aws         ✓ authenticated
db-local        keychain    ✓ unlocked
stripe-test     1password   ✗ token expired
```

In a non-interactive environment (CI, piped output), plain text is printed instead of the TUI table.
