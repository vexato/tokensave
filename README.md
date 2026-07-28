# TokenSave

[![CI](https://github.com/vexato/tokensave/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/vexato/tokensave/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/vexato/tokensave?display_name=tag&sort=semver)](https://github.com/vexato/tokensave/releases)
[![License](https://img.shields.io/github/license/vexato/tokensave)](LICENSE)
[![Go version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](go.mod)

## Project introduction

> Keep terminal noise out of your coding agent's context.

TokenSave is a local Go CLI for developers and coding agents. It wraps verbose commands, stores their complete stdout and stderr locally, and returns a compact summary that is easier to act on. Displayed summaries redact common secrets, the wrapped command keeps its original exit code, and TokenSave does not upload logs or use telemetry.

## Key benefits

- Keep complete command output locally for later inspection.
- Read a compact, redacted summary first.
- Preserve the wrapped command's exit code for scripts and CI.
- Inspect one failure or a narrow log range without rerunning a command.
- Use the same workflow for people and coding agents.

## Why TokenSave exists

Test suites, package managers, and Git commands can produce far more output than a human or agent needs to diagnose the next step. TokenSave keeps the original log available while making progressive inspection practical: summary first, then one failure or a relevant range, then the full log only when needed.

## Installation

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.ps1 | iex
```

The installers download a release asset, verify its SHA-256 against the release `checksums.txt`, and install the bundled global Codex Skill. Go is not required.

Default Skill locations:

```text
Linux and macOS: $HOME/.agents/skills/tokensave
Windows:          %USERPROFILE%\.agents\skills\tokensave
```

Set `TOKENSAVE_SKILL_DIR` to use another global Skill directory. Set `TOKENSAVE_INSTALL_SKILL=0` on Unix-like systems, or pass `-SkipSkill` on PowerShell, to install only the binary. `-SkipCodexSkill` remains a PowerShell compatibility alias.

The installers detect but never delete a legacy `~/.codex/skills/tokensave` installation; migrate it manually after verifying the new directory. PowerShell adds the binary directory to the user `PATH`.

To install from a checkout, use `make build` or `make install` (Go 1.22 or later is required).

## Quick start

```sh
tokensave git status
tokensave npm test
tokensave run vendor/bin/phpunit
tokensave --shell "php artisan test && npm test"
tokensave npm test --json
```

Commands run directly without a shell by default. Use `--shell` only when shell syntax is intended, and use `--` before wrapped arguments that intentionally share a TokenSave option name.

## Inspecting previous runs

```sh
tokensave list
tokensave list --failed
tokensave show 20260727-153045-a81f --failure 1
tokensave show 20260727-153045-a81f --tail 50
tokensave show 20260727-153045-a81f --lines 40:80
tokensave clean --older-than 7d
```

`show --full` prints a complete stored log. Treat it with care because complete logs are intentionally retained without redaction.

## How it works

For each wrapped command, TokenSave captures stdout and stderr, stores both streams and a combined log, selects a parser when it recognizes the command or output, redacts the displayed summary, and writes metadata plus JSON and text summaries. It then returns the child process's exit code.

## Storage locations

Set `TOKENSAVE_HOME` to choose a storage directory. Otherwise TokenSave uses:

- Linux: `$XDG_STATE_HOME/tokensave` or `~/.local/state/tokensave`
- macOS: `~/Library/Application Support/tokensave`
- Windows: `%LOCALAPPDATA%\tokensave`

If the system location cannot be written, TokenSave falls back to an existing `.tokensave/` project store, or creates one when necessary. A run contains `metadata.json`, `stdout.log`, `stderr.log`, `combined.log`, `summary.txt`, and `summary.json`.

## Output limits

The default terminal limits are 80 lines, 12,000 characters, and five failures. Override them for one invocation:

```sh
tokensave npm test --max-lines 120 --max-chars 20000 --max-failures 10
tokensave npm test --quiet
tokensave npm test --no-summary
```

## JSON output

`--json` writes one JSON summary object while preserving the wrapped command's exit code:

```json
{"run_id":"20260727-153045-a81f","status":"failed","exit_code":1,"duration_ms":18400,"parser":"phpunit","summary":{"tests":4,"failed":2},"failures":[{"index":1,"name":"UserTest::testCreate","message":"Expected 201"}],"log_path":"/local/path"}
```

## Configuration

TokenSave reads `~/.config/tokensave/config.yml`, then `.tokensave.yml` in the current project; command-line options take precedence.

```yaml
max_lines: 80
max_chars: 12000
max_failures: 5
retention_days: 14
redact:
  enabled: true
  patterns:
    - "MY_CUSTOM_SECRET=.*"
commands:
  "php test":
    parser: phpunit
```

## Built-in parsers

Built-in parsers cover generic output, Git status and diff, PHPUnit, Pest, Composer, and npm/pnpm/yarn. The parser registry uses a small `Detect`/`Parse` interface; other tools can be added with fixtures and tests.

## Privacy and redaction

TokenSave does not upload logs, call a remote API, or enable telemetry. Displayed summaries redact common authorization headers, token/password/secret/API-key assignments, URL passwords, and configured patterns. Complete logs remain unmodified on disk, so do not publish them if they contain credentials, proprietary source, or other sensitive data.

## Codex Skill

The bundled global Skill is in [`skills/tokensave`](skills/tokensave). Installers copy the complete directory to the configured user-level Skill location. To install it manually, copy the whole directory to `~/.agents/skills/tokensave`, then run:

```sh
codex /skills
```

For a repository-specific Skill, copy it to `.agents/skills/tokensave` in that repository.

## Benchmarks

Run the deterministic, offline correctness and performance benchmark with:

```sh
make benchmark
```

The suite generates both [the Markdown benchmark report](docs/benchmarks.md) and its machine-readable JSON counterpart from the same measured data. It checks exit-code preservation, parser detection, diagnostic retention, secret redaction, output limits, displayed line and byte reduction, and runtime overhead across deterministic generated output, repository fixtures, and a temporary Git repository.

Results depend on the command, parser, configuration, operating system, filesystem, and TokenSave version. They do not represent every command or every coding-agent workload; see the full report for the measured results and limitations.

## Current limitations

- TokenSave is local-only and has no MCP server.
- Shell signal behavior differs by platform.
- Git status is most informative with porcelain v2 output.
- Complete local logs are not redacted.
- Parser coverage is intentionally limited to the built-in set listed above.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Please use sanitized output in issues and pull requests; never publish credentials, complete private logs, or proprietary source code. Security vulnerabilities belong in [private GitHub vulnerability reporting](SECURITY.md), not public issues.

## License

TokenSave is released under the [MIT License](LICENSE).
