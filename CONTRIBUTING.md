# Contributing to TokenSave

Thanks for contributing. TokenSave is a small Go CLI, so focused changes with clear tests are preferred.

## Development prerequisites

- Go 1.22 or later
- Git
- Bash for Unix installer and benchmark checks
- PowerShell 7 or Windows PowerShell for the PowerShell installer checks

## Build, format, and test

```sh
make build
gofmt -w cmd internal
go vet ./...
go test ./...
go test -race ./...
make benchmark
```

The binary is built at `bin/tokensave`. Use `TOKENSAVE_HOME` with a temporary directory while testing commands that create runs.

## Adding a parser

Parsers implement the `Parser` interface in `internal/tokensave/analyze.go`. Keep detection conservative, extract only useful stable fields, and fall back safely to the generic parser. Add representative sanitized fixtures under `fixtures/` and table-driven or focused tests under `internal/tokensave/` for both successful and failing output.

## Installer changes

Run `bash -n scripts/install.sh` and the installer checks in `scripts/test-installers.sh`. On Windows, parse `scripts/install.ps1` with PowerShell and run its path-resolution checks. Use temporary directories and environment-variable overrides; never test an installer against your real home directory.

## Pull requests

Explain the problem and motivation, keep behavior-compatible changes narrow, and include tests for changed behavior. Update documentation when commands, installation, storage, or privacy behavior changes. Parser changes need sanitized fixtures and tests. Preserve the wrapped command's exit code.

Do not include secrets, credentials, complete private logs, or proprietary source code in commits, fixtures, issues, or pull requests.

## Security reports

Do not open a public issue for an undisclosed vulnerability. Follow the private reporting guidance in [SECURITY.md](SECURITY.md).
