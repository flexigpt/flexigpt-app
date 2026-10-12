package model

import (
	"fmt"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// DiscoveryPreparationIntent records the caller's non-destructive discovery
// mutation intent. Generic Source preparation only supports additive work;
// replacement and pruning remain explicit Source operations.
type DiscoveryPreparationIntent string

const (
	DiscoveryPreparationAdditive DiscoveryPreparationIntent = "additive"
)

// DiscoveryRequirement is a family-neutral declaration-discovery requirement.
// It contains no Source driver configuration, package policy, artifact kind,
// decoder implementation, or refresh behavior.
type DiscoveryRequirement struct {
	ExplicitLocators []storeSpec.Locator `json:"explicitLocators,omitempty"`
	DirectoryRoots   []DirectoryRoot     `json:"directoryRoots,omitempty"`
	DecoderHints     []DecoderHint       `json:"decoderHints,omitempty"`

	RequireAuthoritative bool `json:"requireAuthoritative,omitempty"`
}

func (r DiscoveryRequirement) Clone() DiscoveryRequirement {
	output := r
	output.ExplicitLocators = append(
		[]storeSpec.Locator(nil),
		r.ExplicitLocators...,
	)
	output.DirectoryRoots = make(
		[]DirectoryRoot,
		len(r.DirectoryRoots),
	)
	for index, value := range r.DirectoryRoots {
		output.DirectoryRoots[index] = value.Clone()
	}
	output.DecoderHints = make(
		[]DecoderHint,
		len(r.DecoderHints),
	)
	for index, value := range r.DecoderHints {
		output.DecoderHints[index] = value.Clone()
	}
	return output
}

func (r DiscoveryRequirement) Validate() error {
	if len(r.ExplicitLocators) == 0 &&
		len(r.DirectoryRoots) == 0 {
		return fmt.Errorf(
			"%w: declaration discovery preparation requires a locator or directory scope",
			storeSpec.ErrInvalid,
		)
	}
	if len(r.ExplicitLocators) > storeSpec.MaxDiscoveryCandidates ||
		len(r.DirectoryRoots) > storeSpec.MaxDiscoveryCandidates ||
		len(r.DecoderHints) > storeSpec.MaxDecoderHints {
		return fmt.Errorf(
			"%w: declaration discovery preparation exceeds Store limits",
			storeSpec.ErrInvalid,
		)
	}
	for index, locator := range r.ExplicitLocators {
		if err := locator.Validate(false); err != nil {
			return fmt.Errorf(
				"declaration discovery locators[%d]: %w",
				index,
				err,
			)
		}
	}
	for index, directory := range r.DirectoryRoots {
		if err := directory.Validate(); err != nil {
			return fmt.Errorf(
				"declaration discovery directories[%d]: %w",
				index,
				err,
			)
		}
	}
	for index, hint := range r.DecoderHints {
		if err := hint.Validate(); err != nil {
			return fmt.Errorf(
				"declaration discovery decoder hints[%d]: %w",
				index,
				err,
			)
		}
	}
	return nil
}

// DiscoveryPreparation is the revision-checked Source command for additive
// declaration discovery preparation.
type DiscoveryPreparation struct {
	ExpectedRevision uint64                     `json:"expectedRevision"`
	Intent           DiscoveryPreparationIntent `json:"intent"`
	Requirement      DiscoveryRequirement       `json:"requirement"`
}

func (p DiscoveryPreparation) Validate() error {
	if p.ExpectedRevision == 0 ||
		p.ExpectedRevision == ^uint64(0) {
		return fmt.Errorf(
			"%w: expected Source revision is required",
			storeSpec.ErrInvalid,
		)
	}
	if p.Intent != DiscoveryPreparationAdditive {
		return fmt.Errorf(
			"%w: unsupported declaration discovery preparation intent %q",
			storeSpec.ErrInvalid,
			p.Intent,
		)
	}
	return p.Requirement.Validate()
}
