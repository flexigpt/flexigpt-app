package ingest

import (
	"errors"
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	compiledRegistrationTestRootID = rootModel.RootID(
		"0192c4c0-0000-7000-8000-000000000001",
	)
	compiledRegistrationTestSourceID = sourceModel.SourceID(
		"0192c4c0-0001-7000-8000-000000000001",
	)
)

func TestRegisterCompiledDocumentsReplacesPriorRegistrationEvidence(
	t *testing.T,
) {
	registry, err := NewDecoderRegistry(nil)
	if err != nil {
		t.Fatalf("create decoder registry: %v", err)
	}
	engine, err := NewEngine(registry)
	if err != nil {
		t.Fatalf("create ingest engine: %v", err)
	}

	first := compiledRegistrationTestDocument(
		t,
		spec.Locator("first.json"),
		spec.LogicalName("first"),
	)
	if err := engine.RegisterCompiledDocuments(
		t.Context(),
		"test.owner",
		compiledRegistrationTestRootID,
		compiledRegistrationTestSourceID,
		[]CompiledDocument{first},
	); err != nil {
		t.Fatalf("register first compiled document: %v", err)
	}

	second := compiledRegistrationTestDocument(
		t,
		spec.Locator("second.json"),
		spec.LogicalName("second"),
	)
	if err := engine.RegisterCompiledDocuments(
		t.Context(),
		"test.owner",
		compiledRegistrationTestRootID,
		compiledRegistrationTestSourceID,
		[]CompiledDocument{second},
	); err != nil {
		t.Fatalf("replace compiled document registration: %v", err)
	}

	if _, found := engine.compiledDocument(
		compiledRegistrationTestRootID,
		compiledRegistrationTestSourceID,
		first.Locator,
	); found {
		t.Fatal("removed compiled document evidence remains registered")
	}

	actual, found := engine.compiledDocument(
		compiledRegistrationTestRootID,
		compiledRegistrationTestSourceID,
		second.Locator,
	)
	if !found {
		t.Fatal("replacement compiled document evidence is absent")
	}
	if actual.Digest != second.Digest {
		t.Fatalf(
			"replacement compiled document digest is %q, expected %q",
			actual.Digest,
			second.Digest,
		)
	}

	err = engine.RegisterCompiledDocuments(
		t.Context(),
		"other.owner",
		compiledRegistrationTestRootID,
		compiledRegistrationTestSourceID,
		[]CompiledDocument{second},
	)
	if !errors.Is(err, spec.ErrIdentityConflict) {
		t.Fatalf(
			"duplicate locator from another registration error = %v, want identity conflict",
			err,
		)
	}
}

func compiledRegistrationTestDocument(
	t *testing.T,
	locator spec.Locator,
	name spec.LogicalName,
) CompiledDocument {
	t.Helper()

	admitted, err := definition.Admit(definitionModel.Definition{
		Kind:          artifactModel.ArtifactKind("tool"),
		SchemaID:      "test",
		SchemaVersion: "v1",
		LogicalName:   name,
		DisplayName:   string(name),
		Body:          []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("admit test definition: %v", err)
	}

	return CompiledDocument{
		Locator: locator,
		Digest:  cryptoutil.DigestBytes([]byte(locator)),
		Artifacts: []CompiledArtifact{{
			Definition: admitted,
		}},
	}
}
