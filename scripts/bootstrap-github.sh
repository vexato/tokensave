#!/usr/bin/env sh
set -eu

repo="vexato/tokensave"
apply=false
case "${1:-}" in
  "") ;;
  --apply) apply=true ;;
  --help|-h) printf '%s\n' 'Usage: scripts/bootstrap-github.sh [--apply]'; exit 0 ;;
  *) printf '%s\n' 'Usage: scripts/bootstrap-github.sh [--apply]' >&2; exit 2 ;;
esac

description='Local CLI that turns noisy command output into compact, redacted summaries for developers and coding agents.'
topics='cli developer-tools ai-agents coding-agents codex agent-skills terminal logs testing golang privacy context-window'

print_plan() {
  printf '%s\n' "Repository: $repo"
  printf '%s\n' "Set description: $description"
  printf '%s\n' "Enable Issues and add topics: $topics"
  printf '%s\n' 'Create or update the documented labels and create missing roadmap issues only after searching open and closed issues.'
}

if [ "$apply" = false ]; then
  printf '%s\n' 'Dry run: no GitHub changes will be made.'
  print_plan
  exit 0
fi

command -v gh >/dev/null 2>&1 || { printf '%s\n' 'gh is required. Install GitHub CLI and run gh auth login.' >&2; exit 1; }
gh auth status
actual_repo=$(gh repo view "$repo" --json nameWithOwner --jq .nameWithOwner)
[ "$actual_repo" = "$repo" ] || { printf 'Expected repository %s, found %s.\n' "$repo" "$actual_repo" >&2; exit 1; }
print_plan
gh repo edit "$repo" --description "$description" --enable-issues --add-topic cli --add-topic developer-tools --add-topic ai-agents --add-topic coding-agents --add-topic codex --add-topic agent-skills --add-topic terminal --add-topic logs --add-topic testing --add-topic golang --add-topic privacy --add-topic context-window

create_label() { gh label create "$1" --repo "$repo" --color "$2" --description "$3" --force; }
create_label bug d73a4a 'Unexpected behavior or regression'
create_label enhancement a2eeef 'New capability or improvement'
create_label documentation 0075ca 'Documentation improvement'
create_label security b60205 'Security-related work'
create_label parser 7057ff 'Parser support or parser behavior'
create_label installation 0e8a16 'Installation or packaging work'
create_label 'good first issue' 5319e7 'Approachable contribution for a new contributor'
create_label 'help wanted' 008672 'Maintainer would welcome community help'
create_label 'platform: linux' f9d0c4 'Linux-specific work'
create_label 'platform: macos' f9d0c4 'macOS-specific work'
create_label 'platform: windows' f9d0c4 'Windows-specific work'
create_label 'agent integration' c5def5 'Coding-agent integration'
create_label performance fef2c0 'Performance or benchmarking work'

issue_exists() {
  title=$1
  existing=$(gh issue list --repo "$repo" --state all --search "$title in:title" --json title --jq '.[].title')
  [ "$existing" = "$title" ]
}

create_issue() {
  title=$1
  issue_labels=$2
  context=$3
  if issue_exists "$title"; then printf 'Skipping existing issue: %s\n' "$title"; return; fi
  gh issue create --repo "$repo" --title "$title" --label "$issue_labels" --body "## Context

$context

## Expected behavior

The implementation preserves TokenSave's local-first behavior, documented privacy model, and wrapped-command exit codes.

## Suggested implementation direction

Keep the change focused and add sanitized fixtures when parsing output.

## Acceptance criteria

- The behavior is documented.
- Relevant tests pass on supported CI platforms.
- No secrets, private logs, or proprietary source code are added.

## Testing expectations

Add focused tests and run go test ./...; parser work must include representative sanitized fixtures."
}

create_issue 'Add a PHPStan parser' 'parser,enhancement,good first issue' 'Add detection and useful summaries for PHPStan output.'
create_issue 'Add an ESLint parser' 'parser,enhancement,good first issue' 'Add detection and useful summaries for ESLint output.'
create_issue 'Add a TypeScript parser' 'parser,enhancement,good first issue' 'Add detection and useful summaries for TypeScript compiler output.'
create_issue 'Add a Jest parser' 'parser,enhancement,good first issue' 'Add detection and useful summaries for Jest output.'
create_issue 'Add a Vitest parser' 'parser,enhancement,good first issue' 'Add detection and useful summaries for Vitest output.'
create_issue 'Add a Playwright parser' 'parser,enhancement,good first issue' 'Add detection and useful summaries for Playwright output.'
create_issue 'Add Homebrew packaging' 'installation,enhancement' 'Provide a maintained Homebrew formula or tap strategy for release assets.'
create_issue 'Add Scoop packaging' 'installation,enhancement,platform: windows' 'Provide a maintained Scoop manifest strategy for release assets.'
create_issue 'Expand benchmark scenarios' 'performance,enhancement' 'Add deterministic scenarios that exercise more parsers without external ecosystems.'
create_issue 'Improve Windows signal handling' 'enhancement,platform: windows' 'Review and improve interruption behavior for wrapped processes on Windows.'
