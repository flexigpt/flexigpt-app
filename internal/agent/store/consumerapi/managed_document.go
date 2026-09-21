package consumerapi

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// ManagedAgentDocument is the Agent consumer API authoring model.
//
// It deliberately does not expose declaration.Entry. The conversion to the
// portable agentv1 declaration happens inside the Agent consumer API, where
// schema and semantic validation remain authoritative.
//
// A managed Agent is always concrete. It therefore intentionally has no
// top-level declaration locator.
type ManagedAgentDocument struct {
	Name        basespec.LogicalName `json:"name"`
	DisplayName string               `json:"displayName,omitempty"`
	Description string               `json:"description,omitempty"`
	Labels      map[string]string    `json:"labels,omitempty"`
	Metadata    map[string]any       `json:"metadata,omitempty"`

	Members  []ManagedAgentMember `json:"members,omitempty"`
	Loop     *ManagedAgentMember  `json:"loop,omitempty"`
	Workflow *ManagedAgentMember  `json:"workflow,omitempty"`
}

// ManagedAgentMember is an explicit Agent authoring relationship.
//
// Form determines which fields are emitted into the portable declaration:
//
//   - named: type, name, optional locator/scope/server/relationship fields
//   - contained: type, name, parameters, optional locator/relationship fields
//   - selector: type, base, selection patterns, relationship fields
//
// Parameters, Overrides, Use, and Metadata intentionally permit arbitrary
// JSON objects. They are declaration extension data, not Go implementation
// state, and are canonicalized before becoming a declaration.Entry.
type ManagedAgentMember struct {
	Form declaration.MemberForm `json:"form"`
	Type declaration.Type       `json:"type"`

	Name    basespec.LogicalName     `json:"name,omitempty"`
	Locator *declaration.Locator     `json:"locator,omitempty"`
	Scope   declaration.LookupScope  `json:"scope,omitempty"`
	Server  basespec.LogicalName     `json:"server,omitempty"`
	Insert  declaration.InsertTarget `json:"insert,omitempty"`

	Parameters map[string]any `json:"parameters,omitempty"`

	Base        *declaration.Locator `json:"base,omitempty"`
	Include     []string             `json:"include,omitempty"`
	Exclude     []string             `json:"exclude,omitempty"`
	NameInclude []string             `json:"nameInclude,omitempty"`
	NameExclude []string             `json:"nameExclude,omitempty"`

	Overrides map[string]any `json:"overrides,omitempty"`
	Use       map[string]any `json:"use,omitempty"`
}

// Declaration validates and converts the consumer authoring model into the
// canonical Agent declaration consumed by managed publication.
func (d ManagedAgentDocument) Declaration() (
	agentv1.AgentDocument,
	error,
) {
	if err := d.Name.Validate(); err != nil {
		return agentv1.AgentDocument{}, err
	}

	metadata, err := canonicalMetadata(d.Metadata)
	if err != nil {
		return agentv1.AgentDocument{}, err
	}

	output := agentv1.AgentDocument{
		Type:        agentv1.AgentType,
		Name:        string(d.Name),
		DisplayName: d.DisplayName,
		Description: d.Description,
		Labels:      declaration.CloneStringMap(d.Labels),
		Metadata:    metadata,

		Members: make([]declaration.Entry, 0, len(d.Members)),
	}

	for index, member := range d.Members {
		entry, err := member.declarationEntry()
		if err != nil {
			return agentv1.AgentDocument{}, fmt.Errorf(
				"managed Agent members[%d]: %w",
				index,
				err,
			)
		}
		output.Members = append(output.Members, entry)
	}

	if d.Loop != nil {
		entry, err := d.Loop.declarationEntry()
		if err != nil {
			return agentv1.AgentDocument{}, fmt.Errorf(
				"managed Agent loop: %w",
				err,
			)
		}
		output.Loop = &entry
	}

	if d.Workflow != nil {
		entry, err := d.Workflow.declarationEntry()
		if err != nil {
			return agentv1.AgentDocument{}, fmt.Errorf(
				"managed Agent workflow: %w",
				err,
			)
		}
		output.Workflow = &entry
	}

	if err := output.Validate(); err != nil {
		return agentv1.AgentDocument{}, err
	}
	return output, nil
}

func (m ManagedAgentMember) declarationEntry() (
	declaration.Entry,
	error,
) {
	if err := m.Type.Validate(); err != nil {
		return declaration.Entry{}, err
	}

	fields := map[string]any{
		"type": string(m.Type),
	}

	addRelationship := func() {
		if m.Overrides != nil {
			fields["overrides"] = m.Overrides
		}
		if m.Use != nil {
			fields["use"] = m.Use
		}
	}

	addNamedIdentity := func() error {
		if err := m.Name.Validate(); err != nil {
			return err
		}
		fields["name"] = string(m.Name)
		if m.Insert != "" {
			fields["insert"] = string(m.Insert)
		}
		if m.Locator != nil {
			fields["locator"] = *m.Locator
		}
		if m.Scope != "" {
			fields["scope"] = string(m.Scope)
		}
		if m.Server != "" {
			fields["server"] = string(m.Server)
		}
		return nil
	}

	switch m.Form {
	case declaration.MemberNamed:
		if err := addNamedIdentity(); err != nil {
			return declaration.Entry{}, err
		}
		addRelationship()

	case declaration.MemberContained:
		if err := addNamedIdentity(); err != nil {
			return declaration.Entry{}, err
		}
		delete(fields, "scope")
		delete(fields, "server")
		parameters := m.Parameters
		if parameters == nil {
			parameters = map[string]any{}
		}
		fields["parameters"] = parameters
		addRelationship()

	case declaration.MemberSelector:
		if m.Base == nil {
			return declaration.Entry{}, fmt.Errorf(
				"%w: managed Agent selector base is required",
				basespec.ErrInvalid,
			)
		}
		fields["base"] = *m.Base
		if m.Include != nil {
			fields["include"] = append([]string(nil), m.Include...)
		}
		if m.Exclude != nil {
			fields["exclude"] = append([]string(nil), m.Exclude...)
		}
		if m.NameInclude != nil {
			fields["nameInclude"] = append(
				[]string(nil),
				m.NameInclude...,
			)
		}
		if m.NameExclude != nil {
			fields["nameExclude"] = append(
				[]string(nil),
				m.NameExclude...,
			)
		}
		addRelationship()

	default:
		return declaration.Entry{}, fmt.Errorf(
			"%w: unsupported managed Agent member form %q",
			basespec.ErrInvalid,
			m.Form,
		)
	}

	return declaration.NewEntry(fields)
}

func canonicalMetadata(
	values map[string]any,
) (map[string]json.RawMessage, error) {
	if values == nil {
		//nolint:nilnil // Nil out.
		return nil, nil
	}

	output := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf(
				"managed Agent metadata %q: %w",
				key,
				err,
			)
		}
		canonical, err := jsonutil.Canonicalize(raw)
		if err != nil {
			return nil, fmt.Errorf(
				"managed Agent metadata %q: %w",
				key,
				err,
			)
		}
		output[key] = json.RawMessage(canonical)
	}
	return output, nil
}
