package assembly

import (
	"context"
	"errors"
	"fmt"
	"slices"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// PrepareTopologyHydrations reconciles all installer desired states before
// any installer creates topology. It must operate as one batch because
// multiple artifact families can share one protected Root.
func (c *Components) PrepareTopologyHydrations(
	ctx context.Context,
	desiredValues []installModel.Hydration,
) (map[string]bool, error) {
	if c == nil || c.metadata == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: topology hydration context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return nil, err
	}

	currentByInstaller := make(map[string]bool, len(desiredValues))
	seenInstallers := make(map[string]struct{}, len(desiredValues))
	resetInstallerByRoot := make(map[rootModel.RootID]string)

	for _, desired := range desiredValues {
		if err := desired.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seenInstallers[desired.InstallerName]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate topology hydration installer %q",
				spec.ErrInvalid,
				desired.InstallerName,
			)
		}
		seenInstallers[desired.InstallerName] = struct{}{}
		if !c.isProtectedRoot(desired.RootID) {
			return nil, fmt.Errorf(
				"%w: topology hydration root %q is not protected",
				spec.ErrProtected,
				desired.RootID,
			)
		}

		previous, found, err := c.GetTopologyHydration(
			ctx,
			desired.InstallerName,
		)
		if err != nil {
			return nil, err
		}
		if found && equalTopologyHydration(previous, desired) {
			currentByInstaller[desired.InstallerName] = true
			continue
		}

		currentByInstaller[desired.InstallerName] = false
		resetInstallerByRoot[desired.RootID] = desired.InstallerName
		if found {
			resetInstallerByRoot[previous.RootID] = desired.InstallerName
		}
	}

	// A reset invalidates every installer that uses the reset Root, including
	// installers whose persisted hydration marker individually matched.
	for _, desired := range desiredValues {
		if _, reset := resetInstallerByRoot[desired.RootID]; reset {
			currentByInstaller[desired.InstallerName] = false
		}
	}

	orderedRoots := make([]rootModel.RootID, 0, len(resetInstallerByRoot))
	for rootID := range resetInstallerByRoot {
		orderedRoots = append(orderedRoots, rootID)
	}
	slices.Sort(orderedRoots)

	for _, rootID := range orderedRoots {
		if err := c.ResetTopologyHydration(
			ctx,
			resetInstallerByRoot[rootID],
			rootID,
		); err != nil {
			return nil, fmt.Errorf(
				"reset stale topology hydration root %q: %w",
				rootID,
				err,
			)
		}
	}
	return currentByInstaller, nil
}

func (c *Components) GetTopologyHydration(
	ctx context.Context,
	installerName string,
) (installModel.Hydration, bool, error) {
	if c == nil || c.metadata == nil {
		return installModel.Hydration{}, false, spec.ErrClosed
	}
	return c.metadata.GetTopologyHydration(ctx, installerName)
}

// CommitTopologyHydration records successful installation only after the
// generic topology and artifact-family installation paths both complete.
func (c *Components) CommitTopologyHydration(
	ctx context.Context,
	desired installModel.Hydration,
) error {
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return err
	}
	return c.PutTopologyHydration(ctx, desired)
}

func (c *Components) PutTopologyHydration(
	ctx context.Context,
	value installModel.Hydration,
) error {
	if c == nil ||
		c.metadata == nil ||
		c.Roots == nil ||
		c.Sources == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology hydration context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if !c.isProtectedRoot(value.RootID) {
		return fmt.Errorf(
			"%w: topology hydration root %q is not currently protected",
			spec.ErrProtected,
			value.RootID,
		)
	}
	if _, err := c.Roots.Get(ctx, value.RootID); err != nil {
		return err
	}
	if _, err := c.Sources.Get(ctx, value.RootID, value.SourceID); err != nil {
		return err
	}
	return c.metadata.PutTopologyHydration(ctx, value)
}

func equalTopologyHydration(
	left installModel.Hydration,
	right installModel.Hydration,
) bool {
	return left.InstallerName == right.InstallerName &&
		left.RootID == right.RootID &&
		left.SourceID == right.SourceID &&
		left.Fingerprint == right.Fingerprint
}

// ResetTopologyHydration removes all local state for a binary-owned topology.
//
// A reset is authorized only when rootID is the current protected Root or the
// persisted hydration record for installerName owns rootID. The second case is
// required when a newer binary changes its protected Root ID.
//
// The hydration record intentionally survives this method. It is replaced only
// after the next complete installation succeeds, making interrupted upgrades
// converge on the next application start.
func (c *Components) ResetTopologyHydration(
	ctx context.Context,
	installerName string,
	rootID rootModel.RootID,
) error {
	if c == nil ||
		c.metadata == nil ||
		c.Roots == nil ||
		c.managedSources == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology reset context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := installModel.ValidateHydrationInstallerName(installerName); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}

	stored, found, err := c.metadata.GetTopologyHydration(
		ctx,
		installerName,
	)
	if err != nil {
		return err
	}
	ownedByStoredHydration := found &&
		stored.InstallerName == installerName &&
		stored.RootID == rootID
	if !c.isProtectedRoot(rootID) && !ownedByStoredHydration {
		return fmt.Errorf(
			"%w: topology hydration installer %q does not own root %q",
			spec.ErrProtected,
			installerName,
			rootID,
		)
	}

	// Remove source-side package data first. If this fails, metadata remains
	// intact and the next startup can retry safely.
	rootValue, err := c.Roots.Get(ctx, rootID)
	if errors.Is(err, spec.ErrRootNotFound) {
		// Preparation runs before EnsureProtectedTopology. Therefore, a
		// missing protected Root is the normal first-install state. It is
		// also valid after a clean hydration reset completed and a process
		// stopped before topology creation and hydration-marker commit.
		//
		// Do not create topology here. EnsureProtectedTopology owns that
		// step after all stale roots have been reset as one batch.
		//
		// It can also occur after a prior reset completed metadata purging
		// but the process stopped before the replacement topology was
		// installed and its hydration marker was committed. Source-side
		// storage is removed before metadata in the reset sequence, so this
		// is already a clean state and must converge without an error.
		return nil
	}
	if err != nil {
		return fmt.Errorf(
			"read topology root %q before content reset: %w",
			rootID,
			err,
		)
	}
	if err := c.managedSources.RemoveManagedRoot(
		ctx,
		rootValue.StorageKey,
	); err != nil {
		return fmt.Errorf(
			"remove managed source storage for topology root %q: %w",
			rootID,
			err,
		)
	}

	if err := c.metadata.PurgeTopologyRoot(ctx, rootID); err != nil {
		return fmt.Errorf(
			"purge metadata for topology root %q: %w",
			rootID,
			err,
		)
	}
	if c.LocalState != nil {
		_ = c.LocalState.DrainSecretGarbage(
			context.WithoutCancel(ctx),
		)
	}
	return nil
}
