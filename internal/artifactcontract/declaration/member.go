package declaration

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type MemberForm string

const (
	MemberNamed     MemberForm = "named"
	MemberContained MemberForm = "contained"
	MemberSelector  MemberForm = "selector"
)

type LookupScope string

const LookupScopeBuiltin LookupScope = "builtin"

func (s LookupScope) Validate() error {
	switch s {
	case "", LookupScopeBuiltin:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported member lookup scope %q",
			basespec.ErrInvalid,
			s,
		)
	}
}

type Selector struct {
	Type        Type     `json:"type"`
	Base        Locator  `json:"base"`
	Include     []string `json:"include,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
	NameInclude []string `json:"nameInclude,omitempty"`
	NameExclude []string `json:"nameExclude,omitempty"`
}

func (s Selector) Validate() error {
	if err := s.Type.Validate(); err != nil {
		return err
	}
	if s.Type == TypeText || s.Type == TypeWorkspace {
		return fmt.Errorf(
			"%w: Artifact type %q does not support member selectors",
			basespec.ErrInvalid,
			s.Type,
		)
	}
	if err := s.Base.Validate(); err != nil {
		return fmt.Errorf("member selector base: %w", err)
	}
	if err := validateLocalSelectorBase(s.Base); err != nil {
		return err
	}

	if err := basespec.ValidatePathPatterns(
		"member selector include",
		s.Include,
	); err != nil {
		return err
	}
	if err := basespec.ValidatePathPatterns(
		"member selector exclude",
		s.Exclude,
	); err != nil {
		return err
	}
	if err := basespec.ValidatePathPatterns(
		"member selector nameInclude",
		s.NameInclude,
	); err != nil {
		return err
	}
	return basespec.ValidatePathPatterns(
		"member selector nameExclude",
		s.NameExclude,
	)
}

// A selector base identifies a local directory tree inside the declaring
// Source. URL, Git, package, archive, and other external selector bases need
// separate Source and discovery support before they can be enabled.
//
// This restriction applies only to selector base. Ordinary declaration
// locators continue to support every Locator kind.
func validateLocalSelectorBase(value Locator) error {
	switch value.Kind {
	case "":
		parsed, err := url.Parse(value.Scalar)
		if err != nil {
			return fmt.Errorf(
				"%w: invalid member selector base",
				basespec.ErrInvalid,
			)
		}
		if parsed.Scheme != "" {
			return fmt.Errorf(
				"%w: member selector base must be a local relative path",
				basespec.ErrInvalid,
			)
		}
		return ValidatePortableLocatorPath(
			"member selector base",
			value.Scalar,
			true,
		)

	case LocatorKindPath:
		return ValidatePortableLocatorPath(
			"member selector base",
			value.Path,
			true,
		)

	default:
		return fmt.Errorf(
			"%w: member selector base must be a local relative path",
			basespec.ErrInvalid,
		)
	}
}

type Relationship struct {
	Scope     LookupScope                `json:"scope,omitempty"`
	Overrides map[string]json.RawMessage `json:"overrides,omitempty"`
	Use       map[string]json.RawMessage `json:"use,omitempty"`
}

func (r Relationship) Clone() Relationship {
	return Relationship{
		Scope:     r.Scope,
		Overrides: CloneRawMessageMap(r.Overrides),
		Use:       CloneRawMessageMap(r.Use),
	}
}

func (r Relationship) Validate() error {
	if err := r.Scope.Validate(); err != nil {
		return err
	}
	if err := ValidateRawMessageMap(
		"member overrides",
		r.Overrides,
	); err != nil {
		return err
	}
	return ValidateRawMessageMap("member use", r.Use)
}

func (e Entry) MemberForm() (MemberForm, error) {
	if err := e.Validate(); err != nil {
		return "", err
	}
	fields, err := e.fields()
	if err != nil {
		return "", err
	}

	_, hasName := fields["name"]
	_, hasBase := fields["base"]
	_, hasParameters := fields["parameters"]

	switch {
	case hasBase:
		if hasName || hasParameters {
			return "", fmt.Errorf(
				"%w: member selector cannot contain name or parameters",
				basespec.ErrInvalid,
			)
		}
		if err := validateAllowedMemberFields(
			fields,
			"type",
			"base",
			"include",
			"exclude",
			"nameInclude",
			"nameExclude",
			"overrides",
			"use",
		); err != nil {
			return "", err
		}
		selector, err := e.Selector()
		if err != nil {
			return "", err
		}
		if err := selector.Validate(); err != nil {
			return "", err
		}
		if _, err := e.Relationship(); err != nil {
			return "", err
		}

		return MemberSelector, nil

	case hasParameters:
		if !hasName {
			return "", fmt.Errorf(
				"%w: contained member requires name",
				basespec.ErrInvalid,
			)
		}
		if err := validateAllowedMemberFields(
			fields,
			"type",
			"name",
			"insert",
			"locator",
			"parameters",
			"overrides",
			"use",
		); err != nil {
			return "", err
		}
		if err := e.Header().Validate(HeaderValidation{
			RequireName: true,
		}); err != nil {
			return "", err
		}
		if _, err := e.Relationship(); err != nil {
			return "", err
		}
		if _, err := e.ContainedDeclaration(); err != nil {
			return "", err
		}

		return MemberContained, nil

	default:
		if !hasName {
			return "", fmt.Errorf(
				"%w: named member requires name",
				basespec.ErrInvalid,
			)
		}
		if err := validateAllowedMemberFields(
			fields,
			"type",
			"name",
			"insert",
			"locator",
			"scope",
			"server",
			"overrides",
			"use",
		); err != nil {
			return "", err
		}
		if err := e.validateNamedMember(fields); err != nil {
			return "", err
		}
		return MemberNamed, nil
	}
}

func (e Entry) Selector() (Selector, error) {
	var value Selector
	if err := e.decodeProjection(&value); err != nil {
		return Selector{}, err
	}
	if err := value.Validate(); err != nil {
		return Selector{}, err
	}
	return value, nil
}

func (e Entry) Relationship() (Relationship, error) {
	var value Relationship
	if err := e.decodeProjection(&value); err != nil {
		return Relationship{}, err
	}
	if err := value.Validate(); err != nil {
		return Relationship{}, err
	}
	return value, nil
}

func (e Entry) TextInsert() (InsertTarget, error) {
	if e.Header().Type != TypeText {
		return "", fmt.Errorf(
			"%w: insertion identity is valid only for Text",
			basespec.ErrInvalid,
		)
	}
	var value struct {
		Insert InsertTarget `json:"insert"`
	}
	if err := e.decodeProjection(&value); err != nil {
		return "", err
	}
	if err := value.Insert.Validate(); err != nil {
		return "", err
	}
	return value.Insert, nil
}

func (e Entry) ContainedDeclaration() (Entry, error) {
	fields, err := e.fields()
	if err != nil {
		return Entry{}, err
	}
	parametersRaw, found := fields["parameters"]
	if !found {
		return Entry{}, fmt.Errorf(
			"%w: member does not contain parameters",
			basespec.ErrInvalid,
		)
	}

	var target map[string]json.RawMessage
	if err := json.Unmarshal(parametersRaw, &target); err != nil ||
		target == nil {
		return Entry{}, fmt.Errorf(
			"%w: contained member parameters must be an object",
			basespec.ErrInvalid,
		)
	}
	for _, identity := range []string{"type", "name", "locator"} {
		if raw, present := fields[identity]; present {
			target[identity] = append(json.RawMessage(nil), raw...)
		}
	}
	if e.Header().Type == TypeText {
		raw, present := fields["insert"]
		if !present {
			return Entry{}, fmt.Errorf(
				"%w: Text member requires direct insert",
				basespec.ErrInvalid,
			)
		}
		target["insert"] = append(json.RawMessage(nil), raw...)
	}

	raw, err := jsonutil.MarshalCanonicalObject(
		target,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return Entry{}, err
	}
	return DecodeCanonicalEntryJSON(raw)
}

func NewContainedMember(target Entry) (Entry, error) {
	if err := target.Validate(); err != nil {
		return Entry{}, err
	}
	if target.Header().Name == "" {
		return Entry{}, fmt.Errorf(
			"%w: contained declaration requires name",
			basespec.ErrInvalid,
		)
	}

	fields, err := target.fields()
	if err != nil {
		return Entry{}, err
	}
	member := make(map[string]json.RawMessage)
	parameters := make(map[string]json.RawMessage)
	for name, raw := range fields {
		switch name {
		case "type", "name", "locator":
			member[name] = append(json.RawMessage(nil), raw...)
		case "insert":
			if target.Header().Type == TypeText {
				member[name] = append(json.RawMessage(nil), raw...)
			} else {
				parameters[name] = append(json.RawMessage(nil), raw...)
			}
		default:
			parameters[name] = append(json.RawMessage(nil), raw...)
		}
	}
	parametersRaw, err := jsonutil.MarshalCanonicalObject(
		parameters,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return Entry{}, err
	}
	member["parameters"] = parametersRaw

	raw, err := jsonutil.MarshalCanonicalObject(
		member,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return Entry{}, err
	}
	return DecodeCanonicalEntryJSON(raw)
}

func (e Entry) validateNamedMember(
	fields map[string]json.RawMessage,
) error {
	header := e.Header()
	if err := header.Validate(HeaderValidation{RequireName: true}); err != nil {
		return err
	}

	relationship, err := e.Relationship()
	if err != nil {
		return err
	}
	if relationship.Scope != "" && header.Locator != nil {
		return fmt.Errorf(
			"%w: scoped named member cannot contain locator",
			basespec.ErrInvalid,
		)
	}
	if header.Type == TypeText {
		if _, err := e.TextInsert(); err != nil {
			return err
		}
	} else if _, present := fields["insert"]; present {
		return fmt.Errorf(
			"%w: member insert is valid only for Text",
			basespec.ErrInvalid,
		)
	}

	serverRaw, hasServer := fields["server"]
	if !hasServer {
		return nil
	}
	if header.Type != TypeMCP || header.Locator == nil {
		return fmt.Errorf(
			"%w: MCP server selector requires a locator-selected MCP member",
			basespec.ErrInvalid,
		)
	}
	var server basespec.LogicalName
	if err := json.Unmarshal(serverRaw, &server); err != nil {
		return err
	}
	return server.Validate()
}

func (e Entry) fields() (map[string]json.RawMessage, error) {
	raw, err := e.CanonicalJSON()
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// decodeProjection decodes a partial local projection of a canonical Entry.
//
// Entry.DecodeInto remains strict and is used only where the destination
// represents the complete declaration shape. Member projections intentionally
// observe only their owned fields and must ignore sibling target or
// relationship fields.
func (e Entry) decodeProjection(target any) error {
	raw, err := e.CanonicalJSON()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode declaration entry projection: %w", err)
	}
	return nil
}

func validateAllowedMemberFields(
	fields map[string]json.RawMessage,
	allowed ...string,
) error {
	accepted := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		accepted[name] = struct{}{}
	}
	for name := range fields {
		if _, found := accepted[name]; found {
			continue
		}
		return fmt.Errorf(
			"%w: member contains unsupported or flattened target field %q",
			basespec.ErrInvalid,
			name,
		)
	}
	return nil
}
