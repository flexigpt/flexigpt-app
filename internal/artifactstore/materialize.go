package artifactstore

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

// MaterializeSource produces a stable real-directory projection for consumers
// that cannot operate on fs.FS, such as file-oriented runtime libraries.
func (s *Store) MaterializeSource(
	ctx context.Context,
	request spec.MaterializeSourceRequest,
) (spec.MaterializedSource, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	defer finish()

	if s.sourceMaterializer == nil {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: source materializer is not configured",
			spec.ErrMaterializerUnavailable,
		)
	}
	source, err := s.repository.GetSource(ctx, request.SourceID)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	if !source.Enabled {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: source %q is disabled",
			spec.ErrConflict,
			request.SourceID,
		)
	}
	driver, ok := s.driverFor(source.Kind)
	if !ok {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: source kind %q",
			spec.ErrDriverUnavailable,
			source.Kind,
		)
	}
	root := request.Root
	if root == "" {
		root = "."
	}
	result, err := s.sourceMaterializer.Materialize(
		ctx,
		spec.SourceMaterializationInput{
			Source:         source,
			Driver:         driver,
			Root:           root,
			PublicationKey: request.PublicationKey,
			MaxEntries:     request.MaxEntries,
			MaxFiles:       request.MaxFiles,
			MaxBytes:       request.MaxBytes,
		},
	)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	if result.Generation == "" || result.RootPath == "" {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: source materializer returned an incomplete result",
			spec.ErrMaterializerUnavailable,
		)
	}
	confirmed, err := s.repository.GetSource(ctx, request.SourceID)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	if !confirmed.Enabled ||
		confirmed.Kind != source.Kind ||
		confirmed.ObservationRevision != source.ObservationRevision ||
		!equivalentJSONObjects(confirmed.Config, source.Config) {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: source %q changed while it was being materialized",
			spec.ErrConflict,
			request.SourceID,
		)
	}
	return result, nil
}
