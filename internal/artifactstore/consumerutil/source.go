package consumerutil

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type DiscoveryReconciler func(
	current source.DiscoverySpec,
	desired source.DiscoverySpec,
) source.DiscoverySpec

type EnsureAndRefreshSourceRequest struct {
	RootID root.RootID
	Draft  source.Draft

	// ReconcileDiscovery preserves caller-owned dynamic discovery closure when
	// an existing Source is reused. A nil reconciler replaces discovery with
	// Draft.Discovery.
	ReconcileDiscovery DiscoveryReconciler
}

// EnsureAndRefreshSource owns the common Source lifecycle:
//
//	Ensure -> reconcile metadata and discovery -> RefreshSource
//
// Domain callers retain ownership of Source kind, config, display naming, and
// discovery policy.
func EnsureAndRefreshSource(
	ctx context.Context,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	request EnsureAndRefreshSourceRequest,
) (source.Summary, error) {
	if sources == nil || discovery == nil {
		return source.Summary{}, fmt.Errorf(
			"%w: Source lifecycle dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if ctx == nil {
		return source.Summary{}, fmt.Errorf(
			"%w: Source lifecycle context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return source.Summary{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return source.Summary{}, err
	}

	draft := request.Draft
	if !draft.Enabled || draft.Discovery.Empty() {
		return source.Summary{}, fmt.Errorf(
			"%w: Source lifecycle requires an enabled Source with discovery",
			basespec.ErrInvalid,
		)
	}
	draft.Discovery = draft.Discovery.Normalized()
	if err := draft.Discovery.Validate(); err != nil {
		return source.Summary{}, err
	}

	summary, _, err := sources.Ensure(ctx, request.RootID, draft)
	if err != nil {
		return source.Summary{}, err
	}

	desired := draft.Discovery
	if request.ReconcileDiscovery != nil {
		desired = request.ReconcileDiscovery(
			summary.Discovery.Clone(),
			desired,
		)
	}
	desired = desired.Normalized()
	if err := desired.Validate(); err != nil {
		return source.Summary{}, err
	}

	if !summary.Enabled ||
		summary.DisplayName != draft.DisplayName ||
		!summary.Discovery.Equal(desired) {
		summary, err = sources.Update(
			ctx,
			request.RootID,
			summary.ID,
			source.Update{
				ExpectedRevision: summary.Revision,
				DisplayName:      draft.DisplayName,
				Enabled:          true,
				Discovery:        &desired,
			},
		)
		if err != nil {
			return source.Summary{}, err
		}
	}

	if _, err := discovery.RefreshSource(
		ctx,
		request.RootID,
		summary.ID,
	); err != nil {
		return source.Summary{}, err
	}
	return summary, nil
}

func NormalizeFilesystemSourceRoot(
	raw string,
	label string,
) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%w: %s is required", basespec.ErrInvalid, label)
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func FilesystemSourceStorageKey(
	prefix string,
	rootPath string,
) basespec.StorageKey {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(rootPath))),
		cryptoutil.DigestSHA256Prefix,
	)
	return basespec.StorageKey(prefix + "-" + digest[:24])
}
