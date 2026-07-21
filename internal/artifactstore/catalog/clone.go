package catalog

import (
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

// CloneSnapshot returns an owned copy of a snapshot and all mutable members.
func CloneSnapshot(input Snapshot) Snapshot {
	output := input
	output.SourceRevisions = make(
		map[artifactstore.SourceID]uint64,
		len(input.SourceRevisions),
	)
	maps.Copy(output.SourceRevisions, input.SourceRevisions)
	output.SourceGenerations = make(
		map[artifactstore.SourceID]string,
		len(input.SourceGenerations),
	)
	maps.Copy(output.SourceGenerations, input.SourceGenerations)
	output.Diagnostics = artifactstore.CloneDiagnostics(input.Diagnostics)
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
	output.DefinitionDigest = cloneDigest(input.DefinitionDigest)
	output.SourceContentDigest = cloneDigest(input.SourceContentDigest)
	output.Diagnostics = artifactstore.CloneDiagnostics(input.Diagnostics)
	return output
}

func cloneDigest(value *artifactstore.Digest) *artifactstore.Digest {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
