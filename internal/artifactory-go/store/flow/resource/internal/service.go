package internal

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type sourceRefreshInspector interface {
	InspectSourceMetadata(
		ctx context.Context,
		value sourceModel.Source,
	) (refreshModel.Inspection, error)
}

type Service struct {
	artifacts   artifact.Reader
	definitions artifact.DefinitionReader
	refresh     sourceRefreshInspector
	sources     source.Runtime
}

func NewService(
	artifacts artifact.Reader,
	definitions artifact.DefinitionReader,
	refresh sourceRefreshInspector,
	sources source.Runtime,
) (*Service, error) {
	if artifacts == nil ||
		definitions == nil ||
		refresh == nil ||
		sources == nil {
		return nil, fmt.Errorf(
			"%w: Artifact resource service dependencies are incomplete",
			storeSpec.ErrInvalid,
		)
	}
	return &Service{
		artifacts:   artifacts,
		definitions: definitions,
		refresh:     refresh,
		sources:     sources,
	}, nil
}

func (s *Service) ResolveArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	_ resourceModel.ResolveOptions,
) (resourceModel.ResolvedArtifact, error) {
	if err := ref.Validate(); err != nil {
		return resourceModel.ResolvedArtifact{}, err
	}

	record, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return resourceModel.ResolvedArtifact{}, err
	}
	if record.State != artifactModel.StateAvailable ||
		record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return resourceModel.ResolvedArtifact{}, fmt.Errorf(
			"%w: Artifact %q is not currently available",
			storeSpec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if session := verificationSessionFromContext(ctx); session != nil {
		if session.service != s {
			return resourceModel.ResolvedArtifact{}, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				storeSpec.ErrInvalid,
			)
		}
		return s.resolveArtifactInSession(ctx, session, record)
	}

	sessionCtx, lease, err := s.BeginVerificationSession(ctx)
	if err != nil {
		return resourceModel.ResolvedArtifact{}, err
	}
	output, resolveErr := s.resolveArtifactInSession(
		sessionCtx,
		verificationSessionFromContext(sessionCtx),
		record,
	)
	closeErr := lease.Close(context.WithoutCancel(sessionCtx))
	if err := errors.Join(resolveErr, closeErr); err != nil {
		return resourceModel.ResolvedArtifact{}, err
	}
	return output, nil
}

func (s *Service) ResolveVerifiedLocalPath(
	ctx context.Context,
	resolved resourceModel.ResolvedArtifact,
	localLocator storeSpec.Locator,
) (string, error) {
	if err := resolved.Validate(); err != nil {
		return "", err
	}
	if err := localLocator.Validate(true); err != nil {
		return "", err
	}
	if session := verificationSessionFromContext(ctx); session != nil {
		if session.service != s {
			return "", fmt.Errorf(
				"%w: verification session belongs to another resource service",
				storeSpec.ErrInvalid,
			)
		}
		return s.resolveVerifiedLocalPathInSession(ctx, session, resolved, localLocator)
	}

	sessionCtx, lease, err := s.BeginVerificationSession(ctx)
	if err != nil {
		return "", err
	}
	location, resolveErr := s.resolveVerifiedLocalPathInSession(
		sessionCtx,
		verificationSessionFromContext(sessionCtx),
		resolved,
		localLocator,
	)
	closeErr := lease.Close(context.WithoutCancel(sessionCtx))
	if err := errors.Join(resolveErr, closeErr); err != nil {
		return "", err
	}
	return location, nil
}

func (s *Service) ReadSourceEntry(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator storeSpec.Locator,
	maximumBytes int64,
) (_ resourceModel.VerifiedEntry, returnErr error) {
	if err := rootID.Validate(); err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if err := locator.Validate(false); err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if maximumBytes <= 0 || maximumBytes > storeSpec.MaxScanBytes {
		return resourceModel.VerifiedEntry{}, fmt.Errorf(
			"%w: Source entry read limit is invalid",
			storeSpec.ErrInvalid,
		)
	}
	if session := verificationSessionFromContext(ctx); session != nil {
		if session.service != s {
			return resourceModel.VerifiedEntry{}, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				storeSpec.ErrInvalid,
			)
		}
		return s.readSourceEntryInSession(ctx, session, rootID, sourceID, locator, maximumBytes)
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if !value.Enabled {
		return resourceModel.VerifiedEntry{}, fmt.Errorf(
			"%w: Source %q is disabled",
			storeSpec.ErrSourceUnavailable,
			value.ID,
		)
	}
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, snapshot.Close())
	}()

	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if err := entry.Validate(); err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if entry.Locator != locator {
		return resourceModel.VerifiedEntry{}, fmt.Errorf(
			"%w: Source stat for %q returned %q",
			storeSpec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	content, err := source.ReadSnapshotEntry(
		ctx,
		snapshot,
		entry,
		maximumBytes,
	)
	if err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	if err := s.confirmSourceCurrent(ctx, value); err != nil {
		return resourceModel.VerifiedEntry{}, err
	}
	output := resourceModel.VerifiedEntry{
		RootID:           rootID,
		SourceID:         sourceID,
		Locator:          locator,
		SourceRevision:   value.Revision,
		SourceGeneration: snapshot.Generation(),
		Content:          content,
		Digest:           cryptoutil.DigestBytes(content),
	}
	return output, nil
}

// StatSourceEntry confirms one Source snapshot and returns only physical Entry
// metadata. It is used for declaration planning, not as a Resource substitute.
func (s *Service) StatSourceEntry(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator storeSpec.Locator,
) (sourceModel.Entry, error) {
	if err := rootID.Validate(); err != nil {
		return sourceModel.Entry{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return sourceModel.Entry{}, err
	}
	if err := locator.Validate(true); err != nil {
		return sourceModel.Entry{}, err
	}
	if session := verificationSessionFromContext(ctx); session != nil {
		if session.service != s {
			return sourceModel.Entry{}, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				storeSpec.ErrInvalid,
			)
		}
		return s.statSourceEntryInSession(
			ctx,
			session,
			rootID,
			sourceID,
			locator,
		)
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return sourceModel.Entry{}, err
	}
	if !value.Enabled {
		return sourceModel.Entry{}, fmt.Errorf(
			"%w: Source %q is disabled",
			storeSpec.ErrSourceUnavailable,
			sourceID,
		)
	}
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return sourceModel.Entry{}, err
	}

	entry, statErr := snapshot.Stat(ctx, locator)
	confirmErr := snapshot.Confirm(ctx)
	var sourceErr error
	if statErr == nil && confirmErr == nil {
		sourceErr = s.confirmSourceCurrent(ctx, value)
	}
	closeErr := snapshot.Close()
	if err := errors.Join(
		statErr,
		confirmErr,
		sourceErr,
		closeErr,
	); err != nil {
		return sourceModel.Entry{}, err
	}
	if err := entry.Validate(); err != nil {
		return sourceModel.Entry{}, err
	}
	if entry.Locator != locator {
		return sourceModel.Entry{}, fmt.Errorf(
			"%w: Source stat for %q returned %q",
			storeSpec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return entry, nil
}

// ReadSourceTree reads selected regular files from one Source snapshot.
//
// It is intentionally source-oriented and does not know Context, Workspace,
// Skill, or any other contract. Consumers provide a source-relative base and
// portable path patterns. The method preserves verified source generation,
// bounded reads, source containment, and deterministic locator ordering.
func (s *Service) ReadSourceTree(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	base storeSpec.Locator,
	include []string,
	exclude []string,
	maximumEntries int,
	maximumBytes int64,
) (_ []resourceModel.VerifiedEntry, returnErr error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := sourceID.Validate(); err != nil {
		return nil, err
	}
	if err := base.Validate(true); err != nil {
		return nil, err
	}
	selection, err := storeSpec.NewPathSelection(
		include,
		exclude,
	)
	if err != nil {
		return nil, err
	}
	if maximumEntries <= 0 ||
		maximumEntries > storeSpec.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: Source tree entry limit is invalid",
			storeSpec.ErrInvalid,
		)
	}
	if maximumBytes <= 0 ||
		maximumBytes > storeSpec.MaxScanBytes {
		return nil, fmt.Errorf(
			"%w: Source tree byte limit is invalid",
			storeSpec.ErrInvalid,
		)
	}
	if session := verificationSessionFromContext(ctx); session != nil {
		if session.service != s {
			return nil, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				storeSpec.ErrInvalid,
			)
		}
		return s.readSourceTreeInSession(
			ctx,
			session,
			rootID,
			sourceID,
			base,
			include,
			exclude,
			maximumEntries,
			maximumBytes,
		)
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return nil, err
	}
	if !value.Enabled {
		return nil, fmt.Errorf(
			"%w: Source %q is disabled",
			storeSpec.ErrSourceUnavailable,
			value.ID,
		)
	}
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return nil, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, snapshot.Close())
	}()

	generation := snapshot.Generation()
	rootEntry, err := snapshot.Stat(ctx, base)
	if err != nil {
		return nil, err
	}
	if err := rootEntry.Validate(); err != nil {
		return nil, err
	}
	if rootEntry.Locator != base {
		return nil, fmt.Errorf(
			"%w: Source stat for %q returned %q",
			storeSpec.ErrInvalid,
			base,
			rootEntry.Locator,
		)
	}

	type selectedEntry struct {
		entry    sourceModel.Entry
		relative string
	}
	selected := make([]selectedEntry, 0)
	visited := 0

	appendSelected := func(
		entry sourceModel.Entry,
		relative string,
	) error {
		if !entry.IsRegular {
			return nil
		}
		matched, err := selection.Match(relative)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		if len(selected) == maximumEntries {
			return fmt.Errorf(
				"%w: Source tree exceeds %d selected entries",
				storeSpec.ErrInvalid,
				maximumEntries,
			)
		}
		selected = append(selected, selectedEntry{
			entry:    entry,
			relative: relative,
		})
		return nil
	}

	var visit func(
		directory storeSpec.Locator,
		depth int,
	) error
	visit = func(
		directory storeSpec.Locator,
		depth int,
	) error {
		if depth > storeSpec.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: Source tree exceeds depth %d",
				storeSpec.ErrInvalid,
				storeSpec.DefaultMaxDepth,
			)
		}
		entries, err := snapshot.ReadDir(ctx, directory)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(left, right int) bool {
			return entries[left].Locator < entries[right].Locator
		})
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := entry.Validate(); err != nil {
				return err
			}
			if !sourceTreeDirectChild(directory, entry.Locator) {
				return fmt.Errorf(
					"%w: Source snapshot returned non-child %q for %q",
					storeSpec.ErrInvalid,
					entry.Locator,
					directory,
				)
			}
			visited++
			if visited > storeSpec.MaxDiscoveryEntries {
				return fmt.Errorf(
					"%w: Source tree exceeds traversal entry limit",
					storeSpec.ErrInvalid,
				)
			}

			relative, err := sourceTreeRelativeLocator(
				base,
				entry.Locator,
			)
			if err != nil {
				return err
			}
			if entry.IsDirectory {
				if err := visit(entry.Locator, depth+1); err != nil {
					return err
				}
				continue
			}
			if err := appendSelected(entry, relative); err != nil {
				return err
			}
		}
		return nil
	}

	switch {
	case rootEntry.IsRegular:
		if err := appendSelected(
			rootEntry,
			path.Base(string(rootEntry.Locator)),
		); err != nil {
			return nil, err
		}

	case rootEntry.IsDirectory:
		if err := visit(base, 0); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf(
			"%w: Source tree base %q is not a regular file or directory",
			storeSpec.ErrInvalid,
			base,
		)
	}

	sort.Slice(selected, func(left, right int) bool {
		return selected[left].entry.Locator <
			selected[right].entry.Locator
	})

	output := make(
		[]resourceModel.VerifiedEntry,
		0,
		len(selected),
	)
	var consumed int64
	for _, selectedEntry := range selected {
		if selectedEntry.entry.SizeBytes >
			maximumBytes-consumed {
			return nil, fmt.Errorf(
				"%w: Source tree exceeds aggregate byte limit",
				storeSpec.ErrInvalid,
			)
		}
		perEntryLimit := int64(storeSpec.MaxCandidateBytes)
		if remaining := maximumBytes - consumed; remaining < perEntryLimit {
			perEntryLimit = remaining
		}
		content, err := source.ReadSnapshotEntry(
			ctx,
			snapshot,
			selectedEntry.entry,
			perEntryLimit,
		)
		if err != nil {
			return nil, err
		}
		consumed += int64(len(content))
		output = append(output, resourceModel.VerifiedEntry{
			RootID:           rootID,
			SourceID:         sourceID,
			Locator:          selectedEntry.entry.Locator,
			SourceRevision:   value.Revision,
			SourceGeneration: generation,
			Content:          content,
			Digest:           cryptoutil.DigestBytes(content),
		})
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return nil, err
	}
	if err := s.confirmSourceCurrent(ctx, value); err != nil {
		return nil, err
	}
	return output, nil
}

func (s *Service) SupportsLocalPath(
	kind sourceModel.SourceKind,
) bool {
	if s == nil {
		return false
	}
	localPaths, supported := s.sources.(source.LocalPathRuntime)
	return supported && localPaths.SupportsLocalPath(kind)
}

func sourceTreeDirectChild(
	parent storeSpec.Locator,
	child storeSpec.Locator,
) bool {
	if child == "." {
		return false
	}
	if parent == "." {
		return !strings.Contains(string(child), "/")
	}
	prefix := string(parent) + "/"
	relative, found := strings.CutPrefix(
		string(child),
		prefix,
	)
	return found &&
		relative != "" &&
		!strings.Contains(relative, "/")
}

func sourceTreeRelativeLocator(
	base storeSpec.Locator,
	value storeSpec.Locator,
) (string, error) {
	if base == "." {
		return string(value), nil
	}
	prefix := string(base) + "/"
	relative, found := strings.CutPrefix(string(value), prefix)
	if !found || relative == "" {
		return "", fmt.Errorf(
			"%w: Source entry %q is outside tree base %q",
			storeSpec.ErrInvalid,
			value,
			base,
		)
	}
	return relative, nil
}

// confirmSourceCurrent closes the metadata race left after a physical snapshot
// confirms its bytes. A successful resource-only read must not be returned as
// valid if Source lifecycle or discovery metadata changed during that read.
func (s *Service) confirmSourceCurrent(
	ctx context.Context,
	observed sourceModel.Source,
) error {
	current, err := s.sources.Get(
		ctx,
		observed.RootID,
		observed.ID,
	)
	if err != nil {
		return err
	}
	if !current.Enabled ||
		current.Revision != observed.Revision {
		return fmt.Errorf(
			"%w: Source %q changed during verified read",
			storeSpec.ErrRefreshRequired,
			observed.ID,
		)
	}
	return nil
}
