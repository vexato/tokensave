# TokenSave

TokenSave is a local CLI wrapper for verbose commands. It stores complete stdout and stderr on disk, then gives people and coding agents a compact, redacted summary instead of thousands of terminal lines.

It never uploads logs, calls a remote API, or enables telemetry.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.ps1 | iex
```

Both installers download the appropriate pre-built GitHub Release asset; Go is not required. They also install the bundled `tokensave` Codex Skill into `$CODEX_HOME/skills/tokensave` (or `~/.codex/skills/tokensave`). The PowerShell installer adds TokenSave to the user `PATH` automatically and makes it available in the current session. Pin a Unix-like installation with `TOKENSAVE_VERSION=v1.0.0 sh install.sh`, or use PowerShell `-Version v1.0.0`. For contributors building from a checkout, use `make build` / `make install` (Go required).

## Usage

```sh
tokensave git status
tokensave git diff
tokensave run vendor/bin/phpunit
tokensave npm test --json
tokensave --shell "php artisan test && npm test"

tokensave show 20260727-153045-a81f --failure 1
tokensave show 20260727-153045-a81f --tail 50
tokensave show 20260727-153045-a81f --lines 40:80
tokensave list --failed
tokensave clean --older-than 7d
```

`run` is optional. Commands execute directly without a shell, so normal arguments (including spaces) stay safe. Use `--shell` only when shell syntax is intended.
Use `--` before child arguments that intentionally have the same name as TokenSave options, for example `tokensave tool -- --json`.

Each run is stored below `TOKENSAVE_HOME`, or the native state directory:

- Linux: `$XDG_STATE_HOME/tokensave` or `~/.local/state/tokensave`
- macOS: `~/Library/Application Support/tokensave`
- Windows: `%LOCALAPPDATA%\\tokensave`

The run directory contains `metadata.json`, `stdout.log`, `stderr.log`, `combined.log`, `summary.txt`, and `summary.json`.

## Output and JSON

Terminal summaries default to 80 lines, 12,000 characters, five failures, and eight paths. Set `--max-lines`, `--max-chars`, or `--max-failures` to change the invocation. `--quiet` prints only the run id, `--no-summary` prints nothing, and `--json` writes one stable JSON object:

```json
{"run_id":"20260727-153045-a81f","status":"failed","exit_code":1,"duration_ms":18400,"parser":"phpunit","summary":{"tests":428,"failed":2},"failures":[{"index":1,"name":"UserServiceTest::testCreatesUser","message":"Expected status 201, received 500","file":"tests/Unit/UserServiceTest.php","line":84}],"log_path":"/local/path"}
```

TokenSave returns the wrapped command's original exit code.

## Configuration

Optional configuration is read first from `~/.config/tokensave/config.yml`, then `.tokensave.yml` in the current project; CLI options win.

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

## Parsers and privacy

Built-in parsers cover generic output, Git status/diff, PHPUnit, Pest, Composer, and npm/pnpm/yarn. The parser registry has a small `Detect`/`Parse` interface, so PHPStan, ESLint, TypeScript, Jest, Vitest, and Playwright can be added without changing the runner.

Displayed summaries redact common authorization headers, token/password/secret/API-key assignments, URL passwords, and configured patterns. Complete local logs are deliberately retained unmodified, so protect the state directory and use `show --full` only when necessary.

## Codex

Install or copy [`skills/tokensave/SKILL.md`](skills/tokensave/SKILL.md) into a Codex skill directory. It directs agents to run verbose commands through TokenSave, read the summary first, inspect a single failure or range next, and request a complete log only as a last resort.

## Limits

The first release is local-only. Shell signal behavior is platform dependent, and the Git status parser is richest when the recorded command emits porcelain v2. No MCP server is included.
