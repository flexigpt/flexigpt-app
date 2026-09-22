package declaration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// MemberIdentityJSON returns canonical relationship identity bytes. It
// normalizes fields whose ordering or explicit emptiness has no semantic
// meaning without changing the declaration's persisted canonical bytes.
func MemberIdentityJSON(
	value Entry,
) ([]byte, error) {
	_, identity, _, err := normalizedMemberIdentityJSON(value)
	return identity, err
}

func normalizedMemberIdentityJSON(
	value Entry,
) (form MemberForm, identity, raw []byte, err error) {
	form, err = value.MemberForm()
	if err != nil {
		return "", nil, nil, err
	}
	raw = append([]byte(nil), value.raw...)

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", nil, nil, err
	}
	for _, name := range []string{"overrides", "use"} {
		if err := normalizeEmptyMemberObject(fields, name); err != nil {
			return "", nil, nil, err
		}
	}
	if form == MemberSelector {
		for _, name := range []string{
			"include",
			"exclude",
			"nameInclude",
			"nameExclude",
		} {
			if err := normalizeMemberStringSet(fields, name); err != nil {
				return "", nil, nil, err
			}
		}
	}

	identity, err = jsonutil.MarshalCanonicalObject(
		fields,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return "", nil, nil, err
	}
	return form, identity, raw, nil
}

func normalizeEmptyMemberObject(
	fields map[string]json.RawMessage,
	name string,
) error {
	raw, found := fields[name]
	if !found {
		return nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	if len(values) == 0 {
		delete(fields, name)
	}
	return nil
}

func normalizeMemberStringSet(
	fields map[string]json.RawMessage,
	name string,
) error {
	raw, found := fields[name]
	if !found {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	slices.Sort(values)
	values = slices.Compact(values)
	if len(values) == 0 {
		delete(fields, name)
		return nil
	}
	normalized, err := json.Marshal(values)
	if err != nil {
		return err
	}
	fields[name] = normalized
	return nil
}

func SortedMembers(
	label string,
	values []Entry,
) ([]Entry, error) {
	type sortable struct {
		entry    Entry
		identity []byte
		raw      []byte
	}

	ordered := make([]sortable, 0, len(values))
	for index, value := range values {
		_, identity, raw, err := normalizedMemberIdentityJSON(value)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		ordered = append(ordered, sortable{
			entry:    value.Clone(),
			identity: identity,
			raw:      raw,
		})
	}
	sort.SliceStable(ordered, func(left, right int) bool {
		comparison := bytes.Compare(
			ordered[left].identity,
			ordered[right].identity,
		)
		if comparison != 0 {
			return comparison < 0
		}
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
		form, identity, _, err := normalizedMemberIdentityJSON(value)
		if err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if previous, duplicate := exact[string(identity)]; duplicate {
			return fmt.Errorf(
				"%w: %s[%d] duplicates normalized %s[%d]",
				basespec.ErrIdentityConflict,
				label,
				index,
				label,
				previous,
			)
		}
		exact[string(identity)] = index

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

func ValidateMembersWithoutRelationshipBehavior(
	label string,
	values []Entry,
	allowed ...Type,
) error {
	var err error
	if len(allowed) == 0 {
		err = ValidateMemberUniqueness(label, values)
	} else {
		err = ValidateMemberTypes(label, values, allowed...)
	}
	if err != nil {
		return err
	}

	for index, value := range values {
		if err := ValidateNoRelationshipBehavior(
			label+" member",
			value,
		); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
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
