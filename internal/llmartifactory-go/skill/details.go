package skill

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/flexigpt/agentskills-go/document"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/materialize"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

const (
	maxInspectedSkillEntries   = 4096
	maxDisplayedSkillResources = 256
)

type ArtifactSkillResources struct {
	HasResources  bool     `json:"hasResources"`
	TotalCount    int      `json:"totalCount"`
	Locations     []string `json:"locations,omitempty"`
	MoreLocations bool     `json:"moreLocations"`
}

// ArtifactSkillDetails is a verified, read-only inspection result. In
// particular, Document is not an authorization to replace a non-managed
// package, and Locations are not a durable file capability.
type ArtifactSkillDetails struct {
	Artifact        artifactModel.Artifact `json:"artifact"`
	Document        document.SkillDocument `json:"document"`
	Location        string                 `json:"location"`
	Resources       ArtifactSkillResources `json:"resources"`
	RuntimeWarnings []string               `json:"runtimeWarnings,omitempty"`
}

type detailResourceReader struct {
	resourceFlow.API
	resourceFlow.NativePathAPI
}

func (a *Service) GetArtifactSkillDetails(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) (ArtifactSkillDetails, error) {
	if expectedRevision == 0 {
		return ArtifactSkillDetails{}, fmt.Errorf(
			"%w: expected Skill revision is required",
			storeSpec.ErrInvalid,
		)
	}
	if a.nativeResources == nil {
		return ArtifactSkillDetails{}, fmt.Errorf(
			"%w: verified native Skill paths are unavailable",
			storeSpec.ErrUnsupported,
		)
	}

	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (ArtifactSkillDetails, error) {
			value, err := a.GetSkill(sessionCtx, ref)
			if err != nil {
				return ArtifactSkillDetails{}, err
			}
			if value.Revision != expectedRevision {
				return ArtifactSkillDetails{}, storeSpec.ErrConflict
			}

			reader := detailResourceReader{
				API:           a.resources,
				NativePathAPI: a.nativeResources,
			}
			materialized, err := materialize.Resolve(
				sessionCtx,
				reader,
				a.support.Documents,
				value,
			)
			if err != nil {
				return ArtifactSkillDetails{}, err
			}

			resolved, err := a.resources.ResolveArtifact(
				sessionCtx,
				ref,
				resourceModel.ResolveOptions{},
			)
			if err != nil {
				return ArtifactSkillDetails{}, err
			}
			if resolved.Artifact.Ref() != ref ||
				resolved.Artifact.Revision != expectedRevision ||
				resolved.Artifact.Binding != value.Binding {
				return ArtifactSkillDetails{}, storeSpec.ErrRefreshRequired
			}

			entry, err := a.resources.ReadSourceEntry(
				sessionCtx,
				value.RootID,
				value.Binding.SourceID,
				materialized.Locator,
				storeSpec.MaxCandidateBytes,
			)
			if err != nil {
				return ArtifactSkillDetails{}, err
			}
			// Generation alone does not prove this Skill changed.
			// The digest comparison below is the per artifact gate.
			if materialized.Locator == value.Binding.Locator &&
				(value.SourceContentDigest == nil ||
					entry.Digest != *value.SourceContentDigest) {
				return ArtifactSkillDetails{}, storeSpec.ErrRefreshRequired
			}

			_, warnings, err := skillSourceParse(entry.Content, string(value.LogicalName))
			if err != nil {
				return ArtifactSkillDetails{}, err
			}

			resources, err := inspectSkillResources(
				sessionCtx,
				reader,
				resolved,
				materialized.RuntimeLocation,
				materialized.Locator,
			)
			if err != nil {
				return ArtifactSkillDetails{}, err
			}

			current, err := a.GetSkill(sessionCtx, ref)
			if err != nil {
				return ArtifactSkillDetails{}, err
			}
			if current.Revision != expectedRevision ||
				current.Binding != value.Binding {
				return ArtifactSkillDetails{}, storeSpec.ErrRefreshRequired
			}

			return ArtifactSkillDetails{
				Artifact:        current,
				Document:        materialized.Document,
				Location:        materialized.RuntimeLocation,
				Resources:       resources,
				RuntimeWarnings: warnings,
			}, nil
		},
	)
}

func skillSourceParse(content []byte, name string) (document.SkillDocument, []string, error) {
	value, diagnostics, err := skillSource.ParseSkillDocument(content, name)
	if err != nil {
		return document.SkillDocument{}, nil, err
	}
	warnings := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		warnings = append(warnings, diagnostic.Message)
	}
	return value, warnings, nil
}

// Walk only regular, non-hidden files. Every returned path must also pass the
// Store's verified-local-path boundary; symlinks are never followed as
// resources. A bounded walk fails rather than reporting a false total.
func inspectSkillResources(
	ctx context.Context,
	reader detailResourceReader,
	resolved resourceModel.ResolvedArtifact,
	directory string,
	documentLocator storeSpec.Locator,
) (ArtifactSkillResources, error) {
	output := ArtifactSkillResources{}
	packageLocator := path.Dir(string(documentLocator))
	documentName := path.Base(string(documentLocator))
	visited := 0

	var walk func(string, string) error
	walk = func(onDisk, relative string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, err := os.ReadDir(onDisk)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			visited++
			if visited > maxInspectedSkillEntries {
				return fmt.Errorf(
					"%w: Skill resource inspection exceeds its safety limit",
					storeSpec.ErrUnsupported,
				)
			}
			childRelative := path.Join(relative, entry.Name())
			childDisk := filepath.Join(onDisk, entry.Name())
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := walk(childDisk, childRelative); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() ||
				(relative == "" && entry.Name() == documentName) {
				continue
			}

			locator := storeSpec.Locator(path.Join(packageLocator, childRelative))
			if err := locator.ValidatePortable(false); err != nil {
				return err
			}
			approved, err := reader.ResolveVerifiedLocalPath(ctx, resolved, locator)
			if err != nil {
				return err
			}
			scannedReal, err := filepath.EvalSymlinks(childDisk)
			if err != nil {
				return err
			}
			approvedReal, err := filepath.EvalSymlinks(approved)
			if err != nil {
				return err
			}
			if filepath.Clean(scannedReal) != filepath.Clean(approvedReal) {
				return fmt.Errorf(
					"%w: Skill resource resolved outside its verified package",
					storeSpec.ErrRefreshRequired,
				)
			}

			output.TotalCount++
			if len(output.Locations) < maxDisplayedSkillResources {
				output.Locations = append(output.Locations, approved)
			} else {
				output.MoreLocations = true
			}
		}
		return nil
	}

	if err := walk(directory, ""); err != nil {
		return ArtifactSkillResources{}, err
	}
	output.HasResources = output.TotalCount > 0
	return output, nil
}
