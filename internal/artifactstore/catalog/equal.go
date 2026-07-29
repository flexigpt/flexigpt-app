package catalog

import (
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// EqualSnapshot compares the semantic contents of two catalog snapshots.
// Occurrences are compared by occurrence key because their persisted ordering
// is not part of catalog identity.
func EqualSnapshot(left, right Snapshot) bool {
	if left.RootID != right.RootID ||
		left.CollectionID != right.CollectionID ||
		left.Revision != right.Revision ||
		left.CollectionRevision != right.CollectionRevision ||
		left.PlanFingerprint != right.PlanFingerprint ||
		left.DecoderFingerprint != right.DecoderFingerprint ||
		!left.PublishedAt.Equal(right.PublishedAt) ||
		!maps.Equal(left.AttachmentRevisions, right.AttachmentRevisions) ||
		!maps.Equal(left.SourceRevisions, right.SourceRevisions) ||
		!maps.Equal(left.SourceGenerations, right.SourceGenerations) ||
		!diagnostic.EqualDiagnostics(left.Diagnostics, right.Diagnostics) {
		return false
	}

	return EqualOccurrences(left.Occurrences, right.Occurrences)
}

// EqualOccurrences compares occurrence values independently of ordering.
func EqualOccurrences(left, right []Occurrence) bool {
	if len(left) != len(right) {
		return false
	}

	byKey := make(map[OccurrenceKey]Occurrence, len(left))
	for _, value := range left {
		if _, duplicate := byKey[value.Key]; duplicate {
			return false
		}
		byKey[value.Key] = value
	}

	for _, value := range right {
		leftValue, found := byKey[value.Key]
		if !found || !equalOccurrence(leftValue, value) {
			return false
		}
	}
	return true
}

func equalOccurrence(left, right Occurrence) bool {
	return left.RootID == right.RootID &&
		left.CollectionID == right.CollectionID &&
		left.Key == right.Key &&
		left.Kind == right.Kind &&
		left.LogicalName == right.LogicalName &&
		left.LogicalVersion == right.LogicalVersion &&
		cryptoutil.IsDigestEqual(left.DefinitionDigest, right.DefinitionDigest) &&
		cryptoutil.IsDigestEqual(left.SourceContentDigest, right.SourceContentDigest) &&
		left.DecoderID == right.DecoderID &&
		left.State == right.State &&
		left.ObservedAt.Equal(right.ObservedAt) &&
		diagnostic.EqualDiagnostics(left.Diagnostics, right.Diagnostics)
}
