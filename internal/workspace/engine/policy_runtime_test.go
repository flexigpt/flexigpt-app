package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

func TestArtifactPolicyDerivationAndNames(t *testing.T) {
	t.Parallel()

	validatorCalls := 0
	policy, err := NewArtifactPolicy(spec.ArtifactSupport{
		Kind:      "test.kind",
		SchemaID:  "test.schema",
		DecoderID: "test.decoder",
		Validator: func(value definition.Definition) error {
			validatorCalls++
			if value.LogicalName == "bad" {
				return errors.New("invalid test definition")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewArtifactPolicy: %v", err)
	}
	if !policy.Supports("test.kind") || policy.Supports("other.kind") {
		t.Fatalf("Supports returned unexpected result")
	}
	if _, err := NewArtifactPolicy(); !errors.Is(err, spec.ErrInvalidWorkspace) {
		t.Fatalf("empty policy error=%v, want ErrInvalidWorkspace", err)
	}
	if _, err := NewArtifactPolicy(
		spec.ArtifactSupport{
			Kind:      "test.kind",
			SchemaID:  "test.schema",
			DecoderID: "test.decoder",
			Validator: func(definition.Definition) error { return nil },
		},
		spec.ArtifactSupport{
			Kind:      "test.kind",
			SchemaID:  "other.schema",
			DecoderID: "other.decoder",
			Validator: func(definition.Definition) error { return nil },
		},
	); !errors.Is(err, spec.ErrInvalidWorkspace) {
		t.Fatalf("duplicate policy error=%v, want ErrInvalidWorkspace", err)
	}

	key := catalog.OccurrenceKey{
		CollectionID: "019d3150-7101-7a6b-a34e-d9032342bc31",
		SourceID:     "019d3150-7102-7a6b-a34e-d9032342bc31",
		Locator:      "docs/entry.md",
	}
	value := definition.Definition{Kind: "test.kind", SchemaID: "test.schema", LogicalName: "entry"}
	draft, adopt, diagnostics := policy.Derive(
		t.Context(),
		collection.Collection{},
		catalog.Occurrence{Kind: "test.kind", Key: key},
		value,
	)
	if !adopt || len(diagnostics) != 0 || draft.Name == "" || !draft.Enabled || validatorCalls != 1 {
		t.Fatalf(
			"Derive draft=%#v adopt=%t diagnostics=%#v validatorCalls=%d",
			draft,
			adopt,
			diagnostics,
			validatorCalls,
		)
	}
	data, err := DecodeArtifactData(draft.Data)
	if err != nil || data.RuntimeDisabled {
		t.Fatalf("derived data=%#v err=%v", data, err)
	}

	_, adopt, diagnostics = policy.Derive(
		t.Context(),
		collection.Collection{},
		catalog.Occurrence{Kind: "other.kind", Key: key},
		value,
	)
	if adopt || len(diagnostics) != 0 {
		t.Fatalf("unsupported occurrence adopt=%t diagnostics=%#v", adopt, diagnostics)
	}
	mismatch := value
	mismatch.Kind = "other.kind"
	_, adopt, diagnostics = policy.Derive(
		t.Context(),
		collection.Collection{},
		catalog.Occurrence{Kind: "test.kind", Key: key},
		mismatch,
	)
	if adopt || len(diagnostics) != 1 || diagnostics[0].Code != DiagnosticCodeArtifactKindMismatch {
		t.Fatalf("kind mismatch adopt=%t diagnostics=%#v", adopt, diagnostics)
	}
	badSchema := value
	badSchema.SchemaID = "other.schema"
	_, adopt, diagnostics = policy.Derive(
		t.Context(),
		collection.Collection{},
		catalog.Occurrence{Kind: "test.kind", Key: key},
		badSchema,
	)
	if adopt || len(diagnostics) != 1 || diagnostics[0].Code != DiagnosticCodeArtifactSchemaUnsupported {
		t.Fatalf("schema mismatch adopt=%t diagnostics=%#v", adopt, diagnostics)
	}
	invalid := value
	invalid.LogicalName = "bad"
	_, adopt, diagnostics = policy.Derive(
		t.Context(),
		collection.Collection{},
		catalog.Occurrence{Kind: "test.kind", Key: key},
		invalid,
	)
	if adopt || len(diagnostics) != 1 || diagnostics[0].Code != DiagnosticCodeProjectionInvalid {
		t.Fatalf("validator failure adopt=%t diagnostics=%#v", adopt, diagnostics)
	}

	name := artifactName(basespec.LogicalName(strings.Repeat("界", basespec.MaxDisplayNameBytes)), key)
	if len(name) > basespec.MaxDisplayNameBytes || !utf8.ValidString(name) ||
		!strings.Contains(name, artifactNameSeparator) {
		t.Fatalf("bounded artifact name=%q", name)
	}
	if artifactName(
		"",
		key,
	) == artifactName(
		"",
		catalog.OccurrenceKey{CollectionID: key.CollectionID, SourceID: key.SourceID, Locator: "other.md"},
	) {
		t.Fatal("artifact names did not include occurrence identity")
	}
}

func TestRuntimePolicyDecisionsAndValidation(t *testing.T) {
	t.Parallel()

	for _, decision := range []RuntimeDecision{
		{Disposition: RuntimeAllowed},
		{Disposition: RuntimeDenied, Code: "runtime.denied", Message: "blocked"},
		{Disposition: RuntimeUnavailable, Code: "runtime.unavailable", Message: "unavailable"},
	} {
		if err := decision.Validate(); err != nil {
			t.Errorf("valid decision %#v: %v", decision, err)
		}
	}
	for _, decision := range []RuntimeDecision{
		{Disposition: RuntimeAllowed, Code: "unexpected"},
		{Disposition: RuntimeDenied},
		{Disposition: "other"},
	} {
		if err := decision.Validate(); !errors.Is(err, spec.ErrInvalidWorkspace) &&
			!errors.Is(err, basespec.ErrInvalid) {
			t.Errorf("invalid decision %#v error=%v", decision, err)
		}
	}

	policy := NewArtifactRuntimePolicy()
	available := runtimeTestArtifact(t, true, artifact.StateAvailable, false)
	workspace := spec.Workspace{Collection: collection.Collection{Enabled: true}}
	if decision := policy.Decide(
		t.Context(),
		RuntimePolicyRequest{Workspace: workspace, Artifact: available},
	); decision.Disposition != RuntimeAllowed {
		t.Fatalf("allowed decision=%#v", decision)
	}

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if decision := policy.Decide(
		cancelled,
		RuntimePolicyRequest{Workspace: workspace, Artifact: available},
	); decision.Disposition != RuntimeUnavailable ||
		decision.Code != DiagnosticCodeRuntimeUnavailable {
		t.Fatalf("cancelled decision=%#v", decision)
	}
	workspace.Collection.Enabled = false
	if decision := policy.Decide(
		t.Context(),
		RuntimePolicyRequest{Workspace: workspace, Artifact: available},
	); decision.Disposition != RuntimeUnavailable {
		t.Fatalf("disabled workspace decision=%#v", decision)
	}
	workspace.Collection.Enabled = true
	unavailable := runtimeTestArtifact(t, false, artifact.StateAvailable, false)
	if decision := policy.Decide(
		t.Context(),
		RuntimePolicyRequest{Workspace: workspace, Artifact: unavailable},
	); decision.Disposition != RuntimeUnavailable {
		t.Fatalf("disabled artifact decision=%#v", decision)
	}
	disabled := runtimeTestArtifact(t, true, artifact.StateAvailable, true)
	if decision := policy.Decide(
		t.Context(),
		RuntimePolicyRequest{Workspace: workspace, Artifact: disabled},
	); decision.Disposition != RuntimeDenied ||
		decision.Code != DiagnosticCodeRuntimeDenied {
		t.Fatalf("runtime-disabled decision=%#v", decision)
	}
	invalidData := available
	invalidData.Data = []byte(`[]`)
	if decision := policy.Decide(
		t.Context(),
		RuntimePolicyRequest{Workspace: workspace, Artifact: invalidData},
	); decision.Disposition != RuntimeUnavailable {
		t.Fatalf("invalid data decision=%#v", decision)
	}

	d := RuntimeDecisionDiagnostic(
		RuntimeDecision{Disposition: RuntimeUnavailable, Code: "runtime.unavailable", Message: "failure"},
		available,
	)
	if d.Severity != diagnostic.DiagnosticError || d.Location == nil ||
		d.Location.Locator != available.Binding.Locator {
		t.Fatalf("runtime diagnostic=%#v", d)
	}
}

func runtimeTestArtifact(t *testing.T, enabled bool, state artifact.State, runtimeDisabled bool) artifact.Artifact {
	t.Helper()
	raw, err := EncodeArtifactData(spec.ArtifactData{RuntimeDisabled: runtimeDisabled})
	if err != nil {
		t.Fatalf("EncodeArtifactData: %v", err)
	}
	digest := cryptoutil.DigestBytes([]byte("definition"))
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return artifact.Artifact{
		ID:           "019d3150-7103-7a6b-a34e-d9032342bc31",
		RootID:       "019d3150-7104-7a6b-a34e-d9032342bc31",
		CollectionID: "019d3150-7105-7a6b-a34e-d9032342bc31",
		Binding: artifact.SourceBinding{
			SourceID:     "019d3150-7106-7a6b-a34e-d9032342bc31",
			Locator:      "AGENTS.md",
			ExpectedKind: "test.kind",
		},
		Kind:               "test.kind",
		Name:               "Artifact",
		Enabled:            enabled,
		Adoption:           artifact.AdoptionObserved,
		ResolvedDefinition: &digest,
		Data:               raw,
		State:              state,
		Revision:           1,
		CreatedAt:          now,
		ModifiedAt:         now,
	}
}
