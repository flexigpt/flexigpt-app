package artifactbuiltin

import (
	"sync"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// GeneratedPackageCatalog owns one application-shipped generated package-set
// payload. Generic Install owns payload validation and canonicalization; this
// content owner only provides immutable, once-per-process preload lifetime.
type GeneratedPackageCatalog struct {
	payload []byte

	once      sync.Once
	preloaded *installFlow.PreloadedGeneratedPackageSet
	err       error
}

func NewGeneratedPackageCatalog(
	payload []byte,
) *GeneratedPackageCatalog {
	return &GeneratedPackageCatalog{
		payload: append([]byte(nil), payload...),
	}
}

func (c *GeneratedPackageCatalog) PackageSet() (
	installModel.CompiledPackageSet,
	error,
) {
	preloaded, err := c.preload()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	return preloaded.PackageSet()
}

func (c *GeneratedPackageCatalog) Fingerprint() (
	cryptoutil.Digest,
	error,
) {
	preloaded, err := c.preload()
	if err != nil {
		return "", err
	}
	return preloaded.Fingerprint(), nil
}

func (c *GeneratedPackageCatalog) preload() (
	*installFlow.PreloadedGeneratedPackageSet,
	error,
) {
	if c == nil {
		return nil, installFlow.ErrGeneratedPackageSetNotLoaded
	}

	c.once.Do(func() {
		c.preloaded, c.err = installFlow.PreloadGeneratedPackageSet(
			c.payload,
		)
	})
	if c.err != nil {
		return nil, c.err
	}
	if c.preloaded == nil {
		return nil, installFlow.ErrGeneratedPackageSetNotLoaded
	}
	return c.preloaded, nil
}
