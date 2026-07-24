package skilladapter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type SkillArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type SkillSummary struct {
	SchemaVersion string                 `json:"schemaVersion"`
	ID            artifactstore.RecordID `json:"id"`
	Slug          string                 `json:"slug"`
	Name          string                 `json:"name"`
	DisplayName   string                 `json:"displayName"`
	Description   string                 `json:"description"`
	Tags          []string               `json:"tags,omitempty"`
	Insert        string                 `json:"insert"`
	Arguments     []SkillArgument        `json:"arguments,omitempty"`
	IsEnabled     bool                   `json:"isEnabled"`
	CreatedAt     time.Time              `json:"createdAt"`
	ModifiedAt    time.Time              `json:"modifiedAt"`
}

type WorkspaceSkill struct {
	RootID           artifactstore.RootID       `json:"rootID"`
	RecordID         artifactstore.RecordID     `json:"recordID"`
	DefinitionDigest artifactstore.Digest       `json:"definitionDigest"`
	SourceID         artifactstore.SourceID     `json:"sourceID"`
	Locator          artifactstore.Locator      `json:"locator"`
	Skill            SkillSummary               `json:"skill"`
	MarkdownBody     string                     `json:"markdownBody,omitempty"`
	RecordRevision   uint64                     `json:"recordRevision"`
	State            record.State               `json:"state"`
	CatalogCurrent   bool                       `json:"catalogCurrent"`
	RuntimeDisabled  bool                       `json:"runtimeDisabled"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
	RuntimeLocation  string                     `json:"-"`
}

type SkillLoadPlan struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Skills          []WorkspaceSkill           `json:"skills"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type Adapter struct {
	query         *engine.QueryService
	runtimePolicy engine.SourceUsePolicy
	sourceRuntime source.Runtime
}

func NewAdapter(
	query *engine.QueryService,
	runtimePolicy engine.SourceUsePolicy,
	sourceRuntime source.Runtime,
) (*Adapter, error) {
	if query == nil || runtimePolicy == nil || sourceRuntime == nil {
		return nil, fmt.Errorf(
			"%w: Workspace Skill adapter dependencies are incomplete",
			engine.ErrInvalidWorkspace,
		)
	}
	return &Adapter{
		query:         query,
		runtimePolicy: runtimePolicy,
		sourceRuntime: sourceRuntime,
	}, nil
}

func (f *Adapter) List(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]WorkspaceSkill, error) {
	view, err := f.query.Catalog(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSkill, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != skillKind ||
			resourceValue.Definition.SchemaID != skillSchemaID {
			continue
		}
		value, err := projectWorkspaceSkill(
			rootID,
			resourceValue,
			false,
		)
		if err != nil {
			value.Diagnostics = artifactstore.AppendDiagnostics(
				value.Diagnostics,
				skillProjectionDiagnostic(resourceValue.Record, err),
			)
		}
		output = append(output, value)
	}
	sortWorkspaceSkills(output)
	return output, nil
}

func (f *Adapter) Load(
	ctx context.Context,
	rootID artifactstore.RootID,
	recordIDs []artifactstore.RecordID,
) (SkillLoadPlan, error) {
	loadPlan, err := f.query.ComposeLoadPlan(ctx, rootID, recordIDs)
	if err != nil {
		return SkillLoadPlan{}, err
	}
	workspaceValue, err := f.query.GetWorkspace(ctx, rootID)
	if err != nil {
		return SkillLoadPlan{}, err
	}

	output := SkillLoadPlan{
		RootID:          rootID,
		CatalogRevision: loadPlan.CatalogRevision,
		Diagnostics:     loadPlan.Diagnostics,
	}
	for _, item := range loadPlan.Items {
		if err := ValidateSkillDefinition(item.Definition); err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				skillProjectionDiagnostic(item.Record, err),
			)
			continue
		}
		decision := f.runtimePolicy.Decide(ctx, engine.RuntimePolicyRequest{
			Use:              engine.RuntimeUseSkill,
			Workspace:        workspaceValue,
			Record:           item.Record,
			DefinitionDigest: item.Definition.Digest,
			SourceID:         item.Source.ID,
		})
		if err := decision.Validate(); err != nil {
			return SkillLoadPlan{}, err
		}
		if decision.Disposition != engine.RuntimeAllowed {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				engine.RuntimeDecisionDiagnostic(decision, item.Record),
			)
			continue
		}
		resourceValue := engine.Resource{
			Record:          item.Record,
			Definition:      item.Definition,
			Source:          item.Source,
			CatalogCurrent:  item.CatalogCurrent,
			ProjectionValid: true,
		}
		projected, err := projectWorkspaceSkill(
			rootID,
			resourceValue,
			true,
		)
		if err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				skillProjectionDiagnostic(item.Record, err),
			)
			continue
		}
		runtimeLocation, err := f.resolveRuntimeLocation(ctx, item)
		if err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				runtimeLocationDiagnostic(item.Record, err),
			)
			continue
		}
		projected.RuntimeLocation = runtimeLocation
		output.Skills = append(output.Skills, projected)
	}
	sortWorkspaceSkills(output.Skills)
	return output, nil
}

func projectWorkspaceSkill(
	rootID artifactstore.RootID,
	resourceValue engine.Resource,
	includeMarkdown bool,
) (WorkspaceSkill, error) {
	runtimeDisabled, dataErr := engine.RecordRuntimeDisabled(resourceValue.Record)
	output := WorkspaceSkill{
		RootID:           rootID,
		RecordID:         resourceValue.Record.ID,
		RecordRevision:   resourceValue.Record.Revision,
		DefinitionDigest: resourceValue.Definition.Digest,
		SourceID:         resourceValue.Source.ID,
		Locator:          resourceValue.Record.Occurrence.Locator,
		State:            resourceValue.Record.State,
		CatalogCurrent:   resourceValue.CatalogCurrent,
		RuntimeDisabled:  runtimeDisabled,
		Diagnostics: artifactstore.AppendDiagnostics(
			resourceValue.Record.Diagnostics,
			resourceValue.Diagnostics...,
		),
	}
	if dataErr != nil {
		return output, dataErr
	}
	if err := ValidateSkillDefinition(resourceValue.Definition); err != nil {
		return output, err
	}
	body, err := engine.DecodeDefinitionBody[skillDefinition](
		resourceValue.Definition.Body,
	)
	if err != nil {
		return output, err
	}
	markdownBody := ""
	if includeMarkdown {
		markdownBody = body.MarkdownBody
	}
	output.Skill = skillSummary(resourceValue.Record, body)
	output.MarkdownBody = markdownBody
	return output, nil
}

func skillSummary(
	recordValue record.Record,
	value skillDefinition,
) SkillSummary {
	arguments := make([]SkillArgument, 0, len(value.Arguments))
	for _, argument := range value.Arguments {
		arguments = append(arguments, SkillArgument(argument))
	}
	return SkillSummary{
		SchemaVersion: workspaceSkillsSchemaVersionV1,

		ID:   recordValue.ID,
		Slug: value.Name,

		Name:        value.Name,
		DisplayName: value.DisplayName,
		Description: value.Description,
		Tags:        append([]string(nil), value.Tags...),
		Insert:      value.Insert,
		Arguments:   arguments,
		IsEnabled:   recordValue.Enabled,
		CreatedAt:   recordValue.CreatedAt,
		ModifiedAt:  recordValue.ModifiedAt,
	}
}

func sortWorkspaceSkills(values []WorkspaceSkill) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Skill.Name != values[right].Skill.Name {
			return values[left].Skill.Name < values[right].Skill.Name
		}
		return values[left].RecordID < values[right].RecordID
	})
}

// resolveRuntimeLocation performs the Workspace-owned handoff from a selected
// source-linked record to a native filesystem skill package. It does not
// register or execute the skill. Agent Skills runtime lifecycle remains in
// skillruntime.
func (f *Adapter) resolveRuntimeLocation(
	ctx context.Context,
	item engine.LoadPlanItem,
) (string, error) {
	if item.Source.Kind != fsdir.Kind {
		return "", fmt.Errorf(
			"%w: Workspace Skill source kind %q has no native filesystem package",
			artifactstore.ErrUnsupported,
			item.Source.Kind,
		)
	}
	if item.SourceContentDigest == "" ||
		item.OccurrenceDefinitionDigest == "" {
		return "", fmt.Errorf(
			"%w: Workspace Skill has no current source occurrence",
			artifactstore.ErrCatalogStale,
		)
	}
	if item.OccurrenceDefinitionDigest != item.Definition.Digest {
		return "", fmt.Errorf(
			"%w: selected Workspace Skill definition does not match its current source occurrence",
			artifactstore.ErrCatalogStale,
		)
	}

	sourceValue, err := f.sourceRuntime.Get(ctx, item.Source.ID)
	if err != nil {
		return "", err
	}
	if sourceValue.Kind != fsdir.Kind {
		return "", fmt.Errorf(
			"%w: Workspace Skill source kind changed from %q to %q",
			artifactstore.ErrConflict,
			fsdir.Kind,
			sourceValue.Kind,
		)
	}

	localPaths, supported := f.sourceRuntime.(source.LocalPathRuntime)
	if !supported {
		return "", fmt.Errorf(
			"%w: source runtime cannot resolve native filesystem paths",
			artifactstore.ErrUnsupported,
		)
	}
	skillMDPath, err := localPaths.ResolveLocalPath(
		ctx,
		sourceValue,
		item.Record.Occurrence.Locator,
	)
	if err != nil {
		return "", err
	}
	if filepath.Base(skillMDPath) != skillDefinitionFileName {
		return "", fmt.Errorf(
			"%w: Workspace Skill locator %q is not %q",
			artifactstore.ErrInvalid,
			item.Record.Occurrence.Locator,
			skillDefinitionFileName,
		)
	}
	if err := verifySkillMDContent(
		skillMDPath,
		item.SourceContentDigest,
	); err != nil {
		return "", err
	}
	return filepath.Dir(skillMDPath), nil
}

// verifySkillMDContent prevents a selected Workspace record from silently
// becoming a runtime handle for different SKILL.md content after refresh.
// Resource and script contents remain normal live filesystem-provider inputs,
// just as they are for installed filesystem skills.
func verifySkillMDContent(
	location string,
	expected artifactstore.Digest,
) error {
	if err := artifactstore.ValidateDigest(expected); err != nil {
		return err
	}
	info, err := os.Lstat(location)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf(
			"%w: Workspace SKILL.md is not a regular non-symlink file",
			artifactstore.ErrInvalid,
		)
	}
	file, err := os.Open(location)
	if err != nil {
		return err
	}
	content, readErr := io.ReadAll(
		io.LimitReader(file, int64(artifactstore.MaxCandidateBytes)+1),
	)
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if len(content) > artifactstore.MaxCandidateBytes {
		return fmt.Errorf(
			"%w: Workspace SKILL.md exceeds runtime verification limit",
			artifactstore.ErrInvalid,
		)
	}
	if artifactstore.DigestBytes(content) != expected {
		return fmt.Errorf(
			"%w: Workspace SKILL.md changed since the current catalog refresh",
			artifactstore.ErrCatalogStale,
		)
	}
	return nil
}

func runtimeLocationDiagnostic(
	value record.Record,
	err error,
) artifactstore.Diagnostic {
	message := err.Error()
	for len(message) > artifactstore.MaxDiagnosticMessageBytes {
		_, size := utf8.DecodeLastRuneInString(message)
		message = message[:len(message)-size]
	}
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     engine.DiagnosticCodeRuntimeUnavailable,
		Message:  message,
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Occurrence.Locator,
			SubresourceLocator: value.Occurrence.SubresourceLocator,
		},
	}
}

func skillProjectionDiagnostic(
	value record.Record,
	err error,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     engine.DiagnosticCodeProjectionInvalid,
		Message:  err.Error(),
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Occurrence.Locator,
			SubresourceLocator: value.Occurrence.SubresourceLocator,
		},
	}
}
