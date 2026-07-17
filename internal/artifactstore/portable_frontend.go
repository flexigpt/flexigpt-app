package artifactstore

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

type portableDefinitionFrontend struct{}

func (portableDefinitionFrontend) ID() spec.FrontendID {
	return spec.PortableDefinitionFrontendID
}

func (portableDefinitionFrontend) Recognizes(
	_ context.Context,
	candidate spec.ArtifactCandidate,
) spec.Recognition {
	var header struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return spec.RecognitionNone
	}
	if header.Format == spec.ArtifactDefinitionFileFormatV1 {
		return spec.RecognitionPreferred
	}
	return spec.RecognitionNone
}

func (portableDefinitionFrontend) Decode(
	_ context.Context,
	candidate spec.ArtifactCandidate,
) ([]spec.DecodedArtifact, []spec.Diagnostic) {
	canonicalJSON, err := baseutils.CanonicalizeJSON(candidate.Content)
	if err != nil {
		return nil, portableDefinitionDiagnostic(err.Error())
	}
	var file spec.ArtifactDefinitionFile
	decoder := json.NewDecoder(bytes.NewReader(canonicalJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, portableDefinitionDiagnostic(err.Error())
	}
	if err := ensurePortableJSONEOF(decoder); err != nil {
		return nil, portableDefinitionDiagnostic(err.Error())
	}
	if err := validate.ValidateArtifactDefinitionFile(file); err != nil {
		return nil, portableDefinitionDiagnostic(err.Error())
	}
	canonical, err := baseutils.CanonicalizeDefinition(file.Definition)
	if err != nil {
		return nil, portableDefinitionDiagnostic(err.Error())
	}
	canonical.Digest = ""
	return []spec.DecodedArtifact{{Definition: canonical}}, nil
}

func (portableDefinitionFrontend) ValidateStructure(
	_ context.Context,
	_ spec.CanonicalDefinition,
) []spec.Diagnostic {
	return nil
}

func (portableDefinitionFrontend) ValidateSemantic(
	_ context.Context,
	_ spec.CanonicalDefinition,
) []spec.Diagnostic {
	return nil
}

func (portableDefinitionFrontend) ExtractDependencies(
	_ context.Context,
	definition spec.CanonicalDefinition,
) ([]spec.ArtifactSelector, []spec.Diagnostic) {
	return append([]spec.ArtifactSelector(nil), definition.DependencySelectors...), nil
}

func (portableDefinitionFrontend) ValidateRecordData(
	_ context.Context,
	_ spec.CanonicalDefinition,
	_ spec.ArtifactRecordDraft,
) []spec.Diagnostic {
	return nil
}

func (portableDefinitionFrontend) DescribeExportClosure(
	_ context.Context,
	definition spec.CanonicalDefinition,
) (spec.ExportClosure, []spec.Diagnostic) {
	return spec.ExportClosure{
		DefinitionDigests: []spec.Digest{definition.Digest},
		Assets:            append([]spec.AssetManifestEntry(nil), definition.AssetManifest...),
	}, nil
}

func portableDefinitionDiagnostic(message string) []spec.Diagnostic {
	return []spec.Diagnostic{{
		Severity: spec.DiagnosticSeverityError,
		Code:     "artifactstore.portable-definition.invalid",
		Message:  message,
	}}
}

type portableTrailingJSONError struct{}

func (*portableTrailingJSONError) Error() string {
	return "portable definition contains trailing JSON values"
}

func ensurePortableJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return &portableTrailingJSONError{}
	}
	return err
}

var _ spec.ArtifactFrontend = portableDefinitionFrontend{}
