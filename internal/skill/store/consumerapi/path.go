package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
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

	summary, _, err := a.sources.Ensure(
		ctx,
		request.RootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  skillPathStorageKey(rootPath),
			Kind:        source.SourceKindFilesystemDirectory,
			DisplayName: displayName,
			Enabled:     true,
			Config:      config,
			Discovery:   discovery,
		},
	)
	if err != nil {
		return SkillPathRegistrationResult{}, err
	}
	if !summary.Enabled ||
		summary.DisplayName != displayName ||
		!summary.Discovery.Equal(discovery) {
		summary, err = a.sources.Update(
			ctx,
			request.RootID,
			summary.ID,
			source.Update{
				ExpectedRevision: summary.Revision,
				DisplayName:      displayName,
				Enabled:          true,
				Discovery:        &discovery,
			},
		)
		if err != nil {
			return SkillPathRegistrationResult{}, err
		}
	}

	if _, err := a.discovery.RefreshSource(
		ctx,
		request.RootID,
		summary.ID,
	); err != nil {
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
		documentPath := filepath.Join(
			absolute,
			string(skillDomain.SkillDefinitionFileName),
		)
		document, err := os.Stat(documentPath)
		if err != nil {
			return "", "", err
		}
		if !document.Mode().IsRegular() {
			return "", "", fmt.Errorf(
				"%w: Skill directory lacks regular %q",
				basespec.ErrInvalid,
				skillDomain.SkillDefinitionFileName,
			)
		}
		return absolute, skillDomain.SkillDefinitionFileName, nil
	}
	if !info.Mode().IsRegular() ||
		filepath.Base(absolute) != string(skillDomain.SkillDefinitionFileName) {
		return "", "", fmt.Errorf(
			"%w: Skill path must identify a Skill directory or %q",
			basespec.ErrInvalid,
			skillDomain.SkillDefinitionFileName,
		)
	}
	return filepath.Dir(absolute), skillDomain.SkillDefinitionFileName, nil
}

func skillFileDiscovery(
	locator basespec.Locator,
) (source.DiscoverySpec, error) {
	value := source.DiscoverySpec{
		ExplicitLocators: []basespec.Locator{locator},
		DecoderHints: []source.DecoderHint{{
			Locator:   locator,
			Recursive: false,
			DecoderIDs: []basespec.DecoderID{
				skillDomain.MarkdownDecoderID,
			},
		}},
		AllowedDecoderIDs: []basespec.DecoderID{
			skillDomain.MarkdownDecoderID,
		},
		Authoritative: true,
	}
	value = value.Normalized()
	if err := value.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return value, nil
}

func skillPathStorageKey(
	rootPath string,
) basespec.StorageKey {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(rootPath))),
		cryptoutil.DigestSHA256Prefix,
	)
	return basespec.StorageKey("skill-path-" + digest[:24])
}
