package tokensave

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config deliberately supports the small, documented YAML subset without a runtime dependency.
type Config struct {
	MaxLines      int
	MaxChars      int
	MaxFailures   int
	RetentionDays int
	RedactEnabled bool
	Patterns      []string
	Commands      map[string]string
}

func defaultConfig() Config {
	return Config{MaxLines: 80, MaxChars: 12000, MaxFailures: 5, RetentionDays: 14, RedactEnabled: true, Commands: map[string]string{}}
}
func configPaths() []string {
	paths := []string{}
	if h, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(h, ".config", "tokensave", "config.yml"))
	}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, ".tokensave.yml"))
	}
	return paths
}
func LoadConfig() Config {
	c := defaultConfig()
	for _, p := range configPaths() {
		mergeConfig(&c, p)
	}
	return c
}
func mergeConfig(c *Config, path string) {
	f, e := os.Open(path)
	if e != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	section := ""
	command := ""
	for s.Scan() {
		raw := s.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if section == "redact" && strings.HasPrefix(line, "-") {
			c.Patterns = append(c.Patterns, strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "-")), "\"'"))
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		if strings.HasSuffix(line, ":") {
			key := strings.TrimSuffix(line, ":")
			if indent == 0 {
				section = key
				command = ""
			} else if section == "commands" {
				command = strings.Trim(key, "\"")
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		if section == "commands" && command != "" && key == "parser" {
			if c.Commands == nil {
				c.Commands = map[string]string{}
			}
			c.Commands[command] = value
			continue
		}
		if section == "redact" {
			if key == "enabled" {
				c.RedactEnabled = value != "false"
			}
			continue
		}
		switch key {
		case "max_lines":
			c.MaxLines, _ = strconv.Atoi(value)
		case "max_chars":
			c.MaxChars, _ = strconv.Atoi(value)
		case "max_failures":
			c.MaxFailures, _ = strconv.Atoi(value)
		case "retention_days":
			c.RetentionDays, _ = strconv.Atoi(value)
		}
	}
}
func (c Config) Limits() Limits { return Limits{c.MaxLines, c.MaxChars, c.MaxFailures} }
