package source

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type verificationSessionContextKey struct{}

type verificationSessionKey struct {
	RootID         basespec.RootID
	SourceID       basespec.SourceID
	SourceRevision uint64
	SourceKind     basespec.SourceKind
	Generation     string
}

// VerificationSession reuses verified Source snapshots while one caller
// resolves multiple local paths from the same catalogued Source generation.
//
// Snapshot operations are intentionally serialized because Snapshot adapters
// are not required to support concurrent use.
type VerificationSession struct {
	mu        sync.Mutex
	snapshots map[verificationSessionKey]Snapshot
	closed    bool
}

func NewVerificationSession(
	ctx context.Context,
) (context.Context, *VerificationSession, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf(
			"%w: source verification session context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if verificationSessionFromContext(ctx) != nil {
		return nil, nil, fmt.Errorf(
			"%w: source verification session already exists",
			basespec.ErrInvalid,
		)
	}

	session := &VerificationSession{
		snapshots: make(map[verificationSessionKey]Snapshot),
	}
	return context.WithValue(
		ctx,
		verificationSessionContextKey{},
		session,
	), session, nil
}

func (s *VerificationSession) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}

	var output error

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return output
	}
	s.closed = true
	snapshots := s.snapshots
	s.snapshots = nil
	s.mu.Unlock()

	keys := make([]verificationSessionKey, 0, len(snapshots))
	for key := range snapshots {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].RootID != keys[right].RootID {
			return keys[left].RootID < keys[right].RootID
		}
		if keys[left].SourceID != keys[right].SourceID {
			return keys[left].SourceID < keys[right].SourceID
		}
		if keys[left].SourceRevision != keys[right].SourceRevision {
			return keys[left].SourceRevision < keys[right].SourceRevision
		}
		return keys[left].Generation < keys[right].Generation
	})

	for _, key := range keys {
		snapshot := snapshots[key]
		if snapshot == nil {
			continue
		}
		output = errors.Join(
			output,
			snapshot.Confirm(ctx),
			snapshot.Close(),
		)
	}
	return output
}

func ResolveVerifiedLocalPath(
	ctx context.Context,
	runtime Runtime,
	value Source,
	verifiedLocator basespec.Locator,
	localLocator basespec.Locator,
	expectedGeneration string,
	expectedDigest cryptoutil.Digest,
	maximumBytes int64,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: verified source path context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if runtime == nil {
		return "", fmt.Errorf(
			"%w: verified source path runtime is nil",
			basespec.ErrInvalid,
		)
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	if err := basespec.ValidateLocator(verifiedLocator, false); err != nil {
		return "", err
	}
	if err := basespec.ValidateLocator(localLocator, true); err != nil {
		return "", err
	}
	if err := basespec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return "", err
	}
	if err := cryptoutil.ValidateDigest(expectedDigest); err != nil {
		return "", err
	}
	if maximumBytes <= 0 || maximumBytes > basespec.MaxCandidateBytes {
		return "", fmt.Errorf(
			"%w: verified source path byte limit is invalid",
			basespec.ErrInvalid,
		)
	}

	localPaths, supported := runtime.(LocalPathRuntime)
	if !supported || !localPaths.SupportsLocalPath(value.Kind) {
		return "", fmt.Errorf(
			"%w: source kind %q has no trusted native path",
			basespec.ErrUnsupported,
			value.Kind,
		)
	}

	if session := verificationSessionFromContext(ctx); session != nil {
		if err := session.verify(
			ctx,
			runtime,
			value,
			verifiedLocator,
			expectedGeneration,
			expectedDigest,
			maximumBytes,
		); err != nil {
			return "", err
		}
	} else if err := VerifySnapshotContentDigest(
		ctx,
		runtime,
		value,
		verifiedLocator,
		expectedGeneration,
		expectedDigest,
		maximumBytes,
	); err != nil {
		return "", err
	}

	return localPaths.ResolveLocalPath(ctx, value, localLocator)
}

func (s *VerificationSession) verify(
	ctx context.Context,
	runtime Runtime,
	value Source,
	locator basespec.Locator,
	expectedGeneration string,
	expectedDigest cryptoutil.Digest,
	maximumBytes int64,
) error {
	key := verificationSessionKey{
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

	return verifySnapshotEntry(
		ctx,
		snapshot,
		locator,
		expectedDigest,
		maximumBytes,
	)
}

func verificationSessionFromContext(
	ctx context.Context,
) *VerificationSession {
	if ctx == nil {
		return nil
	}
	value, _ := ctx.Value(
		verificationSessionContextKey{},
	).(*VerificationSession)
	return value
}

func readSnapshotLocator(
	ctx context.Context,
	snapshot Snapshot,
	locator basespec.Locator,
	maximumBytes int64,
) ([]byte, error) {
	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return nil, err
	}
	if err := entry.Validate(); err != nil {
		return nil, fmt.Errorf(
			"%w: source snapshot returned an invalid entry: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if entry.Locator != locator {
		return nil, fmt.Errorf(
			"%w: source snapshot stat for %q returned %q",
			basespec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return ReadSnapshotEntry(ctx, snapshot, entry, maximumBytes)
}

func verifySnapshotEntry(
	ctx context.Context,
	snapshot Snapshot,
	locator basespec.Locator,
	expectedDigest cryptoutil.Digest,
	maximumBytes int64,
) error {
	content, err := readSnapshotLocator(
		ctx,
		snapshot,
		locator,
		maximumBytes,
	)
	if err != nil {
		return err
	}
	if cryptoutil.DigestBytes(content) != expectedDigest {
		return fmt.Errorf(
			"%w: source content for %q changed since catalog publication",
			basespec.ErrConflict,
			locator,
		)
	}
	return nil
}
