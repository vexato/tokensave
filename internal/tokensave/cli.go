package tokensave

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Execute(args []string, out, errout io.Writer) error {
	if len(args) == 0 {
		usage(out)
		return nil
	}
	switch args[0] {
	case "show":
		return show(args[1:], out)
	case "list":
		return list(args[1:], out)
	case "clean":
		return clean(args[1:], out)
	case "path":
		return pathCmd(args[1:], out)
	case "help", "--help", "-h":
		usage(out)
		return nil
	}
	return run(args, out)
}
func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: tokensave [run] <command> [arguments...]\n       tokensave show|list|clean|path ...")
}
func run(args []string, out io.Writer) error {
	if len(args) > 0 && args[0] == "run" {
		args = args[1:]
	}
	c := LoadConfig()
	l := c.Limits()
	jsonOut, quiet, noSummary := false, false, false
	shell := ""
	cmd := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			cmd = append(cmd, args[i+1:]...)
			break
		}
		switch a {
		case "--json":
			jsonOut = true
		case "--quiet":
			quiet = true
		case "--no-summary":
			noSummary = true
		case "--shell":
			if i+1 >= len(args) {
				return fmt.Errorf("--shell requires a command")
			}
			i++
			shell = args[i]
		case "--max-lines", "--max-chars", "--max-failures":
			if i+1 >= len(args) {
				return fmt.Errorf("%s requires a value", a)
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil {
				return e
			}
			if a == "--max-lines" {
				l.MaxLines = n
			}
			if a == "--max-chars" {
				l.MaxChars = n
			}
			if a == "--max-failures" {
				l.MaxFailures = n
			}
		default:
			cmd = append(cmd, a)
		}
	}
	m, s, e := ExecuteRun(RunRequest{Command: cmd, Shell: shell}, c)
	if e != nil {
		return e
	}
	if len(s.Failures) > l.MaxFailures {
		s.Failures = s.Failures[:l.MaxFailures]
	}
	if jsonOut {
		fmt.Fprintln(out, SummaryJSON(s))
	} else if quiet {
		fmt.Fprintln(out, m.ID)
	} else if !noSummary {
		fmt.Fprint(out, Render(s, m, l))
	}
	if m.ExitCode != 0 {
		return exitCodeError(m.ExitCode)
	}
	return nil
}

type exitCodeError int

func (e exitCodeError) Error() string     { return "child command failed" }
func ChildExitCode(err error) (int, bool) { e, ok := err.(exitCodeError); return int(e), ok }
func show(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("show requires a run id")
	}
	id := args[0]
	m, d, e := ReadMetadata(id)
	if e != nil {
		return e
	}
	tail, head := 0, 0
	stream := ""
	lines := ""
	failure := 0
	full := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--stdout":
			stream = "stdout.log"
		case "--stderr":
			stream = "stderr.log"
		case "--full":
			full = true
		case "--tail":
			i++
			tail, _ = strconv.Atoi(args[i])
		case "--head":
			i++
			head, _ = strconv.Atoi(args[i])
		case "--lines":
			i++
			lines = args[i]
		case "--failure":
			i++
			failure, _ = strconv.Atoi(args[i])
		}
	}
	if failure > 0 {
		b, e := os.ReadFile(filepath.Join(d, "summary.json"))
		if e != nil {
			return e
		}
		var s Summary
		if e = json.Unmarshal(b, &s); e != nil {
			return e
		}
		if failure > len(s.Failures) || failure < 1 {
			return fmt.Errorf("failure %d does not exist", failure)
		}
		f := s.Failures[failure-1]
		fmt.Fprintf(out, "%d. %s\n%s\n", f.Index, f.Name, f.Message)
		if f.File != "" {
			fmt.Fprintf(out, "%s:%d\n", f.File, f.Line)
		}
		return nil
	}
	if stream == "" && !full && tail == 0 && head == 0 && lines == "" {
		b, e := os.ReadFile(filepath.Join(d, "summary.txt"))
		if e == nil {
			_, e = out.Write(b)
			return e
		}
		fallback, e := readSummaryFallback(m, d)
		if e != nil {
			return e
		}
		_, e = fmt.Fprint(out, fallback)
		return e
	}
	if stream != "" && !full && tail == 0 && head == 0 && lines == "" {
		tail = 100
	}
	if stream == "" {
		stream = "combined.log"
	}
	return printSelection(filepath.Join(d, stream), tail, head, lines, full, out)
}
func readSummaryFallback(m Metadata, d string) (string, error) {
	b, e := os.ReadFile(filepath.Join(d, "summary.json"))
	if e != nil {
		return "", e
	}
	var s Summary
	if e = json.Unmarshal(b, &s); e != nil {
		return "", e
	}
	return Render(s, m, DefaultLimits()), nil
}
func printSelection(path string, tail, head int, rng string, full bool, out io.Writer) error {
	lines := readLines(path)
	start, end := 0, len(lines)
	if tail > 0 && tail < len(lines) {
		start = len(lines) - tail
	}
	if head > 0 && head < end {
		end = head
	}
	if rng != "" {
		p := strings.SplitN(rng, ":", 2)
		if len(p) != 2 {
			return fmt.Errorf("--lines must be A:B")
		}
		start, _ = strconv.Atoi(p[0])
		end, _ = strconv.Atoi(p[1])
		if start < 1 {
			start = 1
		}
		start--
		if end > len(lines) {
			end = len(lines)
		}
	}
	if !full && tail == 0 && head == 0 && rng == "" {
		return fmt.Errorf("use --full to print a complete log")
	}
	for _, x := range lines[start:end] {
		fmt.Fprintln(out, x)
	}
	return nil
}
func list(args []string, out io.Writer) error {
	limit := 0
	failed := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--failed" {
			failed = true
		}
		if args[i] == "--limit" && i+1 < len(args) {
			i++
			limit, _ = strconv.Atoi(args[i])
		}
	}
	runs, e := ListRuns()
	if e != nil {
		return e
	}
	n := 0
	for _, m := range runs {
		if failed && m.ExitCode == 0 {
			continue
		}
		fmt.Fprintf(out, "%s  %s  exit=%d  %s  %s\n", m.ID, m.StartedAt.Local().Format(time.RFC3339), m.ExitCode, duration(m.DurationMS), strings.Join(m.Command, " "))
		n++
		if limit > 0 && n >= limit {
			break
		}
	}
	return nil
}
func pathCmd(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("path requires a run id")
	}
	d, e := RunDir(args[0])
	if e == nil {
		fmt.Fprintln(out, d)
	}
	return e
}
func clean(args []string, out io.Writer) error {
	all := false
	keep := -1
	older := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			all = true
		case "--keep":
			i++
			keep, _ = strconv.Atoi(args[i])
		case "--older-than":
			i++
			older = args[i]
		}
	}
	if (all && (keep >= 0 || older != "")) || (keep >= 0 && older != "") {
		return fmt.Errorf("choose one clean mode")
	}
	runs, e := ListRuns()
	if e != nil {
		return e
	}
	cut := time.Time{}
	if older != "" {
		d, e := time.ParseDuration(older)
		if e != nil {
			return fmt.Errorf("invalid duration: %w", e)
		}
		cut = time.Now().Add(-d)
	}
	deleted := 0
	for i, m := range runs {
		remove := all || (keep >= 0 && i >= keep) || (!cut.IsZero() && m.StartedAt.Before(cut))
		if remove {
			d, _ := RunDir(m.ID)
			if e := os.RemoveAll(d); e != nil {
				return e
			}
			deleted++
		}
	}
	fmt.Fprintf(out, "Removed %d run(s).\n", deleted)
	return nil
}
