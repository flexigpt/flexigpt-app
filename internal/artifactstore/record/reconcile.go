package record

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Reconciler struct {
	ids   artifactstore.IDGenerator
	clock artifactstore.Clock
}

func NewReconciler(
	ids artifactstore.IDGenerator,
	clock artifactstore.Clock,
) (*Reconciler, error) {
	if ids == nil || clock == nil {
		return nil, fmt.Errorf(
			"%w: record reconciler dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Reconciler{ids: ids, clock: clock}, nil
}

func (r *Reconciler) Reconcile(
	ctx context.Context,
	root catalog.Root,
	occurrences []catalog.Occurrence,
	existing []Record,
	definitions definition.Reader,
	policy Policy,
) (Reconciliation, error) {
	if policy == nil {
		return Reconciliation{}, fmt.Errorf(
			"%w: record policy is nil",
			artifactstore.ErrInvalid,
		)
	}
	occurrencesByTypedKey := make(map[string]catalog.Occurrence, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence.Kind == "" {
			continue
		}
		key := TypedOccurrenceKey(root.ID, occurrence.Key, occurrence.Kind)
		occurrencesByTypedKey[key] = occurrence
	}

	recordsByTypedKey := make(map[string]Record, len(existing))
	for _, value := range existing {
		key := TypedOccurrenceKey(value.RootID, value.Occurrence, value.Kind)
		recordsByTypedKey[key] = value
	}

	result := Reconciliation{}
	now := r.clock.Now().UTC()

	for _, current := range existing {
		key := TypedOccurrenceKey(current.RootID, current.Occurrence, current.Kind)
		occurrence, found := occurrencesByTypedKey[key]
		next := current

		if current.Mode == ModePinned {
			next.ResolvedDefinition = cloneDigest(current.PinnedDefinition)
			next.State = StateAvailable
			next.Diagnostics = nil
		} else {
			switch {
			case !found || occurrence.State == catalog.OccurrenceMissing:
				next.State = StateMissing
				next.Diagnostics = []artifactstore.Diagnostic{{
					Severity: artifactstore.DiagnosticWarning,
					Code:     "artifact.record.source-missing",
					Message:  "the source occurrence is missing",
				}}

			case occurrence.State == catalog.OccurrenceInvalid:
				next.State = StateInvalid
				next.Diagnostics = append(
					[]artifactstore.Diagnostic(nil),
					occurrence.Diagnostics...,
				)

			case occurrence.State == catalog.OccurrenceValid &&
				occurrence.Kind != current.Kind:
				next.State = StateIncompatible
				next.Diagnostics = []artifactstore.Diagnostic{{
					Severity: artifactstore.DiagnosticError,
					Code:     "artifact.record.kind-incompatible",
					Message:  "the source occurrence changed artifact kind",
				}}

			case occurrence.State == catalog.OccurrenceValid:
				next.ResolvedDefinition = cloneDigest(occurrence.DefinitionDigest)
				next.State = StateAvailable
				next.Diagnostics = append(
					[]artifactstore.Diagnostic(nil),
					occurrence.Diagnostics...,
				)
			}
		}

		if equivalentSourceState(current, next) {
			continue
		}
		next.Revision++
		next.ModifiedAt = now
		if !next.ModifiedAt.After(current.ModifiedAt) {
			next.ModifiedAt = current.ModifiedAt.Add(1)
		}
		if err := next.Validate(); err != nil {
			return Reconciliation{}, fmt.Errorf(
				"validate reconciled record %q: %w",
				current.ID,
				err,
			)
		}
		result.Updates = append(result.Updates, UpdatePublication{
			Record:           next,
			ExpectedRevision: current.Revision,
		})
	}

	for _, occurrence := range occurrences {
		if occurrence.State != catalog.OccurrenceValid ||
			occurrence.DefinitionDigest == nil {
			continue
		}
		typedKey := TypedOccurrenceKey(root.ID, occurrence.Key, occurrence.Kind)
		if _, exists := recordsByTypedKey[typedKey]; exists {
			continue
		}

		value, err := definitions.Get(ctx, *occurrence.DefinitionDigest)
		if err != nil {
			return Reconciliation{}, err
		}
		draft, create, diagnostics := policy.Derive(
			ctx,
			root,
			occurrence,
			value,
		)
		if err := artifactstore.ValidateDiagnostics(diagnostics); err != nil {
			return Reconciliation{}, fmt.Errorf(
				"record policy diagnostics: %w",
				err,
			)
		}
		result.Diagnostics = artifactstore.AppendDiagnostics(
			result.Diagnostics,
			diagnostics...,
		)
		if !create || artifactstore.ContainsErrorDiagnostic(diagnostics) {
			continue
		}

		data, err := jsoncanon.CanonicalizeObject(
			draft.Data,
			artifactstore.MaxLocalDataBytes,
		)
		if err != nil {
			return Reconciliation{}, err
		}
		id, err := r.ids.NewID(ctx)
		if err != nil {
			return Reconciliation{}, err
		}
		resolved := *occurrence.DefinitionDigest
		created := Record{
			ID:                 artifactstore.RecordID(id),
			RootID:             root.ID,
			Occurrence:         occurrence.Key,
			Kind:               occurrence.Kind,
			Name:               draft.Name,
			Enabled:            draft.Enabled,
			Mode:               ModeLinked,
			ResolvedDefinition: &resolved,
			Data:               json.RawMessage(data),
			State:              StateAvailable,
			Diagnostics: append(
				[]artifactstore.Diagnostic(nil),
				occurrence.Diagnostics...,
			),
			Revision:   1,
			CreatedAt:  now,
			ModifiedAt: now,
		}
		if err := created.Validate(); err != nil {
			return Reconciliation{}, fmt.Errorf(
				"validate derived record: %w",
				err,
			)
		}
		result.Creates = append(result.Creates, created)
		recordsByTypedKey[typedKey] = created
	}
	return result, nil
}

func equivalentSourceState(left, right Record) bool {
	if left.State != right.State {
		return false
	}
	if !digestPointersEqual(left.ResolvedDefinition, right.ResolvedDefinition) {
		return false
	}
	if len(left.Diagnostics) != len(right.Diagnostics) {
		return false
	}
	for index := range left.Diagnostics {
		if left.Diagnostics[index].Severity != right.Diagnostics[index].Severity ||
			left.Diagnostics[index].Code != right.Diagnostics[index].Code ||
			left.Diagnostics[index].Message != right.Diagnostics[index].Message {
			return false
		}
	}
	return true
}

func cloneDigest(value *artifactstore.Digest) *artifactstore.Digest {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func digestPointersEqual(
	left,
	right *artifactstore.Digest,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
