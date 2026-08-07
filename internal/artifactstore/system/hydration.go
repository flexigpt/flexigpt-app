package system

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
)

func (c *Components) GetTopologyHydration(
	ctx context.Context,
	installerName string,
) (topology.Hydration, bool, error) {
	if c == nil || c.metadata == nil {
		return topology.Hydration{}, false, basespec.ErrClosed
	}
	return c.metadata.GetTopologyHydration(ctx, installerName)
}

func (c *Components) PutTopologyHydration(
	ctx context.Context,
	value topology.Hydration,
) error {
	if c == nil ||
		c.metadata == nil ||
		c.Roots == nil ||
		c.Sources == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if !c.isProtectedRoot(value.RootID) {
		return fmt.Errorf(
			"%w: topology hydration root %q is not currently protected",
			basespec.ErrProtected,
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
	rootID basespec.RootID,
) error {
	if c == nil ||
		c.metadata == nil ||
		c.content == nil ||
		c.managedSources == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology reset context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	if err := topology.ValidateHydrationInstallerName(installerName); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
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
			basespec.ErrProtected,
			installerName,
			rootID,
		)
	}

	// Remove source-side package data first. If this fails, metadata remains
	// intact and the next startup can retry safely.
	if err := c.managedSources.RemoveManagedRoot(ctx, rootID); err != nil {
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

	if err := c.content.RemoveRoot(ctx, rootID); err != nil {
		return fmt.Errorf(
			"purge immutable definitions for topology root %q: %w",
			rootID,
			err,
		)
	}
	return nil
}
