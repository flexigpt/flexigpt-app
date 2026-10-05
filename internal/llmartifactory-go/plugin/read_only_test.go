package plugin

import (
	"errors"
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func TestReadOnlyDomainRejectsAuthoringBeforeStoreAccess(t *testing.T) {
	api := &API{
		domain: &Profile{
			Name:     "tool",
			ReadOnly: true,
		},
	}
	ctx := t.Context()

	if _, err := api.Create(ctx, CreateRequest{}); !errors.Is(err, spec.ErrUnsupported) {
		t.Fatalf("Create error = %v", err)
	}
	if _, err := api.EnsureBaseline(ctx, ""); !errors.Is(err, spec.ErrUnsupported) {
		t.Fatalf("EnsureBaseline error = %v", err)
	}
	if _, err := api.loadEditablePlugin(
		ctx,
		artifactModel.ArtifactRef{},
		1,
	); !errors.Is(err, spec.ErrUnsupported) {
		t.Fatalf("loadEditablePlugin error = %v", err)
	}
	if _, err := api.domainManagedSource(ctx, "", ""); !errors.Is(err, spec.ErrUnsupported) {
		t.Fatalf("domainManagedSource error = %v", err)
	}
}
