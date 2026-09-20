package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

// AddSkillPath registers one native Skill directory or one native SKILL.md
// file. The resulting Skill remains a normal Root-scoped source-backed
// Artifact.
func (a *API) AddSkillPath(
	ctx context.Context,
	request SkillPathRegistration,
) (SkillPathRegistrationResult, error) {
	if a == nil {
		return SkillPathRegistrationResult{}, basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return SkillPathRegistrationResult{}, err
	}

	rootPath, skillLocator, err := normalizeSkillPath(request.Path)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}
	displayName := request.SourceDisplayName
	if displayName == "" {
		displayName = "Skill source " + filepath.Base(rootPath)
	}
	if err := basespec.ValidateRequiredText(
		"Skill Source display name",
		displayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return SkillPathRegistrationResult{}, err
	}

	discovery, err := skillFileDiscovery(skillLocator)
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

	summary, err := consumerutil.EnsureAndRefreshSource(
		ctx,
		a.sources,
		a.discovery,
		consumerutil.EnsureAndRefreshSourceRequest{
			RootID: request.RootID,
			Draft: source.Draft{
				ID: source.SourceID(uuidutil.NewUUIDv7()),
				StorageKey: consumerutil.FilesystemSourceStorageKey(
					"skill-path",
					rootPath,
				),
				Kind:        source.SourceKindFilesystemDirectory,
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
		artifact.SourceBinding{
			SourceID: summary.ID,
			Locator:  skillLocator,
		},
		skillDomain.SkillArtifactKind,
	)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}
	if record.State != artifact.StateAvailable {
		return SkillPathRegistrationResult{}, fmt.Errorf(
			"%w: Skill path did not produce an available Artifact",
			basespec.ErrReferenceUnresolved,
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

func normalizeSkillPath(
	raw string,
) (string, basespec.Locator, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", "", fmt.Errorf(
			"%w: Skill path is required",
			basespec.ErrInvalid,
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
		_, locator, err := findSkillDefinitionPath(absolute)
		if err != nil {
			return "", "", err
		}
		return absolute, locator, nil
	}
	if !info.Mode().IsRegular() ||
		!skillDomain.IsSkillDefinitionFile(
			basespec.Locator(filepath.Base(absolute)),
		) {
		return "", "", fmt.Errorf(
			"%w: Skill path must identify a Skill directory or configured Skill document",
			basespec.ErrInvalid,
		)
	}
	return filepath.Dir(absolute),
		basespec.Locator(filepath.Base(absolute)),
		nil
}

func findSkillDefinitionPath(
	directory string,
) (string, basespec.Locator, error) {
	var (
		selectedPath    string
		selectedLocator basespec.Locator
	)
	for _, candidate := range skillDomain.SkillDefinitionFiles() {
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
				basespec.ErrIdentityConflict,
			)
		}
		selectedPath = candidatePath
		selectedLocator = candidate
	}
	if selectedPath == "" {
		return "", "", fmt.Errorf(
			"%w: Skill directory lacks a configured Skill document",
			basespec.ErrInvalid,
		)
	}
	return selectedPath, selectedLocator, nil
}

func skillFileDiscovery(
	locator basespec.Locator,
) (source.DiscoverySpec, error) {
	return documentTopology.DiscoverySpecForLocatorForUse(
		documentTopology.DiscoveryUseSkill,
		locator,
	)
}
