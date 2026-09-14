package artifact

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// SourceBinding is the immutable physical declaration origin for one
// root-local Artifact. Artifact kind is deliberately not part of Binding.
// The persisted typed-origin uniqueness key is:
//
//	root_id + source_id + locator + subresource_locator + kind
//
// This permits one physical declaration origin to later emit a different
// kind while retaining the prior Artifact as incompatible for diagnostics.
type SourceBinding struct {
	SourceID           source.SourceID             `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
}

func (b SourceBinding) Validate() error {
	if err := b.SourceID.Validate(); err != nil {
		return err
	}
	if err := b.Locator.Validate(false); err != nil {
		return err
	}
	return b.SubresourceLocator.Validate()
}
