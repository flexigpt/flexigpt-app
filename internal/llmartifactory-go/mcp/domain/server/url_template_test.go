package server

import "testing"

func TestValidateWholeURLInputReferenceAllowsTextInput(t *testing.T) {
	inputs := map[string]InputDeclaration{
		"MCP_URL": {
			Kind:     InputText,
			Required: true,
		},
	}

	if err := validateWholeURLInputReference(
		"mcpServer.url",
		"${MCP_URL}",
		inputs,
	); err != nil {
		t.Fatalf("whole URL text input was rejected: %v", err)
	}
}

func TestValidateWholeURLInputReferenceRejectsSecretInput(t *testing.T) {
	inputs := map[string]InputDeclaration{
		"MCP_URL": {
			Kind:     InputSecret,
			Required: true,
		},
	}

	if err := validateWholeURLInputReference(
		"mcpServer.url",
		"${MCP_URL}",
		inputs,
	); err == nil {
		t.Fatal("whole URL secret input was accepted")
	}
}
