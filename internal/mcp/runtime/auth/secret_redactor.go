package auth

import (
	"strings"
)

type SecretRedactor struct {
	values []string
}

func NewSecretRedactor(resolved ResolvedTransportAuth) *SecretRedactor {
	seen := make(map[string]struct{})
	values := make([]string, 0, len(resolved.SensitiveValues))
	for _, v := range resolved.SensitiveValues {

		if v == "" || strings.TrimSpace(v) == "" {
			continue
		}
		// Preserve exact value. Trimming can cause redaction misses.
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		values = append(values, v)
	}

	return &SecretRedactor{values: values}
}

func (r *SecretRedactor) Redact(s string) string {
	if r == nil || len(r.values) == 0 || s == "" {
		return s
	}
	for _, v := range r.values {
		s = strings.ReplaceAll(s, v, "[REDACTED]")
	}
	return s
}
