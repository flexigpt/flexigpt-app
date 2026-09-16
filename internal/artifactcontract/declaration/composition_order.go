package declaration

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// SortedCompositionEntries returns deterministic entry order without assigning
// semantic meaning to declaration-array position. Full canonical Entry bytes
// are used as the ordering key, so metadata differences remain distinct.
func SortedCompositionEntries(
	label string,
	values []Entry,
) ([]Entry, error) {
	type sortableEntry struct {
		entry Entry
		raw   []byte
	}

	ordered := make([]sortableEntry, 0, len(values))
	for index, value := range values {
		raw, err := value.CanonicalJSON()
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		ordered = append(ordered, sortableEntry{
			entry: value.Clone(),
			raw:   raw,
		})
	}

	sort.SliceStable(ordered, func(left, right int) bool {
		return bytes.Compare(ordered[left].raw, ordered[right].raw) < 0
	})

	output := make([]Entry, len(ordered))
	for index, value := range ordered {
		output[index] = value.entry.Clone()
	}
	return output, nil
}

// ValidateContainedEntryUniqueness rejects sibling contained declarations
// with the same type and name. External references intentionally remain loose:
// they may repeat, and metadata differences remain distinct occurrences.
func ValidateContainedEntryUniqueness(
	label string,
	values []Entry,
) error {
	seen := make(map[string]int)
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}

		form, err := value.CompositionForm()
		if err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if form != CompositionEntryContained {
			continue
		}

		header := value.Header()
		key := string(header.Type) + "\x00" + header.Name
		if previous, duplicate := seen[key]; duplicate {
			return fmt.Errorf(
				"%w: %s[%d] duplicates contained declaration %s/%s from %s[%d]",
				basespec.ErrIdentityConflict,
				label,
				index,
				header.Type,
				header.Name,
				label,
				previous,
			)
		}
		seen[key] = index
	}
	return nil
}
