package skilladapter

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type SkillArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type SkillSummary struct {
	SchemaVersion string                   `json:"schemaVersion"`
	ID            artifactstore.ArtifactID `json:"id"`
	Slug          string                   `json:"slug"`
	Name          string                   `json:"name"`
	DisplayName   string                   `json:"displayName"`
	Description   string                   `json:"description"`
	Tags          []string                 `json:"tags,omitempty"`
	Insert        string                   `json:"insert"`
	Arguments     []SkillArgument          `json:"arguments,omitempty"`
	IsEnabled     bool                     `json:"isEnabled"`
	CreatedAt     time.Time                `json:"createdAt"`
	ModifiedAt    time.Time                `json:"modifiedAt"`
}

type WorkspaceSkill struct {
	Workspace        artifactstore.CollectionRef `json:"workspace"`
	Artifact         artifactstore.ArtifactRef   `json:"artifact"`
	DefinitionDigest artifactstore.Digest        `json:"definitionDigest"`
	SourceID         artifactstore.SourceID      `json:"sourceID"`
	Locator          artifactstore.Locator       `json:"locator"`
	Skill            SkillSummary                `json:"skill"`
	MarkdownBody     string                      `json:"markdownBody,omitempty"`
	ArtifactRevision uint64                      `json:"artifactRevision"`
	State            artifact.State              `json:"state"`
	CatalogCurrent   bool                        `json:"catalogCurrent"`
	WorkspaceEnabled bool                        `json:"-"`
	RuntimeDisabled  bool                        `json:"runtimeDisabled"`
	Diagnostics      []artifactstore.Diagnostic  `json:"diagnostics,omitempty"`

	ProjectionValid     bool                 `json:"-"`
	RuntimePathBacked   bool                 `json:"-"`
	SourceContentDigest artifactstore.Digest `json:"-"`
	SourceGeneration    string               `json:"-"`
	RuntimeLocation     string               `json:"-"`
}

type SkillLoadPlan struct {
	Workspace       artifactstore.CollectionRef `json:"workspace"`
	CatalogRevision uint64                      `json:"catalogRevision"`
	Skills          []WorkspaceSkill            `json:"skills"`
	Diagnostics     []artifactstore.Diagnostic  `json:"diagnostics,omitempty"`
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
	workspace artifactstore.CollectionRef,
) ([]WorkspaceSkill, error) {
	if err := workspace.Validate(); err != nil {
		return nil, err
	}
	view, err := f.query.Catalog(ctx, workspace)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSkill, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != skillartifact.Kind ||
			resourceValue.Definition.SchemaID != skillartifact.SchemaID {
			continue
		}
		value, err := projectWorkspaceSkill(
			workspace,
			resourceValue,
			view.Workspace.Collection.Enabled,
			false,
			f.supportsRuntimePath(resourceValue.Source.Kind),
		)
		if err != nil {
			value.Diagnostics = artifactstore.AppendDiagnostics(
				value.Diagnostics,
				skillProjectionDiagnostic(resourceValue.Artifact, err),
			)
		}
		output = append(output, value)
	}
	sortWorkspaceSkills(output)
	return output, nil
}

func (f *Adapter) Load(
	ctx context.Context,
	workspace artifactstore.CollectionRef,
	artifactRefs []artifactstore.ArtifactRef,
) (SkillLoadPlan, error) {
	if err := workspace.Validate(); err != nil {
		return SkillLoadPlan{}, err
	}
	loadPlan, err := f.query.ComposeLoadPlan(
		ctx,
		workspace,
		artifactRefs,
	)
	if err != nil {
		return SkillLoadPlan{}, err
	}
	workspaceValue, err := f.query.GetWorkspace(ctx, workspace)
	if err != nil {
		return SkillLoadPlan{}, err
	}

	output := SkillLoadPlan{
		Workspace:       workspace,
		CatalogRevision: loadPlan.CatalogRevision,
		Diagnostics:     artifactstore.CloneDiagnostics(loadPlan.Diagnostics),
	}
	verifiedSources := make(
		map[artifactstore.SourceID]source.Source,
		len(loadPlan.Items),
	)
	sourceFailures := make(map[artifactstore.SourceID]error)

	for _, item := range loadPlan.Items {
		if err := skillartifact.ValidateDefinition(item.Definition); err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				skillProjectionDiagnostic(item.Artifact, err),
			)
			continue
		}
		decision := f.runtimePolicy.Decide(ctx, engine.RuntimePolicyRequest{
			Use:              engine.RuntimeUseSkill,
			Workspace:        workspaceValue,
			Artifact:         item.Artifact,
			DefinitionDigest: item.Definition.Digest,
			SourceID:         item.Source.ID,
		})
		if err := decision.Validate(); err != nil {
			return SkillLoadPlan{}, err
		}
		if decision.Disposition != engine.RuntimeAllowed {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				engine.RuntimeDecisionDiagnostic(decision, item.Artifact),
			)
			continue
		}
		resourceValue := engine.Resource{
			Artifact: item.Artifact,

			Definition:      item.Definition,
			Source:          item.Source,
			CatalogCurrent:  item.CatalogCurrent,
			ProjectionValid: true,
		}
		projected, err := projectWorkspaceSkill(
			workspace,
			resourceValue,
			workspaceValue.Collection.Enabled,
			true,
			f.supportsRuntimePath(item.Source.Kind),
		)
		if err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				skillProjectionDiagnostic(item.Artifact, err),
			)
			continue
		}
		projected.SourceContentDigest = item.SourceContentDigest
		projected.SourceGeneration = item.SourceGeneration
		sourceValue, err := f.runtimeSource(
			ctx,
			item,
			verifiedSources,
			sourceFailures,
		)
		if err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				runtimeLocationDiagnostic(item.Artifact, err),
			)
			continue
		}
		runtimeLocation, err := f.resolveRuntimeLocation(
			ctx,
			item,
			sourceValue,
		)
		if err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				runtimeLocationDiagnostic(item.Artifact, err),
			)
			continue
		}
		projected.RuntimeLocation = runtimeLocation
		output.Skills = append(output.Skills, projected)
	}
	sortWorkspaceSkills(output.Skills)
	return output, nil
}

func (f *Adapter) LoadArtifact(
	ctx context.Context,
	ref artifactstore.ArtifactRef,
) (WorkspaceSkill, error) {
	workspaceValue, resourceValue, err := f.query.ResolveArtifact(ctx, ref)
	if err != nil {
		return WorkspaceSkill{}, err
	}
	workspace := workspaceValue.Collection.Ref()
	if resourceValue.Definition.Kind != skillartifact.Kind ||
		resourceValue.Definition.SchemaID != skillartifact.SchemaID {
		return WorkspaceSkill{}, fmt.Errorf(
			"%w: Artifact %q is not an Agent Skill",
			engine.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	plan, err := f.Load(ctx, workspace, []artifactstore.ArtifactRef{ref})
	if err != nil {
		return WorkspaceSkill{}, err
	}
	if len(plan.Skills) != 1 {
		return WorkspaceSkill{}, fmt.Errorf(
			"%w: Artifact %q is unavailable for runtime loading",
			engine.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return plan.Skills[0], nil
}

func projectWorkspaceSkill(
	workspace artifactstore.CollectionRef,
	resourceValue engine.Resource,
	workspaceEnabled bool,
	includeMarkdown bool,
	runtimePathBacked bool,
) (WorkspaceSkill, error) {
	runtimeDisabled, dataErr := engine.ArtifactRuntimeDisabled(
		resourceValue.Artifact,
	)
	output := WorkspaceSkill{
		Workspace:        workspace,
		Artifact:         resourceValue.Artifact.Ref(),
		ArtifactRevision: resourceValue.Artifact.Revision,
		DefinitionDigest: resourceValue.Definition.Digest,
		SourceID:         resourceValue.Source.ID,
		Locator:          resourceValue.Artifact.Binding.Locator,
		State:            resourceValue.Artifact.State,
		CatalogCurrent:   resourceValue.CatalogCurrent,
		RuntimeDisabled:  runtimeDisabled,
		WorkspaceEnabled: workspaceEnabled,
		Diagnostics: artifactstore.AppendDiagnostics(
			resourceValue.Artifact.Diagnostics,
			resourceValue.Diagnostics...,
		),
		RuntimePathBacked: runtimePathBacked,
	}
	if dataErr != nil {
		return output, dataErr
	}
	if err := skillartifact.ValidateDefinition(resourceValue.Definition); err != nil {
		return output, err
	}
	body, err := skillartifact.DecodeBody(

		resourceValue.Definition.Body,
	)
	if err != nil {
		return output, err
	}
	markdownBody := ""
	if includeMarkdown {
		markdownBody = body.MarkdownBody
	}
	output.Skill = skillSummary(resourceValue.Artifact, body)
	output.MarkdownBody = markdownBody
	if resourceValue.Occurrence != nil &&
		resourceValue.Occurrence.SourceContentDigest != nil {
		output.SourceContentDigest = *resourceValue.Occurrence.SourceContentDigest
	}
	output.ProjectionValid = true
	return output, nil
}

func skillSummary(
	artifactValue artifact.Artifact,
	value skillartifact.Body,
) SkillSummary {
	arguments := make([]SkillArgument, 0, len(value.Arguments))
	for _, argument := range value.Arguments {
		arguments = append(arguments, SkillArgument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	return SkillSummary{
		SchemaVersion: skillartifact.SchemaVersion,
		ID:            artifactValue.ID,

		Slug: value.Name,

		Name:        value.Name,
		DisplayName: value.DisplayName,
		Description: value.Description,
		Tags:        append([]string(nil), value.Tags...),
		Insert:      value.Insert,
		Arguments:   arguments,
		IsEnabled:   artifactValue.Enabled,
		CreatedAt:   artifactValue.CreatedAt,
		ModifiedAt:  artifactValue.ModifiedAt,
	}
}

func sortWorkspaceSkills(values []WorkspaceSkill) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Skill.Name != values[right].Skill.Name {
			return values[left].Skill.Name < values[right].Skill.Name
		}
		return values[left].Artifact.ArtifactID < values[right].Artifact.ArtifactID
	})
}

// resolveRuntimeLocation performs the Workspace-owned handoff from a selected
// source-linked record to a native filesystem skill package. It does not
// register or execute the skill. Agent Skills runtime lifecycle remains in
// skillruntime.
func (f *Adapter) runtimeSource(
	ctx context.Context,
	item engine.LoadPlanItem,
	verified map[artifactstore.SourceID]source.Source,
	failures map[artifactstore.SourceID]error,
) (source.Source, error) {
	if value, found := verified[item.Source.ID]; found {
		return value, nil
	}
	if err, found := failures[item.Source.ID]; found {
		return source.Source{}, err
	}
	sourceValue, err := f.sourceRuntime.Get(
		ctx,
		item.Artifact.RootID,
		item.Source.ID,
	)
	if err != nil {
		failures[item.Source.ID] = err
		return source.Source{}, err
	}
	if sourceValue.ID != item.Source.ID ||
		sourceValue.Revision != item.Source.Revision ||
		sourceValue.Kind != item.Source.Kind {
		err := fmt.Errorf(
			"%w: Workspace Skill source changed after load-plan composition",
			artifactstore.ErrCatalogStale,
		)
		failures[item.Source.ID] = err
		return source.Source{}, err
	}
	localPaths, supported := f.sourceRuntime.(source.LocalPathRuntime)
	if !supported ||
		!localPaths.SupportsLocalPath(sourceValue.Kind) {
		err := fmt.Errorf(
			"%w: Workspace Skill source kind %q has no trusted native path",
			artifactstore.ErrUnsupported,
			sourceValue.Kind,
		)
		failures[item.Source.ID] = err
		return source.Source{}, err
	}
	if err := verifySourceGeneration(
		ctx,
		f.sourceRuntime,
		sourceValue,
		item.SourceGeneration,
	); err != nil {
		failures[item.Source.ID] = err
		return source.Source{}, err
	}
	verified[item.Source.ID] = sourceValue

	return sourceValue, nil
}

func (f *Adapter) resolveRuntimeLocation(
	ctx context.Context,
	item engine.LoadPlanItem,
	sourceValue source.Source,
) (string, error) {
	localPaths, supported := f.sourceRuntime.(source.LocalPathRuntime)
	if !supported ||
		!localPaths.SupportsLocalPath(item.Source.Kind) {
		return "", fmt.Errorf(
			"%w: Workspace Skill source kind %q has no native filesystem package",
			artifactstore.ErrUnsupported,
			item.Source.Kind,
		)
	}
	if item.Artifact.Binding.SubresourceLocator != "" {
		return "", fmt.Errorf(
			"%w: Workspace Skill cannot use a subresource binding",
			artifactstore.ErrUnsupported,
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
	skillMDPath, err := localPaths.ResolveLocalPath(
		ctx,
		sourceValue,
		item.Artifact.Binding.Locator,
	)
	if err != nil {
		return "", err
	}
	if filepath.Base(skillMDPath) != skillartifact.DefinitionFileName {
		return "", fmt.Errorf(
			"%w: Workspace Skill locator %q is not %q",
			artifactstore.ErrInvalid,
			item.Artifact.Binding.Locator,
			skillartifact.DefinitionFileName,
		)
	}
	if err := verifySkillPackage(skillMDPath); err != nil {
		return "", err
	}
	if err := verifySkillMDContent(
		skillMDPath,
		item.SourceContentDigest,
	); err != nil {
		return "", err
	}
	if err := verifySourceGeneration(
		ctx,
		f.sourceRuntime,
		sourceValue,
		item.SourceGeneration,
	); err != nil {
		return "", err
	}
	return filepath.Dir(skillMDPath), nil
}

func verifySourceGeneration(
	ctx context.Context,
	runtime source.Runtime,
	value source.Source,
	expected string,
) error {
	if err := artifactstore.ValidateSourceGeneration(expected); err != nil {
		return err
	}
	snapshot, err := runtime.Open(ctx, value)
	if err != nil {
		return err
	}
	defer snapshot.Close()
	if snapshot.Generation() != expected {
		return fmt.Errorf(
			"%w: Workspace Skill source changed since catalog publication",
			artifactstore.ErrCatalogStale,
		)
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return err
	}
	return nil
}

func (f *Adapter) supportsRuntimePath(
	kind artifactstore.SourceKind,
) bool {
	localPaths, supported := f.sourceRuntime.(source.LocalPathRuntime)
	if !supported {
		return false
	}
	return localPaths.SupportsLocalPath(kind)
}

// verifySkillPackage rejects runtime-visible links and special files before
// handing a package to the ordinary Agent Skills filesystem provider.
//
// Source generation confirmation binds regular package files to the catalog
// observation. This check prevents a package from using a symlink to escape a
// source root after the trusted local-path handoff.
func verifySkillPackage(skillMDPath string) error {
	root := filepath.Dir(skillMDPath)
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf(
			"%w: Workspace Skill package is not a regular directory",
			artifactstore.ErrInvalid,
		)
	}

	var (
		entries int
		bytes   int64
	)

	return filepath.WalkDir(root, func(
		location string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if location == root {
			return nil
		}
		entries++
		if entries > artifactstore.MaxDiscoveryEntries {
			return fmt.Errorf(
				"%w: Workspace Skill package exceeds entry limit",
				artifactstore.ErrInvalid,
			)
		}
		relative, err := filepath.Rel(root, location)
		if err != nil {
			return err
		}

		if filepath.IsAbs(relative) ||
			relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf(
				"%w: Workspace Skill package entry escapes package root",
				artifactstore.ErrInvalid,
			)
		}

		if strings.Count(filepath.ToSlash(relative), "/")+1 > artifactstore.MaxDiscoveryDepth {
			return fmt.Errorf("%w: Workspace Skill package exceeds depth limit", artifactstore.ErrInvalid)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf(
				"%w: Workspace Skill package contains a symbolic link",
				artifactstore.ErrInvalid,
			)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf(
				"%w: Workspace Skill package contains a symbolic link",
				artifactstore.ErrInvalid,
			)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf(
				"%w: Workspace Skill package contains a non-regular file",
				artifactstore.ErrInvalid,
			)
		}
		if info.Size() < 0 || info.Size() > artifactstore.MaxScanBytes-bytes {
			return fmt.Errorf(
				"%w: Workspace Skill package exceeds byte limit",
				artifactstore.ErrInvalid,
			)
		}
		bytes += info.Size()
		return nil
	})
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
	value artifact.Artifact,
	err error,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     engine.DiagnosticCodeRuntimeUnavailable,
		Message:  artifactstore.BoundedDiagnosticMessage(err.Error()),
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}

func skillProjectionDiagnostic(
	value artifact.Artifact,
	err error,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     engine.DiagnosticCodeProjectionInvalid,
		Message:  artifactstore.BoundedDiagnosticMessage(err.Error()),
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}
