package sdkclient

import (
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
)

type secretRedactor struct {
	values []string
}

func newSecretRedactor(resolved runtime.ResolvedTransportAuth) *secretRedactor {
	var values []string

	for _, v := range resolved.Headers {
		v = strings.TrimSpace(v)
		if len(v) >= 8 {
			values = append(values, v)
		}
		if strings.HasPrefix(strings.ToLower(v), "bearer ") {
			token := strings.TrimSpace(v[len("bearer "):])
			if len(token) >= 8 {
				values = append(values, token)
			}
		}
	}

	for _, v := range resolved.Env {
		v = strings.TrimSpace(v)
		if len(v) >= 8 {
			values = append(values, v)
		}
	}

	return &secretRedactor{values: values}
}

func (r *secretRedactor) Redact(s string) string {
	if r == nil || len(r.values) == 0 || s == "" {
		return s
	}
	for _, v := range r.values {
		s = strings.ReplaceAll(s, v, "[REDACTED]")
	}
	return s
}
