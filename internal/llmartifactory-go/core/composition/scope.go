package composition

import (
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

// ScopeBinding is application-supplied composition scope meaning. The portable
// builtin token is never treated as a Root ID in declaration grammar.
type ScopeBinding struct {
	BuiltinRoot rootModel.RootID
}

func (s ScopeBinding) Validate() error {
	if s.BuiltinRoot == "" {
		return nil
	}
	if err := s.BuiltinRoot.Validate(); err != nil {
		return fmt.Errorf("built-in composition Root: %w", err)
	}
	return nil
}
