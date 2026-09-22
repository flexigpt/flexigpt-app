package llmtoolsutil

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/flexigpt/llmtools-go/fstool"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// ReadPortableTextFile reads exactly one selected text file through the
// existing filesystem-tool boundary. It intentionally does not expose a
// reusable arbitrary filesystem reader to consumer APIs.
func ReadPortableTextFile(
	ctx context.Context,
	selectedPath string,
	maximumBytes int,
) ([]byte, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: portable file read context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if selectedPath == "" {
		return nil, fmt.Errorf(
			"%w: selected file path is required",
			basespec.ErrInvalid,
		)
	}
	if maximumBytes <= 0 {
		return nil, fmt.Errorf(
			"%w: portable file byte limit is invalid",
			basespec.ErrInvalid,
		)
	}

	outputs, err := ReadFile(
		ctx,
		fstool.ReadFileArgs{
			Path:     selectedPath,
			Encoding: "text",
		},
	)
	if err != nil {
		return nil, err
	}
	if len(outputs) != 1 {
		return nil, fmt.Errorf(
			"%w: selected file reader returned %d outputs",
			basespec.ErrInvalid,
			len(outputs),
		)
	}

	output := outputs[0]
	if output.Kind != llmtoolsSpec.ToolOutputKindText ||
		output.TextItem == nil ||
		output.ImageItem != nil ||
		output.FileItem != nil {
		return nil, fmt.Errorf(
			"%w: selected file reader did not return one text output",
			basespec.ErrInvalid,
		)
	}

	value := []byte(output.TextItem.Text)
	if len(value) > maximumBytes {
		return nil, fmt.Errorf(
			"%w: selected file exceeds %d bytes",
			basespec.ErrInvalid,
			maximumBytes,
		)
	}
	if !utf8.Valid(value) {
		return nil, fmt.Errorf(
			"%w: selected file is not valid UTF-8 text",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), value...), nil
}
