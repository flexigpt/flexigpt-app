package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

const workspaceIdentityPrefix = "workspace/"

type workspaceAgentProvider struct {
	adapter *skilladapter.Adapter
}

func newWorkspaceAgentProvider(
	adapter *skilladapter.Adapter,
) *workspaceAgentProvider {
	return &workspaceAgentProvider{adapter: adapter}
}

func (*workspaceAgentProvider) Type() string {
	return workspaceSkillProviderType
}

func (p *workspaceAgentProvider) Index(
	ctx context.Context,
	definition agentskillsSpec.SkillDef,
) (agentskillsSpec.ProviderSkillIndexRecord, error) {
	if p == nil || p.adapter == nil {
		return agentskillsSpec.ProviderSkillIndexRecord{}, errors.New(
			"Workspace Skill provider is not configured",
		)
	}
	if definition.Type != workspaceSkillProviderType {
		return agentskillsSpec.ProviderSkillIndexRecord{}, fmt.Errorf(
			"%w: unsupported provider type %q",
			agentskillsSpec.ErrInvalidArgument,
			definition.Type,
		)
	}

	rootID, recordID, digest, err := parseWorkspaceRuntimeLocation(
		definition.Location,
	)
	if err != nil {
		return agentskillsSpec.ProviderSkillIndexRecord{}, err
	}
	value, err := p.load(ctx, rootID, recordID, digest)
	if err != nil {
		return agentskillsSpec.ProviderSkillIndexRecord{}, err
	}
	if value.Skill.Name != definition.Name {
		return agentskillsSpec.ProviderSkillIndexRecord{}, fmt.Errorf(
			"%w: definition name %q does not match Workspace Skill name %q",
			agentskillsSpec.ErrInvalidArgument,
			definition.Name,
			value.Skill.Name,
		)
	}

	document := value.AgentSkillDocument()
	if err := agentskills.ValidateSkillDocument(document); err != nil {
		return agentskillsSpec.ProviderSkillIndexRecord{}, err
	}

	return agentskillsSpec.ProviderSkillIndexRecord{
		Key:         agentskillsSpec.ProviderSkillKey(definition),
		Name:        document.Name,
		Description: document.Description,
		DisplayName: document.DisplayName,
		Insert:      document.Insert,
		Arguments: append(
			[]agentskillsSpec.SkillArgument(nil),
			document.Arguments...,
		),
		Tags:      append([]string(nil), document.Tags...),
		Digest:    string(value.DefinitionDigest),
		SkillBody: document.MarkdownBody,
		Warnings:  diagnosticWarnings(value.Diagnostics),
	}, nil
}

func (p *workspaceAgentProvider) LoadBody(
	ctx context.Context,
	key agentskillsSpec.ProviderSkillKey,
) (string, error) {
	if key.Type != workspaceSkillProviderType {
		return "", fmt.Errorf(
			"%w: unsupported provider type %q",
			agentskillsSpec.ErrInvalidArgument,
			key.Type,
		)
	}
	rootID, recordID, digest, err := parseWorkspaceRuntimeLocation(
		key.Location,
	)
	if err != nil {
		return "", err
	}
	value, err := p.load(ctx, rootID, recordID, digest)
	if err != nil {
		return "", err
	}
	if value.Skill.Name != key.Name {
		return "", agentskillsSpec.ErrSkillNotFound
	}
	return value.MarkdownBody, nil
}

func (*workspaceAgentProvider) ReadResource(
	_ context.Context,
	_ agentskillsSpec.ProviderSkillKey,
	_ string,
	_ agentskillsSpec.ReadResourceEncoding,
) ([]llmtoolsSpec.ToolOutputUnion, error) {
	return nil, fmt.Errorf(
		"%w: Workspace Skills do not expose materialized resources",
		agentskillsSpec.ErrInvalidArgument,
	)
}

func (*workspaceAgentProvider) RunScript(
	_ context.Context,
	_ agentskillsSpec.ProviderSkillKey,
	_ string,
	_ []string,
	_ map[string]string,
	_ string,
) (agentskillsSpec.RunScriptOut, error) {
	return agentskillsSpec.RunScriptOut{},
		agentskillsSpec.ErrRunScriptUnsupported
}

func (p *workspaceAgentProvider) load(
	ctx context.Context,
	rootID artifactstore.RootID,
	recordID artifactstore.RecordID,
	digest artifactstore.Digest,
) (skilladapter.WorkspaceSkill, error) {
	plan, err := p.adapter.Load(
		ctx,
		rootID,
		[]artifactstore.RecordID{recordID},
	)
	if err != nil {
		return skilladapter.WorkspaceSkill{}, err
	}
	if len(plan.Skills) != 1 {
		return skilladapter.WorkspaceSkill{}, fmt.Errorf(
			"%w: Workspace Skill %q is unavailable",
			agentskillsSpec.ErrSkillNotFound,
			recordID,
		)
	}
	value := plan.Skills[0]
	if value.DefinitionDigest != digest {
		return skilladapter.WorkspaceSkill{}, fmt.Errorf(
			"%w: Workspace Skill definition is stale",
			agentskillsSpec.ErrSkillNotFound,
		)
	}
	return value, nil
}

func workspaceRuntimeLocation(
	value skilladapter.WorkspaceSkill,
) string {
	return workspaceIdentity(value.RootID, value.RecordID) +
		"/" + string(value.DefinitionDigest)
}

func parseWorkspaceRuntimeLocation(
	value string,
) (
	artifactstore.RootID,
	artifactstore.RecordID,
	artifactstore.Digest,
	error,
) {
	relative, found := strings.CutPrefix(value, workspaceIdentityPrefix)
	if !found {
		return "", "", "", errors.New(
			"runtime location is not a Workspace Skill",
		)
	}
	parts := strings.Split(relative, "/")
	if len(parts) != 3 || strings.TrimSpace(parts[2]) == "" {
		return "", "", "", errors.New(
			"Workspace Skill runtime location is invalid",
		)
	}
	rootID := artifactstore.RootID(parts[0])
	recordID := artifactstore.RecordID(parts[1])
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return "", "", "", err
	}
	if err := artifactstore.ValidateRecordID(recordID); err != nil {
		return "", "", "", err
	}
	return rootID, recordID, artifactstore.Digest(parts[2]), nil
}

func workspaceRuntimeDefinition(
	value skilladapter.WorkspaceSkill,
) agentskillsSpec.SkillDef {
	return agentskillsSpec.SkillDef{
		Type:     workspaceSkillProviderType,
		Name:     value.Skill.Name,
		Location: workspaceRuntimeLocation(value),
	}
}

func diagnosticWarnings(
	values []artifactstore.Diagnostic,
) []string {
	var output []string
	for _, value := range values {
		if value.Severity == artifactstore.DiagnosticError ||
			strings.TrimSpace(value.Message) == "" {
			continue
		}
		output = append(output, value.Message)
	}
	return output
}
