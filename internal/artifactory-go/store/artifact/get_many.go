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
	for index, value := range values {
		if err := value.ValidateRead(); err != nil {
			return nil, fmt.Errorf("%w: artifact repository returned invalid Artifact: %w", spec.ErrInvalid, err)
		}
		if value.Ref() != owned[index] {
			return nil, fmt.Errorf("%w: artifact repository did not preserve requested ordering", spec.ErrInvalid)
		}
	}
	// Repository values are contractually independently owned. Returning them
	// directly avoids a second deep copy and JSON canonicalization on complete
	// Artifact batch reads.
	return values, nil
}
