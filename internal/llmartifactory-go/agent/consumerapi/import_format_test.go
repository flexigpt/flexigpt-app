package consumerapi

import (
	"bytes"
	"testing"
)

func TestManagedAgentImportFormatForPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    managedAgentImportFormat
		wantErr bool
	}{
		{name: "json", path: "reviewer.json", want: managedAgentImportFormatJSON},
		{name: "yaml", path: "reviewer.yaml", want: managedAgentImportFormatYAML},
		{name: "yml", path: "reviewer.yml", want: managedAgentImportFormatYAML},
		{name: "uppercase", path: "reviewer.JSON", want: managedAgentImportFormatJSON},
		{name: "unsupported", path: "reviewer.md", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := managedAgentImportFormatForPath(test.path)
			if test.wantErr {
				if err == nil {
					t.Fatal("managedAgentImportFormatForPath() returned no error")
				}
				return
			}
			if err != nil {
				t.Fatalf("managedAgentImportFormatForPath() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("managedAgentImportFormatForPath() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCanonicalManagedAgentImportDocumentSelectsParser(t *testing.T) {
	jsonSource := []byte(`{"type":"agent","name":"json-agent"}`)
	yamlSource := []byte("type: agent\nname: json-agent\n")

	jsonCanonical, err := canonicalManagedAgentImportDocument(
		managedAgentImportFormatJSON,
		jsonSource,
	)
	if err != nil {
		t.Fatalf("canonical JSON import document: %v", err)
	}

	yamlCanonical, err := canonicalManagedAgentImportDocument(
		managedAgentImportFormatYAML,
		yamlSource,
	)
	if err != nil {
		t.Fatalf("canonical YAML import document: %v", err)
	}

	if !bytes.Equal(jsonCanonical, yamlCanonical) {
		t.Fatalf(
			"JSON and YAML imports produced different canonical declarations: %s != %s",
			jsonCanonical,
			yamlCanonical,
		)
	}

	if _, err := canonicalManagedAgentImportDocument(
		managedAgentImportFormatJSON,
		yamlSource,
	); err == nil {
		t.Fatal("JSON import parser accepted YAML-only syntax")
	}
}
