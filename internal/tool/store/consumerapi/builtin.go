package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

func (a *API) installBuiltInPackage(
	ctx context.Context,
	request BuiltInPackageInstallRequest,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := a.requireBuiltinSource(
		ctx,
		request.RootID,
		request.SourceID,
	); err != nil {
		return err
	}

	switch request.ExpectedKind {
	case toolDomain.ToolArtifactKind:
		if request.Package.Kind != toolDomain.ToolPackageKind ||
			request.DocumentFile != toolDomain.ToolDocumentFile() {
			return fmt.Errorf(
				"%w: built-in Tool package has invalid kind or document file",
				basespec.ErrInvalid,
			)
		}

	case artifact.ArtifactKind(pluginv1.PluginType):
		if request.Package.Kind != toolDomain.ToolCollectionPackageKind ||
			request.DocumentFile != toolDomain.ToolCollectionDocumentFile() {
			return fmt.Errorf(
				"%w: built-in Tool Collection package has invalid kind or document file",
				basespec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: built-in Tool package kind %q is unsupported",
			basespec.ErrInvalid,
			request.ExpectedKind,
		)
	}

	locator, err := request.Package.FileLocator(request.DocumentFile)
	if err != nil {
		return err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID: request.SourceID,
				Locator:  locator,
			},
			ExpectedKind:        request.ExpectedKind,
			ExpectedLogicalName: request.ExpectedLogicalName,
			ExpectedDefinition:  request.ExpectedDefinition,
			Package: source.ManagedPackagePublication{
				Address: request.Package,
				Files:   request.PackageFiles,
			},
			AllowPackageReplacement: true,
			AllowProtected:          true,
		},
	)
	if err != nil {
		return err
	}

	switch request.ExpectedKind {
	case toolDomain.ToolArtifactKind:
		_, err = a.GetTool(ctx, published.Artifact.Ref())
	case artifact.ArtifactKind(pluginv1.PluginType):
		_, err = a.GetToolCollection(ctx, published.Artifact.Ref())
	}
	return err
}

func (a *API) removeBuiltInPackage(
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
	if err := a.requireBuiltinSource(ctx, rootID, sourceID); err != nil {
		return err
	}
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != toolDomain.ToolPackageKind &&
		address.Kind != toolDomain.ToolCollectionPackageKind {
		return fmt.Errorf(
			"%w: package kind %q is not owned by Tool Store",
			basespec.ErrInvalid,
			address.Kind,
		)
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

func (a *API) ensureBuiltInSourceCurrent(
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
	if err := a.requireBuiltinSource(ctx, rootID, sourceID); err != nil {
		return err
	}
	return compositionapi.EnsureSourceCurrent(
		ctx,
		a.discovery,
		rootID,
		sourceID,
	)
}
