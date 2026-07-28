package tokensave

import (
	"regexp"
	"strings"
)

type Redactor struct{ rules []*regexp.Regexp }

func NewRedactor(c Config) Redactor {
	if !c.RedactEnabled {
		return Redactor{}
	}
	patterns := []string{
		`(?i)(authorization:\s*(?:bearer|basic|token)\s+)[^\s]+`,
		`(?i)((?:token|password|passwd|secret|api[_-]?key)\s*[=:]\s*)[^\s,;]+`,
		`(?i)(https?://[^:/\s]+:)[^@/\s]+(@)`,
		`\b(?:sk|pk)_[A-Za-z0-9_-]{16,}\b`,
	}
	patterns = append(patterns, c.Patterns...)
	r := Redactor{}
	for _, p := range patterns {
		if x, e := regexp.Compile(p); e == nil {
			r.rules = append(r.rules, x)
		}
	}
	return r
}
func (r Redactor) Text(s string) string {
	for _, x := range r.rules {
		s = x.ReplaceAllStringFunc(s, func(m string) string {
			if i := strings.IndexAny(m, "=:"); i >= 0 {
				return m[:i+1] + "[REDACTED]"
			}
			if i := strings.Index(m, "@"); i >= 0 {
				return "[REDACTED]" + m[i:]
			}
			return "[REDACTED]"
		})
	}
	return s
}
