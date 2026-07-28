package tokensave

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func Home() (string, error) {
	if h := os.Getenv("TOKENSAVE_HOME"); h != "" {
		return h, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		if l := os.Getenv("LOCALAPPDATA"); l != "" {
			return filepath.Join(l, "tokensave"), nil
		}
		return filepath.Join(h, "AppData", "Local", "tokensave"), nil
	case "darwin":
		return filepath.Join(h, "Library", "Application Support", "tokensave"), nil
	default:
		if x := os.Getenv("XDG_STATE_HOME"); x != "" {
			return filepath.Join(x, "tokensave"), nil
		}
		return filepath.Join(h, ".local", "state", "tokensave"), nil
	}
}

func runsDir() (string, error) { h, e := Home(); return filepath.Join(h, "runs"), e }
func NewRunID(now time.Time) string {
	return now.UTC().Format("20060102-150405") + "-" + fmt.Sprintf("%04x", now.UnixNano()&0xffff)
}
func RunDir(id string) (string, error) {
	if id == "" || strings.ContainsAny(id, `/\\`) || strings.Contains(id, "..") {
		return "", fmt.Errorf("invalid run id")
	}
	r, e := runsDir()
	return filepath.Join(r, id), e
}
func WriteJSON(path string, value any) error {
	b, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0644)
}
func ReadMetadata(id string) (Metadata, string, error) {
	d, e := RunDir(id)
	if e != nil {
		return Metadata{}, "", e
	}
	b, e := os.ReadFile(filepath.Join(d, "metadata.json"))
	if e != nil {
		return Metadata{}, "", e
	}
	var m Metadata
	e = json.Unmarshal(b, &m)
	return m, d, e
}
func ListRuns() ([]Metadata, error) {
	r, e := runsDir()
	if e != nil {
		return nil, e
	}
	entries, e := os.ReadDir(r)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	out := []Metadata{}
	for _, x := range entries {
		if !x.IsDir() {
			continue
		}
		b, er := os.ReadFile(filepath.Join(r, x.Name(), "metadata.json"))
		if er != nil {
			continue
		}
		var m Metadata
		if json.Unmarshal(b, &m) == nil {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out, nil
}
