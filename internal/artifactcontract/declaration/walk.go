package declaration

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// NamedEntry is one named declaration reachable from a canonical declaration
// document. The root entry uses an empty SubresourceLocator.
//
// Anonymous inline entries remain in the containing declaration body but do
// not become standalone source-backed Artifacts.
type NamedEntry struct {
	SubresourceLocator basespec.SubresourceLocator
	Entry              Entry

	ImplicitLoopBody bool
}

func (e NamedEntry) Clone() NamedEntry {
	return NamedEntry{
		SubresourceLocator: e.SubresourceLocator,
		Entry:              e.Entry.Clone(),
		ImplicitLoopBody:   e.ImplicitLoopBody,
	}
}

func (e NamedEntry) Validate() error {
	if err := e.SubresourceLocator.Validate(); err != nil {
		return err
	}
	if err := e.Entry.Validate(); err != nil {
		return err
	}
	if e.Entry.Header().Name == "" {
		return fmt.Errorf(
			"%w: named declaration has no name",
			basespec.ErrInvalid,
		)
	}
	if e.ImplicitLoopBody &&
		e.Entry.Header().Type != TypeLoop {
		return fmt.Errorf(
			"%w: only a Loop may use an implicit containing body",
			basespec.ErrInvalid,
		)
	}
	return nil
}

// WalkNamedEntries returns the top-level declaration followed by every named
// inline or located nested declaration in stable structural order.
//
// The walker knows only common structural entry positions. It does not decode
// or validate type-specific bodies. Concrete contract packages remain the
// owner of concrete schema validation.
func WalkNamedEntries(
	root Entry,
) ([]NamedEntry, error) {
	if err := root.Validate(); err != nil {
		return nil, err
	}
	if root.Header().Name == "" {
		return nil, fmt.Errorf(
			"%w: top-level declaration requires name",
			basespec.ErrInvalid,
		)
	}

	output := make([]NamedEntry, 0)
	if err := walkNamedEntry(root, nil, true, false, &output); err != nil {
		return nil, err
	}

	result := make([]NamedEntry, len(output))
	for index, value := range output {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		result[index] = value.Clone()
	}
	return result, nil
}

func walkNamedEntry(
	entry Entry,
	path []string,
	topLevel bool,
	implicitLoopBody bool,
	output *[]NamedEntry,
) error {
	if topLevel {
		*output = append(*output, NamedEntry{
			Entry:            entry.Clone(),
			ImplicitLoopBody: implicitLoopBody,
		})
	} else if entry.Header().Name != "" && !entry.IsSymbolic() {
		subresource, err := subresourceForPath(path)
		if err != nil {
			return err
		}
		*output = append(*output, NamedEntry{
			SubresourceLocator: subresource,
			Entry:              entry.Clone(),
			ImplicitLoopBody:   implicitLoopBody,
		})
	}

	raw, err := entry.CanonicalJSON()
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}

	switch entry.Header().Type {
	case TypeSkill:
		return walkEntryArray(
			fields["allowedTools"],
			appendPath(path, "allowedTools"),
			output,
		)

	case TypeCollection:
		return walkEntryArray(
			fields["members"],
			appendPath(path, "members"),
			output,
		)

	case TypeAgent, TypeTeam:
		if err := walkEntryArray(
			fields["members"],
			appendPath(path, "members"),
			output,
		); err != nil {
			return err
		}
		return walkSingleEntry(
			fields["program"],
			appendPath(path, "program"),
			true,
			output,
		)

	case TypeLoop:
		return walkSingleEntry(
			fields["body"],
			appendPath(path, "body"),
			false,
			output,
		)

	case TypeWorkflow:
		return walkWorkflowTargets(
			fields["nodes"],
			appendPath(path, "nodes"),
			output,
		)

	case TypeWorkspace:
		if err := walkEntryArray(
			fields["roots"],
			appendPath(path, "roots"),
			output,
		); err != nil {
			return err
		}
		return walkWorkspaceDeclarationEntries(
			fields["declarations"],
			appendPath(path, "declarations"),
			output,
		)
	default:
	}
	return nil
}

func walkEntryArray(
	raw json.RawMessage,
	base []string,
	output *[]NamedEntry,
) error {
	if len(raw) == 0 {
		return nil
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return err
	}
	for index, value := range entries {
		entry, err := DecodeCanonicalEntryJSON(value)
		if err != nil {
			return err
		}
		if err := walkNamedEntry(
			entry,
			appendPath(base, strconv.Itoa(index)),
			false,
			false,
			output,
		); err != nil {
			return err
		}
	}
	return nil
}

func walkSingleEntry(
	raw json.RawMessage,
	path []string,
	implicitLoopBody bool,
	output *[]NamedEntry,
) error {
	if len(raw) == 0 {
		return nil
	}
	entry, err := DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return err
	}
	return walkNamedEntry(
		entry,
		path,
		false,
		implicitLoopBody && entry.Header().Type == TypeLoop,
		output,
	)
}

func walkWorkflowTargets(
	raw json.RawMessage,
	base []string,
	output *[]NamedEntry,
) error {
	if len(raw) == 0 {
		return nil
	}
	var nodes []struct {
		Target json.RawMessage `json:"target"`
	}
	if err := json.Unmarshal(raw, &nodes); err != nil {
		return err
	}
	for index, node := range nodes {
		if err := walkSingleEntry(
			node.Target,
			appendPath(
				base,
				strconv.Itoa(index),
				"target",
			),
			false,
			output,
		); err != nil {
			return err
		}
	}
	return nil
}

func walkWorkspaceDeclarationEntries(
	raw json.RawMessage,
	base []string,
	output *[]NamedEntry,
) error {
	if len(raw) == 0 {
		return nil
	}
	var declarations []json.RawMessage
	if err := json.Unmarshal(raw, &declarations); err != nil {
		return err
	}
	for index, rawDeclaration := range declarations {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawDeclaration, &fields); err != nil {
			continue
		}
		if _, found := fields["type"]; !found {
			continue
		}
		entry, err := DecodeCanonicalEntryJSON(rawDeclaration)
		if err != nil {
			return err
		}
		if err := walkNamedEntry(
			entry,
			appendPath(base, strconv.Itoa(index)),
			false,
			false,
			output,
		); err != nil {
			return err
		}
	}
	return nil
}

func appendPath(
	current []string,
	segments ...string,
) []string {
	output := append([]string(nil), current...)
	return append(output, segments...)
}

func subresourceForPath(
	path []string,
) (basespec.SubresourceLocator, error) {
	if len(path) == 0 {
		return "", nil
	}
	value := basespec.SubresourceLocator(path[0])
	for _, segment := range path[1:] {
		value += "/" + basespec.SubresourceLocator(segment)
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	return value, nil
}
