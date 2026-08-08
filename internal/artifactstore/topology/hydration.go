package topology

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func ValidateHydrationInstallerName(value string) error {
	return basespec.ValidateIdentifier(
		"topology hydration installer name",
		value,
		basespec.MaxKindBytes,
	)
}

// Hydration records the successfully installed desired state for one
// application-owned topology installer. It intentionally has no foreign keys:
// the record must survive removal of an old Root until replacement hydration
// succeeds and commits a new record.
type Hydration struct {
	InstallerName string            `json:"installerName"`
	RootID        basespec.RootID   `json:"rootID"`
	SourceID      basespec.SourceID `json:"sourceID"`
	Fingerprint   cryptoutil.Digest `json:"fingerprint"`
}

func (h Hydration) Validate() error {
	if err := ValidateHydrationInstallerName(h.InstallerName); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(h.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(h.SourceID); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(h.Fingerprint); err != nil {
		return fmt.Errorf("topology hydration fingerprint: %w", err)
	}
	return nil
}

// HydrationCoordinator reconciles binary-owned topology state before and
// after artifact-family installation. Preparation is batched so one stale
// installer cannot invalidate another installer sharing the same protected
// Root after that installer has already been marked current.
//
// PrepareTopologyHydrations returns current state by installer name. A Root
// reset marks every installer using that Root as non-current.
type HydrationCoordinator interface {
	PrepareTopologyHydrations(
		ctx context.Context,
		desired []Hydration,
	) (currentByInstaller map[string]bool, err error)

	CommitTopologyHydration(
		ctx context.Context,
		desired Hydration,
	) error
}

// HydrationStore persists successful installer hydration state. It is an
// internal composition capability and is never exposed through public APIs.
type HydrationStore interface {
	GetTopologyHydration(
		ctx context.Context,
		installerName string,
	) (Hydration, bool, error)

	PutTopologyHydration(
		ctx context.Context,
		value Hydration,
	) error
}
