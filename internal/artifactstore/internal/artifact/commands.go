package artifactimpl

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// SourceStateUpdate is an internal-refresh publication value. It contains
// only source-derived Artifact fields and cannot modify caller-owned local
// Artifact state such as name, enabled state, or local data.
type SourceStateUpdate struct {
	ArtifactID         basespec.ArtifactID
	RootID             basespec.RootID
	CollectionID       basespec.CollectionID
	ResolvedDefinition *cryptoutil.Digest
	State              artifact.State
	Diagnostics        []providerapi.Diagnostic
	Revision           uint64
	ModifiedAt         time.Time
	ExpectedRevision   uint64
}

func (u SourceStateUpdate) Validate() error {
	if err := basespec.ValidateArtifactID(u.ArtifactID); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(u.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionID(u.CollectionID); err != nil {
		return err
	}
	if err := artifact.ValidateSourceState(u.State, u.ResolvedDefinition); err != nil {
		return err
	}
	if err := providerapi.ValidateDiagnostics(u.Diagnostics); err != nil {
		return err
	}
	if u.ExpectedRevision == 0 ||
		u.Revision != u.ExpectedRevision+1 ||
		u.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: invalid source-derived artifact update",
			basespec.ErrInvalid,
		)
	}
	return nil
}
