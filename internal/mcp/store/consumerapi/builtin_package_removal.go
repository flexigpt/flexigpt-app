package consumerapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

func (a *API) RemoveBuiltInPackage(
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
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := sourceID.Validate(); err != nil {
		return err
	}
	if err := address.Validate(); err != nil {
		return err
	}
	if address.Kind != mcpDomain.MCPCollectionPackageKind {
		return fmt.Errorf(
			"%w: built-in MCP package kind must be %q",
			basespec.ErrInvalid,
			mcpDomain.MCPCollectionPackageKind,
		)
	}
	if !a.protection.IsProtectedRoot(rootID) {
		return fmt.Errorf(
			"%w: MCP built-in Root is not protected",
			basespec.ErrProtected,
		)
	}

	documentLocator, err := address.FileLocator(
		mcpDomain.MCPCollectionDocumentFile,
	)
	if err != nil {
		return err
	}
	previous, err := a.builtInPackageServers(
		ctx,
		rootID,
		sourceID,
		documentLocator,
	)
	if err != nil {
		return err
	}
	if err := a.managedArtifacts.Remove(
		ctx,
		artifact.RemoveArtifactRequest{
			RootID:         rootID,
			SourceID:       sourceID,
			Package:        address,
			AllowProtected: true,
		},
	); err != nil {
		return err
	}

	for _, previousServer := range previous {
		current, err := a.artifacts.Get(ctx, previousServer.Ref())
		if err != nil {
			return err
		}
		if current.State == artifact.StateAvailable {
			return fmt.Errorf(
				"%w: removed built-in MCP package still provides server %q",
				basespec.ErrConflict,
				current.ID,
			)
		}
		if err := a.purgeBuiltInServerInstallation(
			ctx,
			previousServer.Ref(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) builtInPackageServers(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	documentLocator basespec.Locator,
) ([]artifact.Artifact, error) {
	records, err := a.artifacts.ListBySource(ctx, rootID, sourceID)
	if err != nil {
		return nil, err
	}
	output := make([]artifact.Artifact, 0)
	for _, record := range records {
		if record.Kind != mcpDomain.MCPArtifactKind ||
			record.Binding.Locator != documentLocator {
			continue
		}
		output = append(output, record.Clone())
	}
	return output, nil
}

func (a *API) cleanupRemovedBuiltInPackageServers(
	ctx context.Context,
	previous []artifact.Artifact,
	current []artifact.Artifact,
) error {
	retained := make(map[artifact.ArtifactRef]struct{}, len(current))
	for _, record := range current {
		if record.Kind != mcpDomain.MCPArtifactKind ||
			record.State != artifact.StateAvailable {
			continue
		}
		retained[record.Ref()] = struct{}{}
	}

	for _, previousServer := range previous {
		if _, found := retained[previousServer.Ref()]; found {
			continue
		}
		record, err := a.artifacts.Get(ctx, previousServer.Ref())
		if err != nil {
			return err
		}
		if record.State == artifact.StateAvailable {
			continue
		}
		if err := a.purgeBuiltInServerInstallation(
			ctx,
			previousServer.Ref(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) purgeBuiltInServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if a.overlays == nil {
		return fmt.Errorf(
			"%w: protected MCP installation overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}

	overlay, found, err := a.overlays.GetServerOverlay(ctx, ref)
	if err != nil {
		return err
	}
	if found {
		if err := a.cleanupServerSecretReferences(
			ctx,
			ref,
			overlay.ServerData,
		); err != nil {
			return err
		}
		return a.overlays.DeleteServerOverlay(
			ctx,
			ref,
			overlay.Revision,
		)
	}
	return a.cleanupServerSecretReferences(
		ctx,
		ref,
		mcpDomainServer.DefaultServerData(),
	)
}

func (a *API) cleanupServerSecretReferences(
	ctx context.Context,
	ref artifact.ArtifactRef,
	data mcpDomainServer.ServerData,
) error {
	if a == nil || a.secretCleaner == nil {
		return fmt.Errorf(
			"%w: MCP secret cleaner is unavailable",
			basespec.ErrClosed,
		)
	}
	refs, err := data.SecretReferences()
	if err != nil {
		return err
	}

	var result error
	for _, secretRef := range refs {
		result = errors.Join(
			result,
			a.secretCleaner.DeleteSecret(ctx, secretRef),
		)
	}
	tokenRef, err := mcpDomainSecret.NewMCPSecretRefString(
		ref,
		mcpDomainSecret.MCPSecretKindOAuthToken,
		"token",
	)
	if err != nil {
		return errors.Join(result, err)
	}
	return errors.Join(
		result,
		a.secretCleaner.DeleteSecret(ctx, tokenRef),
	)
}
