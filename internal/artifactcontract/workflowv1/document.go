package workflowv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkflowType          = artifactcontract.TypeWorkflow
	WorkflowSchemaID      = "artifact.workflow.v1"
	WorkflowSchemaVersion = artifactcontract.APIVersionV1
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
	ID     string                 `json:"id"`
	Target artifactcontract.Entry `json:"target"`
	Join   Join                   `json:"join,omitempty"`
}

type Edge struct {
	From  string                        `json:"from"`
	To    string                        `json:"to"`
	Match *artifactcontract.OutputMatch `json:"match,omitempty"`
}

type WorkflowDocument struct {
	artifactcontract.Header

	Start []string `json:"start,omitempty"`
	Nodes []Node   `json:"nodes,omitempty"`
	Edges []Edge   `json:"edges,omitempty"`
}

func WorkflowJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeWorkflowJSON(raw []byte) (WorkflowDocument, error) {
	return decodeWorkflow(raw, true)
}

func DecodeWorkflowEntry(
	entry artifactcontract.Entry,
) (WorkflowDocument, error) {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return WorkflowDocument{}, err
	}
	return decodeWorkflow(raw, false)
}

func decodeWorkflow(
	raw []byte,
	requireName bool,
) (WorkflowDocument, error) {
	var value WorkflowDocument
	if err := artifactcontract.DecodeDocumentInto(
		raw,
		compiledWorkflowSchema,
		&value,
	); err != nil {
		return WorkflowDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
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
	return artifactcontract.CanonicalDocumentJSON(v)
}

func (v WorkflowDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return artifactcontract.DocumentDigest(v)
}

func (v WorkflowDocument) Validate() error {
	return v.validate(true)
}

func (v WorkflowDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v WorkflowDocument) validate(requireName bool) error {
	if err := artifactcontract.ValidateDocument(
		compiledWorkflowSchema,
		v,
	); err != nil {
		return fmt.Errorf("workflow schema: %w", err)
	}
	if err := v.Header.Validate(artifactcontract.HeaderValidation{
		ExpectedType: WorkflowType,
		APIVersion:   WorkflowSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if len(v.Start) > basespec.MaxDefinitionDependencies ||
		len(v.Nodes) > basespec.MaxDefinitionDependencies ||
		len(v.Edges) > basespec.MaxDiscoveryEntries {
		return fmt.Errorf(
			"%w: workflow exceeds structural entry limits",
			basespec.ErrInvalid,
		)
	}

	nodes := make(map[string]struct{}, len(v.Nodes))
	for index, node := range v.Nodes {
		if err := artifactcontract.ValidateWorkflowID(
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
		if err := node.Target.Validate(); err != nil {
			return fmt.Errorf(
				"workflow nodes[%d] target: %w",
				index,
				err,
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

	seenStart := make(map[string]struct{}, len(v.Start))
	for index, start := range v.Start {
		if _, found := nodes[start]; !found {
			return fmt.Errorf(
				"%w: workflow start[%d] identifies unknown node %q",
				basespec.ErrInvalid,
				index,
				start,
			)
		}
		if _, duplicate := seenStart[start]; duplicate {
			return fmt.Errorf(
				"%w: workflow start repeats node %q",
				basespec.ErrInvalid,
				start,
			)
		}
		seenStart[start] = struct{}{}
	}

	for index, edge := range v.Edges {
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
			if err := artifactcontract.ValidateOutputMatch(
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
