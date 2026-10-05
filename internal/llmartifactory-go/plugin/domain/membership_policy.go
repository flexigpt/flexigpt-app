// Package domain contains Plugin-owned domain values.
package domain

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

const MembershipPolicyMetadataKey = "flexigpt.site/plugin-membership-policy"

type MembershipMode string

const (
	MembershipModeSingleType MembershipMode = "single-type"
	MembershipModeMixedType  MembershipMode = "mixed-type"

	// MembershipModeLegacyUnconstrained is read-only compatibility for
	// existing Plugin declarations that predate explicit policy metadata.
	MembershipModeLegacyUnconstrained MembershipMode = "legacy-unconstrained"
)

type MembershipPolicy struct {
	Mode MembershipMode `json:"mode"`

	AllowedTypes []declaration.Type       `json:"allowedTypes,omitempty"`
	AllowedForms []declaration.MemberForm `json:"allowedForms,omitempty"`
}

func LegacyMembershipPolicy() MembershipPolicy {
	return MembershipPolicy{
		Mode: MembershipModeLegacyUnconstrained,
	}
}

func (p MembershipPolicy) Validate() error {
	switch p.Mode {
	case MembershipModeLegacyUnconstrained:
		if len(p.AllowedTypes) != 0 || len(p.AllowedForms) != 0 {
			return fmt.Errorf(
				"%w: legacy Plugin policy cannot declare constraints",
				spec.ErrInvalid,
			)
		}
		return nil

	case MembershipModeSingleType:
		if len(p.AllowedTypes) != 1 {
			return fmt.Errorf(
				"%w: single-type Plugin policy requires exactly one type",
				spec.ErrInvalid,
			)
		}

	case MembershipModeMixedType:
		if len(p.AllowedTypes) < 2 {
			return fmt.Errorf(
				"%w: mixed-type Plugin policy requires at least two types",
				spec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: unsupported Plugin membership mode %q",
			spec.ErrInvalid,
			p.Mode,
		)
	}

	seenTypes := make(map[declaration.Type]struct{}, len(p.AllowedTypes))
	for _, value := range p.AllowedTypes {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, duplicate := seenTypes[value]; duplicate {
			return fmt.Errorf(
				"%w: Plugin membership policy repeats type %q",
				spec.ErrIdentityConflict,
				value,
			)
		}
		seenTypes[value] = struct{}{}
	}

	seenForms := make(map[declaration.MemberForm]struct{}, len(p.AllowedForms))
	for _, value := range p.AllowedForms {
		switch value {
		case declaration.MemberNamed,
			declaration.MemberContained,
			declaration.MemberSelector:
		default:
			return fmt.Errorf(
				"%w: Plugin membership policy has invalid member form %q",
				spec.ErrInvalid,
				value,
			)
		}
		if _, duplicate := seenForms[value]; duplicate {
			return fmt.Errorf(
				"%w: Plugin membership policy repeats member form %q",
				spec.ErrIdentityConflict,
				value,
			)
		}
		seenForms[value] = struct{}{}
	}
	return nil
}

func (p MembershipPolicy) Allows(
	entry declaration.Entry,
) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Mode == MembershipModeLegacyUnconstrained {
		return nil
	}

	form, err := entry.MemberForm()
	if err != nil {
		return err
	}
	if len(p.AllowedForms) != 0 {
		found := slices.Contains(p.AllowedForms, form)
		if !found {
			return fmt.Errorf(
				"%w: Plugin membership form %q is not allowed",
				spec.ErrUnsupported,
				form,
			)
		}
	}

	typeFound := false
	for _, allowed := range p.AllowedTypes {
		if allowed == entry.Header().Type {
			typeFound = true
			break
		}
	}
	if !typeFound {
		return fmt.Errorf(
			"%w: Plugin membership type %q is not allowed",
			spec.ErrUnsupported,
			entry.Header().Type,
		)
	}
	return nil
}

func FromMetadata(
	metadata map[string]json.RawMessage,
) (MembershipPolicy, error) {
	raw, found := metadata[MembershipPolicyMetadataKey]
	if !found {
		return LegacyMembershipPolicy(), nil
	}

	var value MembershipPolicy
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		raw,
		&value,
		spec.MaxLocalDataBytes,
	); err != nil {
		return MembershipPolicy{}, fmt.Errorf(
			"%w: decode Plugin membership policy: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if err := value.Validate(); err != nil {
		return MembershipPolicy{}, err
	}
	value.AllowedTypes = append([]declaration.Type(nil), value.AllowedTypes...)
	value.AllowedForms = append(
		[]declaration.MemberForm(nil),
		value.AllowedForms...,
	)
	slices.Sort(value.AllowedTypes)
	slices.Sort(value.AllowedForms)
	return value, nil
}

func PutMetadata(
	metadata map[string]json.RawMessage,
	policy MembershipPolicy,
) (map[string]json.RawMessage, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	output := declaration.CloneRawMessageMap(metadata)
	if output == nil {
		output = make(map[string]json.RawMessage)
	}
	if policy.Mode == MembershipModeLegacyUnconstrained {
		delete(output, MembershipPolicyMetadataKey)
		return output, nil
	}

	raw, err := jsonutil.MarshalCanonicalObject(
		policy,
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	output[MembershipPolicyMetadataKey] = raw
	return output, nil
}
