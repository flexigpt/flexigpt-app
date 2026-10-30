package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
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
	return "", fmt.Errorf("%w: managed Agent import format %q is not supported", storeSpec.ErrInvalid, extension)
}

func canonicalManagedAgentImportDocument(
	format ImportFormat,
	raw []byte,
) ([]byte, error) {
	var canonical []byte
	var err error
	switch format {
	case ImportFormatJSON:
		canonical, err = jsonutil.CanonicalizeObject(
			raw,
			storeSpec.MaxDefinitionBytes,
		)
	case ImportFormatYAML:
		canonical, err = yamlutil.CanonicalObjectJSON(
			raw,
			storeSpec.MaxDefinitionBytes,
		)
	default:
		return nil, fmt.Errorf(
			"%w: unsupported managed Agent import format %q",
			storeSpec.ErrInvalid,
			format,
		)
	}
	if err != nil {
		return nil, err
	}
	return declaration.PrepareSourceDocument(
		agentv1.AgentSchemaKey,
		canonical,
		false,
	)
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
