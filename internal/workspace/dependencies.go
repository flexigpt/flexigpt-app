package workspace

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

// RootLister is the narrow Root capability required to rebuild Workspace
// runtime registrations during application startup.
type RootLister interface {
	List(ctx context.Context) ([]root.Root, error)
}

// Dependencies is the application-composition boundary for Workspace.
// Workspace deliberately depends on Artifact Store ports and services rather
// than the concrete Artifact Store system container.
type Dependencies struct {
	Roots              RootLister
	Sources            *source.Service
	Collections        *collection.Service
	Artifacts          *artifact.Service
	Refresh            refresh.Runner
	Catalogs           catalog.Reader
	Definitions        definition.Reader
	SourceRuntime      source.Runtime
	HasDecoder         func(basespec.DecoderID) bool
	DecoderFingerprint func() (cryptoutil.Digest, error)
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
		d.DecoderFingerprint == nil {
		return fmt.Errorf(
			"%w: Workspace Artifact Store dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}
