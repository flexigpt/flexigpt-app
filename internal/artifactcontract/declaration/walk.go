package declaration

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// NamedEntry is one concrete standalone declaration reachable from a source
// declaration. The root uses an empty SubresourceLocator. Named external
// members and selectors remain relationships and do not emit Artifacts.
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
	if e.Entry.Header().Name == "" {
		return fmt.Errorf(
			"%w: named declaration entry requires name",
			basespec.ErrInvalid,
		)
	}
	return nil
}

// WalkNamedEntries returns the root declaration and every contained
// declaration in stable semantic order. Array position is not used in
// contained declaration subresource identity.
func WalkNamedEntries(root Entry) ([]NamedEntry, error) {
	if err := root.Validate(); err != nil {
		return nil, err
	}
	if root.Header().Name == "" {
		return nil, fmt.Errorf(
			"%w: declaration root requires name",
			basespec.ErrInvalid,
		)
	}

	output := make([]NamedEntry, 0)
	if err := walkDeclaration(root, nil, 0, &output); err != nil {
		return nil, err
	}

	seen := make(map[basespec.SubresourceLocator]struct{}, len(output))
	result := make([]NamedEntry, len(output))
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

func walkDeclaration(
	entry Entry,
	path []string,
	depth int,
	output *[]NamedEntry,
) error {
	if depth > basespec.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: declaration nesting exceeds depth %d",
			basespec.ErrInvalid,
			basespec.MaxDiscoveryDepth,
		)
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	if entry.Header().Name == "" {
		return fmt.Errorf(
			"%w: concrete declaration requires name",
			basespec.ErrInvalid,
		)
	}

	subresource, err := subresourceForPath(path)
	if err != nil {
		return err
	}
	*output = append(*output, NamedEntry{
		SubresourceLocator: subresource,
		Entry:              entry.Clone(),
	})

	fields, err := entry.fields()
	if err != nil {
		return err
	}

	switch entry.Header().Type {
	case TypeSkill:
		return walkMemberArray(
			fields["allowedTools"],
			appendPath(path, "allowedTools"),
			depth+1,
			output,
		)

	case TypePlugin, TypeWorkspace:
		return walkMemberArray(
			fields["members"],
			appendPath(path, "members"),
			depth+1,
			output,
		)

	case TypeAgent, TypeTeam:
		if err := walkMemberArray(
			fields["members"],
			appendPath(path, "members"),
			depth+1,
			output,
		); err != nil {
			return err
		}
		if err := walkSingleMember(
			fields["loop"],
			appendPath(path, "loop"),
			depth+1,
			output,
		); err != nil {
			return err
		}
		return walkSingleMember(
			fields["workflow"],
			appendPath(path, "workflow"),
			depth+1,
			output,
		)

	case TypeLoop:
		return walkSingleMember(
			fields["body"],
			appendPath(path, "body"),
			depth+1,
			output,
		)

	case TypeWorkflow:
		return walkWorkflowNodes(
			fields["nodes"],
			appendPath(path, "nodes"),
			depth+1,
			output,
		)

	default:
		return nil
	}
}

func walkMemberArray(
	raw json.RawMessage,
	base []string,
	depth int,
	output *[]NamedEntry,
) error {
	if len(raw) == 0 {
		return nil
	}

	var rawMembers []json.RawMessage
	if err := json.Unmarshal(raw, &rawMembers); err != nil {
		return err
	}

	members := make([]Entry, 0, len(rawMembers))
	for index, rawMember := range rawMembers {
		member, err := DecodeCanonicalEntryJSON(rawMember)
		if err != nil {
			return fmt.Errorf("members[%d]: %w", index, err)
		}
		members = append(members, member)
	}
	if err := ValidateMemberUniqueness("members", members); err != nil {
		return err
	}

	ordered, err := SortedMembers("members", members)
	if err != nil {
		return err
	}
	for _, member := range ordered {
		if err := walkMember(member, base, depth, output); err != nil {
			return err
		}
	}
	return nil
}

func walkSingleMember(
	raw json.RawMessage,
	base []string,
	depth int,
	output *[]NamedEntry,
) error {
	if len(raw) == 0 {
		return nil
	}
	member, err := DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return err
	}
	form, err := member.MemberForm()
	if err != nil {
		return err
	}
	if form == MemberSelector {
		return fmt.Errorf(
			"%w: singular member position does not allow selectors",
			basespec.ErrInvalid,
		)
	}
	return walkMember(member, base, depth, output)
}

func walkMember(
	member Entry,
	base []string,
	depth int,
	output *[]NamedEntry,
) error {
	form, err := member.MemberForm()
	if err != nil {
		return err
	}
	if form != MemberContained {
		return nil
	}

	target, err := member.ContainedDeclaration()
	if err != nil {
		return err
	}
	path, err := appendMemberIdentity(base, member)
	if err != nil {
		return err
	}
	return walkDeclaration(target, path, depth, output)
}

func walkWorkflowNodes(
	raw json.RawMessage,
	base []string,
	depth int,
	output *[]NamedEntry,
) error {
	if len(raw) == 0 {
		return nil
	}

	var rawNodes []json.RawMessage
	if err := json.Unmarshal(raw, &rawNodes); err != nil {
		return err
	}

	type workflowNode struct {
		id     string
		member Entry
	}
	nodes := make([]workflowNode, 0, len(rawNodes))
	seen := make(map[string]struct{}, len(rawNodes))

	for index, rawNode := range rawNodes {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawNode, &fields); err != nil {
			return fmt.Errorf("workflow nodes[%d]: %w", index, err)
		}

		var id string
		if err := json.Unmarshal(fields["id"], &id); err != nil {
			return fmt.Errorf("workflow nodes[%d].id: %w", index, err)
		}
		if err := ValidateWorkflowID("Workflow node ID", id); err != nil {
			return fmt.Errorf("workflow nodes[%d]: %w", index, err)
		}
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf(
				"%w: duplicate Workflow node ID %q",
				basespec.ErrIdentityConflict,
				id,
			)
		}
		seen[id] = struct{}{}

		delete(fields, "id")
		delete(fields, "join")
		memberRaw, err := jsonutil.MarshalCanonicalObject(
			fields,
			basespec.MaxDefinitionBodyBytes,
		)
		if err != nil {
			return err
		}
		member, err := DecodeCanonicalEntryJSON(memberRaw)
		if err != nil {
			return fmt.Errorf("workflow nodes[%d]: %w", index, err)
		}
		form, err := member.MemberForm()
		if err != nil {
			return fmt.Errorf("workflow nodes[%d]: %w", index, err)
		}
		if form == MemberSelector {
			return fmt.Errorf(
				"%w: Workflow node %q cannot contain a selector",
				basespec.ErrInvalid,
				id,
			)
		}
		nodes = append(nodes, workflowNode{
			id:     id,
			member: member,
		})
	}

	sort.SliceStable(nodes, func(left, right int) bool {
		return nodes[left].id < nodes[right].id
	})
	for _, node := range nodes {
		if err := walkMember(
			node.member,
			appendPath(base, StableWorkflowNodeSegment(node.id)),
			depth,
			output,
		); err != nil {
			return err
		}
	}
	return nil
}

func appendMemberIdentity(
	current []string,
	member Entry,
) ([]string, error) {
	header := member.Header()
	if header.Type == TypeText {
		insert, err := member.TextInsert()
		if err != nil {
			return nil, err
		}
		return appendPath(
			current,
			string(header.Type),
			string(insert),
			header.Name,
		), nil
	}
	return appendPath(current, string(header.Type), header.Name), nil
}

// StableWorkflowNodeSegment uses the node ID when it is portable. Other
// valid workflow IDs remain deterministic without becoming path traversal.
func StableWorkflowNodeSegment(value string) string {
	if basespec.ValidatePortableName("Workflow node ID", value) == nil {
		return value
	}
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(value))),
		cryptoutil.DigestSHA256Prefix,
	)
	return "id-" + digest
}

func appendPath(current []string, segments ...string) []string {
	output := append([]string(nil), current...)
	return append(output, segments...)
}

func subresourceForPath(
	path []string,
) (basespec.SubresourceLocator, error) {
	if len(path) == 0 {
		return "", nil
	}
	value := basespec.SubresourceLocator(strings.Join(path, "/"))
	if err := value.Validate(); err != nil {
		return "", err
	}
	return value, nil
}
