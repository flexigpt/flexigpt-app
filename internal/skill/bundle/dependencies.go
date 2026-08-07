package bundle

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type RootLister interface {
	List(ctx context.Context) ([]root.Root, error)
}

type ManagedSourceStateFunc func(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
) (source.Summary, string, error)

type PublishManagedPackageFunc func(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	publication source.ManagedPackagePublication,
) (source.Summary, string, error)

type RemoveManagedPackageFunc func(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	directory basespec.Locator,
	expectedGeneration string,
) (source.Summary, string, error)

type Dependencies struct {
	Roots                  RootLister
	Sources                *source.Service
	Collections            *collection.Service
	Artifacts              *artifact.Service
	Refresh                refresh.Runner
	Catalogs               catalog.Reader
	Definitions            definition.Reader
	SourceRuntime          source.Runtime
	HasDecoder             func(basespec.DecoderID) bool
	DecoderFingerprint     func() (cryptoutil.Digest, error)
	RootMutationPolicy     protection.RootPolicy
	AutoAdoptionIDProvider artifact.ArtifactIDProvider

	GetManagedSourceState          ManagedSourceStateFunc
	PublishManagedPackage          PublishManagedPackageFunc
	PublishProtectedManagedPackage PublishManagedPackageFunc
	RemoveManagedPackage           RemoveManagedPackageFunc
}

func (d Dependencies) Validate() error {
	if d.Roots == nil ||
		d.Sources == nil ||
		d.Collections == nil ||
		d.Artifacts == nil ||
		d.Refresh == nil ||
		d.Catalogs == nil ||
		d.Definitions == nil ||
		d.SourceRuntime == nil ||
		d.HasDecoder == nil ||
		d.DecoderFingerprint == nil ||
		d.RootMutationPolicy == nil ||
		d.AutoAdoptionIDProvider == nil ||
		d.GetManagedSourceState == nil ||
		d.PublishManagedPackage == nil ||
		d.PublishProtectedManagedPackage == nil ||
		d.RemoveManagedPackage == nil {
		return fmt.Errorf(
			"%w: skill bundle Artifact Store dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return nil
}
