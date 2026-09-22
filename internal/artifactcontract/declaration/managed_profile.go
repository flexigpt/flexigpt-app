package declaration

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const managedProfileSchemaDialect = "https://json-schema.org/draft/2020-12/schema"

// ManagedProfilePolicyDescriptor binds a non-portable managed admission
// restriction schema to one existing portable declaration schema.
type ManagedProfilePolicyDescriptor struct {
	ID string

	DeclarationType Type

	BaseSchemaJSON   []byte
	RestrictionsJSON []byte
}

// ManagedProfilePolicy is a compiled managed admission policy for one
// portable declaration type.
type ManagedProfilePolicy struct {
	id              string
	declarationType Type
	fingerprint     cryptoutil.Digest
	compiled        *jsonschema.Schema
}

func (p *ManagedProfilePolicy) ID() string {
	if p == nil {
		return ""
	}
	return p.id
}

func (p *ManagedProfilePolicy) DeclarationType() Type {
	if p == nil {
		return ""
	}
	return p.declarationType
}

func (p *ManagedProfilePolicy) Fingerprint() cryptoutil.Digest {
	if p == nil {
		return ""
	}
	return p.fingerprint
}

// Validate validates portable declaration bytes against the existing portable
// schema and managed restrictions as one conjunction. It returns canonical
// JSON and does not create another portable schema identity.
func (p *ManagedProfilePolicy) Validate(
	raw []byte,
) ([]byte, error) {
	if p == nil || p.compiled == nil {
		return nil, basespec.ErrClosed
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, err
	}
	if err := jsonutil.ValidateJSONSchema(
		p.compiled,
		json.RawMessage(canonical),
		basespec.MaxDefinitionBytes,
	); err != nil {
		return nil, err
	}

	return append([]byte(nil), canonical...), nil
}

// ManagedProfileRegistry is immutable after construction. One declaration
// type can have only one managed profile policy.
type ManagedProfileRegistry struct {
	byType map[Type]*ManagedProfilePolicy
}

func NewManagedProfileRegistry(
	descriptors ...ManagedProfilePolicyDescriptor,
) (*ManagedProfileRegistry, error) {
	if len(descriptors) == 0 {
		return nil, fmt.Errorf(
			"%w: managed profile registry has no descriptors",
			basespec.ErrInvalid,
		)
	}

	output := &ManagedProfileRegistry{
		byType: make(map[Type]*ManagedProfilePolicy, len(descriptors)),
	}
	for index, descriptor := range descriptors {
		policy, err := compileManagedProfilePolicy(descriptor)
		if err != nil {
			return nil, fmt.Errorf(
				"managed profile policy descriptor %d: %w",
				index,
				err,
			)
		}

		if _, duplicate := output.byType[policy.declarationType]; duplicate {
			return nil, fmt.Errorf(
				"%w: declaration type %q has multiple managed profile policies",
				basespec.ErrConflict,
				policy.declarationType,
			)
		}

		output.byType[policy.declarationType] = policy
	}

	return output, nil
}

func (r *ManagedProfileRegistry) ManagedProfilePolicyFor(
	declarationType Type,
) (*ManagedProfilePolicy, error) {
	if r == nil {
		return nil, basespec.ErrClosed
	}
	if err := declarationType.Validate(); err != nil {
		return nil, err
	}

	policy, found := r.byType[declarationType]
	if !found {
		return nil, fmt.Errorf(
			"%w: no managed profile policy for declaration type %q",
			basespec.ErrNotFound,
			declarationType,
		)
	}

	return policy, nil
}

func compileManagedProfilePolicy(
	descriptor ManagedProfilePolicyDescriptor,
) (*ManagedProfilePolicy, error) {
	if err := basespec.ValidateRequiredText(
		"managed profile policy ID",
		descriptor.ID,
		basespec.MaxKindBytes,
	); err != nil {
		return nil, err
	}
	if err := descriptor.DeclarationType.Validate(); err != nil {
		return nil, err
	}

	base, err := canonicalManagedProfileSchema(
		"managed profile base schema",
		descriptor.BaseSchemaJSON,
	)
	if err != nil {
		return nil, err
	}
	if err := validateManagedProfileBaseSchemaType(
		base,
		descriptor.DeclarationType,
	); err != nil {
		return nil, err
	}

	restrictions, err := canonicalManagedProfileSchema(
		"managed profile restrictions schema",
		descriptor.RestrictionsJSON,
	)
	if err != nil {
		return nil, err
	}

	combined, err := jsonutil.MarshalCanonicalObject(
		map[string]any{
			"$schema": managedProfileSchemaDialect,
			"allOf": []json.RawMessage{
				json.RawMessage(base),
				json.RawMessage(restrictions),
			},
		},
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, err
	}

	compiled, err := jsonutil.CompileJSONSchema(combined)
	if err != nil {
		return nil, fmt.Errorf(
			"compile managed profile policy schema conjunction: %w",
			err,
		)
	}

	return &ManagedProfilePolicy{
		id:              descriptor.ID,
		declarationType: descriptor.DeclarationType,
		fingerprint:     cryptoutil.DigestBytes(combined),
		compiled:        compiled,
	}, nil
}

func canonicalManagedProfileSchema(
	label string,
	raw []byte,
) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf(
			"%w: %s is empty",
			basespec.ErrInvalid,
			label,
		)
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if _, err := jsonutil.CompileJSONSchema(canonical); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}

	return canonical, nil
}

func validateManagedProfileBaseSchemaType(
	raw []byte,
	expected Type,
) error {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return err
	}

	typeRule, found := schema.Properties["type"]
	if !found {
		return fmt.Errorf(
			"%w: managed profile base schema has no type rule",
			basespec.ErrInvalid,
		)
	}

	var value struct {
		Const string `json:"const"`
	}
	if err := json.Unmarshal(typeRule, &value); err != nil {
		return err
	}
	if value.Const != string(expected) {
		return fmt.Errorf(
			"%w: managed profile base schema type is %q, expected %q",
			basespec.ErrInvalid,
			value.Const,
			expected,
		)
	}

	return nil
}
