package catalog

import (
	"context"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type API interface {
	ListByRoot(
		ctx context.Context,
		rootID rootModel.RootID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)

	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)

	FindByIdentity(
		ctx context.Context,
		rootID rootModel.RootID,
		kind artifactModel.ArtifactKind,
		logicalName spec.LogicalName,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)
}
