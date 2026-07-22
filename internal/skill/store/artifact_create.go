package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	userCreatedSkillsDirName = "user-created-skills"
	skillMDFileName          = agentskills.SkillDocumentFileName
)

func (s *SkillStore) PutSkillArtifact(
	ctx context.Context,
	req *spec.PutSkillArtifactRequest,
) (resp *spec.PutSkillArtifactResponse, err error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID, skillSlug and body required", spec.ErrSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, fmt.Errorf("%w: invalid skillSlug", spec.ErrSkillInvalidRequest)
	}

	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrSkillBuiltInReadOnly, req.BundleID)
		}
	}

	name := strings.TrimSpace(req.Body.Name)
	if name == "" {
		name = string(req.SkillSlug)
	}

	insert := req.Body.Insert
	if insert == "" {
		insert = spec.SkillInsertInstructions
	}

	arguments := append([]spec.SkillArgument(nil), req.Body.Arguments...)
	tags := append([]string(nil), req.Body.Tags...)

	displayName := strings.TrimSpace(req.Body.DisplayName)
	if displayName == "" {
		displayName = humanizeSkillName(name)
	}

	skillMD, err := agentskills.MarshalSkillDocument(
		agentskillsSpec.SkillDocument{
			Name:         name,
			DisplayName:  displayName,
			Description:  req.Body.Description,
			Insert:       insert,
			Arguments:    arguments,
			Tags:         tags,
			MarkdownBody: req.Body.MarkdownBody,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: invalid skill artifact document: %w",
			spec.ErrSkillInvalidRequest,
			err,
		)
	}

	document, documentWarnings, err := agentskills.ParseSkillDocument(
		skillMD,
		agentskillsSpec.ParseSkillDocumentOptions{
			ExpectedName: name,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: generated skill artifact is invalid: %w",
			spec.ErrSkillInvalidRequest,
			err,
		)
	}

	location := filepath.Join(
		s.baseDir,
		userCreatedSkillsDirName,
		string(req.BundleID),
		name,
	)
	location = filepath.Clean(location)
	locationAbs, err := filepath.Abs(location)
	if err != nil {
		return nil, err
	}
	location = locationAbs

	var createdDir string
	var created spec.Skill
	defer func() {
		if err != nil && createdDir != "" {
			_ = os.RemoveAll(createdDir)
		}
	}()

	if err := s.withUserWriteSaga(ctx, "putSkillArtifact", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		b, ok := sc.Bundles[req.BundleID]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(b) {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
		}

		sm := sc.Skills[req.BundleID]
		if sm == nil {
			sm = map[spec.SkillSlug]spec.Skill{}
			sc.Skills[req.BundleID] = sm
		}
		if _, exists := sm[req.SkillSlug]; exists {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: duplicate skillSlug in bundle", spec.ErrSkillConflict)
		}

		if err := createManagedSkillPackage(location, string(skillMD)); err != nil {
			return userWriteSagaOutcome{}, err
		}
		createdDir = location

		uuid, err := uuidv7filename.NewUUIDv7String()
		if err != nil {
			return userWriteSagaOutcome{}, err
		}

		now := time.Now().UTC()
		sk := spec.Skill{
			SchemaVersion:  spec.SkillSchemaVersion,
			ID:             bundleitemutils.ItemID(uuid),
			Slug:           req.SkillSlug,
			Type:           spec.SkillTypeFS,
			Location:       location,
			Name:           document.Name,
			DisplayName:    document.DisplayName,
			Description:    document.Description,
			Tags:           tags,
			Insert:         document.Insert,
			Arguments:      append([]spec.SkillArgument(nil), document.Arguments...),
			RawFrontmatter: cloneAnyMap(document.RawFrontmatter),
			RuntimeWarnings: append(
				[]string(nil),
				documentWarnings...,
			),
			Presence:   &spec.SkillPresence{Status: spec.SkillPresenceUnknown},
			IsEnabled:  req.Body.IsEnabled,
			IsBuiltIn:  false,
			CreatedAt:  now,
			ModifiedAt: now,
		}
		if err := validateSkill(&sk); err != nil {
			return userWriteSagaOutcome{}, err
		}

		if b.IsEnabled && sk.IsEnabled {
			def, err := runtimeDefForUserSkill(sk)
			if err != nil {
				return userWriteSagaOutcome{}, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
			}
			if _, rec, rtErr := s.runtimeTryAddForeground(ctx, def); rtErr != nil {
				return userWriteSagaOutcome{}, fmt.Errorf(
					"%w: runtime rejected skill artifact: %w",
					spec.ErrSkillInvalidRequest,
					rtErr,
				)
			} else {
				applyRuntimeRecordToSkill(&sk, rec)
			}
		}

		if err := validateSkill(&sk); err != nil {
			return userWriteSagaOutcome{}, err
		}

		sm[req.SkillSlug] = sk
		sc.Skills[req.BundleID] = sm
		created = sk
		return userWriteSagaOutcome{}, nil
	}); err != nil {
		return nil, err
	}

	return &spec.PutSkillArtifactResponse{
		Body: &spec.PutSkillArtifactResponseBody{
			Skill: cloneSkill(created),
		},
	}, nil
}

func createManagedSkillPackage(dir, skillMD string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("%w: managed skill directory is empty", spec.ErrSkillInvalidRequest)
	}

	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: managed skill directory already exists", spec.ErrSkillConflict)
		}
		return err
	}
	return os.WriteFile(filepath.Join(dir, skillMDFileName), []byte(skillMD), 0o600)
}

func humanizeSkillName(name string) string {
	parts := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(name, "-", " "), "_", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
