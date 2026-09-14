package resource

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifact"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type sourceRefreshInspector interface {
	InspectSource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
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
			basespec.ErrInvalid,
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
	options resource.ResolveOptions,
) (resource.ResolvedArtifact, error) {
	if err := validateContext(ctx, "Artifact resolution"); err != nil {
		return resource.ResolvedArtifact{}, err
	}
	if s == nil {
		return resource.ResolvedArtifact{}, basespec.ErrClosed
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
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	inspection, err := s.refresh.InspectSource(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return resource.ResolvedArtifact{}, err
	}
	if !inspection.IsCurrent() {
		return resource.ResolvedArtifact{}, fmt.Errorf(
			"%w: Artifact Source %q requires refresh",
			basespec.ErrRefreshRequired,
			record.Binding.SourceID,
		)
	}

	value, err := s.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return resource.ResolvedArtifact{}, err
	}
	if !value.Enabled {
		return resource.ResolvedArtifact{}, fmt.Errorf(
			"%w: Artifact Source %q is disabled",
			basespec.ErrSourceUnavailable,
			value.ID,
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
	output := resource.ResolvedArtifact{
		Artifact:     record.Clone(),
		Definition:   definitionValue.Clone(),
		Source:       value.Summary(),
		RefreshState: inspection.State.Clone(),
	}
	if err := output.Validate(); err != nil {
		return resource.ResolvedArtifact{}, err
	}
	if err := sourceimpl.VerifySnapshotContentDigest(
		ctx,
		s.sources,
		value,
		record.Binding.Locator,
		inspection.State.SourceGeneration,
		*record.SourceContentDigest,
		basespec.MaxCandidateBytes,
	); err != nil {
		return resource.ResolvedArtifact{}, err
	}
	return output.Clone(), nil
}

func (s *Service) ResolveVerifiedLocalPath(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
	localLocator basespec.Locator,
) (string, error) {
	if err := validateContext(
		ctx,
		"verified local-path resolution",
	); err != nil {
		return "", err
	}
	if s == nil {
		return "", basespec.ErrClosed
	}
	if err := resolved.Validate(); err != nil {
		return "", err
	}
	if err := localLocator.Validate(true); err != nil {
		return "", err
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
			basespec.ErrSourceUnavailable,
			value.ID,
		)
	}
	if value.Revision != resolved.RefreshState.SourceRevision {
		return "", fmt.Errorf(
			"%w: Source changed after Artifact resolution",
			basespec.ErrRefreshRequired,
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
		basespec.MaxCandidateBytes,
	)
}

func (s *Service) ReadSourceEntry(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	locator basespec.Locator,
	maximumBytes int64,
) (resource.VerifiedEntry, error) {
	if err := validateContext(ctx, "Source entry read"); err != nil {
		return resource.VerifiedEntry{}, err
	}
	if s == nil {
		return resource.VerifiedEntry{}, basespec.ErrClosed
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
	if maximumBytes <= 0 || maximumBytes > basespec.MaxScanBytes {
		return resource.VerifiedEntry{}, fmt.Errorf(
			"%w: Source entry read limit is invalid",
			basespec.ErrInvalid,
		)
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return resource.VerifiedEntry{}, err
	}
	if !value.Enabled {
		return resource.VerifiedEntry{}, fmt.Errorf(
			"%w: Source %q is disabled",
			basespec.ErrSourceUnavailable,
			value.ID,
		)
	}
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return resource.VerifiedEntry{}, err
	}
	defer snapshot.Close()

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
			basespec.ErrInvalid,
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
	if err := output.Validate(); err != nil {
		return resource.VerifiedEntry{}, err
	}
	return output.Clone(), nil
}

// StatSourceEntry confirms one Source snapshot and returns only physical Entry
// metadata. It is used for declaration planning, not as a Resource substitute.
func (s *Service) StatSourceEntry(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	locator basespec.Locator,
) (source.Entry, error) {
	if err := validateContext(ctx, "Source entry stat"); err != nil {
		return source.Entry{}, err
	}
	if s == nil {
		return source.Entry{}, basespec.ErrClosed
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
			basespec.ErrSourceUnavailable,
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
			basespec.ErrInvalid,
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
	base basespec.Locator,
	include []string,
	exclude []string,
	maximumEntries int,
	maximumBytes int64,
) ([]resource.VerifiedEntry, error) {
	if err := validateContext(ctx, "Source tree read"); err != nil {
		return nil, err
	}
	if s == nil || s.sources == nil {
		return nil, basespec.ErrClosed
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
	if err := basespec.ValidatePathPatterns(
		"Source tree include patterns",
		include,
	); err != nil {
		return nil, err
	}
	if err := basespec.ValidatePathPatterns(
		"Source tree exclude patterns",
		exclude,
	); err != nil {
		return nil, err
	}
	if maximumEntries <= 0 ||
		maximumEntries > basespec.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: Source tree entry limit is invalid",
			basespec.ErrInvalid,
		)
	}
	if maximumBytes <= 0 ||
		maximumBytes > basespec.MaxScanBytes {
		return nil, fmt.Errorf(
			"%w: Source tree byte limit is invalid",
			basespec.ErrInvalid,
		)
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return nil, err
	}
	if !value.Enabled {
		return nil, fmt.Errorf(
			"%w: Source %q is disabled",
			basespec.ErrSourceUnavailable,
			value.ID,
		)
	}
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return nil, err
	}
	defer snapshot.Close()

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
			basespec.ErrInvalid,
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
		matched, err := basespec.MatchPathSelection(
			relative,
			include,
			exclude,
		)
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		if len(selected) == maximumEntries {
			return fmt.Errorf(
				"%w: Source tree exceeds %d selected entries",
				basespec.ErrInvalid,
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
		directory basespec.Locator,
		depth int,
	) error
	visit = func(
		directory basespec.Locator,
		depth int,
	) error {
		if depth > basespec.DefaultMaxDepth {
			return fmt.Errorf(
				"%w: Source tree exceeds depth %d",
				basespec.ErrInvalid,
				basespec.DefaultMaxDepth,
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
					basespec.ErrInvalid,
					entry.Locator,
					directory,
				)
			}
			visited++
			if visited > basespec.MaxDiscoveryEntries {
				return fmt.Errorf(
					"%w: Source tree exceeds traversal entry limit",
					basespec.ErrInvalid,
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
			basespec.ErrInvalid,
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
				basespec.ErrInvalid,
			)
		}
		perEntryLimit := int64(basespec.MaxCandidateBytes)
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
	for index := range output {
		if err := output[index].Validate(); err != nil {
			return nil, err
		}
		output[index] = output[index].Clone()
	}
	return output, nil
}

func sourceTreeDirectChild(
	parent basespec.Locator,
	child basespec.Locator,
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
	base basespec.Locator,
	value basespec.Locator,
) (string, error) {
	if base == "." {
		return string(value), nil
	}
	prefix := string(base) + "/"
	relative, found := strings.CutPrefix(string(value), prefix)
	if !found || relative == "" {
		return "", fmt.Errorf(
			"%w: Source entry %q is outside tree base %q",
			basespec.ErrInvalid,
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
			basespec.ErrInvalid,
			operation,
		)
	}
	return ctx.Err()
}
