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

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type Result struct {
	Occurrences []catalog.Occurrence
	Definitions map[cryptoutil.Digest]definition.Definition
	Diagnostics []diagnostic.Diagnostic
	Candidates  int
}

const (
	DiagnosticCodeCandidateTooLarge         = "artifact.discovery.candidate-too-large"
	DiagnosticCodeContentDigestMismatch     = "artifact.discovery.content-digest-mismatch"
	DiagnosticCodeDecoderAmbiguous          = "artifact.discovery.decoder-ambiguous"
	DiagnosticCodeDecoderInvalidRecognition = "artifact.discovery.decoder-invalid-recognition"
	DiagnosticCodeDecoderNoLongerRecognizes = "artifact.discovery.decoder-no-longer-recognizes"
	DiagnosticCodeDefinitionInvalid         = "artifact.discovery.definition-invalid"
	DiagnosticCodeResourceMissing           = "artifact.discovery.resource-missing"
	DiagnosticCodeSubresourceMissing        = "artifact.discovery.subresource-missing"
)

type Engine struct {
	decoders *DecoderRegistry
	clock    clockutil.Clock
}

func NewEngine(
	decoders *DecoderRegistry,
	timeClock clockutil.Clock,
) (*Engine, error) {
	if decoders == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: discovery engine dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Engine{
		decoders: decoders,
		clock:    timeClock,
	}, nil
}

func (e *Engine) DecoderFingerprint() (cryptoutil.Digest, error) {
	return e.decoders.Fingerprint()
}

func (e *Engine) Discover(
	ctx context.Context,
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
	sourceID basespec.SourceID,
	sourceKind basespec.SourceKind,
	snapshot source.Snapshot,
	plan SourcePlan,
	previous []catalog.Occurrence,
) (Result, error) {
	if ctx == nil {
		return Result{}, fmt.Errorf("%w: discovery context is nil", basespec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Result{}, err
	}
	if err := basespec.ValidateCollectionID(collectionID); err != nil {
		return Result{}, err
	}
	if err := basespec.ValidateSourceID(sourceID); err != nil {
		return Result{}, err
	}
	if err := basespec.ValidateSourceKind(sourceKind); err != nil {
		return Result{}, err
	}
	if snapshot == nil {
		return Result{}, fmt.Errorf("%w: source snapshot is nil", basespec.ErrInvalid)
	}
	generation := snapshot.Generation()
	if err := basespec.ValidateSourceGeneration(generation); err != nil {
		return Result{}, fmt.Errorf("%w: invalid source snapshot generation: %w", basespec.ErrInvalid, err)
	}
	if err := plan.Validate(); err != nil {
		return Result{}, err
	}
	plan = plan.Normalized()
	if plan.SourceID != sourceID {
		return Result{}, fmt.Errorf(
			"%w: discovery plan source mismatch",
			basespec.ErrInvalid,
		)
	}
	if plan.ExpectedGeneration != "" &&
		generation != plan.ExpectedGeneration {
		return Result{}, fmt.Errorf(
			"%w: source %q changed after discovery planning",
			basespec.ErrConflict,
			sourceID,
		)
	}

	allowed := make(map[basespec.DecoderID]struct{}, len(plan.AllowedDecoderIDs))
	for _, decoderID := range plan.AllowedDecoderIDs {
		if _, exists := e.decoders.find(decoderID); !exists {
			return Result{}, fmt.Errorf(
				"%w: decoder %q",
				basespec.ErrDecoderUnavailable,
				decoderID,
			)
		}
		allowed[decoderID] = struct{}{}
	}
	for _, hint := range plan.DecoderHints {
		for _, decoderID := range hint.DecoderIDs {
			if _, exists := e.decoders.find(decoderID); !exists {
				return Result{}, fmt.Errorf(
					"%w: decoder %q",
					basespec.ErrDecoderUnavailable,
					decoderID,
				)
			}
			if len(allowed) != 0 {
				if _, permitted := allowed[decoderID]; !permitted {
					return Result{}, fmt.Errorf(
						"%w: hinted decoder %q is not allowed by its source plan",
						basespec.ErrInvalid,
						decoderID,
					)
				}
			}
		}
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
				basespec.ErrInvalid,
				index,
				err,
			)
		}
		if value.CollectionID != collectionID ||
			value.Key.CollectionID != collectionID {
			return Result{}, fmt.Errorf(
				"%w: previous occurrence %d belongs to another collection",
				basespec.ErrInvalid,
				index,
			)
		}
		if value.RootID != rootID {
			return Result{}, fmt.Errorf(
				"%w: previous occurrence %d belongs to another root",
				basespec.ErrInvalid,
				index,
			)
		}
		if _, exists := occurrences[value.Key]; exists {
			return Result{}, fmt.Errorf("%w: duplicate previous occurrence", basespec.ErrInvalid)
		}
		occurrences[value.Key] = catalog.CloneOccurrence(value)
	}

	result := Result{
		Definitions: make(map[cryptoutil.Digest]definition.Definition),
	}
	seenKeys := make(map[catalog.OccurrenceKey]struct{})
	var consumed int64
	now := clockutil.NowUTC(e.clock)

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		result.Candidates++
		if entry.SizeBytes > plan.MaxCandidateBytes {
			diagnostics := []diagnostic.Diagnostic{{
				Severity: diagnostic.DiagnosticError,
				Code:     DiagnosticCodeCandidateTooLarge,
				Message: fmt.Sprintf(
					"candidate exceeds the %d byte limit",
					plan.MaxCandidateBytes,
				),
				Location: &diagnostic.DiagnosticLocation{
					Locator: entry.Locator,
				},
			}}
			applyInvalidForLocator(
				occurrences,
				rootID,
				collectionID,
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
			result.Diagnostics = diagnostic.AppendDiagnostics(
				result.Diagnostics,
				diagnostics...,
			)
			continue
		}
		if entry.SizeBytes > plan.MaxTotalBytes-consumed {
			return Result{}, fmt.Errorf(
				"%w: discovery exceeds total byte limit",
				basespec.ErrInvalid,
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
				basespec.ErrInvalid,
			)
		}
		sourceDigest := cryptoutil.DigestBytes(content)
		if expectedDigest, expected := plan.ExpectedContentDigests[entry.Locator]; expected &&
			sourceDigest != expectedDigest {
			diagnostics := []diagnostic.Diagnostic{{
				Severity: diagnostic.DiagnosticError,
				Code:     DiagnosticCodeContentDigestMismatch,
				Message:  "candidate content does not match its expected portable digest",
				Location: &diagnostic.DiagnosticLocation{
					Locator: entry.Locator,
				},
			}}
			applyInvalidForLocator(
				occurrences,
				rootID,
				collectionID,
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
			result.Diagnostics = diagnostic.AppendDiagnostics(
				result.Diagnostics,
				diagnostics...,
			)
			continue
		}
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
				collectionID,
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
			result.Diagnostics = diagnostic.AppendDiagnostics(
				result.Diagnostics,
				diagnostics...,
			)
			continue
		}
		if decoder == nil {
			diagnostics := markUnrecognizedForLocator(
				occurrences,
				sourceID,
				entry.Locator,
				now,
			)
			if len(diagnostics) != 0 {
				result.Diagnostics = diagnostic.AppendDiagnostics(
					result.Diagnostics,
					diagnostics...,
				)
			}
			markObservedKeysForLocator(
				seenKeys,
				occurrences,
				sourceID,
				entry.Locator,
			)
			continue
		}

		decoded, diagnostics := decoder.Decode(ctx, cloneCandidate(candidate))
		if err := validateCandidateDiagnostics(entry.Locator, diagnostics); err != nil {
			return Result{}, fmt.Errorf(
				"%w: decoder %q returned invalid diagnostics: %w",
				basespec.ErrInvalid,
				decoder.ID(),
				err,
			)
		}
		result.Diagnostics = diagnostic.AppendDiagnostics(
			result.Diagnostics,
			diagnostics...,
		)

		if diagnostic.ContainsErrorDiagnostic(diagnostics) {
			applyInvalidForLocator(
				occurrences,
				rootID,
				collectionID,
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
			if err := basespec.ValidateSubresourceLocator(
				item.SubresourceLocator,
			); err != nil {
				return Result{}, fmt.Errorf(
					"%w: decoder %q emitted invalid subresource: %w",
					basespec.ErrInvalid,
					decoder.ID(),
					err,
				)
			}
			key := catalog.OccurrenceKey{
				CollectionID:       collectionID,
				SourceID:           sourceID,
				Locator:            entry.Locator,
				SubresourceLocator: item.SubresourceLocator,
			}
			if _, duplicate := emittedForLocator[key]; duplicate {
				return Result{}, fmt.Errorf(
					"%w: decoder %q emitted duplicate resource at %q and %q",
					basespec.ErrInvalid,
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
					basespec.ErrInvalid,
					decoder.ID(),
					err,
				)
			}
			result.Diagnostics = diagnostic.AppendDiagnostics(
				result.Diagnostics,
				item.Diagnostics...,
			)
			itemDiagnostics := diagnostic.AppendDiagnostics(
				diagnostics,
				item.Diagnostics...,
			)
			if diagnostic.ContainsErrorDiagnostic(item.Diagnostics) {
				previous, found := occurrences[key]
				if found {
					previous.DefinitionDigest = nil
					previous.SourceContentDigest = &sourceDigest
					previous.DecoderID = decoder.ID()
					previous.State = catalog.OccurrenceInvalid
					previous.Diagnostics = diagnostic.CloneDiagnostics(itemDiagnostics)
					previous.ObservedAt = now
					occurrences[key] = previous
					continue
				}

				occurrences[key] = catalog.Occurrence{
					RootID:              rootID,
					CollectionID:        collectionID,
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
				definitionDiagnostics := []diagnostic.Diagnostic{{
					Severity: diagnostic.DiagnosticError,
					Code:     DiagnosticCodeDefinitionInvalid,
					Message:  diagnostic.BoundedDiagnosticMessage(err.Error()),
					Location: &diagnostic.DiagnosticLocation{
						Locator:            entry.Locator,
						SubresourceLocator: item.SubresourceLocator,
					},
				}}
				itemDiagnostics = diagnostic.AppendDiagnostics(
					itemDiagnostics,
					definitionDiagnostics...,
				)
				if previous, found := occurrences[key]; found {
					previous.DefinitionDigest = nil
					previous.SourceContentDigest = &sourceDigest
					previous.DecoderID = decoder.ID()
					previous.State = catalog.OccurrenceInvalid
					previous.Diagnostics = diagnostic.CloneDiagnostics(itemDiagnostics)
					previous.ObservedAt = now
					occurrences[key] = previous
					result.Diagnostics = diagnostic.AppendDiagnostics(
						result.Diagnostics,
						definitionDiagnostics...,
					)
					continue
				}

				occurrences[key] = catalog.Occurrence{
					RootID:              rootID,
					CollectionID:        collectionID,
					Key:                 key,
					SourceContentDigest: &sourceDigest,
					DecoderID:           decoder.ID(),
					State:               catalog.OccurrenceInvalid,
					Diagnostics:         itemDiagnostics,
					ObservedAt:          now,
				}
				result.Diagnostics = diagnostic.AppendDiagnostics(
					result.Diagnostics,
					definitionDiagnostics...,
				)
				continue
			}

			definitionDigest := canonical.Digest
			occurrences[key] = catalog.Occurrence{
				RootID:              rootID,
				Key:                 key,
				CollectionID:        collectionID,
				Kind:                canonical.Kind,
				LogicalName:         canonical.LogicalName,
				LogicalVersion:      canonical.LogicalVersion,
				DefinitionDigest:    &definitionDigest,
				SourceContentDigest: &sourceDigest,
				DecoderID:           decoder.ID(),
				State:               catalog.OccurrenceValid,
				Diagnostics:         diagnostic.CloneDiagnostics(itemDiagnostics),
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
			seenKeys[key] = struct{}{}
			previousValue.State = catalog.OccurrenceMissing
			previousValue.Diagnostics = []diagnostic.Diagnostic{{
				Severity: diagnostic.DiagnosticWarning,
				Code:     DiagnosticCodeSubresourceMissing,
				Message:  "the decoder no longer emits this subresource",
				Location: &diagnostic.DiagnosticLocation{
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
			previousValue.Diagnostics = []diagnostic.Diagnostic{{
				Severity: diagnostic.DiagnosticWarning,
				Code:     DiagnosticCodeResourceMissing,
				Message:  "the source occurrence was not found during authoritative discovery",
				Location: &diagnostic.DiagnosticLocation{
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
	allowed map[basespec.DecoderID]struct{},
) (Decoder, []diagnostic.Diagnostic) {
	var selected Decoder
	best := RecognitionNone
	tied := make([]basespec.DecoderID, 0)

	for _, decoder := range e.decoders.registered() {
		if len(allowed) != 0 {
			if _, permitted := allowed[decoder.ID()]; !permitted {
				continue
			}
		}

		recognition := decoder.Recognize(ctx, cloneCandidate(candidate))
		if recognition < RecognitionNone ||
			recognition > RecognitionPreferred {
			return nil, []diagnostic.Diagnostic{{
				Severity: diagnostic.DiagnosticError,
				Code:     DiagnosticCodeDecoderInvalidRecognition,
				Message: fmt.Sprintf(
					"decoder %q returned invalid recognition %d",
					decoder.ID(),
					recognition,
				),
				Location: &diagnostic.DiagnosticLocation{
					Locator: candidate.Locator,
				},
			}}
		}
		if recognition > best {
			best = recognition
			selected = decoder
			tied = []basespec.DecoderID{decoder.ID()}
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
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     DiagnosticCodeDecoderAmbiguous,
			Message:  diagnostic.BoundedDiagnosticMessage(message),
			Location: &diagnostic.DiagnosticLocation{
				Locator: candidate.Locator,
			},
		}}
	}
	return selected, nil
}

func cloneCandidate(value Candidate) Candidate {
	output := value
	output.Content = append([]byte(nil), value.Content...)
	output.RequestedDecoderIDs = append([]basespec.DecoderID(nil), value.RequestedDecoderIDs...)
	return output
}

func validateCandidateDiagnostics(
	locator basespec.Locator,
	values []diagnostic.Diagnostic,
) error {
	if err := diagnostic.ValidateDiagnostics(values); err != nil {
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
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	values []diagnostic.Diagnostic,
) error {
	if err := diagnostic.ValidateDiagnostics(values); err != nil {
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
	found := make(map[basespec.Locator]source.Entry)
	visited := 0

	add := func(entry source.Entry) error {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf(
				"%w: source snapshot returned an invalid entry: %w",
				basespec.ErrInvalid,
				err,
			)
		}
		if !entry.IsRegular {
			return nil
		}
		if _, exists := found[entry.Locator]; !exists &&
			len(found) >= plan.MaxCandidates {
			return fmt.Errorf(
				"%w: discovery exceeds %d candidates",
				basespec.ErrInvalid,
				plan.MaxCandidates,
			)
		}
		found[entry.Locator] = entry
		return nil
	}

	for _, locator := range plan.ExplicitLocators {
		entry, err := statEntry(ctx, snapshot, locator)
		if errors.Is(err, basespec.ErrNotFound) {
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
		if errors.Is(err, basespec.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !rootEntry.IsDirectory {
			return nil, fmt.Errorf(
				"%w: discovery root %q is not a directory",
				basespec.ErrInvalid,
				root.Root,
			)
		}

		var visit func(basespec.Locator, int) error
		visit = func(directory basespec.Locator, depth int) error {
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
						basespec.ErrInvalid,
						plan.MaxEntries,
					)
				}
				nextDepth := depth + 1
				if nextDepth > plan.MaxDepth {
					return fmt.Errorf(
						"%w: discovery exceeds depth %d at %q",
						basespec.ErrInvalid,
						plan.MaxDepth,
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

func statEntry(
	ctx context.Context,
	snapshot source.Snapshot,
	locator basespec.Locator,
) (source.Entry, error) {
	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return source.Entry{}, err
	}
	if err := entry.Validate(); err != nil {
		return source.Entry{}, fmt.Errorf(
			"%w: source snapshot returned an invalid stat entry: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if entry.Locator != locator {
		return source.Entry{}, fmt.Errorf(
			"%w: source snapshot stat for %q returned %q",
			basespec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return entry, nil
}

func readDirectoryEntries(
	ctx context.Context,
	snapshot source.Snapshot,
	directory basespec.Locator,
) ([]source.Entry, error) {
	values, err := snapshot.ReadDir(ctx, directory)
	if err != nil {
		return nil, err
	}

	seen := make(map[basespec.Locator]struct{}, len(values))
	output := make([]source.Entry, 0, len(values))
	for _, entry := range values {
		if err := entry.Validate(); err != nil {
			return nil, fmt.Errorf(
				"%w: source snapshot returned an invalid directory entry: %w",
				basespec.ErrInvalid,
				err,
			)
		}
		if !isDirectChild(directory, entry.Locator) {
			return nil, fmt.Errorf(
				"%w: source snapshot returned non-child %q for directory %q",
				basespec.ErrInvalid,
				entry.Locator,
				directory,
			)
		}
		if _, duplicate := seen[entry.Locator]; duplicate {
			return nil, fmt.Errorf(
				"%w: source snapshot returned duplicate directory entry %q",
				basespec.ErrInvalid,
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
			basespec.ErrInvalid,
			entry.Locator,
		)
	}
	if int64(len(content)) != entry.SizeBytes {
		return nil, fmt.Errorf(
			"%w: candidate %q changed size during discovery",
			basespec.ErrConflict,
			entry.Locator,
		)
	}
	return content, nil
}

func applyInvalidForLocator(
	values map[catalog.OccurrenceKey]catalog.Occurrence,
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
	sourceID basespec.SourceID,
	locator basespec.Locator,
	sourceDigest *cryptoutil.Digest,
	decoderID basespec.DecoderID,
	diagnostics []diagnostic.Diagnostic,
	now time.Time,
) {
	matched := false
	for key, previous := range values {
		if previous.Key.SourceID != sourceID ||
			previous.Key.Locator != locator {
			continue
		}
		matched = true
		previous.SourceContentDigest = cryptoutil.CloneDigest(sourceDigest)
		previous.DefinitionDigest = nil
		previous.DecoderID = decoderID
		previous.State = catalog.OccurrenceInvalid
		previous.Diagnostics = diagnostic.CloneDiagnostics(diagnostics)
		previous.ObservedAt = now
		values[key] = previous
	}
	if matched {
		return
	}
	key := catalog.OccurrenceKey{
		CollectionID: collectionID,
		SourceID:     sourceID,
		Locator:      locator,
	}
	values[key] = catalog.Occurrence{
		RootID:              rootID,
		CollectionID:        collectionID,
		Key:                 key,
		SourceContentDigest: cryptoutil.CloneDigest(sourceDigest),
		DecoderID:           decoderID,
		State:               catalog.OccurrenceInvalid,
		Diagnostics:         diagnostic.CloneDiagnostics(diagnostics),
		ObservedAt:          now,
	}
}

func markObservedKeysForLocator(
	seenKeys map[catalog.OccurrenceKey]struct{},
	values map[catalog.OccurrenceKey]catalog.Occurrence,
	sourceID basespec.SourceID,
	locator basespec.Locator,
) {
	for key, value := range values {
		if value.Key.SourceID != sourceID ||
			value.Key.Locator != locator {
			continue
		}
		seenKeys[key] = struct{}{}
	}
}

// markUnrecognizedForLocator reconciles a source candidate which still exists
// but is no longer recognized by any configured decoder. This is distinct
// from an out-of-scope candidate: the candidate was explicitly observed during
// this refresh and must not leave a previous Artifact falsely available.
func markUnrecognizedForLocator(
	values map[catalog.OccurrenceKey]catalog.Occurrence,
	sourceID basespec.SourceID,
	locator basespec.Locator,
	now time.Time,
) []diagnostic.Diagnostic {
	keys := make([]catalog.OccurrenceKey, 0)
	for key, value := range values {
		if value.Key.SourceID != sourceID ||
			value.Key.Locator != locator {
			continue
		}
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].CollectionID != keys[right].CollectionID {
			return keys[left].CollectionID < keys[right].CollectionID
		}
		return keys[left].SubresourceLocator < keys[right].SubresourceLocator
	})

	diagnostics := make([]diagnostic.Diagnostic, 0, len(keys))
	for _, key := range keys {
		previous := values[key]
		d := diagnostic.Diagnostic{
			Severity: diagnostic.DiagnosticWarning,
			Code:     DiagnosticCodeDecoderNoLongerRecognizes,
			Message:  "the source candidate no longer matches any configured decoder",
			Location: &diagnostic.DiagnosticLocation{
				Locator:            previous.Key.Locator,
				SubresourceLocator: previous.Key.SubresourceLocator,
			},
		}
		previous.State = catalog.OccurrenceMissing
		previous.Diagnostics = []diagnostic.Diagnostic{d}
		previous.ObservedAt = now
		values[key] = previous
		diagnostics = diagnostic.AppendDiagnostics(diagnostics, d)
	}
	return diagnostics
}

func locatorInScope(
	locator basespec.Locator,
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
	root spec.DirectoryRoot,
	locator basespec.Locator,
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
