package collection

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
)

func TestReadOnlyDomainDoesNotRequireAuthoringConfiguration(t *testing.T) {
	policy := DomainPolicy{
		Name:        "tool",
		ReadOnly:    true,
		PackageKind: "tool-collection",
		DocumentUse: documentTopology.DocumentUseToolCollection,
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeTool,
		},
		AllowedMemberForms: []declaration.MemberForm{
			declaration.MemberNamed,
		},
	}
	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestReadOnlyDomainRejectsAuthoringBeforeStoreAccess(t *testing.T) {
	api := &API{
		domain: &DomainPolicy{
			Name:     "tool",
			ReadOnly: true,
		},
	}
	ctx := t.Context()

	if _, err := api.Create(ctx, CreateRequest{}); !errors.Is(err, model.ErrUnsupported) {
		t.Fatalf("Create error = %v", err)
	}
	if _, err := api.EnsureBaseline(ctx, ""); !errors.Is(err, model.ErrUnsupported) {
		t.Fatalf("EnsureBaseline error = %v", err)
	}
	if _, err := api.loadEditableCollection(
		ctx,
		artifact.ArtifactRef{},
		1,
	); !errors.Is(err, model.ErrUnsupported) {
		t.Fatalf("loadEditableCollection error = %v", err)
	}
	if _, err := api.domainManagedSource(ctx, "", ""); !errors.Is(err, model.ErrUnsupported) {
		t.Fatalf("domainManagedSource error = %v", err)
	}
}
