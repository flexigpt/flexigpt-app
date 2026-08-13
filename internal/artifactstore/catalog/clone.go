package catalog

import (
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// CloneSnapshot returns an owned copy of a snapshot and all mutable members.
func CloneSnapshot(input Snapshot) Snapshot {
	output := input
	output.AttachmentRevisions = make(
		map[basespec.SourceID]uint64,
		len(input.AttachmentRevisions),
	)
	maps.Copy(output.AttachmentRevisions, input.AttachmentRevisions)
	output.SourceRevisions = make(
		map[basespec.SourceID]uint64,
		len(input.SourceRevisions),
	)
	maps.Copy(output.SourceRevisions, input.SourceRevisions)
	output.SourceGenerations = make(
		map[basespec.SourceID]string,
		len(input.SourceGenerations),
	)
	maps.Copy(output.SourceGenerations, input.SourceGenerations)
	output.Diagnostics = diagnostic.CloneDiagnostics(input.Diagnostics)
	output.Occurrences = make([]Occurrence, len(input.Occurrences))
	for index, occurrence := range input.Occurrences {
		output.Occurrences[index] = CloneOccurrence(occurrence)
	}
	return output
}

// CloneOccurrence returns an owned copy of an occurrence and all mutable
// members.
func CloneOccurrence(input Occurrence) Occurrence {
	output := input
	output.DefinitionDigest = cryptoutil.CloneDigest(input.DefinitionDigest)
	output.SourceContentDigest = cryptoutil.CloneDigest(input.SourceContentDigest)
	if input.Definition != nil {
		value := input.Definition.Clone()
		output.Definition = &value
	}
	output.Diagnostics = diagnostic.CloneDiagnostics(input.Diagnostics)
	return output
}
