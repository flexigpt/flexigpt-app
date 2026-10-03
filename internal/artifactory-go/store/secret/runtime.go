package secret

import (
	"context"

	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
)

// RuntimeAPI returns plaintext and is for trusted runtime composition only.
type RuntimeAPI interface {
	ReadBinding(
		ctx context.Context,
		key secretModel.BindingKey,
		expectedRef secretModel.Ref,
	) (string, secretModel.Binding, error)
}
