---
name: tokensave
description: Reduce terminal context usage by summarizing tests, builds, Git commands, linters and other verbose command output.
---

# TokenSave workflow

Use `tokensave` for commands that can produce substantial output: tests, builds, linters, package managers and Git diffs/status.

1. Run the command through TokenSave and read only its summary first, for example `tokensave npm test`.
2. Keep the emitted run id. For a failure, inspect only the relevant failure with `tokensave show <run-id> --failure 1`.
3. If needed, request the smallest log section: `--tail 50`, `--head 50`, or `--lines 40:80` (and `--stdout`/`--stderr` when useful).
4. Use `--full` only as a last resort. Do not re-run an identical command merely to recover output already stored for its run id.
5. Prefer focused tests after identifying a failure instead of repeatedly running a whole suite.

Use `--json` when a structured result is more useful than a human summary. TokenSave saves full logs locally; summaries are redacted, but full logs may contain sensitive values.
