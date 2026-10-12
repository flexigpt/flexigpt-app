package domain

import (
	"context"
	"encoding/json"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type GoToolDescriptor struct {
	Function    string
	Name        storeSpec.LogicalName
	Version     storeSpec.LogicalVersion
	DisplayName string
	Description string
	Tags        []string
	AutoExecute bool
	InputSchema json.RawMessage
}

func (d GoToolDescriptor) Validate() error {
	if err := storeSpec.ValidateRequiredText(
		"Go Tool function",
		d.Function,
		storeSpec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := d.Name.Validate(); err != nil {
		return err
	}
	if err := d.Version.Validate(false); err != nil {
		return err
	}
	if err := storeSpec.ValidateOptionalText(
		"Go Tool display name",
		d.DisplayName,
		storeSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := storeSpec.ValidateOptionalText(
		"Go Tool description",
		d.Description,
		storeSpec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	return declaration.ValidateJSONSchemaValue(
		"Go Tool inputSchema",
		d.InputSchema,
	)
}

// GoToolLocator is the runtime-neutral Go Tool validation port required by
// Tool Store. Tool Store never imports llmtools-go.
type GoToolLocator interface {
	LookupGoTool(
		ctx context.Context,
		function string,
	) (GoToolDescriptor, error)

	LookupGoToolByName(
		ctx context.Context,
		name storeSpec.LogicalName,
	) (GoToolDescriptor, error)
}
