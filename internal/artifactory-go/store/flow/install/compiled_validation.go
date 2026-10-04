package install

import (
	"fmt"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// validateCompiledSet validates the complete generic installation payload
// before bootstrap can reset or write physical topology.
//
// Definition digest admission remains Ingest's trusted-registration boundary.
// Declaration schema and semantic equivalence remain build-time concerns.
func validateCompiledSet(set installModel.CompiledPackageSet) error {
	if err := validateGeneratedEnvelope(set); err != nil {
		return err
	}
	if len(set.Packages) > spec.MaxDiscoveryCandidates {
		return fmt.Errorf("%w: compiled package count exceeds its limit", spec.ErrInvalid)
	}

	scopes := make(map[spec.Locator]struct{}, len(set.Packages))
	for packageIndex, value := range set.Packages {
		scope, err := value.Address.Directory()
		if err != nil {
			return err
		}
		if _, duplicate := scopes[scope]; duplicate {
			return fmt.Errorf("%w: repeated compiled package %q", spec.ErrIdentityConflict, scope)
		}
		scopes[scope] = struct{}{}
		if err := value.EmbeddedRoot.ValidatePortable(false); err != nil {
			return err
		}
		if err := cryptoutil.ValidateDigest(value.Fingerprint); err != nil {
			return err
		}

		files := make([]managedpackageModel.ManagedPackageFile, len(value.Files))
		witnesses := make(map[spec.Locator]cryptoutil.Digest, len(value.Files))
		for index, file := range value.Files {
			if file.Size != int64(len(file.Content)) ||
				file.Digest != cryptoutil.DigestBytes(file.Content) {
				return fmt.Errorf(
					"%w: compiled package %q file %q witness differs",
					spec.ErrDigestMismatch,
					scope,
					file.Locator,
				)
			}
			files[index] = managedpackageModel.ManagedPackageFile{
				Locator: file.Locator,
				Content: file.Content,
			}
			witnesses[file.Locator] = file.Digest
		}
		if _, err := managedpackageModel.NormalizeManagedPackageFiles(files); err != nil {
			return fmt.Errorf("compiled packages[%d]: %w", packageIndex, err)
		}

		documents := make(map[spec.Locator]struct{}, len(value.Documents))
		for _, document := range value.Documents {
			if _, duplicate := documents[document.Locator]; duplicate {
				return fmt.Errorf(
					"%w: compiled package repeats document %q",
					spec.ErrIdentityConflict,
					document.Locator,
				)
			}
			documents[document.Locator] = struct{}{}
			digest, found := witnesses[document.Locator]
			if !found || digest != document.Digest {
				return fmt.Errorf(
					"%w: compiled document %q has no matching package file",
					spec.ErrDigestMismatch,
					document.Locator,
				)
			}
			if len(document.Artifacts) == 0 ||
				len(document.Artifacts) > spec.MaxDiscoveryCandidates {
				return fmt.Errorf("%w: compiled document artifact count is invalid", spec.ErrInvalid)
			}

			type origin struct {
				subresource spec.SubresourceLocator
				kind        string
			}
			seen := make(map[origin]struct{}, len(document.Artifacts))
			for _, artifact := range document.Artifacts {
				if err := artifact.Subresource.Validate(); err != nil {
					return err
				}
				if err := definitionModel.ValidateAdmitted(artifact.Definition); err != nil {
					return err
				}
				if err := diagnostic.Validate(artifact.Diagnostics); err != nil {
					return err
				}
				key := origin{
					subresource: artifact.Subresource,
					kind:        string(artifact.Definition.Kind),
				}
				if _, duplicate := seen[key]; duplicate {
					return fmt.Errorf(
						"%w: compiled document repeats typed origin",
						spec.ErrIdentityConflict,
					)
				}
				seen[key] = struct{}{}
			}
		}
	}
	return nil
}
