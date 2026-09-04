package protection

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// SetRootPolicy supports multiple protected topology Roots and multiple
// retained application Roots.
//
// Protected Roots reject ordinary descendant mutations. Retained Roots reject
// only Root retirement and purge.
type SetRootPolicy struct {
	protected map[basespec.RootID]struct{}
	retained  map[basespec.RootID]struct{}
}

func NewSetRootPolicy(
	protected []basespec.RootID,
	retained []basespec.RootID,
) (*SetRootPolicy, error) {
	value := &SetRootPolicy{
		protected: make(map[basespec.RootID]struct{}, len(protected)),
		retained:  make(map[basespec.RootID]struct{}, len(retained)),
	}

	for _, rootID := range protected {
		if err := basespec.ValidateRootID(rootID); err != nil {
			return nil, fmt.Errorf("protected root: %w", err)
		}
		value.protected[rootID] = struct{}{}
	}
	for _, rootID := range retained {
		if err := basespec.ValidateRootID(rootID); err != nil {
			return nil, fmt.Errorf("retained root: %w", err)
		}
		value.retained[rootID] = struct{}{}
	}
	return value, nil
}

func (p *SetRootPolicy) IsProtectedRoot(
	rootID basespec.RootID,
) bool {
	if p == nil {
		return false
	}
	_, found := p.protected[rootID]
	return found
}

func (p *SetRootPolicy) IsRootDeletionProtected(
	rootID basespec.RootID,
) bool {
	if p == nil {
		return false
	}
	_, found := p.retained[rootID]
	return found
}
