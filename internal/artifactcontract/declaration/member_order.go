package declaration

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

func SortedMembers(
	label string,
	values []Entry,
) ([]Entry, error) {
	type sortable struct {
		entry Entry
		raw   []byte
	}

	ordered := make([]sortable, 0, len(values))
	for index, value := range values {
		if _, err := value.MemberForm(); err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		raw, err := value.CanonicalJSON()
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		ordered = append(ordered, sortable{
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

func ValidateMemberUniqueness(
	label string,
	values []Entry,
) error {
	exact := make(map[string]int, len(values))
	contained := make(map[string]int, len(values))

	for index, value := range values {
		form, err := value.MemberForm()
		if err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		raw, err := value.CanonicalJSON()
		if err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if previous, duplicate := exact[string(raw)]; duplicate {
			return fmt.Errorf(
				"%w: %s[%d] exactly duplicates %s[%d]",
				basespec.ErrIdentityConflict,
				label,
				index,
				label,
				previous,
			)
		}
		exact[string(raw)] = index

		if form != MemberContained {
			continue
		}
		header := value.Header()
		key := string(header.Type) + "\x00" + header.Name
		if header.Type == TypeText {
			insert, err := value.TextInsert()
			if err != nil {
				return fmt.Errorf("%s[%d]: %w", label, index, err)
			}
			key += "\x00" + string(insert)
		}
		if previous, duplicate := contained[key]; duplicate {
			return fmt.Errorf(
				"%w: %s[%d] duplicates contained declaration identity from %s[%d]",
				basespec.ErrIdentityConflict,
				label,
				index,
				label,
				previous,
			)
		}
		contained[key] = index
	}
	return nil
}

func ValidateMemberTypes(
	label string,
	values []Entry,
	allowed ...Type,
) error {
	accepted := make(map[Type]struct{}, len(allowed))
	for _, value := range allowed {
		accepted[value] = struct{}{}
	}
	for index, value := range values {
		if _, err := value.MemberForm(); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if _, found := accepted[value.Header().Type]; !found {
			return fmt.Errorf(
				"%w: %s[%d] has incompatible type %q",
				basespec.ErrInvalid,
				label,
				index,
				value.Header().Type,
			)
		}
	}
	return ValidateMemberUniqueness(label, values)
}
