---
title: Envoke
toc: false
---

**Per-command secret injection for development teams.**

Secrets are fetched from your vault and injected into a subprocess. Nothing persists. Nothing leaks sideways.

```shell
$ ee -- make dev
$ ee -- psql -h $DB_HOST -U $DB_USER
$ ee -- aider --model claude-sonnet-4-6
```

{{< cards >}}
  {{< card link="docs/getting-started" title="Get Started" icon="play" >}}
  {{< card link="docs/configuration" title="Configuration" icon="adjustments" >}}
  {{< card link="docs/backends" title="Backends" icon="server" >}}
  {{< card link="docs/commands" title="Commands" icon="terminal" >}}
{{< /cards >}}
