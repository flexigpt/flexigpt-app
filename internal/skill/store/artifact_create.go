package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	userCreatedSkillsDirName = "user-created-skills"
	skillMDFileName          = "SKILL.md"
)

var skillArtifactNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

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
	if err := validateSkillArtifactName(name); err != nil {
		return nil, fmt.Errorf("%w: invalid skill artifact name: %w", spec.ErrSkillInvalidRequest, err)
	}

	insert := req.Body.Insert
	if insert == "" {
		insert = spec.SkillInsertInstructions
	}
	switch insert {
	case spec.SkillInsertInstructions, spec.SkillInsertUserMessage:
	default:
		return nil, fmt.Errorf("%w: invalid insert %q", spec.ErrSkillInvalidRequest, insert)
	}

	arguments := append([]spec.SkillArgument(nil), req.Body.Arguments...)
	if err := validateSkillArguments(arguments); err != nil {
		return nil, fmt.Errorf("%w: invalid arguments: %w", spec.ErrSkillInvalidRequest, err)
	}

	markdownBody := req.Body.MarkdownBody
	if strings.TrimSpace(markdownBody) == "" {
		return nil, fmt.Errorf("%w: markdownBody required", spec.ErrSkillInvalidRequest)
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

		if err := createManagedSkillPackage(location, buildManagedSkillMarkdown(managedSkillMarkdownInput{
			Name:         name,
			Description:  req.Body.Description,
			Insert:       insert,
			Arguments:    arguments,
			MarkdownBody: markdownBody,
			DisplayName:  req.Body.DisplayName,
		})); err != nil {
			return userWriteSagaOutcome{}, err
		}
		createdDir = location

		uuid, err := uuidv7filename.NewUUIDv7String()
		if err != nil {
			return userWriteSagaOutcome{}, err
		}

		now := time.Now().UTC()
		rawFrontmatter := map[string]any{
			"name":        name,
			"description": req.Body.Description,
			"insert":      insert,
		}
		if len(arguments) > 0 {
			rawFrontmatter["arguments"] = arguments
		}

		sk := spec.Skill{
			SchemaVersion:  spec.SkillSchemaVersion,
			ID:             bundleitemutils.ItemID(uuid),
			Slug:           req.SkillSlug,
			Type:           spec.SkillTypeFS,
			Location:       location,
			Name:           name,
			DisplayName:    req.Body.DisplayName,
			Description:    req.Body.Description,
			Tags:           append([]string(nil), req.Body.Tags...),
			Insert:         insert,
			Arguments:      append([]spec.SkillArgument(nil), arguments...),
			RawFrontmatter: rawFrontmatter,
			Presence:       &spec.SkillPresence{Status: spec.SkillPresenceUnknown},
			IsEnabled:      req.Body.IsEnabled,
			IsBuiltIn:      false,
			CreatedAt:      now,
			ModifiedAt:     now,
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

func validateSkillArtifactName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is empty")
	}
	if strings.TrimSpace(name) != name {
		return errors.New("name has leading/trailing whitespace")
	}
	if !skillArtifactNameRE.MatchString(name) {
		return errors.New("name must contain only lowercase letters, numbers, and hyphens, max 64 chars")
	}
	return nil
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

type managedSkillMarkdownInput struct {
	Name         string
	Description  string
	DisplayName  string
	Insert       agentskillsSpec.SkillInsert
	Arguments    []agentskillsSpec.SkillArgument
	MarkdownBody string
}

func buildManagedSkillMarkdown(in managedSkillMarkdownInput) string {
	insert := in.Insert
	if insert == "" {
		insert = spec.SkillInsertInstructions
	}

	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = humanizeSkillName(in.Name)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: ")
	b.WriteString(yamlQuotedString(in.Name))
	b.WriteString("\n")
	b.WriteString("description: ")
	b.WriteString(yamlQuotedString(in.Description))
	b.WriteString("\n")
	b.WriteString("insert: ")
	b.WriteString(yamlQuotedString(string(insert)))
	b.WriteString("\n")
	if len(in.Arguments) > 0 {
		b.WriteString("arguments:\n")
		for _, arg := range in.Arguments {
			b.WriteString("  - name: ")
			b.WriteString(yamlQuotedString(arg.Name))
			b.WriteString("\n")
			if strings.TrimSpace(arg.Description) != "" {
				b.WriteString("    description: ")
				b.WriteString(yamlQuotedString(arg.Description))
				b.WriteString("\n")
			}
			if arg.Default != "" {
				b.WriteString("    default: ")
				b.WriteString(yamlQuotedString(arg.Default))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("---\n\n")
	b.WriteString("# ")
	b.WriteString(displayName)
	b.WriteString("\n\n")
	b.WriteString(strings.ReplaceAll(in.MarkdownBody, "\r\n", "\n"))
	if !strings.HasSuffix(in.MarkdownBody, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func yamlQuotedString(value string) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(raw)
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
