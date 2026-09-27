package consumerapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

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
