package consumerapi

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// managedAgentImportFormat selects exactly one input parser. The source path
// remains transient and is never included in a prepared import payload.
type managedAgentImportFormat string

const (
	managedAgentImportFormatJSON managedAgentImportFormat = "json"
	managedAgentImportFormatYAML managedAgentImportFormat = "yaml"
)

func managedAgentImportFormatForPath(
	value string,
) (managedAgentImportFormat, error) {
	switch strings.ToLower(filepath.Ext(value)) {
	case ".json":
		return managedAgentImportFormatJSON, nil
	case ".yaml", ".yml":
		return managedAgentImportFormatYAML, nil
	default:
		return "", fmt.Errorf(
			"%w: managed Agent import accepts .json, .yaml, or .yml files",
			basespec.ErrInvalid,
		)
	}
}

func canonicalManagedAgentImportDocument(
	format managedAgentImportFormat,
	raw []byte,
) ([]byte, error) {
	switch format {
	case managedAgentImportFormatJSON:
		return jsonutil.CanonicalizeObject(
			raw,
			basespec.MaxDefinitionBytes,
		)
	case managedAgentImportFormatYAML:
		return yamlutil.CanonicalObjectJSON(
			raw,
			basespec.MaxDefinitionBytes,
		)
	default:
		return nil, fmt.Errorf(
			"%w: unsupported managed Agent import format %q",
			basespec.ErrInvalid,
			format,
		)
	}
}

func (format managedAgentImportFormat) invalidDocumentIssueCode() string {
	switch format {
	case managedAgentImportFormatJSON:
		return "agent.import.json-invalid"
	case managedAgentImportFormatYAML:
		return "agent.import.yaml-invalid"
	default:
		return "agent.import.document-invalid"
	}
}
