package aggregatecatalog

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

func TestCollectionCatalogIDRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		ref  collection.CollectionRef
	}{
		{
			name: "workspace collection",
			ref: collection.CollectionRef{
				RootID:       "0198f097-0d5b-7000-8000-000000000001",
				CollectionID: "0198f097-0d5b-7000-8000-000000000010",
			},
		},
		{
			name: "built in collection",
			ref: collection.CollectionRef{
				RootID:       "0192c4c0-0000-7000-8000-000000000001",
				CollectionID: "0198f097-0d5b-7000-8000-000000000020",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalogID, err := CollectionCatalogID(test.ref)
			if err != nil {
				t.Fatalf("CollectionCatalogID: %v", err)
			}

			actual, err := collectionRefForCatalogID(catalogID)
			if err != nil {
				t.Fatalf("collectionRefForCatalogID: %v", err)
			}
			if actual != test.ref {
				t.Fatalf("collection ref=%#v, want %#v", actual, test.ref)
			}
		})
	}
}

func TestCatalogSourceRejectsMalformedCatalogIDs(t *testing.T) {
	source := &CatalogSource{
		router: &skillStore.ArtifactRouter{},
	}

	tests := []struct {
		name      string
		catalogID skillRuntime.CatalogID
	}{
		{
			name:      "empty",
			catalogID: "",
		},
		{
			name:      "unknown prefix",
			catalogID: "another-source:catalog",
		},
		{
			name:      "missing collection separator",
			catalogID: "artifact-collection:0198f097-0d5b-7000-8000-000000000001",
		},
		{
			name: "invalid root",
			catalogID: skillRuntime.CatalogID(
				"artifact-collection:not-a-root:0198f097-0d5b-7000-8000-000000000010",
			),
		},
		{
			name: "invalid collection",
			catalogID: skillRuntime.CatalogID(
				"artifact-collection:0198f097-0d5b-7000-8000-000000000001:not-a-collection",
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := source.Skills(t.Context(), test.catalogID)
			if err == nil {
				t.Fatalf("Skills(%q) succeeded", test.catalogID)
			}
		})
	}
}

func TestCollectionCatalogIDRejectsInvalidCollectionRef(t *testing.T) {
	tests := []collection.CollectionRef{
		{},
		{
			RootID:       basespec.RootID("not-a-root"),
			CollectionID: "0198f097-0d5b-7000-8000-000000000010",
		},
	}

	for _, ref := range tests {
		if _, err := CollectionCatalogID(ref); err == nil {
			t.Fatalf("CollectionCatalogID(%#v) succeeded", ref)
		}
	}
}
