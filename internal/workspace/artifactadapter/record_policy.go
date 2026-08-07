package artifactadapter

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const (
	defaultArtifactName      = "artifact"
	artifactNameSeparator    = "-"
	artifactNameDigestLength = 12
	exactVersionConstraintOp = "="
)

type ArtifactPolicy struct {
	ids      artifact.ArtifactIDProvider
	supports map[basespec.ArtifactKind]spec.ArtifactSupport
}

func NewArtifactPolicy(
	ids artifact.ArtifactIDProvider,
	supports ...spec.ArtifactSupport,
) (*ArtifactPolicy, error) {
	if ids == nil || len(supports) == 0 {
		return nil, fmt.Errorf(
			"%w: workspace artifact support is required",
			spec.ErrInvalidWorkspace,
		)
	}
	values := make(map[basespec.ArtifactKind]spec.ArtifactSupport, len(supports))
	for _, support := range supports {
		if err := support.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := values[support.Kind]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate workspace artifact kind %q",
				spec.ErrInvalidWorkspace,
				support.Kind,
			)
		}
		values[support.Kind] = support
	}
	return &ArtifactPolicy{
		ids:      ids,
		supports: values,
	}, nil
}

func (p *ArtifactPolicy) Supports(
	kind basespec.ArtifactKind,
) bool {
	if p == nil {
		return false
	}
	_, supported := p.supports[kind]
	return supported
}

func (p *ArtifactPolicy) Derive(
	ctx context.Context,
	_ collection.Collection,
	occurrence catalog.Occurrence,
	value definition.Definition,
) (artifact.Draft, bool, []diagnostic.Diagnostic, error) {
	support, supported := p.supports[occurrence.Kind]
	if !supported {
		return artifact.Draft{}, false, nil, nil
	}
	if value.Kind != occurrence.Kind {
		return artifact.Draft{}, false, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     DiagnosticCodeArtifactKindMismatch,
			Message: fmt.Sprintf(
				"definition kind %q does not match occurrence kind %q",
				value.Kind,
				occurrence.Kind,
			),
			Location: &diagnostic.DiagnosticLocation{
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
			},
		}}, nil
	}
	if value.SchemaID != support.SchemaID {
		return artifact.Draft{}, false, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     DiagnosticCodeArtifactSchemaUnsupported,
			Message: fmt.Sprintf(
				"definition schema %q is not supported for kind %q",
				value.SchemaID,
				value.Kind,
			),
			Location: &diagnostic.DiagnosticLocation{
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
			},
		}}, nil
	}
	if err := support.Validator(value); err != nil {
		return artifact.Draft{}, false, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     DiagnosticCodeProjectionInvalid,
			Message:  diagnosticMessage(err.Error()),
			Location: &diagnostic.DiagnosticLocation{
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
			},
		}}, nil
	}

	// An empty ArtifactData means runtime use is enabled by default.
	data, err := EncodeArtifactData(spec.ArtifactData{})
	if err != nil {
		return artifact.Draft{}, false, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     DiagnosticCodeProjectionInvalid,
			Message:  diagnosticMessage(err.Error()),
		}}, nil
	}
	id, err := p.ids.NewArtifactID(ctx)
	if err != nil {
		return artifact.Draft{}, false, nil, err
	}
	if err := basespec.ValidateArtifactID(id); err != nil {
		return artifact.Draft{}, false, nil, err
	}

	name := artifactName(value.LogicalName, occurrence.Key)
	return artifact.Draft{
		ID:      id,
		Name:    name,
		Enabled: true,
		Data:    data,
	}, true, nil, nil
}

func artifactName(
	logicalName basespec.LogicalName,
	key catalog.OccurrenceKey,
) string {
	base := strings.TrimSpace(string(logicalName))
	if base == "" {
		base = defaultArtifactName
	}
	digest := cryptoutil.DigestBytes([]byte(occurrenceKeyDigestInput(key)))
	suffix := strings.TrimPrefix(
		string(digest),
		cryptoutil.DigestSHA256Prefix,
	)
	suffix = suffix[:artifactNameDigestLength]
	maximum := basespec.MaxDisplayNameBytes - len(suffix) - len(artifactNameSeparator)
	for len(base) > maximum {
		_, size := utf8.DecodeLastRuneInString(base)
		base = base[:len(base)-size]
	}
	return base + artifactNameSeparator + suffix
}

func diagnosticMessage(value string) string {
	return diagnostic.BoundedDiagnosticMessage(value)
}

func occurrenceKeyDigestInput(key catalog.OccurrenceKey) string {
	return string(key.SourceID) + "\x00" +
		string(key.Locator) + "\x00" +
		string(key.SubresourceLocator)
}
