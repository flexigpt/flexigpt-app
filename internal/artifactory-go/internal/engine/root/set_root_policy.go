package rootimpl

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
)

// SetRootPolicy supports multiple protected topology Roots and multiple
// retained application Roots.
//
// Protected Roots reject ordinary descendant mutations. Retained Roots reject
// only Root retirement and purge.
type SetRootPolicy struct {
	protected map[root.RootID]struct{}
	retained  map[root.RootID]struct{}
}

func NewSetRootPolicy(
	protected []root.RootID,
	retained []root.RootID,
) (*SetRootPolicy, error) {
	value := &SetRootPolicy{
		protected: make(map[root.RootID]struct{}, len(protected)),
		retained:  make(map[root.RootID]struct{}, len(retained)),
	}

	for _, rootID := range protected {
		if err := rootID.Validate(); err != nil {
			return nil, fmt.Errorf("protected root: %w", err)
		}
		value.protected[rootID] = struct{}{}
	}
	for _, rootID := range retained {
		if err := rootID.Validate(); err != nil {
			return nil, fmt.Errorf("retained root: %w", err)
		}
		value.retained[rootID] = struct{}{}
	}
	return value, nil
}

func (p *SetRootPolicy) IsProtectedRoot(
	rootID root.RootID,
) bool {
	if p == nil {
		return false
	}
	_, found := p.protected[rootID]
	return found
}

func (p *SetRootPolicy) IsRootDeletionProtected(
	rootID root.RootID,
) bool {
	if p == nil {
		return false
	}
	_, found := p.retained[rootID]
	return found
}
