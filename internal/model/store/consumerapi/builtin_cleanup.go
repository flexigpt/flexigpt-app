package consumerapi

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/source"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
)

// BuiltinArtifactSnapshot records the identity and kind of an Artifact that
// existed before one generated package hydration plan mutates package bytes.
type BuiltinArtifactSnapshot struct {
	Ref  artifact.ArtifactRef
	Kind artifact.ArtifactKind
}

// BuiltinPackageCleanup is the narrow lifecycle port used by the generated
// Model catalog installer.
//
// It owns Model overlay cleanup and one-time built-in enablement initialization.
// It does not publish package bytes, refresh Sources, compile declarations, or
// mutate protected topology itself.
type BuiltinPackageCleanup interface {
	CaptureBuiltInPackageArtifacts(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		addresses []source.ManagedPackageAddress,
	) ([]BuiltinArtifactSnapshot, error)

	ReconcileBuiltInPackageArtifacts(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		addresses []source.ManagedPackageAddress,
		previous []BuiltinArtifactSnapshot,
	) error
}

type builtinPackageCleanup struct {
	api *API
}

func NewBuiltinPackageCleanup(
	api *API,
) (BuiltinPackageCleanup, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Model built-in package cleanup requires a Model Store API",
			basespec.ErrInvalid,
		)
	}
	return &builtinPackageCleanup{api: api}, nil
}

func (c *builtinPackageCleanup) CaptureBuiltInPackageArtifacts(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	addresses []source.ManagedPackageAddress,
) ([]BuiltinArtifactSnapshot, error) {
	if c == nil || c.api == nil {
		return nil, basespec.ErrClosed
	}
	if !documentTopology.IsBuiltinPackageSource(rootID, sourceID) {
		return nil, fmt.Errorf(
			"%w: Model cleanup does not target the built-in package Source",
			basespec.ErrProtected,
		)
	}

	scopes, err := managedPackageScopes(addresses)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return []BuiltinArtifactSnapshot{}, nil
	}

	entries, err := c.api.artifacts.ListBySource(
		ctx,
		rootID,
		sourceID,
		catalog.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	output := make([]BuiltinArtifactSnapshot, 0)
	seen := make(map[artifact.ArtifactRef]struct{})
	for _, entry := range entries {
		if entry.State != artifact.StateAvailable ||
			!modelBuiltinArtifactKind(entry.Kind) ||
			!entryBelongsToPackageScope(entry, scopes) {
			continue
		}

		if _, duplicate := seen[entry.Ref()]; duplicate {
			continue
		}
		seen[entry.Ref()] = struct{}{}
		output = append(output, BuiltinArtifactSnapshot{
			Ref:  entry.Ref(),
			Kind: entry.Kind,
		})
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Ref.RootID != output[right].Ref.RootID {
			return output[left].Ref.RootID < output[right].Ref.RootID
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}

func (c *builtinPackageCleanup) ReconcileBuiltInPackageArtifacts(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	addresses []source.ManagedPackageAddress,
	previous []BuiltinArtifactSnapshot,
) error {
	if c == nil || c.api == nil {
		return basespec.ErrClosed
	}
	if !documentTopology.IsBuiltinPackageSource(rootID, sourceID) {
		return fmt.Errorf(
			"%w: Model cleanup does not target the built-in package Source",
			basespec.ErrProtected,
		)
	}

	scopes, err := managedPackageScopes(addresses)
	if err != nil {
		return err
	}

	current, err := c.CaptureBuiltInPackageArtifacts(
		ctx,
		rootID,
		sourceID,
		addresses,
	)
	if err != nil {
		return err
	}

	previousByRef := make(
		map[artifact.ArtifactRef]BuiltinArtifactSnapshot,
		len(previous),
	)
	for _, value := range previous {
		previousByRef[value.Ref] = value
	}

	currentByRef := make(
		map[artifact.ArtifactRef]BuiltinArtifactSnapshot,
		len(current),
	)
	for _, value := range current {
		currentByRef[value.Ref] = value
	}

	var result error

	for _, value := range current {
		if _, existed := previousByRef[value.Ref]; existed {
			continue
		}
		result = errors.Join(
			result,
			c.applyInitialBuiltInEnablement(ctx, value.Ref),
		)
	}

	for _, value := range previous {
		if _, stillCurrent := currentByRef[value.Ref]; stillCurrent {
			continue
		}
		result = errors.Join(
			result,
			c.purgeRemovedBuiltInArtifactOverlay(ctx, value),
		)
	}

	// Stale hydration records can identify package scopes that have already
	// disappeared before this process started. Listing all current Source
	// entries is intentionally enough; no Source mutation is attempted here.
	_ = scopes

	return result
}

func (c *builtinPackageCleanup) applyInitialBuiltInEnablement(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	record, err := c.api.artifacts.Get(ctx, ref)
	if err != nil {
		if errors.Is(err, basespec.ErrArtifactNotFound) ||
			errors.Is(err, basespec.ErrRootNotFound) {
			return nil
		}
		return err
	}
	if record.State != artifact.StateAvailable {
		return nil
	}

	definitionValue, err := c.api.artifacts.GetDefinition(ctx, ref)
	if err != nil {
		return err
	}

	raw, declared := definitionValue.Labels[modelDomain.BuiltInInitialEnabledLabel]
	if !declared {
		return nil
	}

	var enabled bool
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		enabled = true
	case "false":
		enabled = false
	default:
		return fmt.Errorf(
			"%w: built-in Model initial enabled label %q is invalid",
			basespec.ErrInvalid,
			raw,
		)
	}

	if record.Enabled == enabled {
		return nil
	}
	_, err = c.api.artifacts.SetEnabled(
		ctx,
		ref,
		record.Revision,
		enabled,
	)
	return err
}

func (c *builtinPackageCleanup) purgeRemovedBuiltInArtifactOverlay(
	ctx context.Context,
	value BuiltinArtifactSnapshot,
) error {
	record, err := c.api.artifacts.Get(ctx, value.Ref)
	if err != nil &&
		!errors.Is(err, basespec.ErrArtifactNotFound) &&
		!errors.Is(err, basespec.ErrRootNotFound) {
		return err
	}
	if err == nil && record.State == artifact.StateAvailable {
		return nil
	}

	switch value.Kind {
	case modelDomain.ModelProviderArtifactKind:
		return c.api.purgeProviderLocalState(ctx, value.Ref)
	case modelDomain.ModelArtifactKind:
		return c.api.purgeModelLocalState(ctx, value.Ref)
	default:
		return nil
	}
}

func managedPackageScopes(
	addresses []source.ManagedPackageAddress,
) (map[basespec.Locator]struct{}, error) {
	output := make(map[basespec.Locator]struct{}, len(addresses))
	for _, address := range addresses {
		if address.Kind != modelDomain.ModelProviderPackageKind &&
			address.Kind != modelDomain.ModelPackageKind {
			continue
		}
		scope, err := address.Directory()
		if err != nil {
			return nil, err
		}
		output[scope] = struct{}{}
	}
	return output, nil
}

func entryBelongsToPackageScope(
	entry catalog.Entry,
	scopes map[basespec.Locator]struct{},
) bool {
	if len(scopes) == 0 {
		return false
	}
	scope := basespec.Locator(path.Dir(string(entry.Binding.Locator)))
	_, found := scopes[scope]
	return found
}

func modelBuiltinArtifactKind(
	kind artifact.ArtifactKind,
) bool {
	return kind == modelDomain.ModelProviderArtifactKind ||
		kind == modelDomain.ModelArtifactKind
}
