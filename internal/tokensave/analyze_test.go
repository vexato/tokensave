package tokensave

import "testing"

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
	a, f, _, _ := phpunitParser{}.Parse([]string{"PHPUnit 10", "Tests: 4, Assertions: 8, Failures: 2, Errors: 0", "1) UserTest::testCreate", "Expected 201", "tests/UserTest.php:84"})
	if a["tests"] != 4 || len(f) != 1 || f[0].Name != "UserTest::testCreate" {
		t.Fatalf("unexpected: %#v %#v", a, f)
	}
}
