package internal

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"sync"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type verificationSessionContextKey struct{}

type verificationSessionSourceKey struct {
	rootID   rootModel.RootID
	sourceID sourceModel.SourceID
}

type verificationSessionSource struct {
	source     sourceModel.Source
	inspection *refreshModel.Inspection
	snapshot   driver.Snapshot
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

type borrowedSnapshot struct {
	driver.Snapshot
}

func (borrowedSnapshot) Confirm(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: borrowed verification snapshot context is nil",
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func (borrowedSnapshot) Close() error {
	return nil
}

type borrowedTreeRuntime struct {
	source   sourceModel.Source
	snapshot driver.Snapshot
}

func (r borrowedTreeRuntime) Get(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (sourceModel.Source, error) {
	if rootID != r.source.RootID || sourceID != r.source.ID {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: Source %q in Root %q",
			spec.ErrSourceNotFound,
			sourceID,
			rootID,
		)
	}
	return r.source.Clone(), nil
}

func (r borrowedTreeRuntime) List(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]sourceModel.Source, error) {
	if rootID != r.source.RootID {
		return []sourceModel.Source{}, nil
	}
	return []sourceModel.Source{r.source.Clone()}, nil
}

func (r borrowedTreeRuntime) Open(
	ctx context.Context,
	value sourceModel.Source,
) (driver.Snapshot, error) {
	if value.RootID != r.source.RootID ||
		value.ID != r.source.ID ||
		value.Revision != r.source.Revision {
		return nil, fmt.Errorf(
			"%w: borrowed verification snapshot Source changed",
			spec.ErrConflict,
		)
	}
	return borrowedSnapshot{Snapshot: r.snapshot}, nil
}

func (s *Service) readSourceTreeInSession(
	ctx context.Context,
	session *verificationSession,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	base spec.Locator,
	include, exclude []string,
	maximumEntries int,
	maximumBytes int64,
) ([]resourceModel.VerifiedEntry, error) {
	var output []resourceModel.VerifiedEntry

	err := session.withSource(
		ctx,
		verificationSessionSourceKey{
			rootID:   rootID,
			sourceID: sourceID,
		},
		func(current *verificationSessionSource) error {
			proxy := *s
			proxy.sources = borrowedTreeRuntime{
				source:   current.source.Clone(),
				snapshot: current.snapshot,
			}

			detached := context.WithValue(
				ctx,
				verificationSessionContextKey{},
				nil,
			)
			values, err := proxy.ReadSourceTree(
				detached,
				rootID,
				sourceID,
				base,
				include,
				exclude,
				maximumEntries,
				maximumBytes,
			)
			if err != nil {
				return err
			}
			output = values
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// BeginVerificationSession creates a request-scoped session that reuses one
// confirmed Source snapshot per Root/Source pair. It is intentionally exposed
// as an optional capability: old ResourceReader test doubles remain valid and
// simply use the existing per-call path.
func (s *Service) BeginVerificationSession(
	ctx context.Context,
) (context.Context, resourceModel.VerificationSession, error) {
	if existing := verificationSessionFromContext(ctx); existing != nil {
		if existing.service != s {
			return nil, nil, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				spec.ErrInvalid,
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
				spec.ErrRefreshRequired,
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
		return spec.ErrClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return spec.ErrClosed
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
			spec.ErrSourceUnavailable,
			value.ID,
		)
	}

	snapshot, err := s.service.sources.Open(ctx, value)
	if err != nil {
		return nil, err
	}

	current := &verificationSessionSource{
		source:   value.Clone(),
		snapshot: snapshot,
	}
	s.sources[key] = current
	return current, nil
}

func (s *verificationSession) ensureRefreshCurrentLocked(
	ctx context.Context,
	current *verificationSessionSource,
) (refreshModel.Inspection, error) {
	if current == nil {
		return refreshModel.Inspection{}, spec.ErrClosed
	}
	if current.inspection != nil {
		return current.inspection.Clone(), nil
	}

	inspection, err := s.service.refresh.InspectSourceMetadata(
		ctx,
		current.source,
	)
	if err != nil {
		return refreshModel.Inspection{}, err
	}
	if !inspection.IsCurrent() ||
		current.snapshot.Generation() != inspection.State.SourceGeneration {
		return refreshModel.Inspection{}, fmt.Errorf(
			"%w: Artifact Source %q changed during batch setup",
			spec.ErrRefreshRequired,
			current.source.ID,
		)
	}

	copyValue := inspection.Clone()
	current.inspection = &copyValue
	return copyValue, nil
}

func readVerificationSessionEntry(
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
		return nil, err
	}
	if entry.Locator != locator {
		return nil, fmt.Errorf(
			"%w: Source snapshot stat for %q returned %q",
			spec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return source.ReadSnapshotEntry(
		ctx,
		snapshot,
		entry,
		maximumBytes,
	)
}

func (s *Service) resolveArtifactInSession(
	ctx context.Context,
	session *verificationSession,
	record artifactModel.Artifact,
) (resourceModel.ResolvedArtifact, error) {
	if record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return resourceModel.ResolvedArtifact{}, fmt.Errorf(
			"%w: Artifact %q is not currently available",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	definitionValue, err := s.definitions.GetDefinition(
		ctx,
		record.RootID,
		*record.ResolvedDefinition,
	)
	if err != nil {
		return resourceModel.ResolvedArtifact{}, err
	}

	var (
		sourceValue sourceModel.Source
		state       refreshModel.State
	)
	err = session.withSource(
		ctx,
		verificationSessionSourceKey{
			rootID:   record.RootID,
			sourceID: record.Binding.SourceID,
		},
		func(current *verificationSessionSource) error {
			inspection, err := session.ensureRefreshCurrentLocked(
				ctx,
				current,
			)
			if err != nil {
				return err
			}
			content, err := readVerificationSessionEntry(
				ctx,
				current.snapshot,
				record.Binding.Locator,
				spec.MaxCandidateBytes,
			)
			if err != nil {
				return err
			}
			if cryptoutil.DigestBytes(content) !=
				*record.SourceContentDigest {
				return fmt.Errorf(
					"%w: Source content for %q changed since refresh",
					spec.ErrConflict,
					record.Binding.Locator,
				)
			}

			sourceValue = current.source.Clone()
			state = inspection.State.Clone()
			return nil
		},
	)
	if err != nil {
		return resourceModel.ResolvedArtifact{}, err
	}

	output := resourceModel.ResolvedArtifact{
		Artifact:     record.Clone(),
		Definition:   definitionValue.Clone(),
		Source:       sourceValue.Summary(),
		RefreshState: state,
	}
	if err := output.Validate(); err != nil {
		return resourceModel.ResolvedArtifact{}, err
	}
	return output.Clone(), nil
}

func (s *Service) readSourceEntryInSession(
	ctx context.Context,
	session *verificationSession,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator spec.Locator,
	maximumBytes int64,
) (resourceModel.VerifiedEntry, error) {
	var output resourceModel.VerifiedEntry

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

			output = resourceModel.VerifiedEntry{
				RootID:           rootID,
				SourceID:         sourceID,
				Locator:          locator,
				SourceRevision:   current.source.Revision,
				SourceGeneration: current.snapshot.Generation(),
				Content:          content,
				Digest:           cryptoutil.DigestBytes(content),
			}
			return nil
		},
	)
	if err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	return output, nil
}

func (s *Service) resolveVerifiedLocalPathInSession(
	ctx context.Context,
	session *verificationSession,
	resolved resourceModel.ResolvedArtifact,
	localLocator spec.Locator,
) (string, error) {
	localPaths, supported := s.sources.(source.LocalPathRuntime)
	if !supported || !localPaths.SupportsLocalPath(resolved.Source.Kind) {
		return "", fmt.Errorf(
			"%w: source kind %q has no trusted native path",
			spec.ErrUnsupported,
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
					spec.ErrRefreshRequired,
				)
			}

			content, err := readVerificationSessionEntry(
				ctx,
				current.snapshot,
				resolved.Artifact.Binding.Locator,
				spec.MaxCandidateBytes,
			)
			if err != nil {
				return err
			}
			if cryptoutil.DigestBytes(content) !=
				*resolved.Artifact.SourceContentDigest {
				return fmt.Errorf(
					"%w: Source content for %q changed since refresh",
					spec.ErrConflict,
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

func (s *Service) statSourceEntryInSession(
	ctx context.Context,
	session *verificationSession,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator spec.Locator,
) (sourceModel.Entry, error) {
	var output sourceModel.Entry

	err := session.withSource(
		ctx,
		verificationSessionSourceKey{
			rootID:   rootID,
			sourceID: sourceID,
		},
		func(current *verificationSessionSource) error {
			entry, err := current.snapshot.Stat(ctx, locator)
			if err != nil {
				return err
			}
			if err := entry.Validate(); err != nil {
				return err
			}
			if entry.Locator != locator {
				return fmt.Errorf(
					"%w: Source stat for %q returned %q",
					spec.ErrInvalid,
					locator,
					entry.Locator,
				)
			}
			output = entry
			return nil
		},
	)
	if err != nil {
		return sourceModel.Entry{}, err
	}
	return output, nil
}
