package secrets

import (
	"regexp"
)

// Redactor replaces sensitive patterns in text
type Redactor struct {
	patterns []*regexp.Regexp
}

// NewRedactor creates a new redactor with the given patterns
func NewRedactor(patterns []string) *Redactor {
	r := &Redactor{
		patterns: make([]*regexp.Regexp, 0, len(patterns)),
	}

	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			r.patterns = append(r.patterns, re)
		}
	}

	return r
}

// Redact replaces all matches with redaction markers
func (r *Redactor) Redact(s string) string {
	for _, re := range r.patterns {
		s = re.ReplaceAllStringFunc(s, func(match string) string {
			if len(match) <= 8 {
				return "[REDACTED]"
			}
			// Show first 4 and last 4 chars for debugging
			return match[:4] + "..." + match[len(match)-4:] + "[REDACTED]"
		})
	}
	return s
}
