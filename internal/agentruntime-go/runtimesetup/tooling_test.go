package runtimesetup

import (
	"testing"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

func TestValidateGoToolBindingCurrentAllowsRevisionOnlyChange(
	t *testing.T,
) {
	binding := goToolBindingForTest(t)
	current := binding.Used
	current.Revision++

	if err := validateGoToolBindingCurrent(binding, &current); err != nil {
		t.Fatalf(
			"validate unchanged Go Tool Definition after local revision change: %v",
			err,
		)
	}
}

func TestValidateGoToolBindingCurrentRejectsChangedDefinitionOrIdentity(
	t *testing.T,
) {
	binding := goToolBindingForTest(t)

	tests := []struct {
		name          string
		mutateBinding func(*goToolBinding)
		mutateCurrent func(*sessionRuntime.ResolvedArtifact)
	}{
		{
			name: "changed definition",
			mutateCurrent: func(value *sessionRuntime.ResolvedArtifact) {
				value.Digest = string(
					cryptoutil.DigestBytes([]byte("changed-definition")),
				)
			},
		},
		{
			name: "changed artifact identity",
			mutateCurrent: func(value *sessionRuntime.ResolvedArtifact) {
				value.ArtifactID = "0192c4c0-0002-7000-8000-000000000002"
			},
		},
		{
			name: "missing persisted witness",
			mutateBinding: func(value *goToolBinding) {
				value.Used.Digest = ""
			},
		},
		{
			name: "target and witness differ",
			mutateBinding: func(value *goToolBinding) {
				value.Used.ArtifactID = "0192c4c0-0002-7000-8000-000000000002"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := binding
			current := candidate.Used

			if test.mutateBinding != nil {
				test.mutateBinding(&candidate)
			}
			if test.mutateCurrent != nil {
				test.mutateCurrent(&current)
			}

			if err := validateGoToolBindingCurrent(candidate, &current); err == nil {
				t.Fatal("validateGoToolBindingCurrent succeeded for stale binding")
			}
		})
	}
}

func goToolBindingForTest(t *testing.T) goToolBinding {
	t.Helper()

	ref := artifactModel.ArtifactRef{
		RootID: rootModel.RootID(
			"0192c4c0-0000-7000-8000-000000000001",
		),
		ArtifactID: artifactModel.ArtifactID(
			"0192c4c0-0002-7000-8000-000000000001",
		),
	}
	targetRef := ref
	digest := cryptoutil.DigestBytes([]byte("tool-definition"))

	return goToolBinding{
		Target: composition.CapabilityTarget{
			Form:       composition.TargetFormArtifact,
			Type:       declaration.TypeTool,
			Name:       storeSpec.LogicalName("tool-one"),
			Provenance: composition.TargetProvenanceCurrentRoot,
			Artifact:   &targetRef,
		},
		Used: sessionRuntime.ResolvedArtifact{
			RootID:     string(ref.RootID),
			ArtifactID: string(ref.ArtifactID),
			Revision:   1,
			Digest:     string(digest),
		},
	}
}
