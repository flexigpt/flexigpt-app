package basespec

import "testing"

func TestValidateIdentifierAllowsLowerCamelSegments(t *testing.T) {
	t.Parallel()

	values := []string{
		"documentSets",
		"managedMCP",
		"artifactStoreDirectory",
		"mcp.policy",
		"mcp.policyV1",
		"agent-managedMCP",
		"v2",
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			if err := ValidateIdentifier(
				"test identifier",
				value,
				MaxKindBytes,
			); err != nil {
				t.Fatalf(
					"ValidateIdentifier(%q) returned error: %v",
					value,
					err,
				)
			}
		})
	}
}

func TestValidateIdentifierRejectsInvalidSegmentForms(t *testing.T) {
	t.Parallel()

	values := []string{
		"",
		"DocumentSets",
		"MCPConfig",
		"mcp.PolicyV1",
		"mcp-PolicyV1",
		"document_sets",
		"document..sets",
		"document--sets",
		"documentSets-",
		"2documentSets",
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			if err := ValidateIdentifier(
				"test identifier",
				value,
				MaxKindBytes,
			); err == nil {
				t.Fatalf(
					"ValidateIdentifier(%q) unexpectedly succeeded",
					value,
				)
			}
		})
	}
}
