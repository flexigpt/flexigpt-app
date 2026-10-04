package artifact

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// GetMany loads complete Artifact entities in caller order. Catalog consumers
// should use catalog.API unless they explicitly need full local state.
func (s *Service) GetMany(ctx context.Context, refs []artifactModel.ArtifactRef) ([]artifactModel.Artifact, error) {
	if s == nil || s.repository == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: artifact batch get context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return []artifactModel.Artifact{}, nil
	}
	owned := append([]artifactModel.ArtifactRef(nil), refs...)
	for index, ref := range owned {
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("artifact refs[%d]: %w", index, err)
		}
	}
	values, err := s.repository.GetMany(ctx, owned)
	if err != nil {
		return nil, err
	}
	if len(values) != len(owned) {
		return nil, fmt.Errorf(
			"%w: artifact repository returned %d values for %d refs",
			spec.ErrInvalid,
			len(values),
			len(owned),
		)
	}
	output := make([]artifactModel.Artifact, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("%w: artifact repository returned invalid Artifact: %w", spec.ErrInvalid, err)
		}
		if value.Ref() != owned[index] {
			return nil, fmt.Errorf("%w: artifact repository did not preserve requested ordering", spec.ErrInvalid)
		}
		output[index] = value.Clone()
	}
	return output, nil
}
