package artifactcontract

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// Entry is a heterogeneous nested declaration or symbolic reference.
//
// The shared package validates only the common header. The independently
// versioned contract selected by Entry.Header().Type validates type-specific
// fields. This prevents the root utility package from becoming a monolithic
// declaration schema.
type Entry struct {
	raw    json.RawMessage
	header Header
}

func DecodeEntryJSON(raw []byte) (Entry, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return Entry{}, fmt.Errorf(
			"%w: canonicalize declaration entry: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	var header Header
	if err := json.Unmarshal(canonical, &header); err != nil {
		return Entry{}, fmt.Errorf(
			"%w: decode declaration entry header: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := header.Validate(HeaderValidation{}); err != nil {
		return Entry{}, err
	}
	return Entry{
		raw:    json.RawMessage(canonical),
		header: header,
	}, nil
}

func NewEntry(value any) (Entry, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return Entry{}, err
	}
	return DecodeEntryJSON(raw)
}

func (e Entry) Header() Header {
	return e.header.Clone()
}

func (e Entry) Validate() error {
	if len(e.raw) == 0 {
		return fmt.Errorf(
			"%w: declaration entry is empty",
			basespec.ErrInvalid,
		)
	}
	return e.header.Validate(HeaderValidation{})
}

func (e Entry) Clone() Entry {
	return Entry{
		raw:    append(json.RawMessage(nil), e.raw...),
		header: e.header.Clone(),
	}
}

func (e Entry) CanonicalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return append([]byte(nil), e.raw...), nil
}

func (e Entry) CalculatedDigest() (cryptoutil.Digest, error) {
	raw, err := e.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(raw), nil
}

func (e Entry) DecodeInto(target any) error {
	if err := e.Validate(); err != nil {
		return err
	}
	return jsonutil.DecodeCanonicalObjectInto(
		e.raw,
		target,
		basespec.MaxDefinitionBodyBytes,
	)
}

// IsSymbolic reports the portable exact reference form:
//
//	type: <type>
//	name: <name>
func (e Entry) IsSymbolic() bool {
	if err := e.Validate(); err != nil {
		return false
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(e.raw, &values); err != nil {
		return false
	}
	if len(values) != 2 {
		return false
	}
	_, hasType := values["type"]
	_, hasName := values["name"]
	return hasType && hasName
}

func (e Entry) MarshalJSON() ([]byte, error) {
	return e.CanonicalJSON()
}

func (e *Entry) UnmarshalJSON(raw []byte) error {
	if e == nil {
		return fmt.Errorf(
			"%w: declaration entry target is nil",
			basespec.ErrInvalid,
		)
	}
	value, err := DecodeEntryJSON(raw)
	if err != nil {
		return err
	}
	*e = value
	return nil
}

func ValidateEntryType(
	entry Entry,
	expected Type,
) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	if entry.Header().Type != expected {
		return fmt.Errorf(
			"%w: declaration entry type is %q, expected %q",
			basespec.ErrInvalid,
			entry.Header().Type,
			expected,
		)
	}
	return nil
}

func ValidateEntryTypes(
	label string,
	values []Entry,
	allowed ...Type,
) error {
	allowedTypes := make(map[Type]struct{}, len(allowed))
	for _, declarationType := range allowed {
		allowedTypes[declarationType] = struct{}{}
	}
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
		if _, supported := allowedTypes[value.Header().Type]; !supported {
			return fmt.Errorf(
				"%w: %s[%d] has incompatible type %q",
				basespec.ErrInvalid,
				label,
				index,
				value.Header().Type,
			)
		}
	}
	return nil
}
