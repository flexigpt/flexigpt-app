package sdkclient

import (
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
)

type secretRedactor struct {
	values []string
}

func newSecretRedactor(resolved runtime.ResolvedTransportAuth) *secretRedactor {
	seen := make(map[string]struct{})
	values := make([]string, 0, len(resolved.SensitiveValues))
	for _, v := range resolved.SensitiveValues {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		values = append(values, v)
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
