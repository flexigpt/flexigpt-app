package toolcatalog

import (
	"encoding/json"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/catalogtest"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/toolruntime"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	goTools, err := toolruntime.NewAdapter()
	if err != nil {
		t.Fatal(err)
	}

	registry, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		t.Fatalf("create Tool catalog interpretation registry: %v", err)
	}

	expected, err := Compile(
		t.Context(),
		t.TempDir(),
		goTools,
		registry,
	)
	if err != nil {
		t.Fatalf("compile embedded Tool packages: %v", err)
	}

	_, expectedFingerprint, err := installFlow.CanonicalGeneratedPackageSet(expected)
	if err != nil {
		t.Fatalf("fingerprint expected generated Tool catalog: %v", err)
	}

	recompiled, err := Compile(
		t.Context(),
		t.TempDir(),
		goTools,
		registry,
	)
	if err != nil {
		t.Fatalf("compile embedded Tool packages again: %v", err)
	}

	_, recompiledFingerprint, err := installFlow.CanonicalGeneratedPackageSet(
		recompiled,
	)
	if err != nil {
		t.Fatalf(
			"fingerprint repeated generated Tool catalog: %v",
			err,
		)
	}

	if expectedFingerprint != recompiledFingerprint {
		t.Fatalf(
			"generated Tool catalog is nondeterministic: first %q, second %q",
			expectedFingerprint,
			recompiledFingerprint,
		)
	}

	actualFingerprint, actualErr := generatedCatalog.Fingerprint()

	candidate, err := catalogtest.CandidatePath(
		"catalog_generated.next.json",
	)
	if err != nil {
		t.Fatalf(
			"resolve generated Tool catalog candidate path: %v",
			err,
		)
	}

	if err := catalogtest.AssertGeneratedPackageSetMatches(
		expected,
		actualErr,
		actualFingerprint,
		"internal/artifactbuiltin/toolcatalog/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedCatalogRetainsGoAndSDKTools(t *testing.T) {
	t.Parallel()

	var catalog struct {
		Packages []struct {
			Address struct {
				Kind string `json:"kind"`
			} `json:"address"`
			Documents []struct {
				Artifacts []struct {
					Definition struct {
						Kind string          `json:"kind"`
						Body json.RawMessage `json:"body"`
					} `json:"definition"`
				} `json:"artifacts"`
			} `json:"documents"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(generatedCatalogJSON, &catalog); err != nil {
		t.Fatalf("decode generated Tool catalog: %v", err)
	}

	index, err := GeneratedToolPluginIndex()
	if err != nil {
		t.Fatalf("load generated Tool Plugin index: %v", err)
	}

	goCount := 0
	sdkCount := 0
	for _, packageValue := range catalog.Packages {
		if packageValue.Address.Kind != string(toolDomain.ToolPackageKind) {
			continue
		}
		for _, documentValue := range packageValue.Documents {
			for _, artifactValue := range documentValue.Artifacts {
				if artifactValue.Definition.Kind !=
					string(toolDomain.ToolArtifactKind) {
					continue
				}

				document, err := toolv1.DecodeAdmittedToolJSON(
					artifactValue.Definition.Body,
				)
				if err != nil {
					t.Fatalf(
						"decode generated Tool declaration: %v",
						err,
					)
				}

				name := spec.LogicalName(document.Name)
				if _, found := index[name]; !found {
					t.Fatalf(
						"generated Tool %q has no generated Plugin membership",
						name,
					)
				}

				switch document.Implementation.Kind {
				case toolv1.ImplementationKindGo:
					goCount++

				case toolv1.ImplementationKindSDK:
					sdkCount++
					if document.Implementation.SDKType == "" {
						t.Fatalf("SDK Tool %q has no sdkType", name)
					}
					if err := document.Implementation.SDKToolType.Validate(); err != nil {
						t.Fatalf(
							"SDK Tool %q sdkToolType: %v",
							name,
							err,
						)
					}

				default:
					t.Fatalf(
						"generated Tool %q has unsupported implementation %q",
						name,
						document.Implementation.Kind,
					)
				}
			}
		}
	}

	if goCount == 0 {
		t.Fatal("generated Tool catalog contains no Go Tools")
	}
	if sdkCount == 0 {
		t.Fatal("generated Tool catalog contains no SDK Tools")
	}
}
