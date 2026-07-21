package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
)

type RecordPolicy struct {
	schemaIDs map[artifactstore.ArtifactKind]artifactstore.SchemaID
}

func NewRecordPolicy(
	supports ...ArtifactSupport,
) (*RecordPolicy, error) {
	if len(supports) == 0 {
		return nil, fmt.Errorf(
			"%w: workspace artifact support is required",
			ErrInvalidWorkspace,
		)
	}
	values := make(map[artifactstore.ArtifactKind]artifactstore.SchemaID, len(supports))
	for _, support := range supports {
		if err := support.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := values[support.Kind]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate workspace artifact kind %q",
				ErrInvalidWorkspace,
				support.Kind,
			)
		}
		values[support.Kind] = support.SchemaID
	}
	return &RecordPolicy{schemaIDs: values}, nil
}

func (p *RecordPolicy) Derive(
	_ context.Context,
	_ catalog.Root,
	occurrence catalog.Occurrence,
	value definition.Definition,
) (record.Draft, bool, []artifactstore.Diagnostic) {
	schemaID, supported := p.schemaIDs[occurrence.Kind]
	if !supported {
		return record.Draft{}, false, nil
	}
	if value.Kind != occurrence.Kind {
		return record.Draft{}, false, []artifactstore.Diagnostic{{
			Severity: artifactstore.DiagnosticError,
			Code:     DiagnosticCodeRecordKindMismatch,
			Message: fmt.Sprintf(
				"definition kind %q does not match occurrence kind %q",
				value.Kind,
				occurrence.Kind,
			),
			Location: &artifactstore.DiagnosticLocation{
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
			},
		}}
	}
	if value.SchemaID != schemaID {
		return record.Draft{}, false, []artifactstore.Diagnostic{{
			Severity: artifactstore.DiagnosticError,
			Code:     DiagnosticCodeRecordSchemaUnsupported,
			Message: fmt.Sprintf(
				"definition schema %q is not supported for kind %q",
				value.SchemaID,
				value.Kind,
			),
			Location: &artifactstore.DiagnosticLocation{
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
			},
		}}
	}
	name := recordName(value.LogicalName, occurrence.Key)
	return record.Draft{
		Name:    name,
		Enabled: true,
		Data:    json.RawMessage(jsoncanon.EmptyObject),
	}, true, nil
}

func recordName(
	logicalName artifactstore.LogicalName,
	key catalog.OccurrenceKey,
) string {
	base := strings.TrimSpace(string(logicalName))
	if base == "" {
		base = defaultRecordName
	}
	digest := definition.DigestBytes([]byte(key.String()))
	suffix := strings.TrimPrefix(
		string(digest),
		artifactstore.DigestSHA256Prefix,
	)
	suffix = suffix[:recordNameDigestLength]
	maximum := artifactstore.MaxDisplayNameBytes - len(suffix) - len(recordNameSeparator)
	if len(base) > maximum {
		base = base[:maximum]
	}
	return base + recordNameSeparator + suffix
}

var _ record.Policy = (*RecordPolicy)(nil)
