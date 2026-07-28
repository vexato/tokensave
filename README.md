# TokenSave

**Keep terminal noise out of your coding agent's context.**

TokenSave is a local CLI wrapper that turns verbose command output into compact, redacted summaries for developers and coding agents.

It runs the original command, preserves its exit code, stores the complete `stdout` and `stderr` locally, and displays only the information needed to understand what happened.

```text
Verbose command
      │
      ▼
 thousands of lines
      │
      ▼
   TokenSave
      │
      ├── Compact summary for you or your coding agent
      └── Complete local logs for deeper inspection
```

* **100% local**
* **Zero telemetry**
* **No remote API**
* **Original exit codes preserved**
* **Secrets redacted from summaries**
* **Complete logs retained locally**
* **Built for developers and coding agents**

## Why TokenSave?

Commands such as test suites, builds, package managers, and Git operations can produce thousands of terminal lines.

For a human, that makes failures harder to find. For a coding agent, it wastes context on repetitive output.

Instead of giving the agent the complete log immediately:

```sh
npm test
```

run:

```sh
tokensave npm test
```

TokenSave returns a compact summary while preserving the full output locally. You can then inspect one failure, a line range, or the complete log only when necessary.

## Install

### Linux and macOS

```sh
curl -fsSL https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.sh | sh
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.ps1 | iex
```

The installers download the appropriate pre-built binary from GitHub Releases and verify its SHA-256 against the release's `checksums.txt` before extracting it. Go is not required.

They also install the bundled TokenSave Codex Skill globally for the current user:

```text
Linux and macOS: ~/.agents/skills/tokensave
Windows:          %USERPROFILE%\.agents\skills\tokensave
```

This makes the skill available to Codex across all repositories.

To install only the CLI, without the Skill:

```sh
curl -fsSL https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.sh | TOKENSAVE_INSTALL_SKILL=0 sh
```

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/vexato/tokensave/main/scripts/install.ps1))) -SkipSkill
```

`-SkipCodexSkill` remains accepted as a PowerShell compatibility alias.

The PowerShell installer adds TokenSave to the user `PATH` automatically and makes it available in the current session.

To install a specific version on Linux or macOS:

```sh
TOKENSAVE_VERSION=<version> sh install.sh
```

To install a specific version with PowerShell:

```powershell
.\install.ps1 -Version <version>
```

For contributors building from a checkout:

```sh
make build
make install
```

Building from source requires Go.

## Quick start

Wrap any command with `tokensave`:

```sh
tokensave git status
tokensave git diff
tokensave vendor/bin/phpunit
tokensave npm test
tokensave pnpm test
tokensave composer install
```

The optional `run` command is also supported:

```sh
tokensave run vendor/bin/phpunit
```

Return a stable JSON object:

```sh
tokensave npm test --json
```

Execute an actual shell expression:

```sh
tokensave --shell "php artisan test && npm test"
```

Commands run directly without a shell by default, so regular arguments—including arguments containing spaces—remain safe.

Use `--shell` only when shell features such as pipes, redirections, environment expansion, or command chaining are intentionally required.

Use `--` before child arguments that have the same name as TokenSave options:

```sh
tokensave tool -- --json
```

## Inspect a run

Every execution receives a run ID.

Use it to inspect only the information you need:

```sh
# Show the first detected failure
tokensave show 20260727-153045-a81f --failure 1

# Show the last 50 lines
tokensave show 20260727-153045-a81f --tail 50

# Show a specific line range
tokensave show 20260727-153045-a81f --lines 40:80

# Show the complete stored output
tokensave show 20260727-153045-a81f --full
```

List previous runs:

```sh
tokensave list
tokensave list --failed
```

Remove old runs:

```sh
tokensave clean --older-than 7d
```

## How it works

For every wrapped command, TokenSave:

1. Executes the command.
2. Captures `stdout` and `stderr`.
3. Detects an appropriate parser.
4. Extracts failures, paths, counts, and useful diagnostics.
5. Redacts sensitive values from the displayed summary.
6. Stores the complete, unmodified logs locally.
7. Returns the original command's exit code.

This lets developers and coding agents follow a progressive inspection workflow:

```text
Summary
   ↓
Single failure
   ↓
Relevant line range
   ↓
Complete log, only when necessary
```

## Storage

Runs are stored below `TOKENSAVE_HOME` when the environment variable is set.

Otherwise, TokenSave uses the native state directory:

* Linux: `$XDG_STATE_HOME/tokensave` or `~/.local/state/tokensave`
* macOS: `~/Library/Application Support/tokensave`
* Windows: `%LOCALAPPDATA%\tokensave`

When the system state directory is unavailable—particularly inside a sandboxed coding agent—TokenSave automatically falls back to:

```text
.tokensave/
```

in the current project.

The directory is Git-ignored by default.

Set `TOKENSAVE_HOME` to force a specific location:

```sh
TOKENSAVE_HOME=/custom/path tokensave npm test
```

Each run directory contains:

```text
metadata.json
stdout.log
stderr.log
combined.log
summary.txt
summary.json
```

## Output limits

Terminal summaries use the following defaults:

| Setting            | Default |
| ------------------ | ------: |
| Maximum lines      |      80 |
| Maximum characters |  12,000 |
| Maximum failures   |       5 |
| Maximum paths      |       8 |

Override the limits for a single invocation:

```sh
tokensave npm test --max-lines 120
tokensave npm test --max-chars 20000
tokensave npm test --max-failures 10
```

Additional output modes:

```sh
# Print only the run ID
tokensave npm test --quiet

# Store the run without printing a summary
tokensave npm test --no-summary

# Print one stable JSON object
tokensave npm test --json
```

## JSON output

Using `--json` produces one stable JSON object:

```json
{
  "run_id": "20260727-153045-a81f",
  "status": "failed",
  "exit_code": 1,
  "duration_ms": 18400,
  "parser": "phpunit",
  "summary": {
    "tests": 428,
    "failed": 2
  },
  "failures": [
    {
      "index": 1,
      "name": "UserServiceTest::testCreatesUser",
      "message": "Expected status 201, received 500",
      "file": "tests/Unit/UserServiceTest.php",
      "line": 84
    }
  ],
  "log_path": "/local/path"
}
```

TokenSave always returns the wrapped command's original exit code.

That means it can be used in scripts, automation, and agent workflows without hiding command failures.

## Configuration

TokenSave reads optional configuration from these locations, in order:

1. `~/.config/tokensave/config.yml`
2. `.tokensave.yml` in the current project
3. CLI options

CLI options have the highest priority.

Example:

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

TokenSave includes parsers for:

* Generic command output
* Git status
* Git diff
* PHPUnit
* Pest
* Composer
* npm
* pnpm
* yarn

The parser registry uses a small `Detect`/`Parse` interface, making it possible to add support for additional tools without changing the command runner.

Good candidates for future parsers include:

* PHPStan
* ESLint
* TypeScript
* Jest
* Vitest
* Playwright

Contributions for additional parsers are welcome.

## Privacy and redaction

TokenSave does not:

* Upload command output
* Call a remote API
* Enable telemetry
* Send usage analytics
* Require an account

Displayed summaries redact common sensitive values, including:

* Authorization headers
* Tokens
* Passwords
* Secrets
* API keys
* Passwords embedded in URLs
* User-configured patterns

Complete local logs are deliberately stored **without modification**.

This is necessary for accurate debugging, but it also means the TokenSave state directory may contain sensitive information. Protect that directory and use `tokensave show --full` only when necessary.

## Codex Skill

The repository includes a global Codex Skill:

```text
skills/tokensave/SKILL.md
```

The installation scripts copy it to the current user's global skill directory:

```text
~/.agents/skills/tokensave
```

The skill teaches Codex to:

1. Run verbose commands through TokenSave.
2. Read the compact summary first.
3. Inspect a single failure when possible.
4. Request only the relevant line range.
5. Read the complete log only as a last resort.

To install it manually:

```sh
mkdir -p ~/.agents/skills/tokensave
cp skills/tokensave/SKILL.md ~/.agents/skills/tokensave/SKILL.md
```

After installation, use `/skills` in Codex to verify that TokenSave is available.

## Project status and limits

TokenSave is currently local-only.

Known limitations:

* Shell signal behavior can vary by platform.
* The Git status parser provides its richest output when the recorded command emits porcelain v2.
* Complete logs are stored locally without redaction.
* Parser coverage is still expanding.
* No MCP server is currently included.

## Contributing

Bug reports, parser contributions, documentation improvements, and platform testing are welcome.

Useful contribution areas include:

* New command parsers
* Additional secret-redaction patterns
* Windows, Linux, and macOS testing
* Installation improvements
* Better summaries for existing parsers
* Packaging and distribution

When reporting a problem, include:

* Your operating system
* The TokenSave version
* The wrapped command
* The selected parser
* The redacted summary or JSON output
* The original exit code

Do not publish complete logs containing secrets or private source code.

## License

See the repository's license file for details.

---

**Does TokenSave keep your terminal noise under control? Give the repository a star and share the command that produces the noisiest output in your workflow.**
