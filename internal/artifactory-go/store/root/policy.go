package root

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// SetRootPolicy supports protected topology Roots and retained application
// Roots. Protected Roots reject ordinary descendant mutation. Retained Roots
// reject only retirement and purge.
type SetRootPolicy struct {
	protected map[rootModel.RootID]struct{}
	retained  map[rootModel.RootID]struct{}
}

func NewSetRootPolicy(
	protected []rootModel.RootID,
	retained []rootModel.RootID,
) (*SetRootPolicy, error) {
	value := &SetRootPolicy{
		protected: make(map[rootModel.RootID]struct{}, len(protected)),
		retained:  make(map[rootModel.RootID]struct{}, len(retained)),
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

func (p *SetRootPolicy) IsProtectedRoot(rootID rootModel.RootID) bool {
	if p == nil {
		return false
	}
	_, found := p.protected[rootID]
	return found
}

func (p *SetRootPolicy) IsRootDeletionProtected(rootID rootModel.RootID) bool {
	if p == nil {
		return false
	}
	_, found := p.retained[rootID]
	return found
}

func RequireMutableRoot(
	ctx context.Context,
	policy Policy,
	rootID rootModel.RootID,
) error {
	if policy == nil || !policy.IsProtectedRoot(rootID) {
		return nil
	}
	if IsInstallerPrivileged(ctx) {
		return nil
	}
	return fmt.Errorf(
		"%w: root %q may only be mutated by a trusted protected-topology installer",
		spec.ErrProtected,
		rootID,
	)
}

func RequireRootDeletion(
	ctx context.Context,
	policy Policy,
	rootID rootModel.RootID,
) error {
	if deletionPolicy, supported := policy.(DeletionPolicy); supported &&
		deletionPolicy.IsRootDeletionProtected(rootID) {
		return fmt.Errorf(
			"%w: root %q is retained and cannot be retired or purged",
			spec.ErrProtected,
			rootID,
		)
	}
	return RequireMutableRoot(ctx, policy, rootID)
}
