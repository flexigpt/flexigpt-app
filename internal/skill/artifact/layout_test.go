package artifact

import (
	"testing"

	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

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
	if string(directory) != string(builtinSchema.AgentSkillPackageKind)+
		"/summarize/"+string(builtinSchema.UnversionedPackageVersion) {
		t.Fatalf("directory=%q", directory)
	}

	locator, err := ManagedPackageLocatorForSkill(address)
	if err != nil {
		t.Fatalf("ManagedPackageLocatorForSkill: %v", err)
	}
	if string(locator) != string(directory)+"/"+string(builtinSchema.AgentSkillDefinitionFileName) {
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
