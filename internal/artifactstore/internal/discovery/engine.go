package discovery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	DiagnosticCodeCandidateTooLarge         = "artifact.discovery.candidate-too-large"
	DiagnosticCodeContentDigestMismatch     = "artifact.discovery.content-digest-mismatch"
	DiagnosticCodeDecoderAmbiguous          = "artifact.discovery.decoder-ambiguous"
	DiagnosticCodeDecoderInvalidRecognition = "artifact.discovery.decoder-invalid-recognition"
	DiagnosticCodeDefinitionInvalid         = "artifact.discovery.definition-invalid"
	DiagnosticCodeSubresourceDuplicate      = "artifact.discovery.subresource-duplicate"
)

type Result struct {
	Observations []Observation
	SeenLocators []basespec.Locator
	Diagnostics  []diagnostic.Diagnostic
	Candidates   int
}

func (r Result) Clone() Result {
	output := r
	output.Observations = make(
		[]Observation,
		len(r.Observations),
	)
	for index, value := range r.Observations {
		output.Observations[index] = value.Clone()
	}
	output.SeenLocators = append(
		[]basespec.Locator(nil),
		r.SeenLocators...,
	)
	output.Diagnostics = diagnostic.Clone(r.Diagnostics)
	return output
}

func (r Result) Validate() error {
	if r.Candidates < 0 {
		return fmt.Errorf(
			"%w: discovery candidate count cannot be negative",
			basespec.ErrInvalid,
		)
	}
	if err := diagnostic.Validate(r.Diagnostics); err != nil {
		return err
	}

	seenLocators := make(map[basespec.Locator]struct{})
	for _, locator := range r.SeenLocators {
		if err := locator.Validate(false); err != nil {
			return err
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovered Source locator %q",
				basespec.ErrInvalid,
				locator,
			)
		}
		seenLocators[locator] = struct{}{}
	}

	seenTyped := make(map[typedOrigin]struct{})
	seenInvalid := make(map[artifact.SourceBinding]struct{})
	for index, observation := range r.Observations {
		if err := observation.Validate(); err != nil {
			return fmt.Errorf(
				"discovery observation %d: %w",
				index,
				err,
			)
		}
		switch observation.State {
		case ObservationValid:
			key := observation.TypedOrigin()
			if _, duplicate := seenTyped[key]; duplicate {
				return fmt.Errorf(
					"%w: duplicate typed Source observation",
					basespec.ErrInvalid,
				)
			}
			seenTyped[key] = struct{}{}

		case ObservationInvalid:
			if _, duplicate := seenInvalid[observation.Binding]; duplicate {
				return fmt.Errorf(
					"%w: duplicate invalid Source observation",
					basespec.ErrInvalid,
				)
			}
			seenInvalid[observation.Binding] = struct{}{}
		}
	}
	return nil
}

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

func (e *Engine) DecoderFingerprint() (
	cryptoutil.Digest,
	error,
) {
	if e == nil || e.decoders == nil {
		return "", basespec.ErrClosed
	}
	return e.decoders.Fingerprint()
}

func (e *Engine) Discover(
	ctx context.Context,
	value source.Source,
	snapshot sourceimpl.Snapshot,
) (Result, error) {
	if e == nil || e.decoders == nil {
		return Result{}, basespec.ErrClosed
	}
	if ctx == nil {
		return Result{}, fmt.Errorf(
			"%w: discovery context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := value.Validate(); err != nil {
		return Result{}, err
	}
	if !value.Enabled {
		return Result{}, fmt.Errorf(
			"%w: disabled Source cannot be refreshed",
			basespec.ErrConflict,
		)
	}
	if snapshot == nil {
		return Result{}, fmt.Errorf(
			"%w: Source snapshot is nil",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(
		snapshot.Generation(),
	); err != nil {
		return Result{}, err
	}

	spec := value.Discovery.Effective()
	if spec.Empty() {
		return Result{}, fmt.Errorf(
			"%w: Source has no declaration discovery configuration",
			basespec.ErrRefreshRequired,
		)
	}
	if err := spec.Validate(); err != nil {
		return Result{}, err
	}

	allowed, err := e.allowedDecoders(spec)
	if err != nil {
		return Result{}, err
	}
	entries, err := collectCandidates(ctx, snapshot, spec)
	if err != nil {
		return Result{}, err
	}

	foundCandidates := make(
		map[basespec.Locator]struct{},
		len(entries),
	)
	for _, entry := range entries {
		foundCandidates[entry.Locator] = struct{}{}
	}
	for locator := range spec.ExpectedContentDigests {
		if _, found := foundCandidates[locator]; !found {
			return Result{}, fmt.Errorf(
				"%w: expected Source content %q was not found",
				basespec.ErrReferenceUnresolved,
				locator,
			)
		}
	}

	result := Result{
		Observations: make([]Observation, 0),
		SeenLocators: make([]basespec.Locator, 0, len(entries)),
		Diagnostics:  make([]diagnostic.Diagnostic, 0),
	}
	var consumed int64
	invalidBindings := make(map[artifact.SourceBinding]struct{})

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		result.Candidates++
		result.SeenLocators = append(
			result.SeenLocators,
			entry.Locator,
		)

		if entry.SizeBytes > spec.MaxCandidateBytes {
			diagnostics := []diagnostic.Diagnostic{{
				Severity: diagnostic.SeverityError,
				Code:     DiagnosticCodeCandidateTooLarge,
				Message: fmt.Sprintf(
					"candidate exceeds the %d byte limit",
					spec.MaxCandidateBytes,
				),
				Location: &diagnostic.Location{
					Locator: entry.Locator,
				},
			}}
			result.Diagnostics = diagnostic.Append(
				result.Diagnostics,
				diagnostics...,
			)
			appendInvalid(
				&result,
				value,
				entry.Locator,
				nil,
				"",
				diagnostics,
			)
			continue
		}
		if entry.SizeBytes > spec.MaxTotalBytes-consumed {
			return Result{}, fmt.Errorf(
				"%w: discovery exceeds total byte limit",
				basespec.ErrInvalid,
			)
		}

		content, err := sourceimpl.ReadSnapshotEntry(
			ctx,
			snapshot,
			entry,
			spec.MaxCandidateBytes,
		)
		if err != nil {
			return Result{}, err
		}
		consumed += int64(len(content))
		if consumed > spec.MaxTotalBytes {
			return Result{}, fmt.Errorf(
				"%w: discovery exceeds total byte limit",
				basespec.ErrInvalid,
			)
		}

		sourceDigest := cryptoutil.DigestBytes(content)
		if expected, found := spec.ExpectedContentDigests[entry.Locator]; found &&
			expected != sourceDigest {
			diagnostics := []diagnostic.Diagnostic{{
				Severity: diagnostic.SeverityError,
				Code:     DiagnosticCodeContentDigestMismatch,
				Message:  "candidate content does not match its expected digest",
				Location: &diagnostic.Location{
					Locator: entry.Locator,
				},
			}}
			result.Diagnostics = diagnostic.Append(
				result.Diagnostics,
				diagnostics...,
			)
			appendInvalid(
				&result,
				value,
				entry.Locator,
				&sourceDigest,
				"",
				diagnostics,
			)
			continue
		}

		candidate := providerapi.Candidate{
			SourceID:            value.ID,
			SourceKind:          value.Kind,
			Locator:             entry.Locator,
			SourceContentDigest: sourceDigest,
			Content:             content,
			RequestedDecoderIDs: spec.RequestedDecoderIDs(
				entry.Locator,
			),
		}
		decoder, diagnostics := e.selectDecoder(
			ctx,
			candidate,
			allowed,
		)
		if len(diagnostics) != 0 {
			result.Diagnostics = diagnostic.Append(
				result.Diagnostics,
				diagnostics...,
			)
			appendInvalid(
				&result,
				value,
				entry.Locator,
				&sourceDigest,
				"",
				diagnostics,
			)
			continue
		}
		if decoder == nil {
			continue
		}

		decoded, decoderDiagnostics := decoder.Decode(
			ctx,
			cloneCandidate(candidate),
		)
		if err := validateCandidateDiagnostics(
			entry.Locator,
			decoderDiagnostics,
		); err != nil {
			return Result{}, fmt.Errorf(
				"%w: decoder %q returned invalid diagnostics: %w",
				basespec.ErrInvalid,
				decoder.ID(),
				err,
			)
		}
		result.Diagnostics = diagnostic.Append(
			result.Diagnostics,
			decoderDiagnostics...,
		)
		if diagnostic.ContainsError(decoderDiagnostics) {
			appendInvalid(
				&result,
				value,
				entry.Locator,
				&sourceDigest,
				decoder.ID(),
				decoderDiagnostics,
			)
			continue
		}

		emitted := make(map[typedOrigin]struct{}, len(decoded))
		emittedBindings := make(map[artifact.SourceBinding]struct{})
		for _, item := range decoded {
			if err := item.SubresourceLocator.Validate(); err != nil {
				return Result{}, fmt.Errorf(
					"%w: decoder %q emitted invalid subresource: %w",
					basespec.ErrInvalid,
					decoder.ID(),
					err,
				)
			}
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

			itemDiagnostics := diagnostic.Append(
				decoderDiagnostics,
				item.Diagnostics...,
			)
			result.Diagnostics = diagnostic.Append(
				result.Diagnostics,
				item.Diagnostics...,
			)
			binding := artifact.SourceBinding{
				SourceID:           value.ID,
				Locator:            entry.Locator,
				SubresourceLocator: item.SubresourceLocator,
			}
			if _, invalid := invalidBindings[binding]; invalid {
				continue
			}
			if diagnostic.ContainsError(item.Diagnostics) {
				result.Observations = removeObservationsForBinding(
					result.Observations,
					binding,
				)
				invalidBindings[binding] = struct{}{}
				appendInvalidBinding(
					&result,
					value,
					binding,
					&sourceDigest,
					decoder.ID(),
					itemDiagnostics,
				)
				continue
			}
			if _, duplicate := emittedBindings[binding]; duplicate {
				result.Observations = removeObservationsForBinding(
					result.Observations,
					binding,
				)
				diagnostics := []diagnostic.Diagnostic{{
					Severity: diagnostic.SeverityError,
					Code:     DiagnosticCodeSubresourceDuplicate,
					Message:  "decoder emitted duplicate declaration subresource",
					Location: &diagnostic.Location{
						Locator:            entry.Locator,
						SubresourceLocator: item.SubresourceLocator,
					},
				}}
				result.Diagnostics = diagnostic.Append(
					result.Diagnostics,
					diagnostics...,
				)
				appendInvalidBinding(
					&result,
					value,
					binding,
					&sourceDigest,
					decoder.ID(),
					diagnostics,
				)
				invalidBindings[binding] = struct{}{}
				continue
			}
			emittedBindings[binding] = struct{}{}
			canonical, err := definition.Canonicalize(item.Definition)
			if err != nil {
				diagnostics := diagnostic.Append(
					itemDiagnostics,
					diagnostic.Diagnostic{
						Severity: diagnostic.SeverityError,
						Code:     DiagnosticCodeDefinitionInvalid,
						Message: diagnostic.BoundedMessage(
							err.Error(),
						),
						Location: &diagnostic.Location{
							Locator: entry.Locator,
							SubresourceLocator: item.
								SubresourceLocator,
						},
					},
				)
				appendInvalidBinding(
					&result,
					value,
					binding,
					&sourceDigest,
					decoder.ID(),
					diagnostics,
				)
				continue
			}

			key := typedOrigin{
				Binding: binding,
				Kind:    canonical.Kind,
			}
			if _, duplicate := emitted[key]; duplicate {
				duplicateDiagnostics := []diagnostic.Diagnostic{{
					Severity: diagnostic.SeverityError,
					Code:     DiagnosticCodeSubresourceDuplicate,
					Message:  "decoder emitted duplicate artifact origin and kind",

					Location: &diagnostic.Location{
						Locator: entry.Locator,
						SubresourceLocator: item.
							SubresourceLocator,
					},
				}}
				appendInvalidBinding(
					&result,
					value,
					binding,
					&sourceDigest,
					decoder.ID(),
					duplicateDiagnostics,
				)
				invalidBindings[binding] = struct{}{}
				continue
			}
			emitted[key] = struct{}{}

			definitionValue := canonical.Clone()
			result.Observations = append(
				result.Observations,
				Observation{
					RootID:              value.RootID,
					Binding:             binding,
					Kind:                canonical.Kind,
					LogicalName:         canonical.LogicalName,
					LogicalVersion:      canonical.LogicalVersion,
					Definition:          &definitionValue,
					SourceContentDigest: &sourceDigest,
					DecoderID:           decoder.ID(),
					State:               ObservationValid,
					Diagnostics:         itemDiagnostics,
				},
			)
		}
	}

	slices.Sort(result.SeenLocators)
	sort.Slice(result.Observations, func(left, right int) bool {
		leftValue := result.Observations[left]
		rightValue := result.Observations[right]
		if leftValue.Binding.Locator != rightValue.Binding.Locator {
			return leftValue.Binding.Locator <
				rightValue.Binding.Locator
		}
		if leftValue.Binding.SubresourceLocator !=
			rightValue.Binding.SubresourceLocator {
			return leftValue.Binding.SubresourceLocator <
				rightValue.Binding.SubresourceLocator
		}
		if leftValue.State != rightValue.State {
			return leftValue.State < rightValue.State
		}
		return leftValue.Kind < rightValue.Kind
	})
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result.Clone(), nil
}

func removeObservationsForBinding(
	values []Observation,
	binding artifact.SourceBinding,
) []Observation {
	output := values[:0]
	for _, value := range values {
		if value.Binding == binding {
			continue
		}
		output = append(output, value)
	}
	return output
}

func appendInvalid(
	r *Result,
	value source.Source,
	locator basespec.Locator,
	sourceDigest *cryptoutil.Digest,
	decoderID basespec.DecoderID,
	diagnostics []diagnostic.Diagnostic,
) {
	appendInvalidBinding(
		r,
		value,
		artifact.SourceBinding{
			SourceID: value.ID,
			Locator:  locator,
		},
		sourceDigest,
		decoderID,
		diagnostics,
	)
}

func appendInvalidBinding(
	r *Result,
	value source.Source,
	binding artifact.SourceBinding,
	sourceDigest *cryptoutil.Digest,
	decoderID basespec.DecoderID,
	diagnostics []diagnostic.Diagnostic,
) {
	r.Observations = append(r.Observations, Observation{
		RootID:              value.RootID,
		Binding:             binding,
		SourceContentDigest: cryptoutil.CloneDigest(sourceDigest),
		DecoderID:           decoderID,
		State:               ObservationInvalid,
		Diagnostics:         diagnostic.Clone(diagnostics),
	})
}

func (e *Engine) allowedDecoders(
	spec source.DiscoverySpec,
) (map[basespec.DecoderID]struct{}, error) {
	allowed := make(
		map[basespec.DecoderID]struct{},
		len(spec.AllowedDecoderIDs),
	)
	for _, decoderID := range spec.AllowedDecoderIDs {
		if _, found := e.decoders.find(decoderID); !found {
			return nil, fmt.Errorf(
				"%w: decoder %q",
				basespec.ErrDecoderUnavailable,
				decoderID,
			)
		}
		allowed[decoderID] = struct{}{}
	}
	for _, hint := range spec.DecoderHints {
		for _, decoderID := range hint.DecoderIDs {
			if _, found := e.decoders.find(decoderID); !found {
				return nil, fmt.Errorf(
					"%w: decoder %q",
					basespec.ErrDecoderUnavailable,
					decoderID,
				)
			}
			if len(allowed) != 0 {
				if _, permitted := allowed[decoderID]; !permitted {
					return nil, fmt.Errorf(
						"%w: hinted decoder %q is not allowed by Source discovery",
						basespec.ErrInvalid,
						decoderID,
					)
				}
			}
		}
	}
	return allowed, nil
}

func (e *Engine) selectDecoder(
	ctx context.Context,
	candidate providerapi.Candidate,
	allowed map[basespec.DecoderID]struct{},
) (providerapi.Decoder, []diagnostic.Diagnostic) {
	var selected providerapi.Decoder
	best := providerapi.RecognitionNone
	tied := make([]basespec.DecoderID, 0)

	for _, decoder := range e.decoders.registered() {
		if len(allowed) != 0 {
			if _, permitted := allowed[decoder.ID()]; !permitted {
				continue
			}
		}
		recognition := decoder.Recognize(
			ctx,
			cloneCandidate(candidate),
		)
		if recognition < providerapi.RecognitionNone ||
			recognition > providerapi.RecognitionPreferred {
			return nil, []diagnostic.Diagnostic{{
				Severity: diagnostic.SeverityError,
				Code:     DiagnosticCodeDecoderInvalidRecognition,
				Message: fmt.Sprintf(
					"decoder %q returned invalid recognition %d",
					decoder.ID(),
					recognition,
				),
				Location: &diagnostic.Location{
					Locator: candidate.Locator,
				},
			}}
		}
		if recognition > best {
			best = recognition
			selected = decoder
			tied = []basespec.DecoderID{decoder.ID()}
			continue
		}
		if recognition == best &&
			recognition != providerapi.RecognitionNone {
			tied = append(tied, decoder.ID())
		}
	}
	if len(tied) <= 1 {
		return selected, nil
	}
	slices.Sort(tied)
	return nil, []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     DiagnosticCodeDecoderAmbiguous,
		Message: diagnostic.BoundedMessage(
			fmt.Sprintf(
				"candidate is equally recognized by decoders %v",
				tied,
			),
		),
		Location: &diagnostic.Location{
			Locator: candidate.Locator,
		},
	}}
}

func collectCandidates(
	ctx context.Context,
	snapshot sourceimpl.Snapshot,
	spec source.DiscoverySpec,
) ([]source.Entry, error) {
	found := make(map[basespec.Locator]source.Entry)
	visited := 0

	add := func(entry source.Entry) error {
		if err := entry.Validate(); err != nil {
			return err
		}
		if !entry.IsRegular {
			return nil
		}
		if _, exists := found[entry.Locator]; !exists &&
			len(found) >= spec.MaxCandidates {
			return fmt.Errorf(
				"%w: discovery exceeds %d candidates",
				basespec.ErrInvalid,
				spec.MaxCandidates,
			)
		}
		found[entry.Locator] = entry
		return nil
	}

	for _, locator := range spec.ExplicitLocators {
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

	for _, root := range spec.DirectoryRoots {
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
			entries, err := readDirectoryEntries(
				ctx,
				snapshot,
				directory,
			)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return err
				}
				visited++
				if visited > spec.MaxEntries {
					return fmt.Errorf(
						"%w: discovery exceeds %d entries",
						basespec.ErrInvalid,
						spec.MaxEntries,
					)
				}
				nextDepth := depth + 1
				if nextDepth > spec.MaxDepth {
					return fmt.Errorf(
						"%w: discovery exceeds depth %d at %q",
						basespec.ErrInvalid,
						spec.MaxDepth,
						entry.Locator,
					)
				}
				if entry.IsDirectory {
					if root.Recursive {
						if err := visit(
							entry.Locator,
							nextDepth,
						); err != nil {
							return err
						}
					}
					continue
				}
				matched, err := root.Matches(entry.Locator)
				if err != nil {
					return err
				}
				if entry.IsRegular && matched {
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
	snapshot sourceimpl.Snapshot,
	locator basespec.Locator,
) (source.Entry, error) {
	entry, err := snapshot.Stat(ctx, locator)
	if err != nil {
		return source.Entry{}, err
	}
	if err := entry.Validate(); err != nil {
		return source.Entry{}, err
	}
	if entry.Locator != locator {
		return source.Entry{}, fmt.Errorf(
			"%w: Source snapshot stat for %q returned %q",
			basespec.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return entry, nil
}

func readDirectoryEntries(
	ctx context.Context,
	snapshot sourceimpl.Snapshot,
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
			return nil, err
		}
		if !isDirectChild(directory, entry.Locator) {
			return nil, fmt.Errorf(
				"%w: Source snapshot returned non-child %q for directory %q",
				basespec.ErrInvalid,
				entry.Locator,
				directory,
			)
		}
		if _, duplicate := seen[entry.Locator]; duplicate {
			return nil, fmt.Errorf(
				"%w: Source snapshot returned duplicate entry %q",
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
	relative, found := strings.CutPrefix(
		string(child),
		prefix,
	)
	return found &&
		relative != "" &&
		!strings.Contains(relative, "/")
}

func cloneCandidate(
	value providerapi.Candidate,
) providerapi.Candidate {
	output := value
	output.Content = append([]byte(nil), value.Content...)
	output.RequestedDecoderIDs = append(
		[]basespec.DecoderID(nil),
		value.RequestedDecoderIDs...,
	)
	return output
}

func validateCandidateDiagnostics(
	locator basespec.Locator,
	values []diagnostic.Diagnostic,
) error {
	if err := diagnostic.Validate(values); err != nil {
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
	if err := diagnostic.Validate(values); err != nil {
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
