package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	maxDisplayNameLen = 256
	maxDescriptionLen = 4096
	maxLocationLen    = 4096
	maxNameLen        = 256
)

func validateSkillBundle(b *spec.SkillBundle) error {
	if b == nil {
		return errors.New("bundle is nil")
	}
	if b.SchemaVersion != spec.SkillSchemaVersion {
		return fmt.Errorf("schemaVersion %q != %q", b.SchemaVersion, spec.SkillSchemaVersion)
	}
	if strings.TrimSpace(string(b.ID)) == "" {
		return errors.New("id is empty")
	}
	if err := bundleitemutils.ValidateBundleSlug(b.Slug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}
	if b.CreatedAt.IsZero() || b.ModifiedAt.IsZero() {
		return errors.New("createdAt/modifiedAt is zero")
	}

	if b.ModifiedAt.Before(b.CreatedAt) {
		return errors.New("modifiedAt is before createdAt")
	}

	if strings.TrimSpace(b.DisplayName) == "" {
		return errors.New("displayName is empty")
	}

	if len(b.DisplayName) > maxDisplayNameLen {
		return fmt.Errorf("displayName too long (>%d)", maxDisplayNameLen)
	}
	if len(b.Description) > maxDescriptionLen {
		return fmt.Errorf("description too long (>%d)", maxDescriptionLen)
	}
	if isSoftDeletedSkillBundle(*b) && b.IsEnabled {
		return errors.New("soft-deleted bundle cannot be enabled")
	}
	return nil
}

func validateSkill(sk *spec.Skill) error {
	if sk == nil {
		return errors.New("skill is nil")
	}
	if sk.SchemaVersion != spec.SkillSchemaVersion {
		return fmt.Errorf("schemaVersion %q != %q", sk.SchemaVersion, spec.SkillSchemaVersion)
	}
	if strings.TrimSpace(string(sk.ID)) == "" {
		return errors.New("id is empty")
	}
	if err := bundleitemutils.ValidateItemSlug(sk.Slug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}
	if strings.TrimSpace(sk.Location) == "" {
		return errors.New("location is empty")
	}
	if len(sk.Location) > maxLocationLen {
		return fmt.Errorf("location too long (>%d)", maxLocationLen)
	}
	if strings.TrimSpace(sk.Name) == "" {
		return errors.New("name is empty")
	}
	if len(sk.Name) > maxNameLen {
		return fmt.Errorf("name too long (>%d)", maxNameLen)
	}
	if len(sk.DisplayName) > maxDisplayNameLen {
		return fmt.Errorf("displayName too long (>%d)", maxDisplayNameLen)
	}
	if len(sk.Description) > maxDescriptionLen {
		return fmt.Errorf("description too long (>%d)", maxDescriptionLen)
	}
	if sk.CreatedAt.IsZero() || sk.ModifiedAt.IsZero() {
		return errors.New("createdAt/modifiedAt is zero")
	}
	if sk.ModifiedAt.Before(sk.CreatedAt) {
		return errors.New("modifiedAt is before createdAt")
	}
	if err := bundleitemutils.ValidateTags(sk.Tags); err != nil {
		return err
	}

	switch sk.Type {
	case spec.SkillTypeFS:
		if sk.IsBuiltIn {
			return errors.New("built-in skill cannot be type=fs")
		}
	case spec.SkillTypeEmbeddedFS:
		if !sk.IsBuiltIn {
			return errors.New("non-built-in skill cannot be type=embeddedfs")
		}
	default:
		return fmt.Errorf("invalid type %q", sk.Type)
	}

	if sk.Presence != nil {
		switch sk.Presence.Status {
		case spec.SkillPresenceUnknown, spec.SkillPresencePresent, spec.SkillPresenceMissing, spec.SkillPresenceError:
		default:
			return fmt.Errorf("invalid presence.status %q", sk.Presence.Status)
		}
	}

	return nil
}
