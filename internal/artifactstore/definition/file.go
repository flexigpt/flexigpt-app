package definition

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

const FileFormatV1 = "artifact-definition/v1"

type File struct {
	Format     string     `json:"format"`
	Definition Definition `json:"definition"`
}

func (f File) Validate() error {
	if f.Format != FileFormatV1 {
		return fmt.Errorf(
			"%w: unsupported definition file format %q",
			artifactstore.ErrInvalid,
			f.Format,
		)
	}
	return f.Definition.Validate()
}
