# TokenSave roadmap

This roadmap describes proposals, not commitments or completed features.

## Current capabilities

- Local command wrapping with complete stdout and stderr storage.
- Compact summaries with common secret redaction for displayed output.
- Generic, Git status/diff, PHPUnit, Pest, Composer, and npm/pnpm/yarn parsers.
- Release installers and a global Codex Skill.

## Short-term priorities

- Improve fixture coverage and parser diagnostics.
- Keep installers, checksums, and release artifacts easy to verify.
- Improve Windows signal handling.
- Expand reproducible benchmark scenarios.

## Parser ecosystem

Potential parser proposals include PHPStan, ESLint, TypeScript, Jest, Vitest, and Playwright. Each proposal should include sanitized representative output, fixtures, and tests.

## Packaging and installation

Potential packaging proposals include Homebrew and Scoop. Packaging work should retain checksum verification and document the Skill-location migration.

## Agent integrations

The bundled Codex Skill is the current agent integration. Additional agent integrations are possible proposals when they can preserve local log handling and clear privacy behavior.

## Explicitly out of scope

- Uploading logs to a hosted service.
- Mandatory accounts, telemetry, or usage analytics.
- Replacing the wrapped command's exit code.
- Claiming token savings from line or byte measurements.
