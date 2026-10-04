package source

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// readSnapshotLocator is Source-owned bounded snapshot evidence access. It is
// intentionally reused by generic generation/digest verification without
// recreating Resource's multi-entity session behavior.
func readSnapshotLocator(
	ctx context.Context,
	snapshot driver.Snapshot,
	locator spec.Locator,
	maximumBytes int64,
) ([]byte, error) {
	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return nil, err
	}
	if err := entry.Validate(); err != nil {
		return nil, fmt.Errorf("%w: source snapshot returned an invalid entry: %w", spec.ErrInvalid, err)
	}
	if entry.Locator != locator {
		return nil, fmt.Errorf("%w: source snapshot stat for %q returned %q", spec.ErrInvalid, locator, entry.Locator)
	}
	return ReadSnapshotEntry(ctx, snapshot, entry, maximumBytes)
}
