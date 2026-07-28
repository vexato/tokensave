# TokenSave benchmark

One or more mandatory correctness assertions failed; details follow.

## Benchmark metadata

- Benchmark date: 2026-07-28
- TokenSave commit: `ed892f8178665867af865d0bd3eda7287700ab82`
- TokenSave version: development build (no embedded version metadata)
- Operating system: Windows 11
- Architecture: AMD64
- Go version: go version go1.26.5 windows/amd64
- Iterations: 20
- Warm-up iterations: 3
- Configuration: default max_lines=80, max_chars=12000, max_failures=5; line-limit override max_lines=12, max_chars=12000, max_failures=20; character-limit override max_lines=80, max_chars=240, max_failures=20; redaction enabled
- Storage location type: isolated automatically removed temporary directory on the local filesystem

The TokenSave binary was built from the current Go source at the commit above. Benchmark state, fixture shims, raw captures, and TokenSave logs were isolated under an automatically removed temporary directory.

## Correctness summary

| Scenario | Parser | Expected diagnosis retained | Exit code preserved | Redaction passed |
|---|---|---|---|---|
| Generic success | `generic` | Yes | Yes | N/A |
| Generic failure | `generic` | Yes | Yes | N/A |
| PHPUnit failure | `phpunit` | No | Yes | N/A |
| Pest failure | `pest` | No | Yes | N/A |
| npm failure | `node` | No | Yes | N/A |
| Composer failure | `composer` | No | Yes | N/A |
| Git status | `git-status` | Yes | Yes | N/A |
| Git diff | `git-diff` | Yes | Yes | N/A |
| Secret redaction | `generic` | Yes | Yes | Yes |
| Large-output line limit | `generic` | No | Yes | N/A |
| Large-output character limit | `generic` | No | Yes | N/A |

### Failed correctness assertions

- **PHPUnit failure - reported failed tests:** expected `2`; observed `null`.
- **Pest failure - detected assertions:** expected `5`; observed `null`.
- **Pest failure - reported failed tests:** expected `1`; observed `null`.
- **Pest failure - failed test name retained:** expected `"Tests\\\\Feature\\\\InvoiceTest"`; observed `{"failures": [], "last_relevant": []}`.
- **npm failure - npm error code retained:** expected `"ERESOLVE"`; observed `{"failures": [], "last_relevant": []}`.
- **npm failure - npm diagnostic retained:** expected `"unable to resolve dependency tree"`; observed `{"failures": [], "last_relevant": []}`.
- **Composer failure - Composer diagnostic retained:** expected `"could not be resolved to an installable set of packages"`; observed `{"failures": [], "last_relevant": []}`.
- **Composer failure - Composer problem retained:** expected `"Problem 1"`; observed `{"failures": [], "last_relevant": []}`.
- **Large-output line limit - maximum summary lines respected:** expected `"<= 12"`; observed `13`.
- **Large-output character limit - maximum summary characters respected:** expected `"<= 240"`; observed `269`.

## Output reduction

| Scenario | Raw lines | Summary lines | Line reduction | Raw bytes | Summary bytes | Byte reduction |
|---|---:|---:|---:|---:|---:|---:|
| Generic success | 400 | 10 | 97.50% | 16800 | 201 | 98.80% |
| Generic failure | 181 | 20 | 88.95% | 7095 | 444 | 93.74% |
| PHPUnit failure | 7 | 19 | -171.43% | 169 | 386 | -128.40% |
| Pest failure | 3 | 11 | -266.67% | 79 | 210 | -165.82% |
| npm failure | 2 | 12 | -500.00% | 66 | 240 | -263.64% |
| Composer failure | 3 | 12 | -300.00% | 140 | 266 | -90.00% |
| Git status | 4 | 21 | -425.00% | 224 | 322 | -43.75% |
| Git diff | 7 | 19 | -171.43% | 173 | 323 | -86.71% |
| Secret redaction | 4 | 22 | -450.00% | 179 | 475 | -165.36% |
| Large-output line limit | 330 | 13 | 96.06% | 35340 | 235 | 99.34% |
| Large-output character limit | 330 | 14 | 95.76% | 35340 | 269 | 99.24% |

These percentages measure displayed line and byte reduction. Negative values are retained when a summary is larger than the raw output.

## Performance

| Scenario | Raw median | TokenSave median | Absolute overhead | Relative overhead | P95 overhead | Stored bytes |
|---|---:|---:|---:|---:|---:|---:|
| Generic success | 9.816 ms | 36.347 ms | 26.531 ms | 270.27% | 31.545 ms | 16800 |
| PHPUnit failure | 9.378 ms | 25.401 ms | 16.023 ms | 170.85% | 17.582 ms | 169 |
| Git status | 40.388 ms | 54.128 ms | 13.740 ms | 34.02% | 17.998 ms | 224 |

## Methodology

The suite runs without network access. A small temporary Go executable emits an explicitly selected repository fixture and exits with the scenario's deterministic code. Copies are named `phpunit`, `pest`, `npm`, and `composer`, placed first on `PATH`, and invoked with representative arguments so command-based parser detection is exercised. Generic, redaction, and limit fixtures are generated deterministically. Git status and diff use a real temporary repository with local-only test identity, one committed file, one modified tracked file, and one untracked file.

Each correctness scenario is executed raw, through TokenSave's text summary, and through TokenSave JSON. Raw and wrapped exit codes are compared. JSON fields are asserted only where the current `Summary`, `Failure`, and parser structures produce them. Expected fixture values come from repository fixtures and parser tests. Secret checks confirm that deterministic fake values are absent from both displayed text and JSON but byte-for-byte present in the complete local combined log.

Line counts treat a final unterminated line as one line; byte counts use the exact captured byte lengths. Reduction is calculated as `100 x (1 - summary / raw)`, with zero-length raw output handled as zero reduction. Line and byte reduction are not exact token reduction: tokenizers split text according to model-specific vocabularies and are not inferred from bytes or lines.

Performance uses 3 warm-up pairs and 20 measured pairs per scenario with Python's monotonic high-resolution `time.perf_counter_ns` timer. Raw commands write combined output to a local file; TokenSave runs with `--quiet` while retaining its normal stdout, stderr, combined log, metadata, and summary writes. Pair order alternates by iteration. Reported overhead is the difference of medians, relative overhead divides that difference by the raw median, and P95 overhead is the nearest-rank 95th percentile of paired TokenSave-minus-raw durations. Individual samples are retained in the JSON report.

## Limitations

- Fixture-based commands measure TokenSave and local process/logging costs, not the runtime cost of the real third-party tool.
- Results depend on the operating system, process scheduler, filesystem, antivirus, and storage cache.
- Very short outputs may not be reduced and can produce negative reduction percentages.
- Parser behavior may change between TokenSave versions; the commit is therefore recorded.
- Timing measurements on shared CI runners and other contended systems can be noisy.
- The benchmark uses deterministic fixture output and a small temporary Git repository; it does not represent every command or coding-agent workload.

## Reproduction

From the repository root:

```sh
make benchmark
```
