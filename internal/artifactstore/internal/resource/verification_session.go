package resource

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type verificationSessionContextKey struct{}

type verificationSessionSourceKey struct {
	rootID   root.RootID
	sourceID source.SourceID
}

type verificationSessionSource struct {
	source     source.Source
	inspection source.RefreshInspection
	snapshot   sourceimpl.Snapshot
}

type verificationSession struct {
	service *Service

	mu      sync.Mutex
	sources map[verificationSessionSourceKey]*verificationSessionSource
	closed  bool
}

// borrowedVerificationSession is returned for a nested request using the
// same Service. The outer session owns confirmation and snapshot closure.
type borrowedVerificationSession struct{}

func (borrowedVerificationSession) Close(context.Context) error {
	return nil
}

// BeginVerificationSession creates a request-scoped session that reuses one
// confirmed Source snapshot per Root/Source pair. It is intentionally exposed
// as an optional capability: old ResourceReader test doubles remain valid and
// simply use the existing per-call path.
func (s *Service) BeginVerificationSession(
	ctx context.Context,
) (context.Context, resource.VerificationSession, error) {
	if err := validateContext(ctx, "resource verification session"); err != nil {
		return nil, nil, err
	}
	if s == nil {
		return nil, nil, basespec.ErrClosed
	}
	if existing := verificationSessionFromContext(ctx); existing != nil {
		if existing.service != s {
			return nil, nil, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				basespec.ErrInvalid,
			)
		}

		// Reuse the outer session. Closing this borrowed lease is deliberately
		// a no-op, so the outer caller remains responsible for confirmation.
		return ctx, borrowedVerificationSession{}, nil
	}

	session := &verificationSession{
		service: s,
		sources: make(map[verificationSessionSourceKey]*verificationSessionSource),
	}
	return context.WithValue(
		ctx,
		verificationSessionContextKey{},
		session,
	), session, nil
}

func (s *verificationSession) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: resource verification session close context is nil",
			basespec.ErrInvalid,
		)
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true

	values := make(
		map[verificationSessionSourceKey]*verificationSessionSource,
		len(s.sources),
	)
	maps.Copy(values, s.sources)
	s.sources = nil
	s.mu.Unlock()

	keys := make([]verificationSessionSourceKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].rootID != keys[right].rootID {
			return keys[left].rootID < keys[right].rootID
		}
		return keys[left].sourceID < keys[right].sourceID
	})

	var output error
	for _, key := range keys {
		value := values[key]

		current, currentErr := s.service.sources.Get(
			ctx,
			key.rootID,
			key.sourceID,
		)
		if currentErr == nil &&
			(!current.Enabled ||
				current.Revision != value.source.Revision) {
			currentErr = fmt.Errorf(
				"%w: Artifact Source %q changed during batch",
				basespec.ErrRefreshRequired,
				key.sourceID,
			)
		}

		output = errors.Join(
			output,
			currentErr,
			value.snapshot.Confirm(ctx),
			value.snapshot.Close(),
		)
	}
	return output
}

func verificationSessionFromContext(
	ctx context.Context,
) *verificationSession {
	if ctx == nil {
		return nil
	}
	value, _ := ctx.Value(
		verificationSessionContextKey{},
	).(*verificationSession)
	return value
}

func (s *verificationSession) withSource(
	ctx context.Context,
	key verificationSessionSourceKey,
	fn func(*verificationSessionSource) error,
) error {
	if s == nil {
		return basespec.ErrClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return basespec.ErrClosed
	}

	value, err := s.sourceLocked(ctx, key)
	if err != nil {
		return err
	}
	return fn(value)
}

func (s *verificationSession) sourceLocked(
	ctx context.Context,
	key verificationSessionSourceKey,
) (*verificationSessionSource, error) {
	if current, found := s.sources[key]; found {
		return current, nil
	}

	inspection, err := s.service.refresh.InspectSource(
		ctx,
		key.rootID,
		key.sourceID,
	)
	if err != nil {
		return nil, err
	}
	if !inspection.IsCurrent() {
		return nil, fmt.Errorf(
			"%w: Artifact Source %q requires refresh",
			basespec.ErrRefreshRequired,
			key.sourceID,
		)
	}

	value, err := s.service.sources.Get(
		ctx,
		key.rootID,
		key.sourceID,
	)
	if err != nil {
		return nil, err
	}
	if !value.Enabled {
		return nil, fmt.Errorf(
			"%w: Artifact Source %q is disabled",
			basespec.ErrSourceUnavailable,
			value.ID,
		)
	}
	if value.Revision != inspection.State.SourceRevision {
		return nil, fmt.Errorf(
			"%w: Artifact Source %q changed during batch setup",
			basespec.ErrRefreshRequired,
			value.ID,
		)
	}

	snapshot, err := s.service.sources.Open(ctx, value)
	if err != nil {
		return nil, err
	}
	if snapshot.Generation() != inspection.State.SourceGeneration {
		return nil, errors.Join(
			fmt.Errorf(
				"%w: Artifact Source %q changed during batch setup",
				basespec.ErrRefreshRequired,
				value.ID,
			),
			snapshot.Close(),
		)
	}

	current := &verificationSessionSource{
		source:     value.Clone(),
		inspection: inspection.Clone(),
		snapshot:   snapshot,
	}
	s.sources[key] = current
	return current, nil
}

func readVerificationSessionEntry(
	ctx context.Context,
	snapshot sourceimpl.Snapshot,
	locator basespec.Locator,
	maximumBytes int64,
) ([]byte, error) {
	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return nil, err
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if entry.Locator != locator {
		return nil, fmt.Errorf(
			"%w: Source snapshot stat for %q returned %q",
			basespec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return sourceimpl.ReadSnapshotEntry(
		ctx,
		snapshot,
		entry,
		maximumBytes,
	)
}

func (s *Service) resolveArtifactInSession(
	ctx context.Context,
	session *verificationSession,
	record artifact.Artifact,
) (resource.ResolvedArtifact, error) {
	if record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return resource.ResolvedArtifact{}, fmt.Errorf(
			"%w: Artifact %q is not currently available",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	definitionValue, err := s.definitions.GetDefinition(
		ctx,
		record.RootID,
		*record.ResolvedDefinition,
	)
	if err != nil {
		return resource.ResolvedArtifact{}, err
	}

	var (
		sourceValue source.Source
		state       source.RefreshState
	)
	err = session.withSource(
		ctx,
		verificationSessionSourceKey{
			rootID:   record.RootID,
			sourceID: record.Binding.SourceID,
		},
		func(current *verificationSessionSource) error {
			content, err := readVerificationSessionEntry(
				ctx,
				current.snapshot,
				record.Binding.Locator,
				basespec.MaxCandidateBytes,
			)
			if err != nil {
				return err
			}
			if cryptoutil.DigestBytes(content) !=
				*record.SourceContentDigest {
				return fmt.Errorf(
					"%w: Source content for %q changed since refresh",
					basespec.ErrConflict,
					record.Binding.Locator,
				)
			}

			sourceValue = current.source.Clone()
			state = current.inspection.State.Clone()
			return nil
		},
	)
	if err != nil {
		return resource.ResolvedArtifact{}, err
	}

	output := resource.ResolvedArtifact{
		Artifact:     record.Clone(),
		Definition:   definitionValue.Clone(),
		Source:       sourceValue.Summary(),
		RefreshState: state,
	}
	if err := output.Validate(); err != nil {
		return resource.ResolvedArtifact{}, err
	}
	return output.Clone(), nil
}

func (s *Service) readSourceEntryInSession(
	ctx context.Context,
	session *verificationSession,
	rootID root.RootID,
	sourceID source.SourceID,
	locator basespec.Locator,
	maximumBytes int64,
) (resource.VerifiedEntry, error) {
	var output resource.VerifiedEntry

	err := session.withSource(
		ctx,
		verificationSessionSourceKey{
			rootID:   rootID,
			sourceID: sourceID,
		},
		func(current *verificationSessionSource) error {
			content, err := readVerificationSessionEntry(
				ctx,
				current.snapshot,
				locator,
				maximumBytes,
			)
			if err != nil {
				return err
			}

			output = resource.VerifiedEntry{
				RootID:           rootID,
				SourceID:         sourceID,
				Locator:          locator,
				SourceRevision:   current.source.Revision,
				SourceGeneration: current.snapshot.Generation(),
				Content:          content,
				Digest:           cryptoutil.DigestBytes(content),
			}
			return output.Validate()
		},
	)
	if err != nil {
		return resource.VerifiedEntry{}, err
	}
	return output.Clone(), nil
}

func (s *Service) resolveVerifiedLocalPathInSession(
	ctx context.Context,
	session *verificationSession,
	resolved resource.ResolvedArtifact,
	localLocator basespec.Locator,
) (string, error) {
	localPaths, supported := s.sources.(sourceimpl.LocalPathRuntime)
	if !supported || !localPaths.SupportsLocalPath(resolved.Source.Kind) {
		return "", fmt.Errorf(
			"%w: source kind %q has no trusted native path",
			basespec.ErrUnsupported,
			resolved.Source.Kind,
		)
	}

	var location string
	err := session.withSource(
		ctx,
		verificationSessionSourceKey{
			rootID:   resolved.Source.RootID,
			sourceID: resolved.Source.ID,
		},
		func(current *verificationSessionSource) error {
			if current.source.Revision !=
				resolved.RefreshState.SourceRevision ||
				current.snapshot.Generation() !=
					resolved.RefreshState.SourceGeneration {
				return fmt.Errorf(
					"%w: Source changed after Artifact resolution",
					basespec.ErrRefreshRequired,
				)
			}

			content, err := readVerificationSessionEntry(
				ctx,
				current.snapshot,
				resolved.Artifact.Binding.Locator,
				basespec.MaxCandidateBytes,
			)
			if err != nil {
				return err
			}
			if cryptoutil.DigestBytes(content) !=
				*resolved.Artifact.SourceContentDigest {
				return fmt.Errorf(
					"%w: Source content for %q changed since refresh",
					basespec.ErrConflict,
					resolved.Artifact.Binding.Locator,
				)
			}

			value, err := localPaths.ResolveLocalPath(
				ctx,
				current.source,
				localLocator,
			)
			if err != nil {
				return err
			}
			location = value
			return nil
		},
	)
	if err != nil {
		return "", err
	}
	return location, nil
}
