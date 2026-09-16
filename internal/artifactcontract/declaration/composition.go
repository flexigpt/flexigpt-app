package declaration

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// CompositionEntryForm identifies how an Entry behaves when it appears in a
// composition position such as Collection.members, Agent.members, a Workflow
// node target, or Workspace.roots.
type CompositionEntryForm string

const (
	// CompositionEntryReference is an edge to an independently declared
	// Artifact. It must never create a source subresource Artifact.
	CompositionEntryReference CompositionEntryForm = "reference"

	// CompositionEntryContained is a complete declaration authored in the
	// containing document. It becomes a source-backed subresource Artifact.
	CompositionEntryContained CompositionEntryForm = "contained"
)

// CompositionForm classifies an Entry only for a composition-entry position.
//
// A common declaration header with an optional non-command locator remains an
// external reference. Header annotations are membership-local presentation
// data and do not overlay the selected target. A type-specific body field
// makes the entry contained. An MCP server selector is part of a reference
// when no executable MCP connection fields are present.
func (e Entry) CompositionForm() (CompositionEntryForm, error) {
	if err := e.Validate(); err != nil {
		return "", err
	}

	raw, err := e.CanonicalJSON()
	if err != nil {
		return "", err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", fmt.Errorf(
			"%w: decode composition entry fields: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	header := e.Header()
	if hasCompositionBody(header, fields) {
		return CompositionEntryContained, nil
	}
	if err := validateCompositionReferenceFields(header, fields); err != nil {
		return "", err
	}
	return CompositionEntryReference, nil
}

func (e Entry) ValidateCompositionReference() error {
	form, err := e.CompositionForm()
	if err != nil {
		return err
	}
	if form != CompositionEntryReference {
		header := e.Header()
		return fmt.Errorf(
			"%w: composition entry %s/%s is a contained declaration",
			basespec.ErrInvalid,
			header.Type,
			header.Name,
		)
	}
	return nil
}

func (e Entry) IsCompositionReference() bool {
	form, err := e.CompositionForm()
	return err == nil && form == CompositionEntryReference
}

func hasCompositionBody(
	header Header,
	fields map[string]json.RawMessage,
) bool {
	// Command locators are executable declaration data. They are not source
	// locations for independently declared Tool or MCP Artifacts.
	if header.Locator != nil &&
		header.Locator.Kind == LocatorKindCommand {
		return true
	}

	switch header.Type {
	case TypeInstruction:
		return containsCompositionField(fields, "content", "mediaType")

	case TypeContext:
		return containsCompositionField(
			fields,
			"content",
			"mediaType",
			"include",
			"exclude",
		)

	case TypeTool:
		return containsCompositionField(
			fields,
			"inputSchema",
			"outputSchema",
		)

	case TypeModel:
		return containsCompositionField(fields, "model", "parameters")

	case TypeSkill:
		return containsCompositionField(fields, "license", "allowedTools")

	case TypeMCP:
		// "server" is a selector for an external multi-server document. It
		// does not turn the entry into a contained MCP declaration.
		return containsCompositionField(
			fields,
			"transport",
			"command",
			"args",
			"env",
			"url",
			"headers",
			"include",
		)

	case TypeMCPPolicy:
		return containsCompositionField(fields, "body")

	case TypeCollection:
		return containsCompositionField(fields, "version", "members")

	case TypeAgent, TypeTeam:
		return containsCompositionField(fields, "members", "program")

	case TypeLoop:
		return containsCompositionField(
			fields,
			"body",
			"maxIterations",
			"until",
		)

	case TypeWorkflow:
		return containsCompositionField(fields, "start", "nodes", "edges")

	case TypeWorkspace:
		return containsCompositionField(
			fields,
			"declarations",
			"roots",
		)
	}
	return false
}

func containsCompositionField(
	fields map[string]json.RawMessage,
	names ...string,
) bool {
	for _, name := range names {
		if _, found := fields[name]; found {
			return true
		}
	}
	return false
}

func validateCompositionReferenceFields(
	header Header,
	fields map[string]json.RawMessage,
) error {
	for name := range fields {
		switch name {
		case "$schema",
			"apiVersion",
			"type",
			"name",
			"description",
			"locator",
			"metadata":
			continue

		case "server":
			if header.Type == TypeMCP {
				continue
			}
		}
		return fmt.Errorf(
			"%w: composition reference %s/%s contains unsupported field %q",
			basespec.ErrInvalid,
			header.Type,
			header.Name,
			name,
		)
	}

	serverRaw, hasServer := fields["server"]
	if !hasServer {
		return nil
	}
	if header.Type != TypeMCP || header.Locator == nil {
		return fmt.Errorf(
			"%w: MCP server selector requires an MCP locator reference",
			basespec.ErrInvalid,
		)
	}

	var server string
	if err := json.Unmarshal(serverRaw, &server); err != nil {
		return fmt.Errorf(
			"%w: decode MCP server selector: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := basespec.LogicalName(server).Validate(); err != nil {
		return fmt.Errorf("MCP server selector: %w", err)
	}
	return nil
}
