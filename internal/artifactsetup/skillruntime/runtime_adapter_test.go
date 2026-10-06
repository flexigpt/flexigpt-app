package skillruntime

import (
	"errors"
	"testing"

	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func testArtifactRef(id string) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     "0192c4c0-0000-7000-8000-000000000002",
		ArtifactID: artifactModel.ArtifactID(id),
	}
}

func TestSelectionRootsRejectsEmptyAndDuplicateSelections(t *testing.T) {
	_, err := selectionRoots(nil)
	if !errors.Is(err, ErrArtifactSkillSelectionRequired) {
		t.Fatalf("empty selection error=%v", err)
	}

	ref := testArtifactRef("0192c4c0-0002-7000-8000-000000000001")
	_, err = selectionRoots([]artifactModel.ArtifactRef{ref, ref})
	if !errors.Is(err, spec.ErrInvalid) {
		t.Fatalf("duplicate selection error=%v, want ErrInvalid", err)
	}
}

func TestSelectionRootsVisitsEachRootOnce(t *testing.T) {
	first := testArtifactRef("0192c4c0-0002-7000-8000-000000000001")
	second := testArtifactRef("0192c4c0-0002-7000-8000-000000000002")

	roots, err := selectionRoots([]artifactModel.ArtifactRef{first, second})
	if err != nil {
		t.Fatalf("selectionRoots: %v", err)
	}
	if len(roots) != 1 || roots[0] != first.RootID {
		t.Fatalf("roots=%+v, want one owning Root", roots)
	}
}

func TestSelectionRejectsDefinitionCollisions(t *testing.T) {
	definition := provider.SkillDef{
		Type:     "fs",
		Name:     "notes",
		Location: "/skills/notes",
	}
	_, err := selectionFromValues([]ResolvedArtifactSkill{
		{
			Artifact:   testArtifactRef("0192c4c0-0002-7000-8000-000000000001"),
			Definition: definition,
		},
		{
			Artifact:   testArtifactRef("0192c4c0-0002-7000-8000-000000000002"),
			Definition: definition,
		},
	})
	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf("collision error=%v, want ErrConflict", err)
	}
}

func TestArtifactRefsCannotEscapeSelection(t *testing.T) {
	ref := testArtifactRef("0192c4c0-0002-7000-8000-000000000001")
	selectedDef := provider.SkillDef{
		Type:     "fs",
		Name:     "selected",
		Location: "/skills/selected",
	}
	unselectedDef := provider.SkillDef{
		Type:     "fs",
		Name:     "unselected",
		Location: "/skills/unselected",
	}

	selected, err := selectionFromValues([]ResolvedArtifactSkill{{
		Artifact:   ref,
		Definition: selectedDef,
	}})
	if err != nil {
		t.Fatalf("selectionFromValues: %v", err)
	}
	refs := selected.ArtifactRefs([]agentskillsRuntimeSpec.SkillRecord{
		{Def: unselectedDef},
		{Def: selectedDef},
	})
	if len(refs) != 1 || refs[0] != ref {
		t.Fatalf("mapped refs=%+v, want only %+v", refs, ref)
	}
}
