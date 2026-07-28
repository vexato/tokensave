package tokensave

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testCommand(ok bool) []string {
	if runtime.GOOS == "windows" {
		if ok {
			return []string{"cmd", "/C", "echo hello"}
		}
		return []string{"cmd", "/C", "echo problem 1>&2 & exit /B 7"}
	}
	if ok {
		return []string{"sh", "-c", "printf hello"}
	}
	return []string{"sh", "-c", "printf 'fatal TOKEN=abc123456789\n' >&2; exit 7"}
}
func withHome(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("TOKENSAVE_HOME", h)
	return h
}
func TestRunCapturesOutputAndExitCode(t *testing.T) {
	h := withHome(t)
	c := defaultConfig()
	m, s, e := ExecuteRun(RunRequest{Command: testCommand(true)}, c)
	if e != nil || m.ExitCode != 0 || s.Status != "succeeded" {
		t.Fatalf("%v %#v %#v", e, m, s)
	}
	b, e := os.ReadFile(filepath.Join(h, "runs", m.ID, "stdout.log"))
	if e != nil || !strings.Contains(string(b), "hello") {
		t.Fatalf("stdout: %q %v", b, e)
	}
	_, _, e = ExecuteRun(RunRequest{Command: testCommand(false)}, c)
	if e != nil {
		t.Fatal(e)
	}
}
func TestExecutePreservesChildExitCodeAndJSON(t *testing.T) {
	withHome(t)
	var b bytes.Buffer
	e := Execute(append(testCommand(false), "--json"), &b, &b)
	code, ok := ChildExitCode(e)
	if !ok || code != 7 {
		t.Fatalf("got %v %v", code, e)
	}
	if !strings.Contains(b.String(), "\"exit_code\":7") {
		t.Fatalf("not json %s", b.String())
	}
}
func TestShowListCleanAndLimits(t *testing.T) {
	h := withHome(t)
	c := defaultConfig()
	m, _, e := ExecuteRun(RunRequest{Command: testCommand(true)}, c)
	if e != nil {
		t.Fatal(e)
	}
	var b bytes.Buffer
	if e = show([]string{m.ID, "--tail", "1"}, &b); e != nil || !strings.Contains(b.String(), "hello") {
		t.Fatalf("show: %v %q", e, b.String())
	}
	b.Reset()
	if e = list(nil, &b); e != nil || !strings.Contains(b.String(), m.ID) {
		t.Fatalf("list: %v %q", e, b.String())
	}
	b.Reset()
	if e = clean([]string{"--all"}, &b); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(h, "runs", m.ID)); !os.IsNotExist(e) {
		t.Fatal("run remains")
	}
}
func TestClip(t *testing.T) {
	s := clip("a\nb\nc\n", Limits{2, 3, 1})
	if len(strings.Split(s, "\n")) < 2 || len(s) < 3 {
		t.Fatal(s)
	}
}

func TestLargeOutputIsStored(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows fixture command")
	}
	withHome(t)
	m, _, err := ExecuteRun(RunRequest{Command: []string{"cmd", "/C", "for /L %i in (1,1,120000) do @echo 0123456789012345678901234567890123456789"}}, defaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if m.StdoutBytes < 4*1024*1024 {
		t.Fatalf("expected multi-megabyte log, got %d bytes", m.StdoutBytes)
	}
}

func TestHomeUsesExistingProjectStore(t *testing.T) {
	t.Setenv("TOKENSAVE_HOME", "")
	project := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".tokensave", "runs"), 0755); err != nil {
		t.Fatal(err)
	}
	home, err := Home()
	expectedHome := filepath.Join(project, ".tokensave")
	resolvedHome, resolveErr := filepath.EvalSymlinks(home)
	resolvedExpectedHome, expectedResolveErr := filepath.EvalSymlinks(expectedHome)
	if err != nil || resolveErr != nil || expectedResolveErr != nil || resolvedHome != resolvedExpectedHome {
		t.Fatalf("home=%q err=%v", home, err)
	}
}
