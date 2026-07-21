package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
)

type RecordPolicy struct {
	descriptors map[artifactstore.ArtifactKind]Descriptor
}

func NewRecordPolicy(
	descriptors ...Descriptor,
) (*RecordPolicy, error) {
	values := map[artifactstore.ArtifactKind]Descriptor{
		DefinitionKind: {
			Kind:     DefinitionKind,
			SchemaID: DefinitionSchemaID,
		},
	}
	for _, descriptor := range descriptors {
		if err := artifactstore.ValidateArtifactKind(descriptor.Kind); err != nil {
			return nil, err
		}
		if err := artifactstore.ValidateSchemaID(descriptor.SchemaID); err != nil {
			return nil, err
		}
		values[descriptor.Kind] = descriptor
	}
	return &RecordPolicy{descriptors: values}, nil
}

func (p *RecordPolicy) Derive(
	_ context.Context,
	_ catalog.Root,
	occurrence catalog.Occurrence,
	value definition.Definition,
) (record.Draft, bool, []artifactstore.Diagnostic) {
	descriptor, supported := p.descriptors[occurrence.Kind]
	if !supported {
		return record.Draft{}, false, nil
	}
	if value.SchemaID != descriptor.SchemaID {
		return record.Draft{}, false, []artifactstore.Diagnostic{{
			Severity: artifactstore.DiagnosticError,
			Code:     "workspace.record.schema-unsupported",
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
		Data:    json.RawMessage("{}"),
	}, true, nil
}

func recordName(
	logicalName artifactstore.LogicalName,
	key catalog.OccurrenceKey,
) string {
	base := strings.TrimSpace(string(logicalName))
	if base == "" {
		base = "artifact"
	}
	digest := definition.DigestBytes([]byte(key.String()))
	suffix := strings.TrimPrefix(string(digest), "sha256:")[:12]
	maximum := artifactstore.MaxDisplayNameBytes - len(suffix) - 1
	if len(base) > maximum {
		base = base[:maximum]
	}
	return base + "-" + suffix
}

var _ record.Policy = (*RecordPolicy)(nil)
