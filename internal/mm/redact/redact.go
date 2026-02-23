package redact

import (
	"regexp"
	"strings"
)

type Redactor struct {
	enabled bool
	rules   []redactionRule
}

type redactionRule struct {
	re    *regexp.Regexp
	label string
}

func New(enabled bool, custom []string) *Redactor {
	rules := []redactionRule{
		{re: regexp.MustCompile(`(?is)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`), label: "[REDACTED_PRIVATE_KEY]"},
		{re: regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`), label: "Bearer [REDACTED]"},
		{re: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), label: "[REDACTED_AWS_KEY]"},
		{re: regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)\s*[:=]\s*['"]?[^\s'"]+`), label: "$1=[REDACTED]"},
		{re: regexp.MustCompile(`(?m)^\s*[A-Za-z_][A-Za-z0-9_]*\s*=\s*.+$`), label: "[REDACTED_ENV_LINE]"},
	}
	for _, pattern := range custom {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		rules = append(rules, redactionRule{re: re, label: "[REDACTED_CUSTOM]"})
	}
	return &Redactor{
		enabled: enabled,
		rules:   rules,
	}
}

func (r *Redactor) Apply(input string) string {
	if r == nil || !r.enabled || input == "" {
		return input
	}
	out := input
	for _, rule := range r.rules {
		out = rule.re.ReplaceAllString(out, rule.label)
	}
	return out
}
