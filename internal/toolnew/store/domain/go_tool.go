package domain

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type GoToolDescriptor struct {
	Function    string
	Name        basespec.LogicalName
	Version     basespec.LogicalVersion
	DisplayName string
	Description string
	Tags        []string
	AutoExecute bool
	InputSchema json.RawMessage
}

func (d GoToolDescriptor) Validate() error {
	if err := basespec.ValidateRequiredText(
		"Go Tool function",
		d.Function,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := d.Name.Validate(); err != nil {
		return err
	}
	if err := d.Version.Validate(false); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"Go Tool display name",
		d.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"Go Tool description",
		d.Description,
		basespec.MaxDescriptionBytes,
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
		name basespec.LogicalName,
	) (GoToolDescriptor, error)
}
