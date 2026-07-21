package discovery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type Engine struct {
	decoders *DecoderRegistry
	clock    artifactstore.Clock
}

func NewEngine(
	decoders *DecoderRegistry,
	clock artifactstore.Clock,
) (*Engine, error) {
	if decoders == nil || clock == nil {
		return nil, fmt.Errorf(
			"%w: discovery engine dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Engine{
		decoders: decoders,
		clock:    clock,
	}, nil
}

func (e *Engine) Discover(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceValue source.Source,
	snapshot source.Snapshot,
	plan SourcePlan,
	previous []catalog.Occurrence,
) (Result, error) {
	plan.ApplyDefaults()
	if err := plan.Validate(); err != nil {
		return Result{}, err
	}
	if plan.SourceID != sourceValue.ID {
		return Result{}, fmt.Errorf(
			"%w: discovery plan source mismatch",
			artifactstore.ErrInvalid,
		)
	}
	if plan.ExpectedGeneration != "" &&
		snapshot.Generation() != plan.ExpectedGeneration {
		return Result{}, fmt.Errorf(
			"%w: source %q changed after discovery planning",
			artifactstore.ErrConflict,
			sourceValue.ID,
		)
	}

	entries, err := collectCandidates(ctx, snapshot, plan)
	if err != nil {
		return Result{}, err
	}

	occurrences := make(map[string]catalog.Occurrence, len(previous))
	for _, value := range previous {
		if value.Key.SourceID != sourceValue.ID {
			continue
		}
		occurrences[value.Key.String()] = value
	}

	result := Result{
		Definitions: make(map[artifactstore.Digest]definition.Definition),
	}
	allowed := make(map[artifactstore.DecoderID]struct{}, len(plan.AllowedDecoderIDs))
	for _, decoderID := range plan.AllowedDecoderIDs {
		if _, exists := e.decoders.Get(decoderID); !exists {
			return Result{}, fmt.Errorf(
				"%w: decoder %q",
				artifactstore.ErrDecoderUnavailable,
				decoderID,
			)
		}
		allowed[decoderID] = struct{}{}
	}

	seenKeys := make(map[string]struct{})
	var consumed int64
	now := e.clock.Now().UTC()

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		result.Candidates++
		if entry.SizeBytes > plan.MaxCandidateBytes {
			diagnostics := []artifactstore.Diagnostic{{
				Severity: artifactstore.DiagnosticError,
				Code:     "artifact.discovery.candidate-too-large",
				Message: fmt.Sprintf(
					"candidate exceeds the %d byte limit",
					plan.MaxCandidateBytes,
				),
				Location: &artifactstore.DiagnosticLocation{
					Locator: entry.Locator,
				},
			}}
			applyInvalidForLocator(
				occurrences,
				rootID,
				sourceValue.ID,
				entry.Locator,
				nil,
				"",
				diagnostics,
				now,
			)
			result.Diagnostics = artifactstore.AppendDiagnostics(
				result.Diagnostics,
				diagnostics...,
			)
			continue
		}
		if entry.SizeBytes > plan.MaxTotalBytes-consumed {
			return Result{}, fmt.Errorf(
				"%w: discovery exceeds total byte limit",
				artifactstore.ErrInvalid,
			)
		}

		content, err := readEntry(
			ctx,
			snapshot,
			entry,
			plan.MaxCandidateBytes,
		)
		if err != nil {
			return Result{}, err
		}
		consumed += int64(len(content))
		if consumed > plan.MaxTotalBytes {
			return Result{}, fmt.Errorf(
				"%w: discovery exceeds total byte limit",
				artifactstore.ErrInvalid,
			)
		}
		sourceDigest := definition.DigestBytes(content)
		candidate := Candidate{
			Source:              sourceValue,
			Locator:             entry.Locator,
			SourceContentDigest: sourceDigest,
			Content:             content,
		}

		decoder, diagnostics := e.selectDecoder(ctx, candidate)
		if len(diagnostics) != 0 {
			applyInvalidForLocator(
				occurrences,
				rootID,
				sourceValue.ID,
				entry.Locator,
				&sourceDigest,
				"",
				diagnostics,
				now,
			)
			result.Diagnostics = artifactstore.AppendDiagnostics(
				result.Diagnostics,
				diagnostics...,
			)
			continue
		}
		if decoder == nil {
			continue
		}
		if len(allowed) != 0 {
			if _, permitted := allowed[decoder.ID()]; !permitted {
				continue
			}
		}

		decoded, diagnostics := decoder.Decode(ctx, candidate)
		if err := artifactstore.ValidateDiagnostics(diagnostics); err != nil {
			return Result{}, fmt.Errorf(
				"%w: decoder %q returned invalid diagnostics: %w",
				artifactstore.ErrInvalid,
				decoder.ID(),
				err,
			)
		}
		result.Diagnostics = artifactstore.AppendDiagnostics(
			result.Diagnostics,
			diagnostics...,
		)

		if artifactstore.ContainsErrorDiagnostic(diagnostics) {
			applyInvalidForLocator(
				occurrences,
				rootID,
				sourceValue.ID,
				entry.Locator,
				&sourceDigest,
				decoder.ID(),
				diagnostics,
				now,
			)
			continue
		}

		emittedForLocator := make(map[string]struct{}, len(decoded))
		for _, item := range decoded {
			if err := artifactstore.ValidateSubresourceLocator(
				item.SubresourceLocator,
			); err != nil {
				return Result{}, fmt.Errorf(
					"%w: decoder %q emitted invalid subresource: %w",
					artifactstore.ErrInvalid,
					decoder.ID(),
					err,
				)
			}
			key := catalog.OccurrenceKey{
				SourceID:           sourceValue.ID,
				Locator:            entry.Locator,
				SubresourceLocator: item.SubresourceLocator,
			}
			if _, duplicate := emittedForLocator[key.String()]; duplicate {
				return Result{}, fmt.Errorf(
					"%w: decoder %q emitted duplicate resource %q",
					artifactstore.ErrInvalid,
					decoder.ID(),
					key.String(),
				)
			}
			emittedForLocator[key.String()] = struct{}{}
			seenKeys[key.String()] = struct{}{}

			canonical, err := definition.Canonicalize(item.Definition)
			if err != nil {
				itemDiagnostics := []artifactstore.Diagnostic{{
					Severity: artifactstore.DiagnosticError,
					Code:     "artifact.discovery.definition-invalid",
					Message:  err.Error(),
					Location: &artifactstore.DiagnosticLocation{
						Locator:            entry.Locator,
						SubresourceLocator: item.SubresourceLocator,
					},
				}}
				occurrences[key.String()] = catalog.Occurrence{
					RootID:              rootID,
					Key:                 key,
					SourceContentDigest: &sourceDigest,
					DecoderID:           decoder.ID(),
					State:               catalog.OccurrenceInvalid,
					Diagnostics:         itemDiagnostics,
					ObservedAt:          now,
				}
				result.Diagnostics = artifactstore.AppendDiagnostics(
					result.Diagnostics,
					itemDiagnostics...,
				)
				continue
			}

			definitionDigest := canonical.Digest
			occurrences[key.String()] = catalog.Occurrence{
				RootID:              rootID,
				Key:                 key,
				Kind:                canonical.Kind,
				LogicalName:         canonical.LogicalName,
				LogicalVersion:      canonical.LogicalVersion,
				DefinitionDigest:    &definitionDigest,
				SourceContentDigest: &sourceDigest,
				DecoderID:           decoder.ID(),
				State:               catalog.OccurrenceValid,
				Diagnostics:         append([]artifactstore.Diagnostic(nil), diagnostics...),
				ObservedAt:          now,
			}
			result.Definitions[canonical.Digest] = canonical
		}

		for key, previousValue := range occurrences {
			if previousValue.Key.SourceID != sourceValue.ID ||
				previousValue.Key.Locator != entry.Locator {
				continue
			}
			if _, stillPresent := emittedForLocator[key]; stillPresent {
				continue
			}
			previousValue.State = catalog.OccurrenceMissing
			previousValue.Diagnostics = []artifactstore.Diagnostic{{
				Severity: artifactstore.DiagnosticWarning,
				Code:     "artifact.discovery.subresource-missing",
				Message:  "the decoder no longer emits this subresource",
				Location: &artifactstore.DiagnosticLocation{
					Locator:            previousValue.Key.Locator,
					SubresourceLocator: previousValue.Key.SubresourceLocator,
				},
			}}
			previousValue.ObservedAt = now
			occurrences[key] = previousValue
		}
	}

	if plan.Authoritative {
		for key, previousValue := range occurrences {
			if previousValue.Key.SourceID != sourceValue.ID {
				continue
			}
			if _, observed := seenKeys[key]; observed {
				continue
			}
			if !locatorInScope(previousValue.Key.Locator, plan) {
				continue
			}
			previousValue.State = catalog.OccurrenceMissing
			previousValue.Diagnostics = []artifactstore.Diagnostic{{
				Severity: artifactstore.DiagnosticWarning,
				Code:     "artifact.discovery.resource-missing",
				Message:  "the source occurrence was not found during authoritative discovery",
				Location: &artifactstore.DiagnosticLocation{
					Locator:            previousValue.Key.Locator,
					SubresourceLocator: previousValue.Key.SubresourceLocator,
				},
			}}
			previousValue.ObservedAt = now
			occurrences[key] = previousValue
		}
	}

	for _, value := range occurrences {
		result.Occurrences = append(result.Occurrences, value)
	}
	catalog.SortOccurrences(result.Occurrences)
	return result, nil
}

func (e *Engine) selectDecoder(
	ctx context.Context,
	candidate Candidate,
) (Decoder, []artifactstore.Diagnostic) {
	var selected Decoder
	best := RecognitionNone
	tied := make([]artifactstore.DecoderID, 0)

	for _, decoder := range e.decoders.All() {
		recognition := decoder.Recognize(ctx, candidate)
		if recognition < RecognitionNone ||
			recognition > RecognitionPreferred {
			return nil, []artifactstore.Diagnostic{{
				Severity: artifactstore.DiagnosticError,
				Code:     "artifact.discovery.decoder-invalid-recognition",
				Message: fmt.Sprintf(
					"decoder %q returned invalid recognition %d",
					decoder.ID(),
					recognition,
				),
				Location: &artifactstore.DiagnosticLocation{
					Locator: candidate.Locator,
				},
			}}
		}
		if recognition > best {
			best = recognition
			selected = decoder
			tied = []artifactstore.DecoderID{decoder.ID()}
		} else if recognition == best && recognition != RecognitionNone {
			tied = append(tied, decoder.ID())
		}
	}
	if len(tied) > 1 {
		slices.Sort(tied)
		return nil, []artifactstore.Diagnostic{{
			Severity: artifactstore.DiagnosticError,
			Code:     "artifact.discovery.decoder-ambiguous",
			Message: fmt.Sprintf(
				"candidate is equally recognized by decoders %v",
				tied,
			),
			Location: &artifactstore.DiagnosticLocation{
				Locator: candidate.Locator,
			},
		}}
	}
	return selected, nil
}

func collectCandidates(
	ctx context.Context,
	snapshot source.Snapshot,
	plan SourcePlan,
) ([]source.Entry, error) {
	found := make(map[artifactstore.Locator]source.Entry)
	visited := 0

	add := func(entry source.Entry) error {
		if !entry.IsRegular {
			return nil
		}
		if _, exists := found[entry.Locator]; !exists &&
			len(found) >= plan.MaxCandidates {
			return fmt.Errorf(
				"%w: discovery exceeds %d candidates",
				artifactstore.ErrInvalid,
				plan.MaxCandidates,
			)
		}
		found[entry.Locator] = entry
		return nil
	}

	for _, locator := range plan.ExplicitLocators {
		entry, err := snapshot.Stat(ctx, locator)
		if errors.Is(err, artifactstore.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if err := add(entry); err != nil {
			return nil, err
		}
	}

	for _, root := range plan.DirectoryRoots {
		rootEntry, err := snapshot.Stat(ctx, root.Root)
		if errors.Is(err, artifactstore.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !rootEntry.IsDirectory {
			return nil, fmt.Errorf(
				"%w: discovery root %q is not a directory",
				artifactstore.ErrInvalid,
				root.Root,
			)
		}

		var visit func(artifactstore.Locator, int) error
		visit = func(directory artifactstore.Locator, depth int) error {
			entries, err := snapshot.ReadDir(ctx, directory)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return err
				}
				visited++
				if visited > plan.MaxEntries {
					return fmt.Errorf(
						"%w: discovery exceeds %d entries",
						artifactstore.ErrInvalid,
						plan.MaxEntries,
					)
				}
				nextDepth := depth + 1
				if nextDepth > plan.MaxDepth {
					return fmt.Errorf(
						"%w: discovery exceeds depth %d at %q",
						artifactstore.ErrInvalid,
						plan.MaxDepth,
						entry.Locator,
					)
				}
				if entry.IsSymlink {
					return fmt.Errorf(
						"%w: discovery refuses symbolic link %q",
						artifactstore.ErrInvalid,
						entry.Locator,
					)
				}
				if entry.IsDirectory {
					if root.Recursive {
						if err := visit(entry.Locator, nextDepth); err != nil {
							return err
						}
					}
					continue
				}
				if entry.IsRegular &&
					matchesDirectoryRoot(root, entry.Locator) {
					if err := add(entry); err != nil {
						return err
					}
				}
			}
			return nil
		}
		if err := visit(root.Root, 0); err != nil {
			return nil, err
		}
	}

	output := make([]source.Entry, 0, len(found))
	for _, value := range found {
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Locator < output[right].Locator
	})
	return output, nil
}

func readEntry(
	ctx context.Context,
	snapshot source.Snapshot,
	entry source.Entry,
	maximum int64,
) ([]byte, error) {
	reader, err := snapshot.Open(ctx, entry.Locator)
	if err != nil {
		return nil, err
	}
	content, readErr := io.ReadAll(io.LimitReader(reader, maximum+1))
	closeErr := reader.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(content)) > maximum {
		return nil, fmt.Errorf(
			"%w: candidate %q exceeds byte limit",
			artifactstore.ErrInvalid,
			entry.Locator,
		)
	}
	if int64(len(content)) != entry.SizeBytes {
		return nil, fmt.Errorf(
			"%w: candidate %q changed size during discovery",
			artifactstore.ErrConflict,
			entry.Locator,
		)
	}
	return content, nil
}

func applyInvalidForLocator(
	values map[string]catalog.Occurrence,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	locator artifactstore.Locator,
	sourceDigest *artifactstore.Digest,
	decoderID artifactstore.DecoderID,
	diagnostics []artifactstore.Diagnostic,
	now time.Time,
) {
	matched := false
	for key, previous := range values {
		if previous.Key.SourceID != sourceID ||
			previous.Key.Locator != locator {
			continue
		}
		matched = true
		previous.SourceContentDigest = cloneDigest(sourceDigest)
		previous.DecoderID = decoderID
		previous.State = catalog.OccurrenceInvalid
		previous.Diagnostics = append([]artifactstore.Diagnostic(nil), diagnostics...)
		previous.ObservedAt = now
		values[key] = previous
	}
	if matched {
		return
	}
	key := catalog.OccurrenceKey{
		SourceID: sourceID,
		Locator:  locator,
	}
	values[key.String()] = catalog.Occurrence{
		RootID:              rootID,
		Key:                 key,
		SourceContentDigest: cloneDigest(sourceDigest),
		DecoderID:           decoderID,
		State:               catalog.OccurrenceInvalid,
		Diagnostics:         append([]artifactstore.Diagnostic(nil), diagnostics...),
		ObservedAt:          now,
	}
}

func cloneDigest(value *artifactstore.Digest) *artifactstore.Digest {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func locatorInScope(
	locator artifactstore.Locator,
	plan SourcePlan,
) bool {
	if slices.Contains(plan.ExplicitLocators, locator) {
		return true
	}
	for _, root := range plan.DirectoryRoots {
		if matchesDirectoryRoot(root, locator) {
			return true
		}
	}
	return false
}

func matchesDirectoryRoot(
	root DirectoryRoot,
	locator artifactstore.Locator,
) bool {
	base := string(root.Root)
	value := string(locator)
	relative := value
	if base != "." {
		prefix := base + "/"
		if !strings.HasPrefix(value, prefix) {
			return false
		}
		relative = strings.TrimPrefix(value, prefix)
	}
	if !root.Recursive && strings.Contains(relative, "/") {
		return false
	}
	if len(root.IncludePatterns) == 0 {
		return true
	}
	for _, pattern := range root.IncludePatterns {
		if matched, _ := path.Match(pattern, relative); matched {
			return true
		}
		if !strings.Contains(pattern, "/") {
			if matched, _ := path.Match(pattern, path.Base(relative)); matched {
				return true
			}
		}
	}
	return false
}
