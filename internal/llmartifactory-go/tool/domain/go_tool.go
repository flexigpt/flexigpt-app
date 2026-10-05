package domain

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type GoToolDescriptor struct {
	Function    string
	Name        spec.LogicalName
	Version     spec.LogicalVersion
	DisplayName string
	Description string
	Tags        []string
	AutoExecute bool
	InputSchema json.RawMessage
}

func (d GoToolDescriptor) Validate() error {
	if err := spec.ValidateRequiredText(
		"Go Tool function",
		d.Function,
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := d.Name.Validate(); err != nil {
		return err
	}
	if err := d.Version.Validate(false); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"Go Tool display name",
		d.DisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"Go Tool description",
		d.Description,
		spec.MaxDescriptionBytes,
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
		name spec.LogicalName,
	) (GoToolDescriptor, error)
}
