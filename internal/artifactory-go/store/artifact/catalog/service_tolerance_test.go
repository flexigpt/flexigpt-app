package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const catalogToleranceRootID rootModel.RootID = "0192c4c0-0000-7000-8000-000000000001"

type tolerantCatalogRepository struct {
	entries []catalogModel.Entry
}

func (r tolerantCatalogRepository) ListCatalogByRoot(
	context.Context,
	rootModel.RootID,
	catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	output := make([]catalogModel.Entry, len(r.entries))
	for index, value := range r.entries {
		output[index] = value.Clone()
	}
	return output, nil
}

func (tolerantCatalogRepository) ListCatalogBySource(
	context.Context,
	rootModel.RootID,
	sourceModel.SourceID,
	catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return nil, nil
}

func (tolerantCatalogRepository) FindCatalogByIdentity(
	context.Context,
	rootModel.RootID,
	artifactModel.ArtifactKind,
	storeSpec.LogicalName,
	catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return nil, nil
}

type tolerantDefinitions struct {
	values      map[definitionModel.Key]definitionModel.Definition
	unavailable definitionModel.Key
}

func (d tolerantDefinitions) GetDefinition(
	ctx context.Context,
	rootID rootModel.RootID,
	digest cryptoutil.Digest,
) (definitionModel.Definition, error) {
	values, err := d.GetDefinitions(
		ctx,
		[]definitionModel.Key{{RootID: rootID, Digest: digest}},
	)
	if err != nil {
		return definitionModel.Definition{}, err
	}
	return values[0], nil
}

func (d tolerantDefinitions) GetDefinitions(
	_ context.Context,
	keys []definitionModel.Key,
) ([]definitionModel.Definition, error) {
	if slices.Contains(keys, d.unavailable) {
		return nil, fmt.Errorf(
			"%w: test Definition is unavailable",
			storeSpec.ErrDefinitionNotFound,
		)
	}

	output := make([]definitionModel.Definition, len(keys))
	for index, key := range keys {
		value, found := d.values[key]
		if !found {
			return nil, fmt.Errorf(
				"%w: test Definition is unavailable",
				storeSpec.ErrDefinitionNotFound,
			)
		}
		output[index] = value.Clone()
	}
	return output, nil
}

func TestListByRootKeepsHealthyRowsWhenOneDefinitionIsMissing(
	t *testing.T,
) {
	healthy := testCatalogDefinition(t, "healthy")
	missing := testCatalogDefinition(t, "missing")

	repository := tolerantCatalogRepository{
		entries: []catalogModel.Entry{
			testCatalogEntry("0192c4c0-0002-7000-8000-000000000001", healthy),
			testCatalogEntry("0192c4c0-0002-7000-8000-000000000002", missing),
		},
	}
	documents := tolerantDefinitions{
		values: map[definitionModel.Key]definitionModel.Definition{
			{
				RootID: catalogToleranceRootID,
				Digest: healthy.Digest,
			}: healthy,
		},
		unavailable: definitionModel.Key{
			RootID: catalogToleranceRootID,
			Digest: missing.Digest,
		},
	}

	service, err := NewService(repository, documents)
	if err != nil {
		t.Fatalf("create catalog service: %v", err)
	}

	values, err := service.ListByRoot(
		t.Context(),
		catalogToleranceRootID,
		catalogModel.ListOptions{IncludeDocument: true},
	)
	if err != nil {
		t.Fatalf("list catalog with one unavailable Definition: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("catalog row count = %d, want 2", len(values))
	}
	if values[0].Document == nil ||
		values[0].Document.Digest != healthy.Digest {
		t.Fatal("healthy catalog row did not retain its attached Definition")
	}
	if values[1].Document != nil {
		t.Fatal("unavailable Definition was attached to the degraded catalog row")
	}
	if values[1].DocumentIssue == nil ||
		values[1].DocumentIssue.Code != "artifact.definition-unavailable" {
		t.Fatalf(
			"degraded catalog issue = %#v, want definition-unavailable",
			values[1].DocumentIssue,
		)
	}
}

func testCatalogDefinition(
	t *testing.T,
	name string,
) definitionModel.Definition {
	t.Helper()

	value, err := definition.Admit(definitionModel.Definition{
		Kind:          artifactModel.ArtifactKind("agent"),
		SchemaID:      schemaModel.SchemaID("artifact.agent.v1"),
		SchemaVersion: "v1",
		LogicalName:   storeSpec.LogicalName(name),
		DisplayName:   name,
		Body: json.RawMessage(
			`{"name":"` + name + `","type":"agent"}`,
		),
	})
	if err != nil {
		t.Fatalf("admit test Definition %q: %v", name, err)
	}
	return value
}

func testCatalogEntry(
	id artifactModel.ArtifactID,
	value definitionModel.Definition,
) catalogModel.Entry {
	return catalogModel.Entry{
		ID:          id,
		RootID:      catalogToleranceRootID,
		Kind:        value.Kind,
		LogicalName: value.LogicalName,
		DisplayName: value.DisplayName,
		State:       artifactModel.StateAvailable,
		Enabled:     true,
		Revision:    1,
		Definition: &catalogModel.DefinitionMetadata{
			Digest:        value.Digest,
			SchemaID:      value.SchemaID,
			SchemaVersion: value.SchemaVersion,
			Description:   value.Description,
		},
	}
}
