package tokensave

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureLines(t *testing.T, name string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestGitStatusParser(t *testing.T) {
	a, _, paths, _ := gitStatusParser{}.Parse([]string{"# branch.head feature/payment", "# branch.ab +2 -0", "1 M. N... 100644 100644 100644 x x src/PaymentService.php", "? tests/NewTest.php"})
	if a["branch"] != "feature/payment" || a["ahead"] != 2 || a["modified"] != 1 || a["untracked"] != 1 || len(paths) != 2 {
		t.Fatalf("unexpected summary: %#v %#v", a, paths)
	}
}
func TestGitDiffParser(t *testing.T) {
	a, _, p, _ := gitDiffParser{}.Parse([]string{"diff --git a/a.go b/a.go", "--- a/a.go", "+++ b/a.go", "+added", "-removed"})
	if a["files"] != 1 || a["insertions"] != 1 || a["deletions"] != 1 || len(p) != 1 {
		t.Fatalf("unexpected: %#v", a)
	}
}
func TestGenericRedactsSecrets(t *testing.T) {
	c := defaultConfig()
	r := NewRedactor(c)
	s := redactSummary(Summary{Failures: []Failure{{Message: "TOKEN=hello-secret-value"}}}, r)
	if s.Failures[0].Message == "TOKEN=hello-secret-value" {
		t.Fatal("secret was not redacted")
	}
}
func TestPHPUnitFailures(t *testing.T) {
	a, f, _, _ := phpunitParser{}.Parse(fixtureLines(t, "phpunit-failures.txt"))
	if a["tests"] != 4 || a["assertions"] != 8 || a["failed"] != 2 || a["errors"] != 0 || len(f) != 1 || f[0].Name != "UserServiceTest::testCreatesUser" {
		t.Fatalf("unexpected: %#v %#v", a, f)
	}
}

func TestPestFailures(t *testing.T) {
	a, f, _, _ := pestParser{}.Parse(fixtureLines(t, "pest.txt"))
	if a["tests"] != 3 || a["assertions"] != 5 || a["failed"] != 1 || len(f) != 1 || f[0].Name != `Tests\\Feature\\InvoiceTest` {
		t.Fatalf("unexpected: %#v %#v", a, f)
	}
}

func TestNodeDiagnostics(t *testing.T) {
	_, f, _, last := nodeParser{}.Parse(fixtureLines(t, "npm-failure.txt"))
	if len(f) != 2 {
		t.Fatalf("unexpected failures: %#v", f)
	}
	combined := strings.Join(append(last, f[0].Message), "\n")
	if !strings.Contains(combined, "ERESOLVE") || !strings.Contains(combined, "unable to resolve dependency tree") {
		t.Fatalf("unexpected: %#v %#v", f, last)
	}
}

func TestComposerDiagnostics(t *testing.T) {
	_, f, _, _ := composerParser{}.Parse(fixtureLines(t, "composer-failure.txt"))
	if len(f) != 1 || f[0].Name != "Problem 1" || !strings.Contains(f[0].Message, "could not be resolved") {
		t.Fatalf("unexpected: %#v", f)
	}
}
