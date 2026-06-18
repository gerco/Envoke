---
title: Backends
weight: 3
---

Backends are the secret stores Envoke fetches from. Each namespace in your `.envoke` file is backed by one.

## Available backends

| Backend | Key | Platforms | Status      |
|---------|-----|-----------|-------------|
| OS Keychain | `keychain` | Windows, macOS, Linux | Stable      |
| AWS Secrets Manager | `aws` | All | Stable      |
| Shell command | `shell` | All | Stable      |
| 1Password | `1password` | All | Source only |
| SOPS | `sops` | All | Planned     |
| HashiCorp Vault | `vault` | All | Planned     |
| Keeper | `keeper` | All | Planned     |
| JumpCloud | `jumpcloud` | All | Planned     |

Select a backend from the sidebar for configuration details.
