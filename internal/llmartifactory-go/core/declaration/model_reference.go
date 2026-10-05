package declaration

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ArtifactNameReference is the portable identity shape shared by the Model
// provider relationship and the Provider best-effort default Model
// relationship.
//
// It intentionally carries no Artifact ID, source identity, revision, or
// cross-root locator. Scope is limited to the current Root lookup policy or
// the protected built-in Root.
type ArtifactNameReference struct {
	Name  spec.LogicalName `json:"name"`
	Scope LookupScope      `json:"scope,omitempty"`
}

func (r ArtifactNameReference) Validate() error {
	if err := r.Name.Validate(); err != nil {
		return fmt.Errorf("artifact name reference: %w", err)
	}
	if err := r.Scope.Validate(); err != nil {
		return fmt.Errorf("artifact name reference scope: %w", err)
	}
	return nil
}

func (r ArtifactNameReference) Clone() ArtifactNameReference {
	return r
}
