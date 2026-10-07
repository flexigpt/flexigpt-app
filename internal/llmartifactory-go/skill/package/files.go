package skillpackage

import (
	"bytes"
	"fmt"
	"path"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

// NormalizeManagedSkillFiles returns a complete portable Skill package and
// its exact SKILL.md bytes.
func NormalizeManagedSkillFiles(
	skillMD []byte,
	input []managedpackageModel.ManagedPackageFile,
	documents support.Documents,
) ([]managedpackageModel.ManagedPackageFile, []byte, error) {
	if err := documents.Validate(); err != nil {
		return nil, nil, err
	}

	documentFile := documents.Default.Locator
	if len(input) == 0 {
		if len(skillMD) == 0 {
			return nil, nil, fmt.Errorf(
				"%w: configured Skill document content is required",
				spec.ErrInvalid,
			)
		}
		return []managedpackageModel.ManagedPackageFile{{
			Locator: documentFile,
			Content: append([]byte(nil), skillMD...),
		}}, append([]byte(nil), skillMD...), nil
	}

	normalized, err := managedpackageModel.NormalizeManagedPackageFiles(input)
	if err != nil {
		return nil, nil, err
	}

	documentIndex := -1
	for index, file := range normalized {
		if path.Dir(string(file.Locator)) != "." ||
			!documents.Matches(file.Locator) {
			continue
		}
		if documentIndex != -1 {
			return nil, nil, fmt.Errorf(
				"%w: managed Skill package has multiple configured Skill documents",
				spec.ErrIdentityConflict,
			)
		}
		documentIndex = index
	}
	if documentIndex == -1 {
		return nil, nil, fmt.Errorf(
			"%w: managed Skill package must contain a configured Skill document",
			spec.ErrInvalid,
		)
	}

	packageSkillMD := append([]byte(nil), normalized[documentIndex].Content...)
	if normalized[documentIndex].Locator != documentFile {
		normalized[documentIndex].Locator = documentFile
		normalized, err = managedpackageModel.NormalizeManagedPackageFiles(
			normalized,
		)
		if err != nil {
			return nil, nil, err
		}
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
	documents support.Documents,
	address managedpackageModel.ManagedPackageAddress,
	input []managedpackageModel.ManagedPackageFile,
) ([]managedpackageModel.ManagedPackageFile, error) {
	if err := documents.Validate(); err != nil {
		return nil, err
	}
	if err := address.Validate(); err != nil {
		return nil, err
	}

	files, err := managedpackageModel.NormalizeManagedPackageFiles(input)
	if err != nil {
		return nil, err
	}

	documentFile := documents.Default.Locator
	documentFound := false
	output := make([]managedpackageModel.ManagedPackageFile, 0, len(files))
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

		output = append(output, managedpackageModel.ManagedPackageFile{
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

	return managedpackageModel.NormalizeManagedPackageFiles(output)
}

func PackageDigest(
	files []managedpackageModel.ManagedPackageFile,
) (cryptoutil.Digest, error) {
	normalized, err := managedpackageModel.NormalizeManagedPackageFiles(files)
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
