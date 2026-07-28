package tokensave

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Parser interface {
	Name() string
	Detect([]string, string) bool
	Parse([]string) (map[string]any, []Failure, []string, []string)
}

func Analyze(m Metadata, root string, c Config) Summary {
	lines := readLines(filepath.Join(root, "combined.log"))
	forced := c.Commands[strings.Join(m.Command, " ")]
	parsers := []Parser{gitStatusParser{}, gitDiffParser{}, phpunitParser{}, pestParser{}, composerParser{}, nodeParser{}}
	for _, p := range parsers {
		if forced == p.Name() || (forced == "" && p.Detect(m.Command, strings.Join(lines[:min(len(lines), 30)], "\n"))) {
			a, f, paths, last := p.Parse(lines)
			return redactSummary(Summary{Parser: p.Name(), Summary: a, Failures: f, ImportantPaths: paths, LastRelevant: last}, NewRedactor(c))
		}
	}
	a, f, paths, last := genericParse(lines)
	return redactSummary(Summary{Parser: "generic", Summary: a, Failures: f, ImportantPaths: paths, LastRelevant: last}, NewRedactor(c))
}
func readLines(path string) []string {
	f, e := os.Open(path)
	if e != nil {
		return nil
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 4*1024*1024)
	out := []string{}
	for s.Scan() {
		out = append(out, s.Text())
	}
	return out
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var importantPattern = regexp.MustCompile(`(?i)([A-Za-z0-9_./\\-]+\.(?:go|php|ts|tsx|js|jsx|json|ya?ml|css|md|py|java|rb))(?:[: ]+line )?:(\d+)|([A-Za-z0-9_./\\-]+\.(?:go|php|ts|tsx|js|jsx|json|ya?ml|css|md|py|java|rb)):(\d+)`)
var errorPattern = regexp.MustCompile(`(?i)\b(error|failed|failure|fatal|exception|warning|cannot find|not found|panic)\b`)

func findLocation(line string) (string, int) {
	m := importantPattern.FindStringSubmatch(line)
	if len(m) == 0 {
		return "", 0
	}
	p := m[1]
	n := m[2]
	if p == "" {
		p = m[3]
		n = m[4]
	}
	i, _ := strconv.Atoi(n)
	return p, i
}
func unique(in []string, n int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range in {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
		if len(out) >= n {
			break
		}
	}
	return out
}
func genericParse(lines []string) (map[string]any, []Failure, []string, []string) {
	paths := []string{}
	f := []Failure{}
	relevant := []string{}
	seen := map[string]bool{}
	warnings := 0
	for i, l := range lines {
		if p, _ := findLocation(l); p != "" {
			paths = append(paths, p)
		}
		if errorPattern.MatchString(l) {
			if strings.Contains(strings.ToLower(l), "warning") {
				warnings++
			}
			key := strings.TrimSpace(l)
			relevant = append(relevant, key)
			if !seen[key] {
				seen[key] = true
				ctx := []string{}
				for j := i + 1; j < len(lines) && j <= i+2; j++ {
					if strings.TrimSpace(lines[j]) != "" {
						ctx = append(ctx, lines[j])
					}
				}
				p, n := findLocation(l)
				f = append(f, Failure{Index: len(f) + 1, Name: "Detected error", Message: key, File: p, Line: n, Context: ctx})
			}
		}
	}
	last := relevant
	if len(last) > 8 {
		last = last[len(last)-8:]
	}
	return map[string]any{"output_lines": len(lines), "errors_detected": len(f), "warnings_detected": warnings}, f, unique(paths, 8), unique(last, 8)
}

type gitStatusParser struct{}

func (gitStatusParser) Name() string { return "git-status" }
func (gitStatusParser) Detect(cmd []string, _ string) bool {
	return len(cmd) >= 2 && cmd[0] == "git" && cmd[1] == "status"
}
func (gitStatusParser) Parse(lines []string) (map[string]any, []Failure, []string, []string) {
	a := map[string]any{"modified": 0, "added": 0, "deleted": 0, "renamed": 0, "untracked": 0, "conflicts": 0, "ahead": 0, "behind": 0}
	paths := []string{}
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "On branch ") {
			a["branch"] = strings.TrimPrefix(trimmed, "On branch ")
		}
		if strings.Contains(trimmed, "ahead of ") && strings.Contains(trimmed, " by ") {
			part := strings.Split(strings.Split(trimmed, " by ")[1], " ")[0]
			a["ahead"], _ = strconv.Atoi(part)
		}
		if strings.Contains(trimmed, "behind ") && strings.Contains(trimmed, " by ") {
			part := strings.Split(strings.Split(trimmed, " by ")[1], " ")[0]
			a["behind"], _ = strconv.Atoi(part)
		}
		for _, entry := range []struct{ prefix, key string }{{"modified:", "modified"}, {"new file:", "added"}, {"deleted:", "deleted"}, {"renamed:", "renamed"}} {
			if strings.HasPrefix(trimmed, entry.prefix) {
				a[entry.key] = a[entry.key].(int) + 1
				paths = append(paths, strings.TrimSpace(strings.TrimPrefix(trimmed, entry.prefix)))
			}
		}
		if strings.HasPrefix(l, "# branch.head ") {
			a["branch"] = strings.TrimPrefix(l, "# branch.head ")
		}
		if strings.HasPrefix(l, "# branch.ab ") {
			x := strings.Fields(l)
			if len(x) >= 4 {
				a["ahead"], _ = strconv.Atoi(strings.TrimPrefix(x[2], "+"))
				a["behind"], _ = strconv.Atoi(strings.TrimPrefix(x[3], "-"))
			}
			continue
		}
		if strings.HasPrefix(l, "?") {
			a["untracked"] = a["untracked"].(int) + 1
			paths = append(paths, strings.TrimSpace(strings.TrimPrefix(l, "?")))
			continue
		}
		if strings.HasPrefix(l, "\t") && strings.Contains(strings.Join(lines, "\n"), "Untracked files:") && !strings.Contains(trimmed, ":") {
			paths = append(paths, trimmed)
		}
		if strings.HasPrefix(l, "u ") {
			a["conflicts"] = a["conflicts"].(int) + 1
			continue
		}
		if strings.HasPrefix(l, "1 ") || strings.HasPrefix(l, "2 ") {
			x := strings.Fields(l)
			if len(x) > 1 {
				status := x[1]
				if strings.Contains(status, "M") {
					a["modified"] = a["modified"].(int) + 1
				}
				if strings.Contains(status, "A") {
					a["added"] = a["added"].(int) + 1
				}
				if strings.Contains(status, "D") {
					a["deleted"] = a["deleted"].(int) + 1
				}
				if strings.HasPrefix(l, "2 ") {
					a["renamed"] = a["renamed"].(int) + 1
				}
				paths = append(paths, x[len(x)-1])
			}
		}
	}
	return a, nil, unique(paths, 8), nil
}

type gitDiffParser struct{}

func (gitDiffParser) Name() string { return "git-diff" }
func (gitDiffParser) Detect(cmd []string, _ string) bool {
	return len(cmd) >= 2 && cmd[0] == "git" && cmd[1] == "diff"
}
func (gitDiffParser) Parse(lines []string) (map[string]any, []Failure, []string, []string) {
	files := []string{}
	adds, dels, binary := 0, 0, 0
	preview := []string{}
	for _, l := range lines {
		if strings.HasPrefix(l, "diff --git ") {
			p := strings.Fields(l)
			if len(p) >= 4 {
				files = append(files, strings.TrimPrefix(p[3], "b/"))
			}
		}
		if strings.HasPrefix(l, "Binary files") || strings.HasPrefix(l, "GIT binary patch") {
			binary++
		}
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			adds++
			if len(preview) < 6 {
				preview = append(preview, l)
			}
		}
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			dels++
			if len(preview) < 6 {
				preview = append(preview, l)
			}
		}
	}
	return map[string]any{"files": len(unique(files, 99999)), "insertions": adds, "deletions": dels, "binary_files": binary}, nil, unique(files, 8), preview
}

type phpunitParser struct{}

func (phpunitParser) Name() string { return "phpunit" }
func (phpunitParser) Detect(cmd []string, out string) bool {
	return strings.Contains(strings.Join(cmd, " "), "phpunit") || strings.Contains(out, "PHPUnit")
}
func (phpunitParser) Parse(lines []string) (map[string]any, []Failure, []string, []string) {
	return testParse(lines, "phpunit")
}

type pestParser struct{}

func (pestParser) Name() string { return "pest" }
func (pestParser) Detect(cmd []string, out string) bool {
	return strings.Contains(strings.Join(cmd, " "), "pest") || strings.Contains(out, "Pest")
}
func (pestParser) Parse(lines []string) (map[string]any, []Failure, []string, []string) {
	return testParse(lines, "pest")
}

var testsPattern = regexp.MustCompile(`(?i)Tests:\s*(\d+).*?(?:Assertions:\s*(\d+))?.*?(?:Failures?:\s*(\d+))?.*?(?:Errors?:\s*(\d+))?`)

func testParse(lines []string, _ string) (map[string]any, []Failure, []string, []string) {
	a := map[string]any{}
	f := []Failure{}
	paths := []string{}
	for i, l := range lines {
		if m := testsPattern.FindStringSubmatch(l); len(m) > 0 {
			for n, k := range []string{"tests", "assertions", "failed", "errors"} {
				if n+1 < len(m) && m[n+1] != "" {
					a[k], _ = strconv.Atoi(m[n+1])
				}
			}
		}
		if strings.HasPrefix(strings.TrimSpace(l), "1)") || regexp.MustCompile(`^\s*\d+\)\s+`).MatchString(l) {
			name := strings.TrimSpace(regexp.MustCompile(`^\s*\d+\)\s*`).ReplaceAllString(l, ""))
			msg := ""
			if i+1 < len(lines) {
				msg = strings.TrimSpace(lines[i+1])
			}
			p, n := findLocation(strings.Join(lines[i:min(i+5, len(lines))], "\n"))
			if p != "" {
				paths = append(paths, p)
			}
			f = append(f, Failure{Index: len(f) + 1, Name: name, Message: msg, File: p, Line: n})
		}
		if strings.Contains(l, "OK (") {
			a["passed"] = true
		}
		if strings.Contains(strings.ToLower(l), "skipped") {
			a["skipped"] = true
		}
	}
	if _, ok := a["tests"]; !ok {
		a["tests"] = 0
	}
	return a, f, unique(paths, 8), nil
}

type composerParser struct{}

func (composerParser) Name() string { return "composer" }
func (composerParser) Detect(cmd []string, out string) bool {
	return len(cmd) > 0 && cmd[0] == "composer" || strings.Contains(out, "Composer")
}
func (composerParser) Parse(lines []string) (map[string]any, []Failure, []string, []string) {
	a, f, p, l := genericParse(lines)
	installed, updated, removed := 0, 0, 0
	for _, x := range lines {
		z := strings.ToLower(x)
		if strings.Contains(z, "installing ") {
			installed++
		}
		if strings.Contains(z, "updating ") {
			updated++
		}
		if strings.Contains(z, "removing ") {
			removed++
		}
	}
	a["installed"] = installed
	a["updated"] = updated
	a["removed"] = removed
	return a, f, p, l
}

type nodeParser struct{}

func (nodeParser) Name() string { return "node" }
func (nodeParser) Detect(cmd []string, out string) bool {
	x := strings.Join(cmd, " ")
	return strings.HasPrefix(x, "npm ") || strings.HasPrefix(x, "pnpm ") || strings.HasPrefix(x, "yarn ") || strings.Contains(out, "npm ERR!")
}
func (nodeParser) Parse(lines []string) (map[string]any, []Failure, []string, []string) {
	a, f, p, l := genericParse(lines)
	v := 0
	for _, x := range lines {
		if strings.Contains(strings.ToLower(x), "vulnerabilit") {
			v++
		}
	}
	a["vulnerability_lines"] = v
	return a, f, p, l
}

func redactSummary(s Summary, r Redactor) Summary {
	for i := range s.Failures {
		s.Failures[i].Name = r.Text(s.Failures[i].Name)
		s.Failures[i].Message = r.Text(s.Failures[i].Message)
		for j := range s.Failures[i].Context {
			s.Failures[i].Context[j] = r.Text(s.Failures[i].Context[j])
		}
	}
	for i := range s.LastRelevant {
		s.LastRelevant[i] = r.Text(s.LastRelevant[i])
	}
	sort.Strings(s.ImportantPaths)
	return s
}
