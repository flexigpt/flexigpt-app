package artifact

import "testing"

func TestManagedSkillPackageLayout(t *testing.T) {
	t.Parallel()

	address, err := ManagedPackageAddressForSkill("summarize", "")
	if err != nil {
		t.Fatalf("ManagedPackageAddressForSkill: %v", err)
	}

	directory, err := address.Directory()
	if err != nil {
		t.Fatalf("Directory: %v", err)
	}
	if directory != "agent.skill/summarize/unversioned" {
		t.Fatalf("directory=%q", directory)
	}

	locator, err := ManagedPackageLocatorForSkill(address)
	if err != nil {
		t.Fatalf("ManagedPackageLocatorForSkill: %v", err)
	}
	if locator != "agent.skill/summarize/unversioned/SKILL.md" {
		t.Fatalf("locator=%q", locator)
	}

	parsed, err := ManagedPackageAddressFromSkillLocator(locator)
	if err != nil {
		t.Fatalf("ManagedPackageAddressFromSkillLocator: %v", err)
	}
	if parsed != address {
		t.Fatalf("parsed=%#v want %#v", parsed, address)
	}
}
