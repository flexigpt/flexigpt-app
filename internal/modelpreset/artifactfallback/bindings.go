package artifactfallback

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

// Binding explicitly maps one Artifact logical model name to one built-in
// provider and model preset. It is used for duplicate model names and aliases.
type Binding struct {
	Name          basespec.LogicalName
	ProviderName  inferenceSpec.ProviderName
	ModelPresetID modelpreset.ModelPresetID
}

func (b Binding) Validate() error {
	if err := b.Name.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Model fallback binding provider name",
		string(b.ProviderName),
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return bundleitemutils.ValidateTag(string(b.ModelPresetID))
}

// Populate this list when two built-in providers expose the same portable
// model name, or when a desired artifact name is an alias.
var defaultBindings []Binding

func DefaultBindings() []Binding {
	return append([]Binding(nil), defaultBindings...)
}
