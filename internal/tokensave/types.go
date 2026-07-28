package tokensave

import "time"

type Metadata struct {
	ID                      string    `json:"id"`
	Command                 []string  `json:"command"`
	WorkingDirectory        string    `json:"working_directory"`
	StartedAt               time.Time `json:"started_at"`
	FinishedAt              time.Time `json:"finished_at"`
	DurationMS              int64     `json:"duration_ms"`
	ExitCode                int       `json:"exit_code"`
	StdoutBytes             int64     `json:"stdout_bytes"`
	StderrBytes             int64     `json:"stderr_bytes"`
	TruncatedTerminalOutput bool      `json:"truncated_terminal_output"`
	Parser                  string    `json:"parser"`
}

type Failure struct {
	Index   int      `json:"index"`
	Name    string   `json:"name"`
	Message string   `json:"message"`
	File    string   `json:"file,omitempty"`
	Line    int      `json:"line,omitempty"`
	Context []string `json:"context,omitempty"`
}

type Summary struct {
	RunID          string         `json:"run_id"`
	Status         string         `json:"status"`
	ExitCode       int            `json:"exit_code"`
	DurationMS     int64          `json:"duration_ms"`
	Parser         string         `json:"parser"`
	Summary        map[string]any `json:"summary"`
	Failures       []Failure      `json:"failures,omitempty"`
	ImportantPaths []string       `json:"important_paths,omitempty"`
	LastRelevant   []string       `json:"last_relevant,omitempty"`
	LogPath        string         `json:"log_path"`
}

type Limits struct{ MaxLines, MaxChars, MaxFailures int }

func DefaultLimits() Limits { return Limits{80, 12000, 5} }
