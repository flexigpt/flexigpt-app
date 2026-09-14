package domain

import (
	"bytes"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// NormalizeManagedSkillFiles returns a complete portable Skill package and
// its exact SKILL.md bytes.
func NormalizeManagedSkillFiles(
	skillMD []byte,
	input []source.ManagedPackageFile,
) ([]source.ManagedPackageFile, []byte, error) {
	if len(input) == 0 {
		if len(skillMD) == 0 {
			return nil, nil, fmt.Errorf(
				"%w: SKILL.md content is required",
				basespec.ErrInvalid,
			)
		}
		return []source.ManagedPackageFile{{
			Locator: SkillDefinitionFileName,
			Content: append([]byte(nil), skillMD...),
		}}, append([]byte(nil), skillMD...), nil
	}

	normalized, err := source.NormalizeManagedPackageFiles(input)
	if err != nil {
		return nil, nil, err
	}

	var packageSkillMD []byte
	for _, file := range normalized {
		if file.Locator != SkillDefinitionFileName {
			continue
		}
		packageSkillMD = append([]byte(nil), file.Content...)
		break
	}
	if len(packageSkillMD) == 0 {
		return nil, nil, fmt.Errorf(
			"%w: managed Skill package must contain %q",
			basespec.ErrInvalid,
			SkillDefinitionFileName,
		)
	}
	if len(skillMD) != 0 &&
		!bytes.Equal(skillMD, packageSkillMD) {
		return nil, nil, fmt.Errorf(
			"%w: requested SKILL.md differs from package SKILL.md",
			basespec.ErrInvalid,
		)
	}
	return normalized, packageSkillMD, nil
}

func PackageDigest(
	files []source.ManagedPackageFile,
) (cryptoutil.Digest, error) {
	normalized, err := source.NormalizeManagedPackageFiles(files)
	if err != nil {
		return "", err
	}
	return cryptoutil.CanonicalDigest(normalized)
}
