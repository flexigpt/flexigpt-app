package source

import (
	"fmt"
	"path"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
)

type Entry struct {
	Locator    basespec.Locator
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
		return fmt.Errorf("%w: source entry name is empty", basespec.ErrInvalid)
	}
	if e.Locator != "." &&
		e.Name != path.Base(string(e.Locator)) {
		return fmt.Errorf(
			"%w: source entry name does not match locator",
			basespec.ErrInvalid,
		)
	}
	if e.SizeBytes < 0 {
		return fmt.Errorf("%w: source entry size is negative", basespec.ErrInvalid)
	}
	if e.IsDirectory == e.IsRegular {
		return fmt.Errorf(
			"%w: source entry must identify exactly one entry type",
			basespec.ErrInvalid,
		)
	}
	return nil
}
