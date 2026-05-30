package auth

import (
	"strings"
	"testing"
)

func TestSecretRedactorRedactsSensitiveValues(t *testing.T) {
	redactor := NewSecretRedactor(ResolvedTransportAuth{
		SensitiveValues: []string{
			"secret-one",
			"secret-two",
			"secret-one",
			"",
			"   ",
		},
	})

	got := redactor.Redact("prefix secret-one middle secret-two suffix")

	if strings.Contains(got, "secret-one") || strings.Contains(got, "secret-two") {
		t.Fatalf("redacted string leaked secret: %q", got)
	}
	if got != "prefix [REDACTED] middle [REDACTED] suffix" {
		t.Fatalf("redacted = %q", got)
	}
}

func TestSecretRedactorPreservesExactSecretValue(t *testing.T) {
	redactor := NewSecretRedactor(ResolvedTransportAuth{
		SensitiveValues: []string{"  exact-secret  "},
	})

	got := redactor.Redact("value=  exact-secret  ")
	if strings.Contains(got, "exact-secret") {
		t.Fatalf("redacted string leaked secret: %q", got)
	}
}
