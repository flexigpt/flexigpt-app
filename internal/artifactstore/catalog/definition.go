package catalog

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

// DefinitionForOccurrence returns the SQLite-cached canonical definition for
// one current catalog occurrence. It does not perform CAS lookup.
func DefinitionForOccurrence(
	snapshot Snapshot,
	key OccurrenceKey,
) (definition.Definition, error) {
	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Key != key {
			continue
		}
		if occurrence.State != OccurrenceValid ||
			occurrence.Definition == nil {
			return definition.Definition{}, fmt.Errorf(
				"%w: occurrence %q has no available definition",
				basespec.ErrDefinitionNotFound,
				key.Locator,
			)
		}
		return occurrence.Definition.Clone(), nil
	}
	return definition.Definition{}, fmt.Errorf(
		"%w: occurrence %q",
		basespec.ErrDefinitionNotFound,
		key.Locator,
	)
}
