package internal

import (
	"fmt"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

// compiledDocumentsFromPackages is the only Install-to-Ingest translation.
// Package-relative generated locators become source-relative admitted-document
// evidence before scanner registration.
func compiledDocumentsFromPackages(values []installModel.CompiledPackage) ([]ingest.CompiledDocument, error) {
	output := make([]ingest.CompiledDocument, 0)
	seen := make(map[spec.Locator]struct{})
	for _, packageValue := range values {
		for _, document := range packageValue.Documents {
			locator, err := packageValue.Address.FileLocator(document.Locator)
			if err != nil {
				return nil, err
			}
			if _, duplicate := seen[locator]; duplicate {
				return nil, fmt.Errorf(
					"%w: compiled package catalog repeats source locator %q",
					spec.ErrIdentityConflict,
					locator,
				)
			}
			seen[locator] = struct{}{}
			artifacts := make([]ingest.CompiledArtifact, len(document.Artifacts))
			for artifactIndex, value := range document.Artifacts {
				artifacts[artifactIndex] = ingest.CompiledArtifact{
					Subresource: value.Subresource,
					Definition:  value.Definition.Clone(),
					Diagnostics: diagnostic.Clone(value.Diagnostics),
				}
			}
			output = append(
				output,
				ingest.CompiledDocument{Locator: locator, Digest: document.Digest, Artifacts: artifacts},
			)
		}
	}
	return output, nil
}
