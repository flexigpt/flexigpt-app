package source

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

func TestManagedPackageAddressUsesDirectSemanticDirectory(t *testing.T) {
	t.Parallel()

	address, err := NewManagedPackageAddress(
		"agent.skill",
		"meeting-summary",
		"unversioned",
	)
	if err != nil {
		t.Fatalf("NewManagedPackageAddress: %v", err)
	}

	directory, err := address.Directory()
	if err != nil {
		t.Fatalf("Directory: %v", err)
	}
	if directory != "agent.skill/meeting-summary/unversioned" {
		t.Fatalf("Directory=%q", directory)
	}

	if directory == "packages/agent.skill/meeting-summary/unversioned" {
		t.Fatal("generic packages directory must not exist")
	}

	locator, err := address.FileLocator("SKILL.md")
	if err != nil {
		t.Fatalf("FileLocator: %v", err)
	}
	if locator != "agent.skill/meeting-summary/unversioned/SKILL.md" {
		t.Fatalf("FileLocator=%q", locator)
	}

	parsed, err := ParseManagedPackageAddressDirectory(directory)
	if err != nil {
		t.Fatalf("ParseManagedPackageAddressDirectory: %v", err)
	}
	if parsed != address {
		t.Fatalf("parsed=%#v want %#v", parsed, address)
	}

	if _, err := NewManagedPackageAddress(
		basespec.PackageKind("agent.skill"),
		"meeting-summary",
		"",
	); err == nil {
		t.Fatal("empty package version was accepted")
	}
}
