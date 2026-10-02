package domain

import (
	"bytes"
	"fmt"
	"path"

	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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
				spec.ErrInvalid,
			)
		}
		return []source.ManagedPackageFile{{
			Locator: SkillDefinitionFileName(),
			Content: append([]byte(nil), skillMD...),
		}}, append([]byte(nil), skillMD...), nil
	}

	normalized, err := source.NormalizeManagedPackageFiles(input)
	if err != nil {
		return nil, nil, err
	}

	documentFile := SkillDefinitionFileName()
	var packageSkillMD []byte
	for _, file := range normalized {
		if file.Locator != documentFile {
			continue
		}
		packageSkillMD = append([]byte(nil), file.Content...)
		break
	}
	if len(packageSkillMD) == 0 {
		return nil, nil, fmt.Errorf(
			"%w: managed Skill package must contain %q",
			spec.ErrInvalid,
			documentFile,
		)
	}
	if len(skillMD) != 0 &&
		!bytes.Equal(skillMD, packageSkillMD) {
		return nil, nil, fmt.Errorf(
			"%w: requested SKILL.md differs from package SKILL.md",
			spec.ErrInvalid,
		)
	}
	return normalized, packageSkillMD, nil
}

// ManagedSkillStorageFiles converts logical Skill-directory files into the
// physical files stored in one managed package.
//
// Callers provide a logical Agent Skill directory:
//
//	SKILL.md
//	references/example.md
//	scripts/check.py
//
// The managed package stores all files below a directory named after the
// Skill. This satisfies the Agent Skills filesystem requirement that the
// directory containing SKILL.md matches frontmatter.name:
//
//	<skill-name>/SKILL.md
//	<skill-name>/references/example.md
//	<skill-name>/scripts/check.py
func ManagedSkillStorageFiles(
	address source.ManagedPackageAddress,
	input []source.ManagedPackageFile,
) ([]source.ManagedPackageFile, error) {
	if err := validateManagedSkillPackageAddress(address); err != nil {
		return nil, err
	}

	files, err := source.NormalizeManagedPackageFiles(input)
	if err != nil {
		return nil, err
	}

	documentFile := SkillDefinitionFileName()
	documentFound := false
	output := make([]source.ManagedPackageFile, 0, len(files))
	for _, file := range files {
		if file.Locator == documentFile {
			documentFound = true
		}

		locator := spec.Locator(path.Join(
			string(address.Name),
			string(file.Locator),
		))
		if err := locator.ValidatePortable(false); err != nil {
			return nil, err
		}

		output = append(output, source.ManagedPackageFile{
			Locator: locator,
			Content: append([]byte(nil), file.Content...),
		})
	}
	if !documentFound {
		return nil, fmt.Errorf(
			"%w: logical managed Skill package must contain %q",
			spec.ErrInvalid,
			documentFile,
		)
	}

	return source.NormalizeManagedPackageFiles(output)
}

func PackageDigest(
	files []source.ManagedPackageFile,
) (cryptoutil.Digest, error) {
	normalized, err := source.NormalizeManagedPackageFiles(files)
	if err != nil {
		return "", err
	}
	type fileManifest struct {
		Locator spec.Locator      `json:"locator"`
		Size    int64             `json:"size"`
		Digest  cryptoutil.Digest `json:"digest"`
	}
	manifest := make([]fileManifest, 0, len(normalized))
	for _, file := range normalized {
		manifest = append(manifest, fileManifest{
			Locator: file.Locator,
			Size:    int64(len(file.Content)),
			Digest:  cryptoutil.DigestBytes(file.Content),
		})
	}
	raw, err := jsonutil.MarshalCanonicalObject(map[string]any{
		"format": "skill-package-content/v1",
		"files":  manifest,
	}, spec.MaxDefinitionBytes)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(raw), nil
}
