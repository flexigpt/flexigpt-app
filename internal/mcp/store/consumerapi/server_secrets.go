package consumerapi

import (
	"context"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

func (a *API) GetServerSecrets(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerSecretsView, error) {
	material, err := a.resolveServerMaterial(ctx, ref)
	if err != nil {
		return ServerSecretsView{}, err
	}

	names := make([]string, 0)
	for name, declaration := range material.Document.Configuration.Install.Inputs {
		switch declaration.Kind {
		case mcpDomainServer.InputSecret,
			mcpDomainServer.InputOAuthClientCredentials:
			names = append(names, name)
		default:
		}
	}
	sort.Strings(names)

	output := ServerSecretsView{
		Inputs: make([]ServerSecretInputView, 0, len(names)),
	}
	for _, name := range names {
		declaration := material.Document.Configuration.Install.Inputs[name]
		binding, found := material.Installation.Inputs[name]
		output.Inputs = append(output.Inputs, ServerSecretInputView{
			Name:        name,
			Label:       declaration.Label,
			Description: declaration.Description,
			Required:    declaration.Required,
			Kind:        declaration.Kind,
			Configured:  found && binding.SecretRef != "",
		})
	}
	return output, nil
}
