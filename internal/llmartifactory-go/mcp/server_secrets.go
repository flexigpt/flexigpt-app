package mcp

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

func (a *Service) GetServerSecrets(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ServerSecretsView, error) {
	inputs, err := a.installation.SecretInputs(ctx, ref)
	if err != nil {
		return ServerSecretsView{}, err
	}

	output := ServerSecretsView{
		Inputs: make([]ServerSecretInputView, 0, len(inputs)),
	}
	for _, input := range inputs {
		output.Inputs = append(output.Inputs, ServerSecretInputView{
			Name:        input.Name,
			Label:       input.Label,
			Description: input.Description,
			Required:    input.Required,
			Kind:        input.Kind,
			Configured:  input.Configured,
		})
	}
	return output, nil
}
