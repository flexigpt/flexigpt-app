package mcpv1

import "testing"

func TestValidateMCPURLAllowsWholeInstallationInputReference(t *testing.T) {
	if err := validateMCPURL("${MCP_URL}"); err != nil {
		t.Fatalf("whole installation input reference was rejected: %v", err)
	}
}

func TestValidateMCPURLRejectsNonURLText(t *testing.T) {
	if err := validateMCPURL("not-a-url"); err == nil {
		t.Fatal("non-URL text was accepted")
	}
}
