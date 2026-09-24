package domain_test

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

func TestManagedSkillStorageLayoutUsesNamedRuntimeDirectory(
	t *testing.T,
) {
	address, err := source.NewManagedPackageAddress(
		skillDomain.ManagedSkillPackageKind,
		"release-notes",
		"unversioned",
	)
	if err != nil {
		t.Fatalf("NewManagedPackageAddress: %v", err)
	}

	directory, err := skillDomain.ManagedSkillDirectoryLocator(address)
	if err != nil {
		t.Fatalf("ManagedSkillDirectoryLocator: %v", err)
	}
	wantDirectory := basespec.Locator(
		"skill/release-notes/unversioned/release-notes",
	)
	if directory != wantDirectory {
		t.Fatalf(
			"runtime directory=%q, want %q",
			directory,
			wantDirectory,
		)
	}

	documentLocator, err := skillDomain.ManagedPackageLocatorForSkill(
		address,
	)
	if err != nil {
		t.Fatalf("ManagedPackageLocatorForSkill: %v", err)
	}
	wantDocumentLocator := basespec.Locator(
		"skill/release-notes/unversioned/release-notes/SKILL.md",
	)
	if documentLocator != wantDocumentLocator {
		t.Fatalf(
			"document locator=%q, want %q",
			documentLocator,
			wantDocumentLocator,
		)
	}

	roundTripped, err := skillDomain.ManagedPackageAddressFromSkillLocator(
		documentLocator,
	)
	if err != nil {
		t.Fatalf("ManagedPackageAddressFromSkillLocator: %v", err)
	}
	if roundTripped != address {
		t.Fatalf(
			"round-tripped address=%+v, want %+v",
			roundTripped,
			address,
		)
	}

	files, err := skillDomain.ManagedSkillStorageFiles(
		address,
		[]source.ManagedPackageFile{
			{
				Locator: "SKILL.md",
				Content: []byte(
					"---\nname: release-notes\ndescription: Notes.\n---\n",
				),
			},
			{
				Locator: "references/checklist.md",
				Content: []byte("Check release dates.\n"),
			},
		},
	)
	if err != nil {
		t.Fatalf("ManagedSkillStorageFiles: %v", err)
	}

	contents := make(map[basespec.Locator]string, len(files))
	for _, file := range files {
		contents[file.Locator] = string(file.Content)
	}

	if contents["release-notes/SKILL.md"] == "" {
		t.Fatal("stored files do not contain named runtime SKILL.md")
	}
	if contents["release-notes/references/checklist.md"] !=
		"Check release dates.\n" {
		t.Fatalf(
			"stored checklist=%q, want expected content",
			contents["release-notes/references/checklist.md"],
		)
	}
	if _, found := contents["SKILL.md"]; found {
		t.Fatal("stored files unexpectedly contain root-level SKILL.md")
	}

	_, err = skillDomain.ManagedPackageAddressFromSkillLocator(
		"skill/release-notes/unversioned/not-release-notes/SKILL.md",
	)
	if !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf(
			"mismatched runtime directory error=%v, want ErrInvalid",
			err,
		)
	}
}
