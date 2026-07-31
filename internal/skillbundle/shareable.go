package skillbundle

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

// ShareableSkillPackage is the portable package form. It intentionally has no
// Artifact Store identity, installation state, local path, revision,
// enablement, role, runtime state, idempotency key, or user metadata fields.
//
// PackageSHA256 is supplied by the sharing boundary and verified against the
// canonical portable package payload below. It is not a local Artifact digest.
type ShareableSkillPackage struct {
	Bundle        basespec.LogicalName    `json:"bundle"`
	BundleVersion basespec.LogicalVersion `json:"bundleVersion,omitempty"`
	Skill         basespec.LogicalName    `json:"skill"`
	SkillVersion  basespec.LogicalVersion `json:"skillVersion,omitempty"`
	PackageSHA256 cryptoutil.Digest       `json:"packageSHA256"`
	Files         []ShareablePackageFile  `json:"files"`
}

type ShareablePackageFile struct {
	Locator basespec.Locator `json:"locator"`
	Content []byte           `json:"content"`
}

func CanonicalizeShareableSkillPackage(
	input ShareableSkillPackage,
) (ShareableSkillPackage, error) {
	output := input
	output.Files = make([]ShareablePackageFile, len(input.Files))
	for index, file := range input.Files {
		output.Files[index] = ShareablePackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		}
	}
	sort.Slice(output.Files, func(left, right int) bool {
		return output.Files[left].Locator < output.Files[right].Locator
	})
	if err := output.Validate(); err != nil {
		return ShareableSkillPackage{}, err
	}

	raw, err := json.Marshal(struct {
		Bundle        basespec.LogicalName    `json:"bundle"`
		BundleVersion basespec.LogicalVersion `json:"bundleVersion,omitempty"`
		Skill         basespec.LogicalName    `json:"skill"`
		SkillVersion  basespec.LogicalVersion `json:"skillVersion,omitempty"`
		Files         []ShareablePackageFile  `json:"files"`
	}{
		Bundle:        output.Bundle,
		BundleVersion: output.BundleVersion,
		Skill:         output.Skill,
		SkillVersion:  output.SkillVersion,
		Files:         output.Files,
	})
	if err != nil {
		return ShareableSkillPackage{}, err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return ShareableSkillPackage{}, err
	}
	actual := cryptoutil.DigestBytes(canonical)
	if actual != output.PackageSHA256 {
		return ShareableSkillPackage{}, fmt.Errorf(
			"%w: supplied package SHA-256 %q does not match package bytes %q",
			basespec.ErrDigestMismatch,
			output.PackageSHA256,
			actual,
		)
	}
	return output, nil
}

func (p ShareableSkillPackage) Validate() error {
	if err := basespec.ValidateLogicalName(p.Bundle); err != nil {
		return err
	}
	if err := basespec.ValidateLogicalVersion(p.BundleVersion, true); err != nil {
		return err
	}
	if err := basespec.ValidateLogicalName(p.Skill); err != nil {
		return err
	}
	if err := basespec.ValidateLogicalVersion(p.SkillVersion, true); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(p.PackageSHA256); err != nil {
		return err
	}
	if len(p.Files) == 0 {
		return fmt.Errorf("%w: shareable Skill package has no files", basespec.ErrInvalid)
	}
	if len(p.Files) > basespec.MaxDiscoveryEntries {
		return fmt.Errorf(
			"%w: shareable Skill package exceeds file-count limit",
			basespec.ErrInvalid,
		)
	}

	seen := make(map[basespec.Locator]struct{}, len(p.Files))
	var skillMD []byte
	var totalBytes int64
	for index, file := range p.Files {
		if err := basespec.ValidatePortableLocator(file.Locator, false); err != nil {
			return fmt.Errorf("shareable package files[%d]: %w", index, err)
		}
		if int64(len(file.Content)) > basespec.MaxScanBytes-totalBytes {
			return fmt.Errorf(
				"%w: shareable Skill package exceeds byte limit",
				basespec.ErrInvalid,
			)
		}
		totalBytes += int64(len(file.Content))
		if _, duplicate := seen[file.Locator]; duplicate {
			return fmt.Errorf("%w: duplicate shareable package file %q", basespec.ErrInvalid, file.Locator)
		}

		seen[file.Locator] = struct{}{}
		if file.Locator == skillartifact.DefinitionFileName {
			skillMD = append([]byte(nil), file.Content...)
		}
	}
	if len(skillMD) == 0 {
		return fmt.Errorf("%w: shareable Skill package has no SKILL.md", basespec.ErrInvalid)
	}
	definitionValue, _, err := skillartifact.DecodeSkillDocument(
		skillMD,
		string(p.Skill),
	)
	if err != nil {
		return fmt.Errorf("%w: shareable SKILL.md: %w", basespec.ErrInvalid, err)
	}
	if definitionValue.LogicalName != p.Skill {
		return fmt.Errorf("%w: shareable Skill name differs from SKILL.md", basespec.ErrInvalid)
	}
	return nil
}
