package tokensave

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func Render(s Summary, m Metadata, limits Limits) string {
	var b strings.Builder
	status := "succeeded"
	if m.ExitCode != 0 {
		status = "failed"
	}
	fmt.Fprintf(&b, "TokenSave: command %s\nRun: %s\n", status, m.ID)
	if m.ExitCode != 0 {
		fmt.Fprintf(&b, "Command: %s\nExit code: %d\n", strings.Join(m.Command, " "), m.ExitCode)
	}
	fmt.Fprintf(&b, "Duration: %s\n\n", duration(m.DurationMS))
	switch s.Parser {
	case "git-status":
		renderGitStatus(&b, s)
	case "git-diff":
		renderGitDiff(&b, s)
	case "phpunit", "pest":
		renderTests(&b, s)
	default:
		renderGeneric(&b, s)
	}
	if len(s.Failures) > 0 {
		b.WriteString("\nFailures:\n")
		for _, f := range s.Failures[:min(len(s.Failures), limits.MaxFailures)] {
			fmt.Fprintf(&b, "%d. %s\n   %s\n", f.Index, f.Name, f.Message)
			if f.File != "" {
				fmt.Fprintf(&b, "   %s", f.File)
				if f.Line > 0 {
					fmt.Fprintf(&b, ":%d", f.Line)
				}
				b.WriteByte('\n')
			}
		}
	}
	if len(s.ImportantPaths) > 0 {
		b.WriteString("\nImportant paths:\n")
		for _, p := range s.ImportantPaths[:min(len(s.ImportantPaths), 8)] {
			fmt.Fprintf(&b, "- %s\n", p)
		}
	}
	if len(s.LastRelevant) > 0 && len(s.Failures) == 0 {
		b.WriteString("\nLast relevant lines:\n")
		for _, x := range s.LastRelevant {
			fmt.Fprintf(&b, "%s\n", x)
		}
	}
	b.WriteString("\nFull log saved locally.\n")
	if len(s.Failures) > 0 {
		fmt.Fprintf(&b, "Inspect: tokensave show %s --failure 1\n", m.ID)
	} else {
		fmt.Fprintf(&b, "Inspect: tokensave show %s --tail 100\n", m.ID)
	}
	return clip(b.String(), limits)
}
func value(a map[string]any, k string) any {
	if x, ok := a[k]; ok {
		return x
	}
	return 0
}
func renderGitStatus(b *strings.Builder, s Summary) {
	fmt.Fprintln(b, "Git status")
	if x, ok := s.Summary["branch"]; ok {
		fmt.Fprintf(b, "Branch: %v\n", x)
	}
	for _, p := range [][2]string{{"ahead", "Ahead"}, {"behind", "Behind"}, {"modified", "Modified"}, {"added", "Added"}, {"deleted", "Deleted"}, {"renamed", "Renamed"}, {"untracked", "Untracked"}, {"conflicts", "Conflicts"}} {
		fmt.Fprintf(b, "%s: %v\n", p[1], value(s.Summary, p[0]))
	}
}
func renderGitDiff(b *strings.Builder, s Summary) {
	fmt.Fprintln(b, "Git diff")
	fmt.Fprintf(b, "Files: %v\nInsertions: %v\nDeletions: %v\nBinary files: %v\n", value(s.Summary, "files"), value(s.Summary, "insertions"), value(s.Summary, "deletions"), value(s.Summary, "binary_files"))
}
func renderTests(b *strings.Builder, s Summary) {
	fmt.Fprintln(b, "PHP tests")
	for _, p := range [][2]string{{"tests", "Tests"}, {"assertions", "Assertions"}, {"passed", "Passed"}, {"failed", "Failed"}, {"errors", "Errors"}, {"skipped", "Skipped"}} {
		if _, ok := s.Summary[p[0]]; ok {
			fmt.Fprintf(b, "%s: %v\n", p[1], s.Summary[p[0]])
		}
	}
}
func renderGeneric(b *strings.Builder, s Summary) {
	fmt.Fprintf(b, "Output: %v lines\nDetected errors: %v\nWarnings: %v\n", value(s.Summary, "output_lines"), value(s.Summary, "errors_detected"), value(s.Summary, "warnings_detected"))
}
func duration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return (time.Duration(ms) * time.Millisecond).Round(100 * time.Millisecond).String()
}
func clip(s string, l Limits) string {
	lines := strings.Split(s, "\n")
	if len(lines) > l.MaxLines {
		lines = append(lines[:l.MaxLines], "[terminal output truncated]")
	}
	s = strings.Join(lines, "\n")
	if len(s) > l.MaxChars {
		return s[:l.MaxChars] + "\n[terminal output truncated]\n"
	}
	return s
}
func SummaryJSON(s Summary) string { b, _ := json.Marshal(s); return string(bytes.TrimSpace(b)) }
