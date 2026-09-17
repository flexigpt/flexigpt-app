package workflowv1

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkflowType          = declaration.TypeWorkflow
	WorkflowSchemaID      = "artifact.workflow.v1"
	WorkflowSchemaVersion = declaration.SchemaVersionV1
)

type Join string

const (
	JoinAll Join = "all"
	JoinAny Join = "any"
)

//go:embed workflow-v1.schema.json
var schemaJSON []byte

var compiledWorkflowSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var WorkflowSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(WorkflowType),
	schema.SchemaID(WorkflowSchemaID),
	WorkflowSchemaVersion,
)

type Node struct {
	ID     string            `json:"-"`
	Join   Join              `json:"-"`
	Member declaration.Entry `json:"-"`
}

func (n Node) MarshalJSON() ([]byte, error) {
	raw, err := n.Member.CanonicalJSON()
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	fields["id"], _ = json.Marshal(n.ID)
	if n.Join != "" {
		fields["join"], _ = json.Marshal(n.Join)
	}
	return jsonutil.MarshalCanonicalObject(
		fields,
		basespec.MaxDefinitionBodyBytes,
	)
}

func (n *Node) UnmarshalJSON(raw []byte) error {
	if n == nil {
		return fmt.Errorf("%w: Workflow node target is nil", basespec.ErrInvalid)
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &fields); err != nil {
		return err
	}
	if err := json.Unmarshal(fields["id"], &n.ID); err != nil {
		return err
	}
	if joinRaw, found := fields["join"]; found {
		if err := json.Unmarshal(joinRaw, &n.Join); err != nil {
			return err
		}
	}
	delete(fields, "id")
	delete(fields, "join")
	memberRaw, err := jsonutil.MarshalCanonicalObject(
		fields,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return err
	}
	n.Member, err = declaration.DecodeCanonicalEntryJSON(memberRaw)
	return err
}

type Edge struct {
	From  string                   `json:"from"`
	To    string                   `json:"to"`
	Match *declaration.OutputMatch `json:"match,omitempty"`
}

type WorkflowDocument struct {
	declaration.Header

	Start []string `json:"start,omitempty"`
	Nodes []Node   `json:"nodes,omitempty"`
	Edges []Edge   `json:"edges,omitempty"`
}

func WorkflowJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeWorkflowJSON(raw []byte) (WorkflowDocument, error) {
	return decodeWorkflow(raw)
}

func DecodeWorkflowEntry(
	entry declaration.Entry,
) (WorkflowDocument, error) {
	var value WorkflowDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledWorkflowSchema,
		&value,
	); err != nil {
		return WorkflowDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return WorkflowDocument{}, err
	}
	return value, nil
}

func decodeWorkflow(
	raw []byte,
) (WorkflowDocument, error) {
	var value WorkflowDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledWorkflowSchema,
		&value,
	); err != nil {
		return WorkflowDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return WorkflowDocument{}, err
	}
	return value, nil
}

func (v WorkflowDocument) Clone() (WorkflowDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v WorkflowDocument) Canonicalize() (
	WorkflowDocument,
	error,
) {
	return v.Clone()
}

func (v WorkflowDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v WorkflowDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v WorkflowDocument) Validate() error {
	return v.validate()
}

func (v WorkflowDocument) ValidateEntry() error {
	return v.validate()
}

func (v WorkflowDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledWorkflowSchema,
		v,
	); err != nil {
		return fmt.Errorf("workflow schema: %w", err)
	}
	return v.validateFields()
}

func (v WorkflowDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: WorkflowType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"Workflow",
		v.Locator,
		v.Start != nil ||
			v.Nodes != nil ||
			v.Edges != nil,
	); err != nil {
		return err
	}

	nodes := make(map[string]struct{}, len(v.Nodes))
	for index, node := range v.Nodes {
		if err := declaration.ValidateWorkflowID(
			"Workflow node ID",
			node.ID,
		); err != nil {
			return fmt.Errorf("workflow nodes[%d]: %w", index, err)
		}
		if _, duplicate := nodes[node.ID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate Workflow node ID %q",
				basespec.ErrInvalid,
				node.ID,
			)
		}
		nodes[node.ID] = struct{}{}
		form, err := node.Member.MemberForm()
		if err != nil {
			return fmt.Errorf(
				"workflow nodes[%d] member: %w",
				index,
				err,
			)
		}
		if form == declaration.MemberSelector {
			return fmt.Errorf(
				"%w: Workflow node %q cannot contain a member selector",
				basespec.ErrInvalid,
				node.ID,
			)
		}
		switch node.Join {
		case "", JoinAll, JoinAny:
		default:
			return fmt.Errorf(
				"%w: workflow node %q has invalid join %q",
				basespec.ErrInvalid,
				node.ID,
				node.Join,
			)
		}
	}

	starts := make(map[string]int, len(v.Start))
	for index, start := range v.Start {
		if previous, duplicate := starts[start]; duplicate {
			return fmt.Errorf(
				"%w: Workflow start[%d] duplicates start[%d]",
				basespec.ErrIdentityConflict,
				index,
				previous,
			)
		}
		starts[start] = index
		if _, found := nodes[start]; !found {
			return fmt.Errorf(
				"%w: workflow start[%d] identifies unknown node %q",
				basespec.ErrInvalid,
				index,
				start,
			)
		}
	}

	edges := make(map[string]int, len(v.Edges))
	for index, edge := range v.Edges {
		match := ""
		if edge.Match != nil {
			raw, err := declaration.CanonicalDocumentJSON(edge.Match)
			if err != nil {
				return err
			}
			match = string(raw)
		}
		key := edge.From + "\x00" + edge.To + "\x00" + match
		if previous, duplicate := edges[key]; duplicate {
			return fmt.Errorf(
				"%w: Workflow edges[%d] duplicates edges[%d]",
				basespec.ErrIdentityConflict,
				index,
				previous,
			)
		}
		edges[key] = index
		if _, found := nodes[edge.From]; !found {
			return fmt.Errorf(
				"%w: workflow edges[%d].from identifies unknown node %q",
				basespec.ErrInvalid,
				index,
				edge.From,
			)
		}
		if _, found := nodes[edge.To]; !found {
			return fmt.Errorf(
				"%w: workflow edges[%d].to identifies unknown node %q",
				basespec.ErrInvalid,
				index,
				edge.To,
			)
		}
		if edge.Match != nil {
			if err := declaration.ValidateOutputMatch(
				"Workflow edge match",
				*edge.Match,
			); err != nil {
				return fmt.Errorf(
					"workflow edges[%d]: %w",
					index,
					err,
				)
			}
		}
	}
	return nil
}
