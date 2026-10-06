package refresh

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

// RegisterCompiledDocuments accepts only trusted installer work and passes
// Ingest-owned source witnesses to the owner-local scanner implementation.
func (s *Service) RegisterCompiledDocuments(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	documents []ingest.CompiledDocument,
) error {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return err
	}
	return s.compiled.RegisterCompiledDocuments(ctx, rootID, sourceID, documents)
}
