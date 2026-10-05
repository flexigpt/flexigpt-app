package install

import (
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// PreloadedGeneratedPackageSet is immutable generic generated-content state.
// It owns one decoded canonical package-set copy and never exposes its mutable
// internal slices directly.
type PreloadedGeneratedPackageSet struct {
	value       installModel.CompiledPackageSet
	fingerprint cryptoutil.Digest
}

// PreloadGeneratedPackageSet validates and canonicalizes a generated package
// payload once for the lifetime of the owning application installation setup.
func PreloadGeneratedPackageSet(
	raw []byte,
) (*PreloadedGeneratedPackageSet, error) {
	value, err := DecodeGeneratedPackageSet(raw)
	if err != nil {
		return nil, err
	}
	if err := validateCompiledSet(value); err != nil {
		return nil, err
	}

	normalized, fingerprint, err := CanonicalGeneratedPackageSet(value)
	if err != nil {
		return nil, err
	}

	return &PreloadedGeneratedPackageSet{
		value:       normalized,
		fingerprint: fingerprint,
	}, nil
}

func (p *PreloadedGeneratedPackageSet) PackageSet() (
	installModel.CompiledPackageSet,
	error,
) {
	if p == nil {
		return installModel.CompiledPackageSet{}, ErrGeneratedPackageSetNotLoaded
	}
	return p.value.Clone(), nil
}

func (p *PreloadedGeneratedPackageSet) Fingerprint() cryptoutil.Digest {
	if p == nil {
		return ""
	}
	return p.fingerprint
}
