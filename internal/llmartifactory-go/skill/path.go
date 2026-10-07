package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

// AddSkillPath registers one native Skill directory or one native SKILL.md
// file. The resulting Skill remains a normal Root-scoped source-backed
// Artifact.
func (a *Service) AddSkillPath(
	ctx context.Context,
	request SkillPathRegistration,
) (SkillPathRegistrationResult, error) {
	if err := request.RootID.Validate(); err != nil {
		return SkillPathRegistrationResult{}, err
	}

	rootPath, skillLocator, err := a.normalizeSkillPath(request.Path)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}
	displayName := request.SourceDisplayName
	if displayName == "" {
		displayName = "Skill source " + filepath.Base(rootPath)
	}
	if err := spec.ValidateRequiredText(
		"Skill Source display name",
		displayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return SkillPathRegistrationResult{}, err
	}

	discovery, err := a.skillFileDiscovery(skillLocator)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}
	config, err := json.Marshal(struct {
		RootPath string `json:"rootPath"`
	}{
		RootPath: rootPath,
	})
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}

	summary, err := refreshFlow.EnsureAndRefreshSource(
		ctx,
		a.sources,
		a.discovery,
		refreshFlow.EnsureAndRefreshSourceRequest{
			RootID: request.RootID,
			Draft: sourceModel.Draft{
				ID: sourceModel.SourceID(uuidutil.NewUUIDv7()),
				StorageKey: fsdir.FilesystemSourceStorageKey(
					"skill-path",
					rootPath,
				),
				Kind:        fsdir.Kind,
				DisplayName: displayName,
				Enabled:     true,
				Config:      config,
				Discovery:   discovery,
			},
		},
	)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}

	record, err := a.artifacts.FindByOrigin(
		ctx,
		request.RootID,
		artifactModel.SourceBinding{
			SourceID: summary.ID,
			Locator:  skillLocator,
		},
		skillSource.SkillArtifactKind,
	)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}
	if record.State != artifactModel.StateAvailable {
		return SkillPathRegistrationResult{}, fmt.Errorf(
			"%w: Skill path did not produce an available Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return SkillPathRegistrationResult{}, err
		}
	}

	summary, err = a.sources.Get(ctx, request.RootID, summary.ID)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}
	return SkillPathRegistrationResult{
		Source:   summary,
		Artifact: record,
	}, nil
}

func (a *Service) normalizeSkillPath(
	raw string,
) (string, spec.Locator, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", "", fmt.Errorf(
			"%w: Skill path is required",
			spec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", "", err
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		_, locator, err := a.findSkillDefinitionPath(absolute)
		if err != nil {
			return "", "", err
		}
		return absolute, locator, nil
	}
	if !info.Mode().IsRegular() ||
		!skillSource.IsSkillDefinitionFile(
			a.support.Documents,
			spec.Locator(filepath.Base(absolute)),
		) {
		return "", "", fmt.Errorf(
			"%w: Skill path must identify a Skill directory or configured Skill document",
			spec.ErrInvalid,
		)
	}
	return filepath.Dir(absolute),
		spec.Locator(filepath.Base(absolute)),
		nil
}

func (a *Service) findSkillDefinitionPath(
	directory string,
) (string, spec.Locator, error) {
	var (
		selectedPath    string
		selectedLocator spec.Locator
	)
	for _, candidate := range skillSource.SkillDefinitionFiles(a.support.Documents) {
		candidatePath := filepath.Join(directory, string(candidate))
		info, err := os.Stat(candidatePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", "", err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if selectedPath != "" {
			return "", "", fmt.Errorf(
				"%w: Skill directory has multiple configured Skill documents",
				spec.ErrIdentityConflict,
			)
		}
		selectedPath = candidatePath
		selectedLocator = candidate
	}
	if selectedPath == "" {
		return "", "", fmt.Errorf(
			"%w: Skill directory lacks a configured Skill document",
			spec.ErrInvalid,
		)
	}
	return selectedPath, selectedLocator, nil
}

func (a *Service) skillFileDiscovery(
	locator spec.Locator,
) (sourceModel.DiscoverySpec, error) {
	value := sourceModel.DiscoverySpec{
		ExplicitLocators: []spec.Locator{locator},
		DecoderHints: []sourceModel.DecoderHint{{
			Locator: locator,
			DecoderIDs: []spec.DecoderID{
				a.support.Package.Document.DecoderID,
			},
		}},
		AllowedDecoderIDs: []spec.DecoderID{
			a.support.Package.Document.DecoderID,
		},
		Authoritative: a.support.Discovery.Authoritative,
	}.Normalized()
	if err := value.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return value, nil
}
