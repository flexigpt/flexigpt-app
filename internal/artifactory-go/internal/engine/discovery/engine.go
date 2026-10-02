package discovery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/providerapi"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/internal/engine/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	DiagnosticCodeCandidateTooLarge         = "artifact.discovery.candidate-too-large"
	DiagnosticCodeContentDigestMismatch     = "artifact.discovery.content-digest-mismatch"
	DiagnosticCodeDecoderAmbiguous          = "artifact.discovery.decoder-ambiguous"
	DiagnosticCodeDecoderInvalidRecognition = "artifact.discovery.decoder-invalid-recognition"
	DiagnosticCodeDefinitionInvalid         = "artifact.discovery.definition-invalid"
	DiagnosticCodeSubresourceDuplicate      = "artifact.discovery.subresource-duplicate"
	DiagnosticCodeOriginConflict            = "artifact.discovery.origin-conflict"
)

type Result struct {
	Observations []Observation
	SeenLocators []model.Locator
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
		[]model.Locator(nil),
		r.SeenLocators...,
	)
	output.Diagnostics = diagnostic.Clone(r.Diagnostics)
	return output
}

func (r Result) Validate() error {
	if r.Candidates < 0 {
		return fmt.Errorf(
			"%w: discovery candidate count cannot be negative",
			model.ErrInvalid,
		)
	}
	if err := diagnostic.Validate(r.Diagnostics); err != nil {
		return err
	}

	seenLocators := make(map[model.Locator]struct{})
	for _, locator := range r.SeenLocators {
		if err := locator.Validate(false); err != nil {
			return err
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovered Source locator %q",
				model.ErrInvalid,
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
					model.ErrInvalid,
				)
			}
			seenTyped[key] = struct{}{}

		case ObservationInvalid:
			if _, duplicate := seenInvalid[observation.Binding]; duplicate {
				return fmt.Errorf(
					"%w: duplicate invalid Source observation",
					model.ErrInvalid,
				)
			}
			seenInvalid[observation.Binding] = struct{}{}
		}
	}
	return nil
}

type Engine struct {
	decoders *DecoderRegistry
	compiled compiledDocumentRegistry
}

func NewEngine(
	decoders *DecoderRegistry,
) (*Engine, error) {
	if decoders == nil {
		return nil, fmt.Errorf(
			"%w: discovery engine dependencies are incomplete",
			model.ErrInvalid,
		)
	}
	return &Engine{
		decoders: decoders,
	}, nil
}

func (e *Engine) DecoderFingerprint() (
	cryptoutil.Digest,
	error,
) {
	if e == nil || e.decoders == nil {
		return "", model.ErrClosed
	}
	return e.decoders.Fingerprint()
}

func (e *Engine) Discover(
	ctx context.Context,
	value source.Source,
	snapshot sourceimpl.Snapshot,
) (Result, error) {
	if e == nil || e.decoders == nil {
		return Result{}, model.ErrClosed
	}
	if ctx == nil {
		return Result{}, fmt.Errorf(
			"%w: discovery context is nil",
			model.ErrInvalid,
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
			model.ErrConflict,
		)
	}
	if snapshot == nil {
		return Result{}, fmt.Errorf(
			"%w: Source snapshot is nil",
			model.ErrInvalid,
		)
	}
	if err := model.ValidateSourceGeneration(
		snapshot.Generation(),
	); err != nil {
		return Result{}, err
	}

	spec := value.Discovery.Effective()
	if spec.Empty() {
		return Result{}, fmt.Errorf(
			"%w: Source has no declaration discovery configuration",
			model.ErrRefreshRequired,
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

	result := Result{
		Observations: make([]Observation, 0),
		Diagnostics:  make([]diagnostic.Diagnostic, 0),
	}
	foundCandidates := make(
		map[model.Locator]struct{},
		len(entries),
	)
	for _, entry := range entries {
		foundCandidates[entry.Locator] = struct{}{}
	}
	for locator := range spec.ExpectedContentDigests {
		if _, found := foundCandidates[locator]; !found {
			result.Diagnostics = diagnostic.Append(
				result.Diagnostics,
				diagnostic.Diagnostic{
					Severity: diagnostic.SeverityError,
					Code:     "artifact.discovery.expected-content-missing",
					Message:  "expected Source content was not found",
					Location: &diagnostic.Location{Locator: locator},
				},
			)
		}
	}

	seenLocators := make(map[model.Locator]struct{}, len(entries))
	validOrigins := make(map[typedOrigin]Observation)
	invalidBindings := make(map[artifact.SourceBinding]struct{})
	var consumed int64

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		result.Candidates++
		seenLocators[entry.Locator] = struct{}{}

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
				model.ErrInvalid,
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
				model.ErrInvalid,
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

		if compiled, found := e.compiledDocument(
			value.RootID,
			value.ID,
			entry.Locator,
		); found {
			if compiled.Digest != sourceDigest {
				diagnostics := []diagnostic.Diagnostic{{
					Severity: diagnostic.SeverityError,
					Code:     DiagnosticCodeContentDigestMismatch,
					Message:  "compiled declaration bytes differ from the binary catalog",
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
					compiledDecoderID,
					diagnostics,
				)
				continue
			}

			// The generated catalog already completed canonicalization, schema
			// validation, declaration-tree validation, and Definition creation.
			// Runtime checks source bytes against the generated digest, then
			// publishes the immutable generated Definitions directly.
			for _, item := range compiled.Artifacts {
				definitionValue := item.Definition.Clone()
				result.Observations = append(
					result.Observations,
					Observation{
						RootID: value.RootID,
						Binding: artifact.SourceBinding{
							SourceID:           value.ID,
							Locator:            entry.Locator,
							SubresourceLocator: item.Subresource,
						},
						Kind:                definitionValue.Kind,
						LogicalName:         definitionValue.LogicalName,
						LogicalVersion:      definitionValue.LogicalVersion,
						Definition:          &definitionValue,
						SourceContentDigest: cryptoutil.CloneDigest(&sourceDigest),
						DecoderID:           compiledDecoderID,
						State:               ObservationValid,
						Diagnostics:         diagnostic.Clone(item.Diagnostics),
					},
				)
			}
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

		var decoded []providerapi.Decoded
		var decoderDiagnostics []diagnostic.Diagnostic
		if sourceAware, supported := decoder.(providerapi.SourceAwareDecoder); supported {
			decoded, decoderDiagnostics = sourceAware.DecodeWithSource(
				ctx,
				cloneCandidate(candidate),
				snapshotEntryReader{
					snapshot:     snapshot,
					maximumBytes: spec.MaxCandidateBytes,
				},
			)
		} else {
			decoded, decoderDiagnostics = decoder.Decode(ctx, cloneCandidate(candidate))
		}
		if err := validateCandidateDiagnostics(
			entry.Locator,
			decoderDiagnostics,
		); err != nil {
			return Result{}, fmt.Errorf(
				"%w: decoder %q returned invalid diagnostics: %w",
				model.ErrInvalid,
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
		for _, item := range decoded {
			if err := item.SubresourceLocator.Validate(); err != nil {
				return Result{}, fmt.Errorf(
					"%w: decoder %q emitted invalid subresource: %w",
					model.ErrInvalid,
					decoder.ID(),
					err,
				)
			}
			binding, itemSourceDigest, err := decodedBinding(
				value.ID,
				entry.Locator,
				sourceDigest,
				item,
			)
			if err != nil {
				return Result{}, err
			}
			seenLocators[binding.Locator] = struct{}{}
			if err := validateDecodedDiagnostics(
				binding.Locator,
				item.SubresourceLocator,
				item.Diagnostics,
			); err != nil {
				return Result{}, fmt.Errorf(
					"%w: decoder %q returned invalid decoded diagnostics: %w",
					model.ErrInvalid,
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
			if _, invalid := invalidBindings[binding]; invalid {
				continue
			}
			if diagnostic.ContainsError(item.Diagnostics) {
				result.Observations = removeObservationsForBinding(
					result.Observations,
					binding,
				)
				deleteValidOriginsForBinding(validOrigins, binding)
				invalidBindings[binding] = struct{}{}
				appendInvalidBinding(
					&result,
					value,
					binding,
					itemSourceDigest,
					decoder.ID(),
					itemDiagnostics,
				)
				continue
			}
			canonical, err := definition.Canonicalize(item.Definition)
			if err != nil {
				definitionDiagnostic := diagnostic.Diagnostic{
					Severity: diagnostic.SeverityError,
					Code:     DiagnosticCodeDefinitionInvalid,
					Message: diagnostic.BoundedMessage(
						err.Error(),
					),
					Location: &diagnostic.Location{
						Locator: binding.Locator,
						SubresourceLocator: binding.
							SubresourceLocator,
					},
				}
				diagnostics := diagnostic.Append(
					itemDiagnostics,
					definitionDiagnostic,
				)
				deleteValidOriginsForBinding(validOrigins, binding)
				invalidBindings[binding] = struct{}{}
				result.Diagnostics = diagnostic.Append(
					result.Diagnostics,
					definitionDiagnostic,
				)
				result.Observations = removeObservationsForBinding(
					result.Observations,
					binding,
				)
				appendInvalidBinding(
					&result,
					value,
					binding,
					itemSourceDigest,
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
				result.Observations = removeObservationsForBinding(
					result.Observations,
					binding,
				)
				result.Diagnostics = diagnostic.Append(
					result.Diagnostics,
					duplicateDiagnostics...,
				)
				appendInvalidBinding(
					&result,
					value,
					binding,
					itemSourceDigest,
					decoder.ID(),
					duplicateDiagnostics,
				)
				deleteValidOriginsForBinding(validOrigins, binding)
				invalidBindings[binding] = struct{}{}
				continue
			}
			emitted[key] = struct{}{}

			if previous, duplicate := validOrigins[key]; duplicate {
				if equivalentObservedOrigin(
					previous,
					canonical,
					itemSourceDigest,
				) {
					continue
				}
				result.Observations = removeObservationsForBinding(
					result.Observations,
					binding,
				)
				deleteValidOriginsForBinding(validOrigins, binding)
				diagnostics := []diagnostic.Diagnostic{{
					Severity: diagnostic.SeverityError,
					Code:     DiagnosticCodeOriginConflict,
					Message:  "multiple decoders emitted different definitions for one artifact origin",
					Location: &diagnostic.Location{
						Locator:            binding.Locator,
						SubresourceLocator: binding.SubresourceLocator,
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
					itemSourceDigest,
					decoder.ID(),
					diagnostics,
				)
				invalidBindings[binding] = struct{}{}
				continue
			}

			definitionValue := canonical.Clone()
			observed := Observation{
				RootID:              value.RootID,
				Binding:             binding,
				Kind:                canonical.Kind,
				LogicalName:         canonical.LogicalName,
				LogicalVersion:      canonical.LogicalVersion,
				Definition:          &definitionValue,
				SourceContentDigest: cryptoutil.CloneDigest(itemSourceDigest),
				DecoderID:           decoder.ID(),
				State:               ObservationValid,
				Diagnostics:         itemDiagnostics,
			}
			result.Observations = append(
				result.Observations,
				observed,
			)
			validOrigins[key] = observed.Clone()
		}
	}

	result.SeenLocators = make([]model.Locator, 0, len(seenLocators))
	for locator := range seenLocators {
		result.SeenLocators = append(result.SeenLocators, locator)
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

func decodedBinding(
	sourceID source.SourceID,
	candidateLocator model.Locator,
	candidateDigest cryptoutil.Digest,
	item providerapi.Decoded,
) (artifact.SourceBinding, *cryptoutil.Digest, error) {
	if (item.OriginLocator == "") !=
		(item.OriginContentDigest == nil) {
		return artifact.SourceBinding{}, nil, fmt.Errorf(
			"%w: declaration origin locator and digest must be supplied together",
			model.ErrInvalid,
		)
	}

	originLocator := candidateLocator
	if item.OriginLocator != "" {
		if err := item.OriginLocator.Validate(false); err != nil {
			return artifact.SourceBinding{}, nil, fmt.Errorf(
				"%w: decoder emitted invalid declaration origin: %w",
				model.ErrInvalid,
				err,
			)
		}
		originLocator = item.OriginLocator
	}

	originDigest := candidateDigest
	if item.OriginContentDigest != nil {
		if err := cryptoutil.ValidateDigest(*item.OriginContentDigest); err != nil {
			return artifact.SourceBinding{}, nil, err
		}
		originDigest = *item.OriginContentDigest
	}

	return artifact.SourceBinding{
		SourceID:           sourceID,
		Locator:            originLocator,
		SubresourceLocator: item.SubresourceLocator,
	}, &originDigest, nil
}

func equivalentObservedOrigin(
	previous Observation,
	current definition.Definition,
	currentDigest *cryptoutil.Digest,
) bool {
	return previous.Definition != nil &&
		previous.Definition.Digest == current.Digest &&
		cryptoutil.IsDigestEqual(
			previous.SourceContentDigest,
			currentDigest,
		)
}

func deleteValidOriginsForBinding(
	values map[typedOrigin]Observation,
	binding artifact.SourceBinding,
) {
	for key := range values {
		if key.Binding == binding {
			delete(values, key)
		}
	}
}

type snapshotEntryReader struct {
	snapshot     sourceimpl.Snapshot
	maximumBytes int64
}

func (r snapshotEntryReader) ReadSourceEntry(
	ctx context.Context,
	locator model.Locator,
) (providerapi.SourceContent, error) {
	entry, err := statEntry(ctx, r.snapshot, locator)
	if err != nil {
		return providerapi.SourceContent{}, err
	}
	content, err := sourceimpl.ReadSnapshotEntry(
		ctx,
		r.snapshot,
		entry,
		r.maximumBytes,
	)
	if err != nil {
		return providerapi.SourceContent{}, err
	}
	return providerapi.SourceContent{
		Locator: entry.Locator,
		Content: content,
		Digest:  cryptoutil.DigestBytes(content),
	}, nil
}

func appendInvalid(
	r *Result,
	value source.Source,
	locator model.Locator,
	sourceDigest *cryptoutil.Digest,
	decoderID model.DecoderID,
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
	decoderID model.DecoderID,
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
) (map[model.DecoderID]struct{}, error) {
	allowed := make(
		map[model.DecoderID]struct{},
		len(spec.AllowedDecoderIDs),
	)
	for _, decoderID := range spec.AllowedDecoderIDs {
		if _, found := e.decoders.find(decoderID); !found {
			return nil, fmt.Errorf(
				"%w: decoder %q",
				model.ErrDecoderUnavailable,
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
					model.ErrDecoderUnavailable,
					decoderID,
				)
			}
			if len(allowed) != 0 {
				if _, permitted := allowed[decoderID]; !permitted {
					return nil, fmt.Errorf(
						"%w: hinted decoder %q is not allowed by Source discovery",
						model.ErrInvalid,
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
	allowed map[model.DecoderID]struct{},
) (providerapi.Decoder, []diagnostic.Diagnostic) {
	var selected providerapi.Decoder
	best := providerapi.RecognitionNone
	tied := make([]model.DecoderID, 0)

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
			tied = []model.DecoderID{decoder.ID()}
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
	found := make(map[model.Locator]source.Entry)
	visited := make(map[model.Locator]struct{})

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
				model.ErrInvalid,
				spec.MaxCandidates,
			)
		}
		found[entry.Locator] = entry
		return nil
	}

	for _, locator := range spec.ExplicitLocators {
		entry, err := statEntry(ctx, snapshot, locator)
		if errors.Is(err, model.ErrNotFound) {
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
		if errors.Is(err, model.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !rootEntry.IsDirectory {
			return nil, fmt.Errorf(
				"%w: discovery root %q is not a directory",
				model.ErrInvalid,
				root.Root,
			)
		}
		selection, err := model.NewPathSelection(
			root.IncludePatterns,
			root.ExcludePatterns,
		)
		if err != nil {
			return nil, err
		}

		var visit func(model.Locator, int) error
		visit = func(directory model.Locator, depth int) error {
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
				if _, seen := visited[entry.Locator]; !seen {
					visited[entry.Locator] = struct{}{}
					if len(visited) > spec.MaxEntries {
						return fmt.Errorf(
							"%w: discovery exceeds %d entries",
							model.ErrInvalid,
							spec.MaxEntries,
						)
					}
				}
				nextDepth := depth + 1
				if nextDepth > spec.MaxDepth {
					return fmt.Errorf(
						"%w: discovery exceeds depth %d at %q",
						model.ErrInvalid,
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

				relative := string(entry.Locator)
				if root.Root != "." {
					prefix := string(root.Root) + "/"
					var found bool
					relative, found = strings.CutPrefix(
						relative,
						prefix,
					)
					if !found || relative == "" {
						continue
					}
				}
				matched, err := selection.Match(relative)
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
	locator model.Locator,
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
			model.ErrInvalid,
			locator,
			entry.Locator,
		)
	}
	return entry, nil
}

func readDirectoryEntries(
	ctx context.Context,
	snapshot sourceimpl.Snapshot,
	directory model.Locator,
) ([]source.Entry, error) {
	values, err := snapshot.ReadDir(ctx, directory)
	if err != nil {
		return nil, err
	}
	seen := make(map[model.Locator]struct{}, len(values))
	output := make([]source.Entry, 0, len(values))
	for _, entry := range values {
		if err := entry.Validate(); err != nil {
			return nil, err
		}
		if !isDirectChild(directory, entry.Locator) {
			return nil, fmt.Errorf(
				"%w: Source snapshot returned non-child %q for directory %q",
				model.ErrInvalid,
				entry.Locator,
				directory,
			)
		}
		if _, duplicate := seen[entry.Locator]; duplicate {
			return nil, fmt.Errorf(
				"%w: Source snapshot returned duplicate entry %q",
				model.ErrInvalid,
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

func cloneCandidate(
	value providerapi.Candidate,
) providerapi.Candidate {
	output := value
	output.Content = append([]byte(nil), value.Content...)
	output.RequestedDecoderIDs = append(
		[]model.DecoderID(nil),
		value.RequestedDecoderIDs...,
	)
	return output
}

func validateCandidateDiagnostics(
	locator model.Locator,
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
	locator model.Locator,
	subresource model.SubresourceLocator,
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
