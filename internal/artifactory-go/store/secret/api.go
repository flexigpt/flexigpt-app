package secret

import (
	"context"

	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
)

type API interface {
	GetBinding(
		ctx context.Context,
		key secretModel.BindingKey,
	) (secretModel.Binding, bool, error)

	ReplaceBinding(
		ctx context.Context,
		request secretModel.ReplaceBindingRequest,
	) (secretModel.Binding, error)

	ClearBinding(
		ctx context.Context,
		request secretModel.ClearBindingRequest,
	) error
}
