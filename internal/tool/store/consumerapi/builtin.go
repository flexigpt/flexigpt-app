package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

func (a *API) requireBuiltinInstaller(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := a.protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	return a.requireBuiltinSource(ctx, rootID, sourceID)
}

func (a *API) installBuiltInPackage(
	ctx context.Context,
	request BuiltInPackageInstallRequest,
) error {
	if err := a.requireBuiltinInstaller(
		ctx,
		request.RootID,
		request.SourceID,
	); err != nil {
		return err
	}
	if err := validateBuiltInPackage(request); err != nil {
		return err
	}

	locator, err := request.Package.FileLocator(request.DocumentFile)
	if err != nil {
		return err
	}
	binding := artifact.SourceBinding{
		SourceID: request.SourceID,
		Locator:  locator,
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID:              request.RootID,
			Binding:             binding,
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

	record := published.Artifact
	if record.RootID != request.RootID ||
		record.Binding != binding ||
		record.Kind != request.ExpectedKind ||
		record.LogicalName != request.ExpectedLogicalName ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != request.ExpectedDefinition {
		return fmt.Errorf(
			"%w: published Tool catalog Artifact does not match its package",
			basespec.ErrReferenceUnresolved,
		)
	}

	switch request.ExpectedKind {
	case toolDomain.ToolArtifactKind:
		_, err = a.GetTool(ctx, record.Ref())
	case artifact.ArtifactKind(pluginv1.PluginType):
		_, err = a.GetToolCollection(ctx, record.Ref())
	}
	return err
}

func validateBuiltInPackage(request BuiltInPackageInstallRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	files, err := source.NormalizeManagedPackageFiles(request.PackageFiles)
	if err != nil {
		return err
	}
	if len(files) != 1 || files[0].Locator != request.DocumentFile {
		return fmt.Errorf(
			"%w: built-in Tool packages must contain exactly their declaration document",
			basespec.ErrInvalid,
		)
	}

	raw, err := yamlutil.CanonicalObjectJSON(
		files[0].Content,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return err
	}
	entry, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return err
	}
	if err := decoder.ValidateEntryTree(entry); err != nil {
		return err
	}
	definitionValue, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return err
	}
	if definitionValue.Kind != request.ExpectedKind ||
		definitionValue.LogicalName != request.ExpectedLogicalName ||
		definitionValue.Digest != request.ExpectedDefinition {
		return fmt.Errorf(
			"%w: Tool package declaration differs from publication expectations",
			basespec.ErrDigestMismatch,
		)
	}

	var expectedAddress source.ManagedPackageAddress
	switch request.ExpectedKind {
	case toolDomain.ToolArtifactKind:
		if request.DocumentFile != toolDomain.ToolDocumentFile() {
			return fmt.Errorf(
				"%w: unsupported Tool package document",
				basespec.ErrInvalid,
			)
		}
		document, err := toolv1.DecodeToolEntry(entry)
		if err != nil {
			return err
		}
		expectedAddress, err = toolDomain.ToolPackageAddress(
			basespec.LogicalName(document.Name),
			document.Version,
		)
		if err != nil {
			return err
		}

	case artifact.ArtifactKind(pluginv1.PluginType):
		if request.DocumentFile != toolDomain.ToolCollectionDocumentFile() {
			return fmt.Errorf(
				"%w: unsupported Tool Collection document",
				basespec.ErrInvalid,
			)
		}
		document, err := pluginv1.DecodePluginEntry(entry)
		if err != nil {
			return err
		}
		if _, err := toolDomain.ValidateToolCollectionDocument(document); err != nil {
			return err
		}
		expectedAddress, err = toolDomain.ToolCollectionPackageAddress(
			basespec.LogicalName(document.Name),
		)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf(
			"%w: unsupported built-in Tool package kind %q",
			basespec.ErrInvalid,
			request.ExpectedKind,
		)
	}

	if request.Package != expectedAddress {
		return fmt.Errorf(
			"%w: Tool package address differs from declaration identity",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (a *API) removeBuiltInPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if err := a.requireBuiltinInstaller(ctx, rootID, sourceID); err != nil {
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
	if err := a.requireBuiltinInstaller(ctx, rootID, sourceID); err != nil {
		return err
	}
	return compositionapi.EnsureSourceCurrent(
		ctx,
		a.discovery,
		rootID,
		sourceID,
	)
}
