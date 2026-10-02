package resource

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/internal/engine/artifact"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/internal/engine/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type sourceRefreshInspector interface {
	InspectSourceMetadata(
		ctx context.Context,
		value source.Source,
	) (source.RefreshInspection, error)
}

type Service struct {
	artifacts   artifactimpl.Reader
	definitions artifactimpl.DefinitionReader
	refresh     sourceRefreshInspector
	sources     sourceimpl.Runtime
}

func NewService(
	artifacts artifactimpl.Reader,
	definitions artifactimpl.DefinitionReader,
	refresh sourceRefreshInspector,
	sources sourceimpl.Runtime,
) (*Service, error) {
	if artifacts == nil ||
		definitions == nil ||
		refresh == nil ||
		sources == nil {
		return nil, fmt.Errorf(
			"%w: Artifact resource service dependencies are incomplete",
			model.ErrInvalid,
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
	ref artifact.ArtifactRef,
	_ resource.ResolveOptions,
) (resource.ResolvedArtifact, error) {
	if err := validateContext(ctx, "Artifact resolution"); err != nil {
		return resource.ResolvedArtifact{}, err
	}
	if s == nil {
		return resource.ResolvedArtifact{}, model.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return resource.ResolvedArtifact{}, err
	}

	record, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return resource.ResolvedArtifact{}, err
	}
	if record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return resource.ResolvedArtifact{}, fmt.Errorf(
			"%w: Artifact %q is not currently available",
			model.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if session := verificationSessionFromContext(ctx); session != nil {
		if session.service != s {
			return resource.ResolvedArtifact{}, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				model.ErrInvalid,
			)
		}
		return s.resolveArtifactInSession(ctx, session, record)
	}

	sessionCtx, lease, err := s.BeginVerificationSession(ctx)
	if err != nil {
		return resource.ResolvedArtifact{}, err
	}
	output, resolveErr := s.resolveArtifactInSession(
		sessionCtx,
		verificationSessionFromContext(sessionCtx),
		record,
	)
	closeErr := lease.Close(context.WithoutCancel(sessionCtx))
	if err := errors.Join(resolveErr, closeErr); err != nil {
		return resource.ResolvedArtifact{}, err
	}
	return output, nil
}

func (s *Service) ResolveVerifiedLocalPath(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
	localLocator model.Locator,
) (string, error) {
	if err := validateContext(
		ctx,
		"verified local-path resolution",
	); err != nil {
		return "", err
	}
	if s == nil {
		return "", model.ErrClosed
	}
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
				model.ErrInvalid,
			)
		}
		return s.resolveVerifiedLocalPathInSession(ctx, session, resolved, localLocator)
	}

	value, err := s.sources.Get(
		ctx,
		resolved.Source.RootID,
		resolved.Source.ID,
	)
	if err != nil {
		return "", err
	}
	if !value.Enabled {
		return "", fmt.Errorf(
			"%w: Artifact Source %q is disabled",
			model.ErrSourceUnavailable,
			value.ID,
		)
	}
	if value.Revision != resolved.RefreshState.SourceRevision {
		return "", fmt.Errorf(
			"%w: Source changed after Artifact resolution",
			model.ErrRefreshRequired,
		)
	}
	return sourceimpl.ResolveVerifiedLocalPath(
		ctx,
		s.sources,
		value,
		resolved.Artifact.Binding.Locator,
		localLocator,
		resolved.RefreshState.SourceGeneration,
		*resolved.Artifact.SourceContentDigest,
		model.MaxCandidateBytes,
	)
}

func (s *Service) ReadSourceEntry(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	locator model.Locator,
	maximumBytes int64,
) (_ resource.VerifiedEntry, returnErr error) {
	if err := validateContext(ctx, "Source entry read"); err != nil {
		return resource.VerifiedEntry{}, err
	}
	if s == nil {
		return resource.VerifiedEntry{}, model.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return resource.VerifiedEntry{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return resource.VerifiedEntry{}, err
	}
	if err := locator.Validate(false); err != nil {
		return resource.VerifiedEntry{}, err
	}
	if maximumBytes <= 0 || maximumBytes > model.MaxScanBytes {
		return resource.VerifiedEntry{}, fmt.Errorf(
			"%w: Source entry read limit is invalid",
			model.ErrInvalid,
		)
	}
	if session := verificationSessionFromContext(ctx); session != nil {
		if session.service != s {
			return resource.VerifiedEntry{}, fmt.Errorf(
				"%w: verification session belongs to another resource service",
				model.ErrInvalid,
			)
		}
		return s.readSourceEntryInSession(ctx, session, rootID, sourceID, locator, maximumBytes)
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return resource.VerifiedEntry{}, err
	}
	if !value.Enabled {
		return resource.VerifiedEntry{}, fmt.Errorf(
			"%w: Source %q is disabled",
			model.ErrSourceUnavailable,
			value.ID,
		)
	}
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return resource.VerifiedEntry{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, snapshot.Close())
	}()

	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return resource.VerifiedEntry{}, err
	}
	if err := entry.Validate(); err != nil {
		return resource.VerifiedEntry{}, err
	}
	if entry.Locator != locator {
		return resource.VerifiedEntry{}, fmt.Errorf(
			"%w: Source stat for %q returned %q",
			model.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	content, err := sourceimpl.ReadSnapshotEntry(
		ctx,
		snapshot,
		entry,
		maximumBytes,
	)
	if err != nil {
		return resource.VerifiedEntry{}, err
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return resource.VerifiedEntry{}, err
	}
	output := resource.VerifiedEntry{
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
	rootID root.RootID,
	sourceID source.SourceID,
	locator model.Locator,
) (source.Entry, error) {
	if err := validateContext(ctx, "Source entry stat"); err != nil {
		return source.Entry{}, err
	}
	if s == nil {
		return source.Entry{}, model.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return source.Entry{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return source.Entry{}, err
	}
	if err := locator.Validate(true); err != nil {
		return source.Entry{}, err
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return source.Entry{}, err
	}
	if !value.Enabled {
		return source.Entry{}, fmt.Errorf(
			"%w: Source %q is disabled",
			model.ErrSourceUnavailable,
			sourceID,
		)
	}
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return source.Entry{}, err
	}

	entry, statErr := snapshot.Stat(ctx, locator)
	confirmErr := snapshot.Confirm(ctx)
	closeErr := snapshot.Close()
	if err := errors.Join(statErr, confirmErr, closeErr); err != nil {
		return source.Entry{}, err
	}
	if err := entry.Validate(); err != nil {
		return source.Entry{}, err
	}
	if entry.Locator != locator {
		return source.Entry{}, fmt.Errorf(
			"%w: Source stat for %q returned %q",
			model.ErrInvalid,
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
	rootID root.RootID,
	sourceID source.SourceID,
	base model.Locator,
	include []string,
	exclude []string,
	maximumEntries int,
	maximumBytes int64,
) (_ []resource.VerifiedEntry, returnErr error) {
	if err := validateContext(ctx, "Source tree read"); err != nil {
		return nil, err
	}
	if s == nil || s.sources == nil {
		return nil, model.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := sourceID.Validate(); err != nil {
		return nil, err
	}
	if err := base.Validate(true); err != nil {
		return nil, err
	}
	selection, err := model.NewPathSelection(
		include,
		exclude,
	)
	if err != nil {
		return nil, err
	}
	if maximumEntries <= 0 ||
		maximumEntries > model.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: Source tree entry limit is invalid",
			model.ErrInvalid,
		)
	}
	if maximumBytes <= 0 ||
		maximumBytes > model.MaxScanBytes {
		return nil, fmt.Errorf(
			"%w: Source tree byte limit is invalid",
			model.ErrInvalid,
		)
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return nil, err
	}
	if !value.Enabled {
		return nil, fmt.Errorf(
			"%w: Source %q is disabled",
			model.ErrSourceUnavailable,
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
			model.ErrInvalid,
			base,
			rootEntry.Locator,
		)
	}

	type selectedEntry struct {
		entry    source.Entry
		relative string
	}
	selected := make([]selectedEntry, 0)
	visited := 0

	appendSelected := func(
		entry source.Entry,
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
				model.ErrInvalid,
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
		directory model.Locator,
		depth int,
	) error
	visit = func(
		directory model.Locator,
		depth int,
	) error {
		if depth > model.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: Source tree exceeds depth %d",
				model.ErrInvalid,
				model.DefaultMaxDepth,
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
					model.ErrInvalid,
					entry.Locator,
					directory,
				)
			}
			visited++
			if visited > model.MaxDiscoveryEntries {
				return fmt.Errorf(
					"%w: Source tree exceeds traversal entry limit",
					model.ErrInvalid,
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
			model.ErrInvalid,
			base,
		)
	}

	sort.Slice(selected, func(left, right int) bool {
		return selected[left].entry.Locator <
			selected[right].entry.Locator
	})

	output := make(
		[]resource.VerifiedEntry,
		0,
		len(selected),
	)
	var consumed int64
	for _, selectedEntry := range selected {
		if selectedEntry.entry.SizeBytes >
			maximumBytes-consumed {
			return nil, fmt.Errorf(
				"%w: Source tree exceeds aggregate byte limit",
				model.ErrInvalid,
			)
		}
		perEntryLimit := int64(model.MaxCandidateBytes)
		if remaining := maximumBytes - consumed; remaining < perEntryLimit {
			perEntryLimit = remaining
		}
		content, err := sourceimpl.ReadSnapshotEntry(
			ctx,
			snapshot,
			selectedEntry.entry,
			perEntryLimit,
		)
		if err != nil {
			return nil, err
		}
		consumed += int64(len(content))
		output = append(output, resource.VerifiedEntry{
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
	return output, nil
}

func sourceTreeDirectChild(
	parent model.Locator,
	child model.Locator,
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
	base model.Locator,
	value model.Locator,
) (string, error) {
	if base == "." {
		return string(value), nil
	}
	prefix := string(base) + "/"
	relative, found := strings.CutPrefix(string(value), prefix)
	if !found || relative == "" {
		return "", fmt.Errorf(
			"%w: Source entry %q is outside tree base %q",
			model.ErrInvalid,
			value,
			base,
		)
	}
	return relative, nil
}

func (s *Service) SupportsLocalPath(
	kind source.SourceKind,
) bool {
	if s == nil {
		return false
	}
	localPaths, supported := s.sources.(sourceimpl.LocalPathRuntime)
	return supported && localPaths.SupportsLocalPath(kind)
}

func validateContext(
	ctx context.Context,
	operation string,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: %s context is nil",
			model.ErrInvalid,
			operation,
		)
	}
	return ctx.Err()
}
