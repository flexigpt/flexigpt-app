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

type Result struct {
	Occurrences []catalog.Occurrence
	Definitions map[artifactstore.Digest]definition.Definition
	Diagnostics []artifactstore.Diagnostic
	Candidates  int
}

const (
	DiagnosticCodeCandidateTooLarge         = "artifact.discovery.candidate-too-large"
	DiagnosticCodeDecoderAmbiguous          = "artifact.discovery.decoder-ambiguous"
	DiagnosticCodeDecoderInvalidRecognition = "artifact.discovery.decoder-invalid-recognition"
	DiagnosticCodeDefinitionInvalid         = "artifact.discovery.definition-invalid"
	DiagnosticCodeResourceMissing           = "artifact.discovery.resource-missing"
	DiagnosticCodeSubresourceMissing        = "artifact.discovery.subresource-missing"
)

type DirectoryRoot struct {
	Root            artifactstore.Locator
	Recursive       bool
	IncludePatterns []string
}

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
	sourceID artifactstore.SourceID,
	sourceKind artifactstore.SourceKind,
	snapshot source.Snapshot,
	plan SourcePlan,
	previous []catalog.Occurrence,
) (Result, error) {
	if ctx == nil {
		return Result{}, fmt.Errorf("%w: discovery context is nil", artifactstore.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return Result{}, err
	}
	if err := artifactstore.ValidateSourceID(sourceID); err != nil {
		return Result{}, err
	}
	if err := artifactstore.ValidateSourceKind(sourceKind); err != nil {
		return Result{}, err
	}
	if snapshot == nil {
		return Result{}, fmt.Errorf("%w: source snapshot is nil", artifactstore.ErrInvalid)
	}
	generation := snapshot.Generation()
	if err := artifactstore.ValidateSourceGeneration(generation); err != nil {
		return Result{}, fmt.Errorf("%w: invalid source snapshot generation: %w", artifactstore.ErrInvalid, err)
	}
	if err := plan.Validate(); err != nil {
		return Result{}, err
	}
	plan = plan.Normalized()
	if plan.SourceID != sourceID {
		return Result{}, fmt.Errorf(
			"%w: discovery plan source mismatch",
			artifactstore.ErrInvalid,
		)
	}
	if plan.ExpectedGeneration != "" &&
		generation != plan.ExpectedGeneration {
		return Result{}, fmt.Errorf(
			"%w: source %q changed after discovery planning",
			artifactstore.ErrConflict,
			sourceID,
		)
	}

	allowed := make(map[artifactstore.DecoderID]struct{}, len(plan.AllowedDecoderIDs))
	for _, decoderID := range plan.AllowedDecoderIDs {
		if _, exists := e.decoders.find(decoderID); !exists {
			return Result{}, fmt.Errorf(
				"%w: decoder %q",
				artifactstore.ErrDecoderUnavailable,
				decoderID,
			)
		}
		allowed[decoderID] = struct{}{}
	}

	entries, err := collectCandidates(ctx, snapshot, plan)
	if err != nil {
		return Result{}, err
	}

	occurrences := make(map[catalog.OccurrenceKey]catalog.Occurrence, len(previous))
	for index, value := range previous {
		if value.Key.SourceID != sourceID {
			continue
		}
		if err := value.Validate(); err != nil {
			return Result{}, fmt.Errorf(
				"%w: previous occurrence %d is invalid: %w",
				artifactstore.ErrInvalid,
				index,
				err,
			)
		}
		if value.RootID != rootID {
			return Result{}, fmt.Errorf(
				"%w: previous occurrence %d belongs to another root",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if _, exists := occurrences[value.Key]; exists {
			return Result{}, fmt.Errorf("%w: duplicate previous occurrence", artifactstore.ErrInvalid)
		}
		occurrences[value.Key] = catalog.CloneOccurrence(value)
	}

	result := Result{
		Definitions: make(map[artifactstore.Digest]definition.Definition),
	}
	seenKeys := make(map[catalog.OccurrenceKey]struct{})
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
				Code:     DiagnosticCodeCandidateTooLarge,
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
				sourceID,
				entry.Locator,
				nil,
				"",
				diagnostics,
				now,
			)
			markObservedKeysForLocator(
				seenKeys,
				occurrences,
				sourceID,
				entry.Locator,
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
		sourceDigest := artifactstore.DigestBytes(content)
		candidate := Candidate{
			SourceID:            sourceID,
			SourceKind:          sourceKind,
			Locator:             entry.Locator,
			SourceContentDigest: sourceDigest,
			Content:             content,
			RequestedDecoderIDs: plan.RequestedDecoderIDs(entry.Locator),
		}

		decoder, diagnostics := e.selectDecoder(ctx, candidate, allowed)
		if len(diagnostics) != 0 {
			applyInvalidForLocator(
				occurrences,
				rootID,
				sourceID,
				entry.Locator,
				&sourceDigest,
				"",
				diagnostics,
				now,
			)
			markObservedKeysForLocator(
				seenKeys,
				occurrences,
				sourceID,
				entry.Locator,
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

		decoded, diagnostics := decoder.Decode(ctx, cloneCandidate(candidate))
		if err := validateCandidateDiagnostics(entry.Locator, diagnostics); err != nil {
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
				sourceID,
				entry.Locator,
				&sourceDigest,
				decoder.ID(),
				diagnostics,
				now,
			)
			markObservedKeysForLocator(
				seenKeys,
				occurrences,
				sourceID,
				entry.Locator,
			)
			continue
		}

		emittedForLocator := make(map[catalog.OccurrenceKey]struct{}, len(decoded))
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
				SourceID:           sourceID,
				Locator:            entry.Locator,
				SubresourceLocator: item.SubresourceLocator,
			}
			if _, duplicate := emittedForLocator[key]; duplicate {
				return Result{}, fmt.Errorf(
					"%w: decoder %q emitted duplicate resource at %q and %q",
					artifactstore.ErrInvalid,
					decoder.ID(),
					key.Locator,
					key.SubresourceLocator,
				)
			}
			emittedForLocator[key] = struct{}{}
			seenKeys[key] = struct{}{}
			if err := validateDecodedDiagnostics(
				entry.Locator,
				item.SubresourceLocator,
				item.Diagnostics,
			); err != nil {
				return Result{}, fmt.Errorf(
					"%w: decoder %q returned invalid decoded diagnostics: %w",
					artifactstore.ErrInvalid,
					decoder.ID(),
					err,
				)
			}
			result.Diagnostics = artifactstore.AppendDiagnostics(
				result.Diagnostics,
				item.Diagnostics...,
			)
			itemDiagnostics := artifactstore.AppendDiagnostics(
				diagnostics,
				item.Diagnostics...,
			)
			if artifactstore.ContainsErrorDiagnostic(item.Diagnostics) {
				occurrences[key] = catalog.Occurrence{
					RootID:              rootID,
					Key:                 key,
					SourceContentDigest: &sourceDigest,
					DecoderID:           decoder.ID(),
					State:               catalog.OccurrenceInvalid,
					Diagnostics:         itemDiagnostics,
					ObservedAt:          now,
				}
				continue
			}

			canonical, err := definition.Canonicalize(item.Definition)
			if err != nil {
				definitionDiagnostics := []artifactstore.Diagnostic{{
					Severity: artifactstore.DiagnosticError,
					Code:     DiagnosticCodeDefinitionInvalid,
					Message:  artifactstore.BoundedDiagnosticMessage(err.Error()),
					Location: &artifactstore.DiagnosticLocation{
						Locator:            entry.Locator,
						SubresourceLocator: item.SubresourceLocator,
					},
				}}
				itemDiagnostics = artifactstore.AppendDiagnostics(
					itemDiagnostics,
					definitionDiagnostics...,
				)
				occurrences[key] = catalog.Occurrence{
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
					definitionDiagnostics...,
				)
				continue
			}

			definitionDigest := canonical.Digest
			occurrences[key] = catalog.Occurrence{
				RootID:              rootID,
				Key:                 key,
				Kind:                canonical.Kind,
				LogicalName:         canonical.LogicalName,
				LogicalVersion:      canonical.LogicalVersion,
				DefinitionDigest:    &definitionDigest,
				SourceContentDigest: &sourceDigest,
				DecoderID:           decoder.ID(),
				State:               catalog.OccurrenceValid,
				Diagnostics:         artifactstore.CloneDiagnostics(itemDiagnostics),
				ObservedAt:          now,
			}
			result.Definitions[canonical.Digest] = canonical
		}

		for key, previousValue := range occurrences {
			if previousValue.Key.SourceID != sourceID ||
				previousValue.Key.Locator != entry.Locator {
				continue
			}
			if _, stillPresent := emittedForLocator[key]; stillPresent {
				continue
			}
			previousValue.State = catalog.OccurrenceMissing
			previousValue.Diagnostics = []artifactstore.Diagnostic{{
				Severity: artifactstore.DiagnosticWarning,
				Code:     DiagnosticCodeSubresourceMissing,
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
			if previousValue.Key.SourceID != sourceID {
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
				Code:     DiagnosticCodeResourceMissing,
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
	allowed map[artifactstore.DecoderID]struct{},
) (Decoder, []artifactstore.Diagnostic) {
	var selected Decoder
	best := RecognitionNone
	tied := make([]artifactstore.DecoderID, 0)

	for _, decoder := range e.decoders.registered() {
		if len(allowed) != 0 {
			if _, permitted := allowed[decoder.ID()]; !permitted {
				continue
			}
		}

		recognition := decoder.Recognize(ctx, cloneCandidate(candidate))
		if recognition < RecognitionNone ||
			recognition > RecognitionPreferred {
			return nil, []artifactstore.Diagnostic{{
				Severity: artifactstore.DiagnosticError,
				Code:     DiagnosticCodeDecoderInvalidRecognition,
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
		listed := tied
		const maximumListedDecoders = 16
		if len(listed) > maximumListedDecoders {
			listed = listed[:maximumListedDecoders]
		}
		message := fmt.Sprintf(
			"candidate is equally recognized by decoders %v",
			listed,
		)
		if len(tied) > len(listed) {
			message = fmt.Sprintf(
				"candidate is equally recognized by %v and %d additional decoders",
				listed,
				len(tied)-len(listed),
			)
		}
		return nil, []artifactstore.Diagnostic{{
			Severity: artifactstore.DiagnosticError,
			Code:     DiagnosticCodeDecoderAmbiguous,
			Message:  artifactstore.BoundedDiagnosticMessage(message),
			Location: &artifactstore.DiagnosticLocation{
				Locator: candidate.Locator,
			},
		}}
	}
	return selected, nil
}

func cloneCandidate(value Candidate) Candidate {
	output := value
	output.Content = append([]byte(nil), value.Content...)
	output.RequestedDecoderIDs = append([]artifactstore.DecoderID(nil), value.RequestedDecoderIDs...)
	return output
}

func validateCandidateDiagnostics(
	locator artifactstore.Locator,
	values []artifactstore.Diagnostic,
) error {
	if err := artifactstore.ValidateDiagnostics(values); err != nil {
		return err
	}
	for index, value := range values {
		if value.Location == nil {
			continue
		}
		if value.Location.Locator != "" &&
			value.Location.Locator != locator {
			return fmt.Errorf(
				"diagnostics[%d]: location %q does not belong to candidate %q",
				index,
				value.Location.Locator,
				locator,
			)
		}
		if value.Location.SubresourceLocator != "" {
			return fmt.Errorf(
				"diagnostics[%d]: candidate diagnostic cannot target a subresource",
				index,
			)
		}
	}
	return nil
}

func validateDecodedDiagnostics(
	locator artifactstore.Locator,
	subresource artifactstore.SubresourceLocator,
	values []artifactstore.Diagnostic,
) error {
	if err := artifactstore.ValidateDiagnostics(values); err != nil {
		return err
	}
	for index, value := range values {
		if value.Location == nil {
			continue
		}
		if value.Location.Locator != "" &&
			value.Location.Locator != locator {
			return fmt.Errorf(
				"diagnostics[%d]: location %q does not belong to candidate %q",
				index,
				value.Location.Locator,
				locator,
			)
		}
		if value.Location.SubresourceLocator != "" &&
			value.Location.SubresourceLocator != subresource {
			return fmt.Errorf(
				"diagnostics[%d]: subresource %q does not belong to decoded resource %q",
				index,
				value.Location.SubresourceLocator,
				subresource,
			)
		}
	}
	return nil
}

func collectCandidates(
	ctx context.Context,
	snapshot source.Snapshot,
	plan SourcePlan,
) ([]source.Entry, error) {
	found := make(map[artifactstore.Locator]source.Entry)
	visited := 0

	add := func(entry source.Entry) error {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf(
				"%w: source snapshot returned an invalid entry: %w",
				artifactstore.ErrInvalid,
				err,
			)
		}
		if entry.IsSymlink {
			// A symlink is not a discoverable candidate. It is deliberately
			// ignored rather than making an otherwise valid Workspace fail.
			return nil
		}
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
		entry, err := statEntry(ctx, snapshot, locator)
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
		rootEntry, err := statEntry(ctx, snapshot, root.Root)
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
			entries, err := readDirectoryEntries(ctx, snapshot, directory)
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
					continue
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

func statEntry(
	ctx context.Context,
	snapshot source.Snapshot,
	locator artifactstore.Locator,
) (source.Entry, error) {
	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return source.Entry{}, err
	}
	if err := entry.Validate(); err != nil {
		return source.Entry{}, fmt.Errorf(
			"%w: source snapshot returned an invalid stat entry: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	if entry.Locator != locator {
		return source.Entry{}, fmt.Errorf(
			"%w: source snapshot stat for %q returned %q",
			artifactstore.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return entry, nil
}

func readDirectoryEntries(
	ctx context.Context,
	snapshot source.Snapshot,
	directory artifactstore.Locator,
) ([]source.Entry, error) {
	values, err := snapshot.ReadDir(ctx, directory)
	if err != nil {
		return nil, err
	}

	seen := make(map[artifactstore.Locator]struct{}, len(values))
	output := make([]source.Entry, 0, len(values))
	for _, entry := range values {
		if err := entry.Validate(); err != nil {
			return nil, fmt.Errorf(
				"%w: source snapshot returned an invalid directory entry: %w",
				artifactstore.ErrInvalid,
				err,
			)
		}
		if !isDirectChild(directory, entry.Locator) {
			return nil, fmt.Errorf(
				"%w: source snapshot returned non-child %q for directory %q",
				artifactstore.ErrInvalid,
				entry.Locator,
				directory,
			)
		}
		if _, duplicate := seen[entry.Locator]; duplicate {
			return nil, fmt.Errorf(
				"%w: source snapshot returned duplicate directory entry %q",
				artifactstore.ErrInvalid,
				entry.Locator,
			)
		}
		seen[entry.Locator] = struct{}{}
		output = append(output, entry)
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Locator < output[right].Locator
	})
	return output, nil
}

func isDirectChild(
	parent artifactstore.Locator,
	child artifactstore.Locator,
) bool {
	if child == "." {
		return false
	}
	if parent == "." {
		return !strings.Contains(string(child), "/")
	}
	prefix := string(parent) + "/"
	relative, found := strings.CutPrefix(string(child), prefix)
	return found && relative != "" && !strings.Contains(relative, "/")
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
	values map[catalog.OccurrenceKey]catalog.Occurrence,
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
		previous.Diagnostics = artifactstore.CloneDiagnostics(diagnostics)
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
	values[key] = catalog.Occurrence{
		RootID:              rootID,
		Key:                 key,
		SourceContentDigest: cloneDigest(sourceDigest),
		DecoderID:           decoderID,
		State:               catalog.OccurrenceInvalid,
		Diagnostics:         artifactstore.CloneDiagnostics(diagnostics),
		ObservedAt:          now,
	}
}

func markObservedKeysForLocator(
	seenKeys map[catalog.OccurrenceKey]struct{},
	values map[catalog.OccurrenceKey]catalog.Occurrence,
	sourceID artifactstore.SourceID,
	locator artifactstore.Locator,
) {
	for key, value := range values {
		if value.Key.SourceID != sourceID ||
			value.Key.Locator != locator {
			continue
		}
		seenKeys[key] = struct{}{}
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
