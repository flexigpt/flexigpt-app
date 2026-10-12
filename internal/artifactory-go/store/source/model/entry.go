package model

import (
	"fmt"
	"path"
	"time"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Entry struct {
	Locator    storeSpec.Locator
	Name       string
	SizeBytes  int64
	Mode       uint32
	ModifiedAt time.Time

	IsDirectory bool
	IsRegular   bool
}

func (e Entry) Validate() error {
	if err := e.Locator.Validate(true); err != nil {
		return err
	}
	if e.Name == "" {
		return fmt.Errorf("%w: source entry name is empty", storeSpec.ErrInvalid)
	}
	if e.Locator != "." &&
		e.Name != path.Base(string(e.Locator)) {
		return fmt.Errorf(
			"%w: source entry name does not match locator",
			storeSpec.ErrInvalid,
		)
	}
	if e.SizeBytes < 0 {
		return fmt.Errorf("%w: source entry size is negative", storeSpec.ErrInvalid)
	}
	if e.IsDirectory == e.IsRegular {
		return fmt.Errorf(
			"%w: source entry must identify exactly one entry type",
			storeSpec.ErrInvalid,
		)
	}
	return nil
}
