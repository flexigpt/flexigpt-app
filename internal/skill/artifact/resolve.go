package artifact

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type runtimePackageResolutionSessionContextKey struct{}

type runtimePackageResolutionSessionKey struct {
	RootID         basespec.RootID
	SourceID       basespec.SourceID
	SourceRevision uint64
	SourceKind     basespec.SourceKind
	Generation     string
}

// RuntimePackageResolutionSession reuses confirmed source snapshots while a
// caller resolves multiple Agent Skill packages. Callers must Close it before
// returning any successfully resolved package locations to their caller.
type RuntimePackageResolutionSession struct {
	mu        sync.Mutex
	snapshots map[runtimePackageResolutionSessionKey]source.Snapshot
	closed    bool
}

func NewRuntimePackageResolutionSession(
	ctx context.Context,
) (context.Context, *RuntimePackageResolutionSession, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf(
			"%w: Skill runtime package session context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if runtimePackageResolutionSessionFromContext(ctx) != nil {
		return nil, nil, fmt.Errorf(
			"%w: Skill runtime package session already exists",
			basespec.ErrInvalid,
		)
	}

	session := &RuntimePackageResolutionSession{
		snapshots: make(
			map[runtimePackageResolutionSessionKey]source.Snapshot,
		),
	}
	return context.WithValue(
		ctx,
		runtimePackageResolutionSessionContextKey{},
		session,
	), session, nil
}

func (s *RuntimePackageResolutionSession) Close(
	ctx context.Context,
) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	snapshots := s.snapshots
	s.snapshots = nil
	s.mu.Unlock()

	var closeErr error
	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}
		closeErr = errors.Join(
			closeErr,
			snapshot.Confirm(ctx),
			snapshot.Close(),
		)
	}
	return closeErr
}

func runtimePackageResolutionSessionFromContext(
	ctx context.Context,
) *RuntimePackageResolutionSession {
	if ctx == nil {
		return nil
	}
	value, _ := ctx.Value(
		runtimePackageResolutionSessionContextKey{},
	).(*RuntimePackageResolutionSession)
	return value
}

// ResolveRuntimePackage converts a verified Skill source locator into the
// native package directory required by the Agent Skills filesystem provider.
//
// Source adapters retain ownership of native path resolution, containment, and
// platform-specific behavior. Agentskills-go retains ownership of SKILL.md
// parsing, resource access, script policy, sandboxing, and execution.
func ResolveRuntimePackage(
	ctx context.Context,
	runtime source.Runtime,
	value source.Source,
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	expectedGeneration string,
	expectedContentDigest cryptoutil.Digest,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: Skill runtime package context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if runtime == nil {
		return "", fmt.Errorf(
			"%w: Skill runtime package source runtime is nil",
			basespec.ErrInvalid,
		)
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	if err := basespec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return "", err
	}
	if err := cryptoutil.ValidateDigest(expectedContentDigest); err != nil {
		return "", err
	}
	if err := basespec.ValidateLocator(locator, false); err != nil {
		return "", err
	}
	if err := basespec.ValidateSubresourceLocator(subresource); err != nil {
		return "", err
	}
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Agent Skill bindings cannot target a subresource",
			basespec.ErrUnsupported,
		)
	}
	if !strings.EqualFold(
		path.Base(string(locator)),
		DefinitionFileName,
	) {
		return "", fmt.Errorf(
			"%w: Agent Skill locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			DefinitionFileName,
		)
	}

	packageLocator := basespec.Locator(path.Dir(string(locator)))
	if packageLocator == "." {
		return "", fmt.Errorf(
			"%w: Agent Skill package cannot be the Source root",
			basespec.ErrInvalid,
		)
	}

	localPaths, supported := runtime.(source.LocalPathRuntime)
	if !supported || !localPaths.SupportsLocalPath(value.Kind) {
		return "", fmt.Errorf(
			"%w: Source kind %q has no trusted native package path",
			basespec.ErrUnsupported,
			value.Kind,
		)
	}

	// Verify the exact catalogued entry before requesting a native package
	// path. Source adapters remain responsible for containment and path
	// resolution, while the artifact boundary verifies the generation and
	// bytes used to derive this Skill definition.
	var verifyErr error
	if session := runtimePackageResolutionSessionFromContext(ctx); session != nil {
		verifyErr = session.verify(
			ctx,
			runtime,
			value,
			expectedGeneration,
			locator,
			expectedContentDigest,
		)
	} else {
		verifyErr = source.VerifySnapshotContentDigest(
			ctx,
			runtime,
			value,
			locator,
			expectedGeneration,
			expectedContentDigest,
			basespec.MaxCandidateBytes,
		)
	}
	if verifyErr != nil {
		return "", verifyErr
	}

	location, err := localPaths.ResolveLocalPath(
		ctx,
		value,
		packageLocator,
	)
	if err != nil {
		return "", err
	}
	return location, nil
}

func (s *RuntimePackageResolutionSession) verify(
	ctx context.Context,
	runtime source.Runtime,
	value source.Source,
	expectedGeneration string,
	locator basespec.Locator,
	expectedContentDigest cryptoutil.Digest,
) error {
	key := runtimePackageResolutionSessionKey{
		RootID:         value.RootID,
		SourceID:       value.ID,
		SourceRevision: value.Revision,
		SourceKind:     value.Kind,
		Generation:     expectedGeneration,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return basespec.ErrClosed
	}

	snapshot, found := s.snapshots[key]
	if !found {
		opened, err := runtime.Open(ctx, value)
		if err != nil {
			return err
		}
		if opened.Generation() != expectedGeneration {
			return errors.Join(
				fmt.Errorf(
					"%w: source generation changed since it was observed",
					basespec.ErrConflict,
				),
				opened.Close(),
			)
		}
		snapshot = opened
		s.snapshots[key] = snapshot
	}

	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return err
	}
	if entry.Locator != locator {
		return fmt.Errorf(
			"%w: source snapshot stat for %q returned %q",
			basespec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	content, err := source.ReadSnapshotEntry(
		ctx,
		snapshot,
		entry,
		basespec.MaxCandidateBytes,
	)
	if err != nil {
		return err
	}
	if cryptoutil.DigestBytes(content) != expectedContentDigest {
		return fmt.Errorf(
			"%w: source content for %q changed since catalog publication",
			basespec.ErrConflict,
			locator,
		)
	}
	return nil
}
