package skillpackage_test

import (
	"errors"
	"testing"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	skillPackage "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/package"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

func TestManagedSkillStorageLayoutUsesNamedRuntimeDirectory(
	t *testing.T,
) {
	layout := support.PackageLayout{
		Kind:           skillSource.ManagedSkillPackageKind,
		DefaultVersion: "unversioned",
		Document: support.Document{
			Locator:   "SKILL.md",
			DecoderID: skillSource.MarkdownDecoderID,
		},
	}
	documents := support.Documents{
		Default: layout.Document,
		Files:   []spec.Locator{"SKILL.md"},
	}

	address, err := managedpackageModel.NewManagedPackageAddress(
		layout.Kind,
		"release-notes",
		"unversioned",
	)
	if err != nil {
		t.Fatalf("NewManagedPackageAddress: %v", err)
	}

	directory, err := skillPackage.ManagedSkillDirectoryLocator(layout, address)
	if err != nil {
		t.Fatalf("ManagedSkillDirectoryLocator: %v", err)
	}
	wantDirectory := spec.Locator(
		"skill/release-notes/unversioned/release-notes",
	)
	if directory != wantDirectory {
		t.Fatalf(
			"runtime directory=%q, want %q",
			directory,
			wantDirectory,
		)
	}

	documentLocator, err := skillPackage.ManagedPackageLocatorForSkill(
		layout,
		address,
	)
	if err != nil {
		t.Fatalf("ManagedPackageLocatorForSkill: %v", err)
	}
	wantDocumentLocator := spec.Locator(
		"skill/release-notes/unversioned/release-notes/SKILL.md",
	)
	if documentLocator != wantDocumentLocator {
		t.Fatalf(
			"document locator=%q, want %q",
			documentLocator,
			wantDocumentLocator,
		)
	}

	roundTripped, err := skillPackage.ManagedPackageAddressFromSkillLocator(
		layout,
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

	files, err := skillPackage.ManagedSkillStorageFiles(
		documents,
		address,
		[]managedpackageModel.ManagedPackageFile{
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

	contents := make(map[spec.Locator]string, len(files))
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

	_, err = skillPackage.ManagedPackageAddressFromSkillLocator(
		layout,
		"skill/release-notes/unversioned/not-release-notes/SKILL.md",
	)
	if !errors.Is(err, spec.ErrInvalid) {
		t.Fatalf(
			"mismatched runtime directory error=%v, want ErrInvalid",
			err,
		)
	}
}
