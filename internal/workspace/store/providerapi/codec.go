package providerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/loopv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/teamv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workflowv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type documentCanonicalizer func([]byte) ([]byte, error)

type declarationCodec struct {
	key          schema.Key
	schema       []byte
	canonicalize documentCanonicalizer
}

func NewSchemaCodecs() []providerapi.SchemaCodec {
	return []providerapi.SchemaCodec{
		declarationCodec{
			key:          instructionv1.InstructionSchemaKey,
			schema:       instructionv1.InstructionJSONSchema(),
			canonicalize: canonicalInstruction,
		},
		declarationCodec{
			key:          contextv1.ContextSchemaKey,
			schema:       contextv1.ContextJSONSchema(),
			canonicalize: canonicalContext,
		},
		declarationCodec{
			key:          toolv1.ToolSchemaKey,
			schema:       toolv1.ToolJSONSchema(),
			canonicalize: canonicalTool,
		},
		declarationCodec{
			key:          modelv1.ModelSchemaKey,
			schema:       modelv1.ModelJSONSchema(),
			canonicalize: canonicalModel,
		},
		declarationCodec{
			key:          collectionv1.CollectionSchemaKey,
			schema:       collectionv1.CollectionJSONSchema(),
			canonicalize: canonicalCollection,
		},
		declarationCodec{
			key:          agentv1.AgentSchemaKey,
			schema:       agentv1.AgentJSONSchema(),
			canonicalize: canonicalAgent,
		},
		declarationCodec{
			key:          teamv1.TeamSchemaKey,
			schema:       teamv1.TeamJSONSchema(),
			canonicalize: canonicalTeam,
		},
		declarationCodec{
			key:          loopv1.LoopSchemaKey,
			schema:       loopv1.LoopJSONSchema(),
			canonicalize: canonicalLoop,
		},
		declarationCodec{
			key:          workflowv1.WorkflowSchemaKey,
			schema:       workflowv1.WorkflowJSONSchema(),
			canonicalize: canonicalWorkflow,
		},
		declarationCodec{
			key:          workspacev1.WorkspaceSchemaKey,
			schema:       workspacev1.WorkspaceJSONSchema(),
			canonicalize: canonicalWorkspace,
		},
	}
}

func (c declarationCodec) Key() schema.Key {
	return c.key
}

func (c declarationCodec) JSONSchema() []byte {
	return append([]byte(nil), c.schema...)
}

func (c declarationCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	if ctx == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: canonical declaration codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schema.ParsedDocument{}, err
	}
	if c.canonicalize == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: canonical declaration codec has no canonicalizer",
			basespec.ErrInvalid,
		)
	}
	canonical, err := c.canonicalize(raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	return schema.ParsedDocument{
		Key:    c.key,
		Digest: cryptoutil.DigestBytes(canonical),
		Raw:    canonical,
	}, nil
}

func canonicalInstruction(raw []byte) ([]byte, error) {
	value, err := instructionv1.DecodeInstructionJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalContext(raw []byte) ([]byte, error) {
	value, err := contextv1.DecodeContextJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalTool(raw []byte) ([]byte, error) {
	value, err := toolv1.DecodeToolJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalModel(raw []byte) ([]byte, error) {
	value, err := modelv1.DecodeModelJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalCollection(raw []byte) ([]byte, error) {
	value, err := collectionv1.DecodeCollectionJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalAgent(raw []byte) ([]byte, error) {
	value, err := agentv1.DecodeAgentJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalTeam(raw []byte) ([]byte, error) {
	value, err := teamv1.DecodeTeamJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalLoop(raw []byte) ([]byte, error) {
	value, err := loopv1.DecodeLoopJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalWorkflow(raw []byte) ([]byte, error) {
	value, err := workflowv1.DecodeWorkflowJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}

func canonicalWorkspace(raw []byte) ([]byte, error) {
	value, err := workspacev1.DecodeWorkspaceJSON(raw)
	if err != nil {
		return nil, err
	}
	return value.CanonicalJSON()
}
