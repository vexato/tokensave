package tokensave

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type streamWriter struct {
	file     *os.File
	combined *os.File
	lock     *sync.Mutex
	bytes    int64
}

func (w *streamWriter) Write(p []byte) (int, error) {
	n, e := w.file.Write(p)
	w.bytes += int64(n)
	w.lock.Lock()
	_, ce := w.combined.Write(p)
	w.lock.Unlock()
	if e != nil {
		return n, e
	}
	return n, ce
}

type RunRequest struct {
	Command []string
	Shell   string
	CWD     string
}

func ExecuteRun(req RunRequest, c Config) (Metadata, Summary, error) {
	if len(req.Command) == 0 && req.Shell == "" {
		return Metadata{}, Summary{}, fmt.Errorf("no command provided")
	}
	if req.CWD == "" {
		req.CWD, _ = os.Getwd()
	}
	started := time.Now().UTC()
	id := NewRunID(started)
	root, e := RunDir(id)
	if e != nil {
		return Metadata{}, Summary{}, e
	}
	if e = os.MkdirAll(root, 0755); e != nil {
		return Metadata{}, Summary{}, e
	}
	stdout, e := os.Create(filepath.Join(root, "stdout.log"))
	if e != nil {
		return Metadata{}, Summary{}, e
	}
	defer stdout.Close()
	stderr, e := os.Create(filepath.Join(root, "stderr.log"))
	if e != nil {
		return Metadata{}, Summary{}, e
	}
	defer stderr.Close()
	combined, e := os.Create(filepath.Join(root, "combined.log"))
	if e != nil {
		return Metadata{}, Summary{}, e
	}
	defer combined.Close()
	ctx, stop := signalContext()
	defer stop()
	var cmd *exec.Cmd
	if req.Shell != "" {
		cmd = shellCommand(ctx, req.Shell)
	} else {
		cmd = exec.CommandContext(ctx, req.Command[0], req.Command[1:]...)
	}
	cmd.Dir = req.CWD
	cmd.Env = os.Environ()
	var lock sync.Mutex
	outw := &streamWriter{file: stdout, combined: combined, lock: &lock}
	errw := &streamWriter{file: stderr, combined: combined, lock: &lock}
	cmd.Stdout = outw
	cmd.Stderr = errw
	e = cmd.Run()
	finished := time.Now().UTC()
	code := 0
	if e != nil {
		if x, ok := e.(*exec.ExitError); ok {
			code = x.ExitCode()
		} else {
			code = 127
		}
	}
	meta := Metadata{ID: id, Command: req.Command, WorkingDirectory: req.CWD, StartedAt: started, FinishedAt: finished, DurationMS: finished.Sub(started).Milliseconds(), ExitCode: code, StdoutBytes: outw.bytes, StderrBytes: errw.bytes}
	if req.Shell != "" {
		meta.Command = []string{req.Shell}
	}
	s := Analyze(meta, root, c)
	meta.Parser = s.Parser
	meta.TruncatedTerminalOutput = true
	s.RunID = id
	s.Status = map[bool]string{true: "succeeded", false: "failed"}[code == 0]
	s.ExitCode = code
	s.DurationMS = meta.DurationMS
	s.LogPath = root
	if e := WriteJSON(filepath.Join(root, "metadata.json"), meta); e != nil {
		return meta, s, e
	}
	if e := WriteJSON(filepath.Join(root, "summary.json"), s); e != nil {
		return meta, s, e
	}
	if e := os.WriteFile(filepath.Join(root, "summary.txt"), []byte(Render(s, meta, c.Limits())), 0644); e != nil {
		return meta, s, e
	}
	return meta, s, nil
}

// References kept here so platform files have a small, explicit contract.
var _ io.Writer
