package model

import (
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
	SourceID           source.SourceID         `json:"sourceID"`
	Locator            spec.Locator            `json:"locator"`
	SubresourceLocator spec.SubresourceLocator `json:"subresourceLocator,omitempty"`
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
