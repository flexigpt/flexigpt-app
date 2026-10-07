package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// ImportFormat selects exactly one input parser. The source path
// remains transient and is never included in a prepared import payload.
func managedAgentImportFormatForPath(
	value string,
	formats map[string]ImportFormat,
) (ImportFormat, error) {
	extension := strings.ToLower(filepath.Ext(value))
	if format, found := formats[extension]; found {
		return format, nil
	}
	return "", fmt.Errorf("%w: managed Agent import format %q is not supported", spec.ErrInvalid, extension)
}

func canonicalManagedAgentImportDocument(
	format ImportFormat,
	raw []byte,
) ([]byte, error) {
	switch format {
	case ImportFormatJSON:
		return jsonutil.CanonicalizeObject(
			raw,
			spec.MaxDefinitionBytes,
		)
	case ImportFormatYAML:
		return yamlutil.CanonicalObjectJSON(
			raw,
			spec.MaxDefinitionBytes,
		)
	default:
		return nil, fmt.Errorf(
			"%w: unsupported managed Agent import format %q",
			spec.ErrInvalid,
			format,
		)
	}
}

func (format ImportFormat) invalidDocumentIssueCode() string {
	switch format {
	case ImportFormatJSON:
		return "agent.import.json-invalid"
	case ImportFormatYAML:
		return "agent.import.yaml-invalid"
	default:
		return "agent.import.document-invalid"
	}
}
