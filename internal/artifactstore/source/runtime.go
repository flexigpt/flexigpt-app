package source

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// Runtime is a trusted internal capability for consumers that need an
// operational source, including its normalized adapter configuration.
//
// It is intentionally separate from Service, whose query methods return
// Summary values and do not expose opaque source configuration.
type Runtime interface {
	Get(
		ctx context.Context,
		rootID basespec.RootID,
		id basespec.SourceID,
	) (Source, error)

	Open(
		ctx context.Context,
		value Source,
	) (Snapshot, error)
}

// LocalPathRuntime is an optional extension implemented by the trusted source
// runtime when its opener supports LocalPathResolver.
//
// It intentionally accepts a full Source rather than a public Summary because
// source configuration remains internal to Artifact Store consumers.
type LocalPathRuntime interface {
	ResolveLocalPath(
		ctx context.Context,
		value Source,
		locator basespec.Locator,
	) (string, error)

	SupportsLocalPath(
		kind basespec.SourceKind,
	) bool
}

type runtime struct {
	reader     Reader
	opener     Opener
	localPaths LocalPathResolver
	localKinds LocalPathCapability
}

func NewRuntime(
	reader Reader,
	opener Opener,
) (Runtime, error) {
	if reader == nil || opener == nil {
		return nil, fmt.Errorf(
			"%w: source runtime dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	value := &runtime{
		reader: reader,
		opener: opener,
	}
	if resolver, supported := opener.(LocalPathResolver); supported {
		value.localPaths = resolver
	}
	if capabilities, supported := opener.(LocalPathCapability); supported {
		value.localKinds = capabilities
	}
	return value, nil
}

func (r *runtime) Get(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
) (Source, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Source{}, err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
		return Source{}, err
	}
	value, err := r.reader.Get(ctx, rootID, id)
	if err != nil {
		return Source{}, err
	}
	if value.ID != id {
		return Source{}, fmt.Errorf(
			"%w: source reader returned %q for requested source %q",
			basespec.ErrInvalid,
			value.ID,
			id,
		)
	}
	if value.RootID != rootID {
		return Source{}, fmt.Errorf(
			"%w: source reader returned root %q for requested root %q",
			basespec.ErrInvalid,
			value.RootID,
			rootID,
		)
	}
	if err := value.Validate(); err != nil {
		return Source{}, fmt.Errorf("invalid source returned by runtime reader: %w", err)
	}
	return value.Clone(), nil
}

func (r *runtime) Open(
	ctx context.Context,
	value Source,
) (Snapshot, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	snapshot, err := r.opener.Open(ctx, value.Clone())
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot); err != nil {
		_ = snapshot.Close()
		return nil, err
	}
	return snapshot, nil
}

func (r *runtime) SupportsLocalPath(
	kind basespec.SourceKind,
) bool {
	if r == nil || r.localKinds == nil {
		return false
	}
	return r.localKinds.SupportsLocalPath(kind)
}

// ResolveLocalPath delegates only to adapters that explicitly support native
// filesystem paths. Non-filesystem sources remain source-backed but do not
// become path-backed implicitly.
func (r *runtime) ResolveLocalPath(
	ctx context.Context,
	value Source,
	locator basespec.Locator,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("%w: source local-path context is nil", basespec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	if err := basespec.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	if r.localPaths == nil ||
		!r.SupportsLocalPath(value.Kind) {
		return "", fmt.Errorf(
			"%w: source runtime has no native path resolver",
			basespec.ErrUnsupported,
		)
	}
	location, err := r.localPaths.ResolveLocalPath(
		ctx,
		value.Clone(),
		locator,
	)
	if err != nil {
		return "", err
	}
	location = filepath.Clean(location)
	if !filepath.IsAbs(location) {
		return "", fmt.Errorf(
			"%w: source runtime returned a non-absolute local path",
			basespec.ErrInvalid,
		)
	}
	return location, nil
}

func validateSnapshot(snapshot Snapshot) error {
	if snapshot == nil {
		return fmt.Errorf("%w: source opener returned a nil snapshot", basespec.ErrInvalid)
	}
	if err := basespec.ValidateSourceGeneration(snapshot.Generation()); err != nil {
		return fmt.Errorf(
			"%w: source snapshot returned an invalid generation: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return nil
}
