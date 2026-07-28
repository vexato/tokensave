#!/usr/bin/env python3
"""Deterministic, offline TokenSave correctness and performance benchmark."""

from __future__ import annotations

import argparse
import json
import math
import os
import platform
import shutil
import statistics
import subprocess
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


WARMUP_ITERATIONS = 3
MEASURED_ITERATIONS = 20
DEFAULT_MAX_LINES = 80
DEFAULT_MAX_CHARS = 12_000
DEFAULT_MAX_FAILURES = 5
LIMIT_MAX_LINES = 12
LIMIT_MAX_CHARS = 240
LIMIT_MAX_FAILURES = 20


@dataclass
class Check:
    name: str
    passed: bool
    expected: Any
    actual: Any

    def as_dict(self) -> dict[str, Any]:
        return {
            "name": self.name,
            "passed": self.passed,
            "expected": self.expected,
            "actual": self.actual,
        }


@dataclass
class Scenario:
    id: str
    label: str
    command: list[str]
    cwd: Path
    environment: dict[str, str]
    expected_exit_code: int
    expected_parser: str
    expected_status: str
    tokensave_options: list[str] = field(default_factory=list)
    checks: list[Check] = field(default_factory=list)
    diagnosis_check_names: list[str] = field(default_factory=list)
    redaction_check_names: list[str] = field(default_factory=list)
    raw_exit_code: int = 0
    tokensave_exit_code: int = 0
    json_exit_code: int = 0
    raw_data: bytes = b""
    summary_data: bytes = b""
    summary_json: dict[str, Any] = field(default_factory=dict)
    stored_data: bytes = b""


def run(
    command: list[str],
    *,
    cwd: Path,
    environment: dict[str, str],
    stdout: Any = subprocess.PIPE,
    stderr: Any = subprocess.STDOUT,
    check: bool = False,
) -> subprocess.CompletedProcess[bytes]:
    executable = None
    if os.name == "nt" and not Path(command[0]).is_absolute():
        executable = shutil.which(command[0], path=environment.get("PATH"))
    return subprocess.run(
        command,
        cwd=cwd,
        env=environment,
        stdout=stdout,
        stderr=stderr,
        check=check,
        executable=executable,
    )


def command_output(command: list[str], *, cwd: Path, environment: dict[str, str]) -> str:
    result = run(command, cwd=cwd, environment=environment, check=True)
    return result.stdout.decode("utf-8", errors="replace").strip()


def line_count(data: bytes) -> int:
    if not data:
        return 0
    return data.count(b"\n") + (0 if data.endswith(b"\n") else 1)


def reduction(raw: int, summary: int) -> float:
    if raw == 0:
        return 0.0
    return 100.0 * (1.0 - summary / raw)


def add_check(
    scenario: Scenario,
    name: str,
    passed: bool,
    expected: Any,
    actual: Any,
    *,
    diagnosis: bool = False,
    redaction: bool = False,
) -> None:
    scenario.checks.append(Check(name, bool(passed), expected, actual))
    if diagnosis:
        scenario.diagnosis_check_names.append(name)
    if redaction:
        scenario.redaction_check_names.append(name)


def check_value(
    scenario: Scenario,
    name: str,
    actual: Any,
    expected: Any,
    *,
    diagnosis: bool = False,
) -> None:
    add_check(
        scenario,
        name,
        actual == expected,
        expected,
        actual,
        diagnosis=diagnosis,
    )


def contains_diagnostic(summary: dict[str, Any], needle: str) -> bool:
    searchable = {
        "summary": summary.get("summary", {}),
        "failures": summary.get("failures", []),
        "important_paths": summary.get("important_paths", []),
        "last_relevant": summary.get("last_relevant", []),
    }
    return needle.casefold() in json.dumps(searchable, sort_keys=True).casefold()


def create_fixture_shim(temp_dir: Path, environment: dict[str, str]) -> Path:
    source = temp_dir / "fixture-shim.go"
    source.write_text(
        """package main

import (
    "fmt"
    "os"
    "strconv"
)

func main() {
    fixture := os.Getenv("TOKENSAVE_BENCHMARK_FIXTURE")
    data, err := os.ReadFile(fixture)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(125)
    }
    if _, err = os.Stdout.Write(data); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(125)
    }
    code, err := strconv.Atoi(os.Getenv("TOKENSAVE_BENCHMARK_EXIT"))
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(125)
    }
    os.Exit(code)
}
""",
        encoding="utf-8",
        newline="\n",
    )
    suffix = ".exe" if os.name == "nt" else ""
    executable = temp_dir / f"fixture-shim{suffix}"
    run(
        ["go", "build", "-o", str(executable), str(source)],
        cwd=temp_dir,
        environment=environment,
        check=True,
    )
    return executable


def install_shim(source: Path, shim_dir: Path, name: str) -> None:
    suffix = ".exe" if os.name == "nt" else ""
    target = shim_dir / f"{name}{suffix}"
    shutil.copyfile(source, target)
    target.chmod(0o755)


def write_generated_fixtures(temp_dir: Path) -> dict[str, Path]:
    fixture_dir = temp_dir / "generated-fixtures"
    fixture_dir.mkdir(parents=True, exist_ok=True)
    fixtures: dict[str, str] = {
        "generic-success": "".join(
            f"successful deterministic output line {index:04d}\n"
            for index in range(1, 401)
        ),
        "generic-failure": "".join(
            f"routine deterministic output line {index:04d}\n"
            for index in range(1, 181)
        )
        + "fatal: deterministic benchmark failure at internal/benchmark_failure.go:73\n",
        "secret-redaction": (
            "error: Authorization: Bearer benchmark-secret-value\n"
            "error: API_KEY=benchmark-api-key\n"
            "error: password=benchmark-password\n"
            "error: https://user:benchmark-url-password@example.invalid\n"
        ),
        "large-output-limits": "".join(
            f"large deterministic output line {index:04d} "
            + ("x" * 72)
            + "\n"
            for index in range(1, 301)
        )
        + "".join(
            f"error: retained large-output diagnostic {index:02d} "
            f"at internal/large_output_{index:02d}.go:{100 + index}\n"
            for index in range(1, 31)
        ),
    }
    paths: dict[str, Path] = {}
    for name, content in fixtures.items():
        path = fixture_dir / f"{name}.txt"
        path.write_text(content, encoding="utf-8", newline="\n")
        paths[name] = path
    return paths


def fixture_environment(
    base_environment: dict[str, str], fixture: Path, exit_code: int
) -> dict[str, str]:
    environment = base_environment.copy()
    environment["TOKENSAVE_BENCHMARK_FIXTURE"] = str(fixture)
    environment["TOKENSAVE_BENCHMARK_EXIT"] = str(exit_code)
    return environment


def create_git_repository(temp_dir: Path, environment: dict[str, str]) -> Path:
    repository = temp_dir / "git-repository"
    repository.mkdir()
    run(
        ["git", "init", "-b", "benchmark-main"],
        cwd=repository,
        environment=environment,
        check=True,
    )
    run(
        ["git", "config", "user.name", "TokenSave Benchmark"],
        cwd=repository,
        environment=environment,
        check=True,
    )
    run(
        ["git", "config", "user.email", "benchmark@example.invalid"],
        cwd=repository,
        environment=environment,
        check=True,
    )
    tracked = repository / "tracked.txt"
    tracked.write_text("original benchmark content\n", encoding="utf-8", newline="\n")
    run(["git", "add", "tracked.txt"], cwd=repository, environment=environment, check=True)
    run(
        ["git", "commit", "-m", "benchmark fixture"],
        cwd=repository,
        environment=environment,
        check=True,
    )
    tracked.write_text("modified benchmark content\n", encoding="utf-8", newline="\n")
    (repository / "untracked.txt").write_text(
        "untracked benchmark content\n", encoding="utf-8", newline="\n"
    )
    return repository


def execute_scenario(scenario: Scenario, tokensave_bin: Path, temp_dir: Path) -> None:
    scenario_dir = temp_dir / "measurements" / scenario.id
    scenario_dir.mkdir(parents=True, exist_ok=True)

    raw_result = run(
        scenario.command,
        cwd=scenario.cwd,
        environment=scenario.environment,
    )
    scenario.raw_exit_code = raw_result.returncode
    scenario.raw_data = raw_result.stdout

    text_command = [
        str(tokensave_bin),
        *scenario.command,
        *scenario.tokensave_options,
    ]
    text_result = run(
        text_command,
        cwd=scenario.cwd,
        environment=scenario.environment,
    )
    scenario.tokensave_exit_code = text_result.returncode
    scenario.summary_data = text_result.stdout

    json_command = [*text_command, "--json"]
    json_result = run(
        json_command,
        cwd=scenario.cwd,
        environment=scenario.environment,
    )
    scenario.json_exit_code = json_result.returncode
    try:
        scenario.summary_json = json.loads(json_result.stdout.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        scenario.summary_json = {"json_parse_error": str(error)}

    log_path = scenario.summary_json.get("log_path")
    if isinstance(log_path, str):
        combined_log = Path(log_path) / "combined.log"
        if combined_log.is_file():
            scenario.stored_data = combined_log.read_bytes()

    check_value(
        scenario,
        "raw exit code",
        scenario.raw_exit_code,
        scenario.expected_exit_code,
    )
    check_value(
        scenario,
        "TokenSave text exit code",
        scenario.tokensave_exit_code,
        scenario.raw_exit_code,
    )
    check_value(
        scenario,
        "TokenSave JSON exit code",
        scenario.json_exit_code,
        scenario.raw_exit_code,
    )
    check_value(
        scenario,
        "parser",
        scenario.summary_json.get("parser"),
        scenario.expected_parser,
    )
    check_value(
        scenario,
        "status",
        scenario.summary_json.get("status"),
        scenario.expected_status,
    )


def add_scenario_specific_checks(scenario: Scenario) -> None:
    summary = scenario.summary_json
    parsed = summary.get("summary", {})
    failures = summary.get("failures", [])
    paths = summary.get("important_paths", [])

    if scenario.id == "generic-success":
        check_value(
            scenario,
            "reported output line count",
            parsed.get("output_lines"),
            line_count(scenario.raw_data),
            diagnosis=True,
        )
        add_check(
            scenario,
            "line output reduced",
            line_count(scenario.summary_data) < line_count(scenario.raw_data),
            "summary lines < raw lines",
            f"{line_count(scenario.summary_data)} < {line_count(scenario.raw_data)}",
            diagnosis=True,
        )
        add_check(
            scenario,
            "byte output reduced",
            len(scenario.summary_data) < len(scenario.raw_data),
            "summary bytes < raw bytes",
            f"{len(scenario.summary_data)} < {len(scenario.raw_data)}",
            diagnosis=True,
        )

    elif scenario.id == "generic-failure":
        add_check(
            scenario,
            "failure detected",
            len(failures) == 1,
            1,
            len(failures),
            diagnosis=True,
        )
        add_check(
            scenario,
            "failure message retained",
            contains_diagnostic(summary, "deterministic benchmark failure"),
            "deterministic benchmark failure",
            failures,
            diagnosis=True,
        )
        if failures:
            check_value(
                scenario,
                "failure file",
                failures[0].get("file"),
                "internal/benchmark_failure.go",
                diagnosis=True,
            )
            check_value(
                scenario,
                "failure line",
                failures[0].get("line"),
                73,
                diagnosis=True,
            )

    elif scenario.id == "phpunit-failure":
        check_value(scenario, "detected tests", parsed.get("tests"), 4, diagnosis=True)
        check_value(
            scenario, "reported failed tests", parsed.get("failed"), 2, diagnosis=True
        )
        add_check(
            scenario,
            "detailed failure count",
            len(failures) == 1,
            1,
            len(failures),
            diagnosis=True,
        )
        if failures:
            check_value(
                scenario,
                "failure name",
                failures[0].get("name"),
                "UserServiceTest::testCreatesUser",
                diagnosis=True,
            )
            check_value(
                scenario,
                "failure message",
                failures[0].get("message"),
                "Expected status 201, received 500",
                diagnosis=True,
            )
            check_value(
                scenario,
                "failure file",
                failures[0].get("file"),
                "tests/Unit/UserServiceTest.php",
                diagnosis=True,
            )
            check_value(
                scenario,
                "failure line",
                failures[0].get("line"),
                84,
                diagnosis=True,
            )

    elif scenario.id == "pest-failure":
        check_value(scenario, "detected tests", parsed.get("tests"), 3, diagnosis=True)
        check_value(
            scenario, "detected assertions", parsed.get("assertions"), 5, diagnosis=True
        )
        check_value(
            scenario, "reported failed tests", parsed.get("failed"), 1, diagnosis=True
        )
        add_check(
            scenario,
            "failed test name retained",
            contains_diagnostic(summary, r"Tests\\Feature\\InvoiceTest"),
            r"Tests\\Feature\\InvoiceTest",
            {
                "failures": failures,
                "last_relevant": summary.get("last_relevant", []),
            },
            diagnosis=True,
        )

    elif scenario.id == "npm-failure":
        add_check(
            scenario,
            "npm error code retained",
            contains_diagnostic(summary, "ERESOLVE"),
            "ERESOLVE",
            {
                "failures": failures,
                "last_relevant": summary.get("last_relevant", []),
            },
            diagnosis=True,
        )
        add_check(
            scenario,
            "npm diagnostic retained",
            contains_diagnostic(summary, "unable to resolve dependency tree"),
            "unable to resolve dependency tree",
            {
                "failures": failures,
                "last_relevant": summary.get("last_relevant", []),
            },
            diagnosis=True,
        )

    elif scenario.id == "composer-failure":
        add_check(
            scenario,
            "Composer diagnostic retained",
            contains_diagnostic(
                summary,
                "could not be resolved to an installable set of packages",
            ),
            "could not be resolved to an installable set of packages",
            {
                "failures": failures,
                "last_relevant": summary.get("last_relevant", []),
            },
            diagnosis=True,
        )
        add_check(
            scenario,
            "Composer problem retained",
            contains_diagnostic(summary, "Problem 1"),
            "Problem 1",
            {
                "failures": failures,
                "last_relevant": summary.get("last_relevant", []),
            },
            diagnosis=True,
        )

    elif scenario.id == "git-status":
        check_value(
            scenario, "branch", parsed.get("branch"), "benchmark-main", diagnosis=True
        )
        check_value(scenario, "modified paths", parsed.get("modified"), 1, diagnosis=True)
        check_value(scenario, "untracked paths", parsed.get("untracked"), 1, diagnosis=True)
        add_check(
            scenario,
            "modified path retained",
            "tracked.txt" in paths,
            "tracked.txt",
            paths,
            diagnosis=True,
        )
        add_check(
            scenario,
            "untracked path retained",
            "untracked.txt" in paths,
            "untracked.txt",
            paths,
            diagnosis=True,
        )

    elif scenario.id == "git-diff":
        check_value(scenario, "modified files", parsed.get("files"), 1, diagnosis=True)
        check_value(scenario, "insertions", parsed.get("insertions"), 1, diagnosis=True)
        check_value(scenario, "deletions", parsed.get("deletions"), 1, diagnosis=True)
        add_check(
            scenario,
            "modified path retained",
            "tracked.txt" in paths,
            "tracked.txt",
            paths,
            diagnosis=True,
        )
        add_check(
            scenario,
            "diff preview retained",
            contains_diagnostic(summary, "modified benchmark content"),
            "modified benchmark content",
            summary.get("last_relevant", []),
            diagnosis=True,
        )

    elif scenario.id == "secret-redaction":
        secret_values = [
            ("bearer secret", "benchmark-secret-value"),
            ("API key", "benchmark-api-key"),
            ("password assignment", "benchmark-password"),
            ("URL password", "benchmark-url-password"),
        ]
        text_summary = scenario.summary_data.decode("utf-8", errors="replace")
        json_summary = json.dumps(summary, sort_keys=True)
        stored_text = scenario.stored_data.decode("utf-8", errors="replace")
        for label, secret in secret_values:
            add_check(
                scenario,
                f"{label} absent from text summary",
                secret not in text_summary,
                "absent",
                "absent" if secret not in text_summary else "present",
                redaction=True,
            )
            add_check(
                scenario,
                f"{label} absent from JSON summary",
                secret not in json_summary,
                "absent",
                "absent" if secret not in json_summary else "present",
                redaction=True,
            )
            add_check(
                scenario,
                f"{label} retained in complete log",
                secret in stored_text,
                "present",
                "present" if secret in stored_text else "absent",
                redaction=True,
            )

    elif scenario.id == "large-output-line-limit":
        summary_text = scenario.summary_data.decode("utf-8", errors="replace")
        add_check(
            scenario,
            "maximum summary lines respected",
            line_count(scenario.summary_data) <= LIMIT_MAX_LINES,
            f"<= {LIMIT_MAX_LINES}",
            line_count(scenario.summary_data),
            diagnosis=True,
        )
        add_check(
            scenario,
            "truncation reported",
            "[terminal output truncated]" in summary_text,
            "truncation marker present",
            "present"
            if "[terminal output truncated]" in summary_text
            else "absent",
            diagnosis=True,
        )
        add_check(
            scenario,
            "complete log retained",
            scenario.stored_data == scenario.raw_data,
            {
                "bytes": len(scenario.raw_data),
                "content": "identical to raw output",
            },
            {
                "bytes": len(scenario.stored_data),
                "content": "identical"
                if scenario.stored_data == scenario.raw_data
                else "different",
            },
            diagnosis=True,
        )

    elif scenario.id == "large-output-character-limit":
        summary_text = scenario.summary_data.decode("utf-8", errors="replace")
        add_check(
            scenario,
            "maximum summary characters respected",
            len(summary_text) <= LIMIT_MAX_CHARS,
            f"<= {LIMIT_MAX_CHARS}",
            len(summary_text),
            diagnosis=True,
        )
        add_check(
            scenario,
            "truncation reported",
            "[terminal output truncated]" in summary_text,
            "truncation marker present",
            "present"
            if "[terminal output truncated]" in summary_text
            else "absent",
            diagnosis=True,
        )
        add_check(
            scenario,
            "complete log retained",
            scenario.stored_data == scenario.raw_data,
            {
                "bytes": len(scenario.raw_data),
                "content": "identical to raw output",
            },
            {
                "bytes": len(scenario.stored_data),
                "content": "identical"
                if scenario.stored_data == scenario.raw_data
                else "different",
            },
            diagnosis=True,
        )


def nearest_rank_percentile(values: list[float], percentile: float) -> float:
    ordered = sorted(values)
    index = max(0, math.ceil(percentile * len(ordered)) - 1)
    return ordered[index]


def timed_run(
    command: list[str],
    *,
    cwd: Path,
    environment: dict[str, str],
    output_file: Path | None,
) -> tuple[int, float, bytes]:
    started = time.perf_counter_ns()
    if output_file is None:
        result = run(command, cwd=cwd, environment=environment)
        output = result.stdout
    else:
        with output_file.open("wb") as stream:
            result = run(
                command,
                cwd=cwd,
                environment=environment,
                stdout=stream,
            )
        output = b""
    elapsed_ms = (time.perf_counter_ns() - started) / 1_000_000.0
    return result.returncode, elapsed_ms, output


def performance_measurement(
    scenario: Scenario, tokensave_bin: Path, temp_dir: Path
) -> dict[str, Any]:
    perf_dir = temp_dir / "performance" / scenario.id
    perf_dir.mkdir(parents=True, exist_ok=True)
    raw_log = perf_dir / "raw.log"
    tokensave_command = [
        str(tokensave_bin),
        *scenario.command,
        *scenario.tokensave_options,
        "--quiet",
    ]

    for _ in range(WARMUP_ITERATIONS):
        timed_run(
            scenario.command,
            cwd=scenario.cwd,
            environment=scenario.environment,
            output_file=raw_log,
        )
        timed_run(
            tokensave_command,
            cwd=scenario.cwd,
            environment=scenario.environment,
            output_file=None,
        )

    raw_samples: list[float] = []
    tokensave_samples: list[float] = []
    overhead_samples: list[float] = []
    raw_exit_codes: list[int] = []
    tokensave_exit_codes: list[int] = []
    last_run_id = ""

    for iteration in range(MEASURED_ITERATIONS):
        if iteration % 2 == 0:
            raw_code, raw_ms, _ = timed_run(
                scenario.command,
                cwd=scenario.cwd,
                environment=scenario.environment,
                output_file=raw_log,
            )
            tokensave_code, tokensave_ms, output = timed_run(
                tokensave_command,
                cwd=scenario.cwd,
                environment=scenario.environment,
                output_file=None,
            )
        else:
            tokensave_code, tokensave_ms, output = timed_run(
                tokensave_command,
                cwd=scenario.cwd,
                environment=scenario.environment,
                output_file=None,
            )
            raw_code, raw_ms, _ = timed_run(
                scenario.command,
                cwd=scenario.cwd,
                environment=scenario.environment,
                output_file=raw_log,
            )
        raw_samples.append(raw_ms)
        tokensave_samples.append(tokensave_ms)
        overhead_samples.append(tokensave_ms - raw_ms)
        raw_exit_codes.append(raw_code)
        tokensave_exit_codes.append(tokensave_code)
        decoded = output.decode("utf-8", errors="replace").strip()
        if decoded:
            last_run_id = decoded.splitlines()[-1]

    raw_median = statistics.median(raw_samples)
    tokensave_median = statistics.median(tokensave_samples)
    absolute_overhead = tokensave_median - raw_median
    relative_overhead = (
        (absolute_overhead / raw_median) * 100.0 if raw_median else 0.0
    )
    stored_bytes = 0
    if last_run_id:
        stored_log = (
            Path(scenario.environment["TOKENSAVE_HOME"])
            / "runs"
            / last_run_id
            / "combined.log"
        )
        if stored_log.is_file():
            stored_bytes = stored_log.stat().st_size

    return {
        "scenario_id": scenario.id,
        "scenario": scenario.label,
        "warmup_iterations": WARMUP_ITERATIONS,
        "measured_iterations": MEASURED_ITERATIONS,
        "timer": "Python time.perf_counter_ns",
        "raw_samples_ms": [round(value, 6) for value in raw_samples],
        "tokensave_samples_ms": [
            round(value, 6) for value in tokensave_samples
        ],
        "paired_overhead_samples_ms": [
            round(value, 6) for value in overhead_samples
        ],
        "raw_median_ms": round(raw_median, 6),
        "tokensave_median_ms": round(tokensave_median, 6),
        "absolute_overhead_ms": round(absolute_overhead, 6),
        "relative_overhead_percent": round(relative_overhead, 6),
        "p95_overhead_ms": round(
            nearest_rank_percentile(overhead_samples, 0.95), 6
        ),
        "stored_log_bytes": stored_bytes,
        "raw_exit_codes": raw_exit_codes,
        "tokensave_exit_codes": tokensave_exit_codes,
        "exit_codes_preserved": all(
            raw_code == tokensave_code == scenario.expected_exit_code
            for raw_code, tokensave_code in zip(
                raw_exit_codes, tokensave_exit_codes
            )
        ),
    }


def scenario_to_dict(scenario: Scenario) -> dict[str, Any]:
    raw_lines = line_count(scenario.raw_data)
    summary_lines = line_count(scenario.summary_data)
    raw_bytes = len(scenario.raw_data)
    summary_bytes = len(scenario.summary_data)
    check_map = {check.name: check.passed for check in scenario.checks}
    diagnosis_retained = all(
        check_map[name] for name in scenario.diagnosis_check_names
    )
    redaction_passed: bool | None = None
    if scenario.redaction_check_names:
        redaction_passed = all(
            check_map[name] for name in scenario.redaction_check_names
        )
    exit_code_preserved = (
        scenario.raw_exit_code
        == scenario.tokensave_exit_code
        == scenario.json_exit_code
        == scenario.expected_exit_code
    )
    return {
        "id": scenario.id,
        "name": scenario.label,
        "command": scenario.command,
        "expected_parser": scenario.expected_parser,
        "parser": scenario.summary_json.get("parser"),
        "expected_status": scenario.expected_status,
        "status": scenario.summary_json.get("status"),
        "expected_exit_code": scenario.expected_exit_code,
        "raw_exit_code": scenario.raw_exit_code,
        "tokensave_exit_code": scenario.tokensave_exit_code,
        "json_exit_code": scenario.json_exit_code,
        "raw": {
            "lines": raw_lines,
            "bytes": raw_bytes,
        },
        "summary": {
            "lines": summary_lines,
            "bytes": summary_bytes,
        },
        "reduction": {
            "line_percent": round(reduction(raw_lines, summary_lines), 6),
            "byte_percent": round(reduction(raw_bytes, summary_bytes), 6),
        },
        "stored_log_bytes": len(scenario.stored_data),
        "correctness": {
            "passed": all(check.passed for check in scenario.checks),
            "expected_diagnosis_retained": diagnosis_retained,
            "exit_code_preserved": exit_code_preserved,
            "redaction_passed": redaction_passed,
            "checks": [check.as_dict() for check in scenario.checks],
        },
    }


def format_percent(value: float) -> str:
    return f"{value:.2f}%"


def yes_no(value: bool | None) -> str:
    if value is None:
        return "N/A"
    return "Yes" if value else "No"


def markdown_report(report: dict[str, Any]) -> str:
    metadata = report["metadata"]
    scenarios = report["scenarios"]
    performance = report["performance"]
    status = (
        "All mandatory correctness assertions passed."
        if report["all_mandatory_correctness_passed"]
        else "One or more mandatory correctness assertions failed; details follow."
    )
    lines = [
        "# TokenSave benchmark",
        "",
        status,
        "",
        "## Benchmark metadata",
        "",
        f"- Benchmark date: {metadata['benchmark_date']}",
        f"- TokenSave commit: `{metadata['tokensave_commit']}`",
        f"- TokenSave version: {metadata['tokensave_version']}",
        f"- Operating system: {metadata['operating_system']}",
        f"- Architecture: {metadata['architecture']}",
        f"- Go version: {metadata['go_version']}",
        f"- Iterations: {metadata['iterations']}",
        f"- Warm-up iterations: {metadata['warmup_iterations']}",
        f"- Configuration: {metadata['configuration']}",
        f"- Storage location type: {metadata['storage_location_type']}",
        "",
        "The TokenSave binary was built from the current Go source at the commit above. "
        "Benchmark state, fixture shims, raw captures, and TokenSave logs were isolated "
        "under an automatically removed temporary directory.",
        "",
        "## Correctness summary",
        "",
        "| Scenario | Parser | Expected diagnosis retained | Exit code preserved | Redaction passed |",
        "|---|---|---|---|---|",
    ]
    for scenario in scenarios:
        correctness = scenario["correctness"]
        parser = scenario["parser"] or "unavailable"
        lines.append(
            f"| {scenario['name']} | `{parser}` | "
            f"{yes_no(correctness['expected_diagnosis_retained'])} | "
            f"{yes_no(correctness['exit_code_preserved'])} | "
            f"{yes_no(correctness['redaction_passed'])} |"
        )

    failed_checks = [
        (scenario["name"], check)
        for scenario in scenarios
        for check in scenario["correctness"]["checks"]
        if not check["passed"]
    ]
    if failed_checks:
        lines.extend(["", "### Failed correctness assertions", ""])
        for scenario_name, check in failed_checks:
            expected = json.dumps(check["expected"], ensure_ascii=False, sort_keys=True)
            actual = json.dumps(check["actual"], ensure_ascii=False, sort_keys=True)
            lines.append(
                f"- **{scenario_name} - {check['name']}:** expected `{expected}`; "
                f"observed `{actual}`."
            )

    lines.extend(
        [
            "",
            "## Output reduction",
            "",
            "| Scenario | Raw lines | Summary lines | Line reduction | Raw bytes | Summary bytes | Byte reduction |",
            "|---|---:|---:|---:|---:|---:|---:|",
        ]
    )
    for scenario in scenarios:
        lines.append(
            f"| {scenario['name']} | {scenario['raw']['lines']} | "
            f"{scenario['summary']['lines']} | "
            f"{format_percent(scenario['reduction']['line_percent'])} | "
            f"{scenario['raw']['bytes']} | {scenario['summary']['bytes']} | "
            f"{format_percent(scenario['reduction']['byte_percent'])} |"
        )

    lines.extend(
        [
            "",
            "These percentages measure displayed line and byte reduction. Negative "
            "values are retained when a summary is larger than the raw output.",
            "",
            "## Performance",
            "",
            "| Scenario | Raw median | TokenSave median | Absolute overhead | Relative overhead | P95 overhead | Stored bytes |",
            "|---|---:|---:|---:|---:|---:|---:|",
        ]
    )
    for measurement in performance:
        lines.append(
            f"| {measurement['scenario']} | "
            f"{measurement['raw_median_ms']:.3f} ms | "
            f"{measurement['tokensave_median_ms']:.3f} ms | "
            f"{measurement['absolute_overhead_ms']:.3f} ms | "
            f"{measurement['relative_overhead_percent']:.2f}% | "
            f"{measurement['p95_overhead_ms']:.3f} ms | "
            f"{measurement['stored_log_bytes']} |"
        )

    lines.extend(
        [
            "",
            "## Methodology",
            "",
            "The suite runs without network access. A small temporary Go executable "
            "emits an explicitly selected repository fixture and exits with the "
            "scenario's deterministic code. Copies are named `phpunit`, `pest`, "
            "`npm`, and `composer`, placed first on `PATH`, and invoked with "
            "representative arguments so command-based parser detection is exercised. "
            "Generic, redaction, and limit fixtures are generated deterministically. "
            "Git status and diff use a real temporary repository with local-only test "
            "identity, one committed file, one modified tracked file, and one untracked file.",
            "",
            "Each correctness scenario is executed raw, through TokenSave's text "
            "summary, and through TokenSave JSON. Raw and wrapped exit codes are "
            "compared. JSON fields are asserted only where the current `Summary`, "
            "`Failure`, and parser structures produce them. Expected fixture values "
            "come from repository fixtures and parser tests. Secret checks confirm "
            "that deterministic fake values are absent from both displayed text and "
            "JSON but byte-for-byte present in the complete local combined log.",
            "",
            "Line counts treat a final unterminated line as one line; byte counts use "
            "the exact captured byte lengths. Reduction is calculated as "
            "`100 x (1 - summary / raw)`, with zero-length raw output handled as "
            "zero reduction. Line and byte reduction are not exact token reduction: "
            "tokenizers split text according to model-specific vocabularies and are "
            "not inferred from bytes or lines.",
            "",
            f"Performance uses {WARMUP_ITERATIONS} warm-up pairs and "
            f"{MEASURED_ITERATIONS} measured pairs per scenario with Python's "
            "monotonic high-resolution `time.perf_counter_ns` timer. Raw commands "
            "write combined output to a local file; TokenSave runs with `--quiet` "
            "while retaining its normal stdout, stderr, combined log, metadata, and "
            "summary writes. Pair order alternates by iteration. Reported overhead "
            "is the difference of medians, relative overhead divides that difference "
            "by the raw median, and P95 overhead is the nearest-rank 95th percentile "
            "of paired TokenSave-minus-raw durations. Individual samples are retained "
            "in the JSON report.",
            "",
            "## Limitations",
            "",
            "- Fixture-based commands measure TokenSave and local process/logging costs, not the runtime cost of the real third-party tool.",
            "- Results depend on the operating system, process scheduler, filesystem, antivirus, and storage cache.",
            "- Very short outputs may not be reduced and can produce negative reduction percentages.",
            "- Parser behavior may change between TokenSave versions; the commit is therefore recorded.",
            "- Timing measurements on shared CI runners and other contended systems can be noisy.",
            "- The benchmark uses deterministic fixture output and a small temporary Git repository; it does not represent every command or coding-agent workload.",
            "",
            "## Reproduction",
            "",
            "From the repository root:",
            "",
            "```sh",
            "make benchmark",
            "```",
            "",
        ]
    )
    return "\n".join(lines)


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--tokensave-bin", required=True, type=Path)
    parser.add_argument("--temp-dir", required=True, type=Path)
    parser.add_argument("--markdown", required=True, type=Path)
    parser.add_argument("--json", required=True, type=Path)
    return parser.parse_args()


def main() -> int:
    arguments = parse_arguments()
    repo_root = Path(__file__).resolve().parent.parent
    temp_dir = arguments.temp_dir.resolve()
    tokensave_bin = arguments.tokensave_bin.resolve()
    markdown_path = arguments.markdown.resolve()
    json_path = arguments.json.resolve()

    base_environment = os.environ.copy()
    base_environment["TOKENSAVE_HOME"] = str(temp_dir / "state")
    base_environment["GIT_CONFIG_NOSYSTEM"] = "1"
    base_environment["GIT_TERMINAL_PROMPT"] = "0"

    generated = write_generated_fixtures(temp_dir)
    shim_dir = temp_dir / "bin"
    shim_dir.mkdir()
    shim_source = create_fixture_shim(temp_dir, base_environment)
    for name in [
        "benchmark-success",
        "benchmark-failure",
        "phpunit",
        "pest",
        "npm",
        "composer",
        "benchmark-secrets",
        "benchmark-large-output",
    ]:
        install_shim(shim_source, shim_dir, name)
    base_environment["PATH"] = str(shim_dir) + os.pathsep + base_environment["PATH"]

    git_repository = create_git_repository(temp_dir, base_environment)
    fixture_dir = repo_root / "fixtures"
    scenarios = [
        Scenario(
            "generic-success",
            "Generic success",
            ["benchmark-success", "--emit-large"],
            repo_root,
            fixture_environment(base_environment, generated["generic-success"], 0),
            0,
            "generic",
            "succeeded",
        ),
        Scenario(
            "generic-failure",
            "Generic failure",
            ["benchmark-failure", "--mode", "fail"],
            repo_root,
            fixture_environment(base_environment, generated["generic-failure"], 7),
            7,
            "generic",
            "failed",
        ),
        Scenario(
            "phpunit-failure",
            "PHPUnit failure",
            ["phpunit", "--colors=never"],
            repo_root,
            fixture_environment(
                base_environment, fixture_dir / "phpunit-failures.txt", 1
            ),
            1,
            "phpunit",
            "failed",
        ),
        Scenario(
            "pest-failure",
            "Pest failure",
            ["pest", "--colors=never"],
            repo_root,
            fixture_environment(base_environment, fixture_dir / "pest.txt", 1),
            1,
            "pest",
            "failed",
        ),
        Scenario(
            "npm-failure",
            "npm failure",
            ["npm", "install", "--offline"],
            repo_root,
            fixture_environment(
                base_environment, fixture_dir / "npm-failure.txt", 1
            ),
            1,
            "node",
            "failed",
        ),
        Scenario(
            "composer-failure",
            "Composer failure",
            ["composer", "install", "--no-interaction", "--no-progress"],
            repo_root,
            fixture_environment(
                base_environment, fixture_dir / "composer-failure.txt", 2
            ),
            2,
            "composer",
            "failed",
        ),
        Scenario(
            "git-status",
            "Git status",
            ["git", "status", "--porcelain=v2", "--branch"],
            git_repository,
            base_environment.copy(),
            0,
            "git-status",
            "succeeded",
        ),
        Scenario(
            "git-diff",
            "Git diff",
            ["git", "diff"],
            git_repository,
            base_environment.copy(),
            0,
            "git-diff",
            "succeeded",
        ),
        Scenario(
            "secret-redaction",
            "Secret redaction",
            ["benchmark-secrets", "--fake-values-only"],
            repo_root,
            fixture_environment(base_environment, generated["secret-redaction"], 7),
            7,
            "generic",
            "failed",
        ),
        Scenario(
            "large-output-line-limit",
            "Large-output line limit",
            ["benchmark-large-output", "--deterministic"],
            repo_root,
            fixture_environment(
                base_environment, generated["large-output-limits"], 7
            ),
            7,
            "generic",
            "failed",
            [
                "--max-lines",
                str(LIMIT_MAX_LINES),
                "--max-chars",
                str(DEFAULT_MAX_CHARS),
                "--max-failures",
                str(LIMIT_MAX_FAILURES),
            ],
        ),
        Scenario(
            "large-output-character-limit",
            "Large-output character limit",
            ["benchmark-large-output", "--deterministic"],
            repo_root,
            fixture_environment(
                base_environment, generated["large-output-limits"], 7
            ),
            7,
            "generic",
            "failed",
            [
                "--max-lines",
                str(DEFAULT_MAX_LINES),
                "--max-chars",
                str(LIMIT_MAX_CHARS),
                "--max-failures",
                str(LIMIT_MAX_FAILURES),
            ],
        ),
    ]

    for scenario in scenarios:
        execute_scenario(scenario, tokensave_bin, temp_dir)
        add_scenario_specific_checks(scenario)

    performance_scenarios = {
        "generic-success",
        "phpunit-failure",
        "git-status",
    }
    performance = [
        performance_measurement(scenario, tokensave_bin, temp_dir)
        for scenario in scenarios
        if scenario.id in performance_scenarios
    ]

    scenario_results = [scenario_to_dict(scenario) for scenario in scenarios]
    for measurement in performance:
        if not measurement["exit_codes_preserved"]:
            matching = next(
                scenario
                for scenario in scenario_results
                if scenario["id"] == measurement["scenario_id"]
            )
            matching["correctness"]["passed"] = False
            matching["correctness"]["checks"].append(
                {
                    "name": "performance iteration exit codes",
                    "passed": False,
                    "expected": matching["expected_exit_code"],
                    "actual": {
                        "raw": measurement["raw_exit_codes"],
                        "tokensave": measurement["tokensave_exit_codes"],
                    },
                }
            )

    commit = command_output(
        ["git", "-c", f"safe.directory={repo_root}", "rev-parse", "HEAD"],
        cwd=repo_root,
        environment=base_environment,
    )
    go_version = command_output(
        ["go", "version"], cwd=repo_root, environment=base_environment
    )
    operating_system = f"{platform.system()} {platform.release()}"
    report: dict[str, Any] = {
        "schema_version": 1,
        "metadata": {
            "benchmark_date": datetime.now(timezone.utc).date().isoformat(),
            "tokensave_commit": commit,
            "tokensave_version": "development build (no embedded version metadata)",
            "operating_system": operating_system,
            "architecture": platform.machine(),
            "go_version": go_version,
            "iterations": MEASURED_ITERATIONS,
            "warmup_iterations": WARMUP_ITERATIONS,
            "configuration": (
                f"default max_lines={DEFAULT_MAX_LINES}, "
                f"max_chars={DEFAULT_MAX_CHARS}, "
                f"max_failures={DEFAULT_MAX_FAILURES}; "
                f"line-limit override max_lines={LIMIT_MAX_LINES}, "
                f"max_chars={DEFAULT_MAX_CHARS}, "
                f"max_failures={LIMIT_MAX_FAILURES}; "
                f"character-limit override max_lines={DEFAULT_MAX_LINES}, "
                f"max_chars={LIMIT_MAX_CHARS}, "
                f"max_failures={LIMIT_MAX_FAILURES}; redaction enabled"
            ),
            "storage_location_type": (
                "isolated automatically removed temporary directory on the local filesystem"
            ),
            "timer": "Python time.perf_counter_ns",
        },
        "scenarios": scenario_results,
        "performance": performance,
    }
    report["all_mandatory_correctness_passed"] = all(
        scenario["correctness"]["passed"] for scenario in scenario_results
    )

    markdown_path.parent.mkdir(parents=True, exist_ok=True)
    json_path.parent.mkdir(parents=True, exist_ok=True)
    json_path.write_text(
        json.dumps(report, indent=2, sort_keys=True, ensure_ascii=False) + "\n",
        encoding="utf-8",
        newline="\n",
    )
    markdown_path.write_text(
        markdown_report(report),
        encoding="utf-8",
        newline="\n",
    )

    print(f"Markdown report: {markdown_path}")
    print(f"JSON report: {json_path}")
    if report["all_mandatory_correctness_passed"]:
        print("All mandatory correctness assertions passed.")
        return 0
    print("One or more mandatory correctness assertions failed.", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
