package declaration

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// NamedEntry is one named declaration reachable from a canonical declaration
// document. The root entry uses an empty SubresourceLocator.
//
// Composition references, including locator-bearing references, remain edges
// and do not become standalone source-backed Artifacts. A body-less Loop
// nested directly under an Agent or Team program remains contextual because
// its body is the containing owner.
type NamedEntry struct {
	SubresourceLocator basespec.SubresourceLocator
	Entry              Entry
}

func (e NamedEntry) Clone() NamedEntry {
	return NamedEntry{
		SubresourceLocator: e.SubresourceLocator,
		Entry:              e.Entry.Clone(),
	}
}

func (e NamedEntry) Validate() error {
	if err := e.SubresourceLocator.Validate(); err != nil {
		return err
	}
	if err := e.Entry.Validate(); err != nil {
		return err
	}
	return nil
}

// WalkNamedEntries returns the top-level declaration followed by every named
// inline or located nested declaration in stable structural order.
//
// The walker knows only common structural entry positions. It does not decode
// type-specific bodies. Concrete contract packages remain the owner of
// concrete schema validation.
func WalkNamedEntries(
	root Entry,
) ([]NamedEntry, error) {
	if err := root.Validate(); err != nil {
		return nil, err
	}

	output := make([]NamedEntry, 0)
	if err := walkNamedEntry(root, nil, true, false, false, &output); err != nil {
		return nil, err
	}

	result := make([]NamedEntry, len(output))
	seen := make(map[basespec.SubresourceLocator]struct{}, len(output))
	for index, value := range output {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[value.SubresourceLocator]; duplicate {
			return nil, fmt.Errorf(
				"%w: declaration emits duplicate named subresource %q",
				basespec.ErrIdentityConflict,
				value.SubresourceLocator,
			)
		}
		seen[value.SubresourceLocator] = struct{}{}
		result[index] = value.Clone()
	}
	return result, nil
}

func walkNamedEntry(
	entry Entry,
	path []string,
	topLevel bool,
	implicitLoopBody bool,
	compositionEntry bool,
	output *[]NamedEntry,
) error {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return err
	}
	if !topLevel && compositionEntry {
		form, err := entry.CompositionForm()
		if err != nil {
			return err
		}
		if form == CompositionEntryReference {
			return nil
		}
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}

	_, hasLoopBody := fields["body"]
	contextualBodylessLoop := implicitLoopBody &&
		entry.Header().Type == TypeLoop &&
		!hasLoopBody

	// A body-less Loop nested under Agent.program or Team.program uses its
	// containing Agent or Team as its body. It is graph-local and cannot be
	// independently resolved, so it must not become a standalone Artifact.
	if topLevel {
		*output = append(*output, NamedEntry{
			Entry: entry.Clone(),
		})
	} else if !entry.IsSymbolic() &&
		!entry.IsDeclarationLocatorReference() {
		if !contextualBodylessLoop {
			subresource, err := subresourceForPath(path)
			if err != nil {
				return err
			}
			*output = append(*output, NamedEntry{
				SubresourceLocator: subresource,
				Entry:              entry.Clone(),
			})
		}
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
	decoded := make([]Entry, 0, len(entries))
	for _, value := range entries {
		entry, err := DecodeCanonicalEntryJSON(value)
		if err != nil {
			return err
		}
		decoded = append(decoded, entry)
	}
	ordered, err := SortedCompositionEntries(
		"composition entries",
		decoded,
	)
	if err != nil {
		return err
	}
	for _, entry := range ordered {
		if err := walkNamedEntry(
			entry,
			appendEntryPath(base, entry),
			false,
			false,
			true,
			output,
		); err != nil {
			return err
		}
	}
	return nil
}

func walkSingleEntry(
	raw json.RawMessage,
	base []string,
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
		appendEntryPath(base, entry),
		false,
		implicitLoopBody && entry.Header().Type == TypeLoop,
		true,
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
		ID     string          `json:"id"`
		Target json.RawMessage `json:"target"`
	}
	if err := json.Unmarshal(raw, &nodes); err != nil {
		return err
	}
	sort.SliceStable(nodes, func(left, right int) bool {
		return nodes[left].ID < nodes[right].ID
	})
	for _, node := range nodes {
		if err := walkSingleEntry(
			node.Target,
			appendPath(
				base,
				stableWorkflowNodeSegment(node.ID),
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
	for _, rawDeclaration := range declarations {
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
			appendEntryPath(base, entry),
			false,
			false,
			false,
			output,
		); err != nil {
			return err
		}
	}
	return nil
}

func appendEntryPath(
	current []string,
	entry Entry,
) []string {
	header := entry.Header()
	return appendPath(
		current,
		string(header.Type),
		header.Name,
	)
}

func stableWorkflowNodeSegment(value string) string {
	if basespec.ValidatePortableName(
		"Workflow node ID",
		value,
	) == nil {
		return value
	}
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(value))),
		cryptoutil.DigestSHA256Prefix,
	)
	return "id-" + digest
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
