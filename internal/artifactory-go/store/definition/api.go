package definition

import (
	"context"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type API interface {
	GetDefinition(
		ctx context.Context,
		rootID rootModel.RootID,
		digest cryptoutil.Digest,
	) (definitionModel.Definition, error)

	GetDefinitions(
		ctx context.Context,
		keys []definitionModel.Key,
	) ([]definitionModel.Definition, error)
}
