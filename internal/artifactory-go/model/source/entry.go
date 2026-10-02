package source

import (
	"fmt"
	"path"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
)

type Entry struct {
	Locator    model.Locator
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
		return fmt.Errorf("%w: source entry name is empty", model.ErrInvalid)
	}
	if e.Locator != "." &&
		e.Name != path.Base(string(e.Locator)) {
		return fmt.Errorf(
			"%w: source entry name does not match locator",
			model.ErrInvalid,
		)
	}
	if e.SizeBytes < 0 {
		return fmt.Errorf("%w: source entry size is negative", model.ErrInvalid)
	}
	if e.IsDirectory == e.IsRegular {
		return fmt.Errorf(
			"%w: source entry must identify exactly one entry type",
			model.ErrInvalid,
		)
	}
	return nil
}
