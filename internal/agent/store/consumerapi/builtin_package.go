package consumerapi

import (
	"context"
	"fmt"
	"sort"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (a *API) installBuiltInAgentPackage(
	ctx context.Context,
	request BuiltInAgentPackageInstallRequest,
) ([]artifact.Artifact, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: built-in Agent package context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return nil, err
	}

	expectations, rootExpectation, err := normalizeBuiltInAgentExpectations(
		request.PluginDocumentFile,
		request.Expectations,
	)
	if err != nil {
		return nil, err
	}
	if err := a.validateBuiltInAgentPackageSource(
		ctx,
		request.RootID,
		request.SourceID,
		request.PackageAddress,
	); err != nil {
		return nil, err
	}

	documentLocator, err := request.PackageAddress.FileLocator(
		request.PluginDocumentFile,
	)
	if err != nil {
		return nil, err
	}

	if _, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID:           request.SourceID,
				Locator:            documentLocator,
				SubresourceLocator: rootExpectation.Subresource,
			},
			ExpectedKind:        rootExpectation.Kind,
			ExpectedLogicalName: rootExpectation.LogicalName,
			ExpectedDefinition:  rootExpectation.DefinitionDigest,
			Package: source.ManagedPackagePublication{
				Address: request.PackageAddress,
				Files:   request.PackageFiles,
			},
			AllowPackageReplacement: true,
			AllowProtected:          true,
		},
	); err != nil {
		return nil, err
	}

	records, _, err := a.verifyBuiltInAgentArtifacts(
		ctx,
		request.RootID,
		request.SourceID,
		request.PackageAddress,
		expectations,
		rootExpectation,
	)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (a *API) validateBuiltInAgentPackage(
	ctx context.Context,
	request BuiltInAgentPackageInstallRequest,
) error {
	return a.validateBuiltInAgentPackages(
		ctx,
		[]BuiltInAgentPackageInstallRequest{request},
	)
}

func (a *API) validateBuiltInAgentPackages(
	ctx context.Context,
	requests []BuiltInAgentPackageInstallRequest,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: built-in Agent package batch validation context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if len(requests) == 0 {
		return nil
	}
	if a.declarationResolver == nil {
		return basespec.ErrClosed
	}

	type validationPlan struct {
		request         BuiltInAgentPackageInstallRequest
		expectations    []BuiltInAgentArtifactExpectation
		rootExpectation BuiltInAgentArtifactExpectation
	}

	plans := make([]validationPlan, 0, len(requests))
	var (
		sharedRootID   root.RootID
		sharedSourceID source.SourceID
	)
	for index, request := range requests {
		expectations, rootExpectation, err := normalizeBuiltInAgentExpectations(
			request.PluginDocumentFile,
			request.Expectations,
		)
		if err != nil {
			return fmt.Errorf(
				"built-in Agent package validation request %d: %w",
				index,
				err,
			)
		}
		if err := a.validateBuiltInAgentPackageSource(
			ctx,
			request.RootID,
			request.SourceID,
			request.PackageAddress,
		); err != nil {
			return fmt.Errorf(
				"built-in Agent package validation request %d: %w",
				index,
				err,
			)
		}

		if index == 0 {
			sharedRootID = request.RootID
			sharedSourceID = request.SourceID
		} else if request.RootID != sharedRootID ||
			request.SourceID != sharedSourceID {
			return fmt.Errorf(
				"%w: built-in Agent validation batch spans multiple Sources",
				basespec.ErrInvalid,
			)
		}

		plans = append(plans, validationPlan{
			request:         request,
			expectations:    expectations,
			rootExpectation: rootExpectation,
		})
	}

	if err := a.ensureBuiltInAgentSourceCurrent(
		ctx,
		sharedRootID,
		sharedSourceID,
	); err != nil {
		return err
	}

	_, err := consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (struct{}, error) {
			for index, plan := range plans {
				if err := sessionCtx.Err(); err != nil {
					return struct{}{}, err
				}

				_, rootArtifact, err := a.verifyBuiltInAgentArtifacts(
					sessionCtx,
					plan.request.RootID,
					plan.request.SourceID,
					plan.request.PackageAddress,
					plan.expectations,
					plan.rootExpectation,
				)
				if err != nil {
					return struct{}{}, fmt.Errorf(
						"verify built-in Agent package %d: %w",
						index,
						err,
					)
				}

				capabilities, err := a.declarationResolver.ResolvePluginCapabilities(
					sessionCtx,
					rootArtifact.Ref(),
				)
				if err != nil {
					return struct{}{}, fmt.Errorf(
						"resolve built-in Agent package %d: %w",
						index,
						err,
					)
				}
				if err := resolve.RequireComplete(
					capabilities.Occurrences,
				); err != nil {
					return struct{}{}, fmt.Errorf(
						"validate built-in Agent Collection %q resolution: %w",
						rootArtifact.LogicalName,
						err,
					)
				}
			}
			return struct{}{}, nil
		},
	)
	return err
}

func (a *API) removeBuiltInAgentPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := a.validateBuiltInAgentPackageSource(
		ctx,
		rootID,
		sourceID,
		address,
	); err != nil {
		return err
	}

	return a.managedArtifacts.Remove(
		ctx,
		artifact.RemoveArtifactRequest{
			RootID:         rootID,
			SourceID:       sourceID,
			Package:        address,
			AllowProtected: true,
		},
	)
}

func (a *API) ensureBuiltInAgentSourceCurrent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := a.requireMutable(ctx, rootID, true); err != nil {
		return err
	}
	return compositionapi.EnsureSourceCurrent(
		ctx,
		a.discovery,
		rootID,
		sourceID,
	)
}

func (a *API) validateBuiltInAgentPackageSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := sourceID.Validate(); err != nil {
		return err
	}
	if err := address.Validate(); err != nil {
		return err
	}
	if !documentTopology.IsBuiltinPackageSource(rootID, sourceID) {
		return fmt.Errorf(
			"%w: Agent built-in package does not target the declared built-in package Source",
			basespec.ErrInvalid,
		)
	}
	if address.Kind != agentDomain.BuiltinAgentCollectionPackageKind {
		return fmt.Errorf(
			"%w: built-in Agent package kind must be %q",
			basespec.ErrInvalid,
			agentDomain.BuiltinAgentCollectionPackageKind,
		)
	}
	if err := a.requireMutable(ctx, rootID, true); err != nil {
		return err
	}

	sourceValue, err := a.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return fmt.Errorf(
			"%w: built-in Agent Source must be managed",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (a *API) verifyBuiltInAgentArtifacts(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
	expectations []BuiltInAgentArtifactExpectation,
	rootExpectation BuiltInAgentArtifactExpectation,
) (records []artifact.Artifact, rootArtifact artifact.Artifact, err error) {
	output := make([]artifact.Artifact, 0, len(expectations))

	for _, expected := range expectations {
		originLocator, err := address.FileLocator(expected.Locator)
		if err != nil {
			return nil, artifact.Artifact{}, err
		}

		record, err := a.artifacts.FindByOrigin(
			ctx,
			rootID,
			artifact.SourceBinding{
				SourceID:           sourceID,
				Locator:            originLocator,
				SubresourceLocator: expected.Subresource,
			},
			expected.Kind,
		)
		if err != nil {
			return nil, artifact.Artifact{}, err
		}
		if record.State != artifact.StateAvailable ||
			record.LogicalName != expected.LogicalName ||
			record.LogicalVersion != expected.LogicalVersion ||
			record.ResolvedDefinition == nil ||
			*record.ResolvedDefinition != expected.DefinitionDigest {
			return nil, artifact.Artifact{}, fmt.Errorf(
				"%w: built-in Agent Artifact %q does not match package expectation",
				basespec.ErrReferenceUnresolved,
				record.ID,
			)
		}

		if expected.Locator == rootExpectation.Locator &&
			expected.Subresource == rootExpectation.Subresource &&
			expected.Kind == rootExpectation.Kind {
			rootArtifact = record.Clone()
		}
		output = append(output, record.Clone())
	}

	if rootArtifact.ID == "" {
		return nil, artifact.Artifact{}, fmt.Errorf(
			"%w: built-in Agent package has no resolved root Plugin",
			basespec.ErrReferenceUnresolved,
		)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Binding.Locator != output[right].Binding.Locator {
			return output[left].Binding.Locator <
				output[right].Binding.Locator
		}
		if output[left].Binding.SubresourceLocator !=
			output[right].Binding.SubresourceLocator {
			return output[left].Binding.SubresourceLocator <
				output[right].Binding.SubresourceLocator
		}
		return output[left].Kind < output[right].Kind
	})
	return output, rootArtifact, nil
}

func normalizeBuiltInAgentExpectations(
	documentFile basespec.Locator,
	values []BuiltInAgentArtifactExpectation,
) (
	expectations []BuiltInAgentArtifactExpectation,
	rootExpectation BuiltInAgentArtifactExpectation,
	err error,
) {
	if err := documentFile.ValidatePortable(false); err != nil {
		return nil, BuiltInAgentArtifactExpectation{}, err
	}
	if !documentTopology.IsCollectionDocumentFile(documentFile) {
		return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
			"%w: built-in Agent Collection document is not declared in topology", basespec.ErrInvalid,
		)
	}
	if len(values) == 0 || len(values) > basespec.MaxDiscoveryEntries {
		return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
			"%w: built-in Agent package has invalid Artifact expectation count",
			basespec.ErrInvalid,
		)
	}

	type origin struct {
		locator     basespec.Locator
		subresource basespec.SubresourceLocator
		kind        artifact.ArtifactKind
	}

	output := append([]BuiltInAgentArtifactExpectation(nil), values...)
	seen := make(map[origin]struct{}, len(output))
	rootFound := false

	for index, expected := range output {
		if err := expected.Locator.ValidatePortable(false); err != nil {
			return nil, BuiltInAgentArtifactExpectation{}, err
		}
		if err := expected.Subresource.Validate(); err != nil {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"built-in Agent expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := expected.Kind.Validate(); err != nil {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"built-in Agent expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := expected.LogicalName.Validate(); err != nil {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"built-in Agent expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := expected.LogicalVersion.Validate(true); err != nil {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"built-in Agent expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := cryptoutil.ValidateDigest(expected.DefinitionDigest); err != nil {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"built-in Agent expectations[%d]: %w",
				index,
				err,
			)
		}

		key := origin{
			locator:     expected.Locator,
			subresource: expected.Subresource,
			kind:        expected.Kind,
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"%w: built-in Agent package repeats Artifact origin %q/%q",
				basespec.ErrInvalid,
				expected.Subresource,
				expected.Kind,
			)
		}
		seen[key] = struct{}{}

		if expected.Locator != documentFile ||
			expected.Subresource != "" {
			continue
		}
		if expected.Kind != artifact.ArtifactKind(declaration.TypePlugin) ||
			expected.LogicalVersion != "" {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"%w: built-in Agent package root must be an unversioned Plugin",
				basespec.ErrInvalid,
			)
		}
		if rootFound {
			return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
				"%w: built-in Agent package has multiple root Plugins",
				basespec.ErrInvalid,
			)
		}
		rootExpectation = expected
		rootFound = true
	}

	if !rootFound {
		return nil, BuiltInAgentArtifactExpectation{}, fmt.Errorf(
			"%w: built-in Agent package has no root Plugin Artifact",
			basespec.ErrInvalid,
		)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Locator != output[right].Locator {
			return output[left].Locator < output[right].Locator
		}
		if output[left].Subresource != output[right].Subresource {
			return output[left].Subresource < output[right].Subresource
		}
		return output[left].Kind < output[right].Kind
	})
	return output, rootExpectation, nil
}
