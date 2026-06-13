---
title: "Quick start"
description: "Run your first elife command."
weight: 30
---

Once `elife` is on your `PATH`:

```bash
elife --help       # see the command tree
elife version      # build info
```

This is a fresh scaffold, so the command tree is just `version` for now. Add
your first real command in `cli/`, build on the `elife` library package,
and document it here.

A good first command usually fetches one thing and prints it as JSON, so the
output pipes straight into `jq` and the rest of your tools.
