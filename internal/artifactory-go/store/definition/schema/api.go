package schema

import (
	"context"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
)

type API interface {
	CanonicalizeExpected(
		ctx context.Context,
		expected schemaModel.Key,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}
