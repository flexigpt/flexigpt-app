package attachment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/imagetool"
)

// ImageRef carries metadata for image attachments.
type ImageRef struct {
	Path    string     `json:"path"`
	Name    string     `json:"name"`
	Exists  bool       `json:"exists"`
	IsDir   bool       `json:"isDir"`
	Size    int64      `json:"size,omitempty"`
	ModTime *time.Time `json:"modTime,omitempty"`

	Width    int      `json:"width,omitempty"`
	Height   int      `json:"height,omitempty"`
	Format   string   `json:"format,omitempty"`   // e.g. "jpeg", "png"
	MIMEType MIMEType `json:"mimeType,omitempty"` // e.g. "image/jpeg"

	// Original snapshot (for change detection across turns).
	OrigPath    string    `json:"origPath"`
	OrigSize    int64     `json:"origSize"`
	OrigModTime time.Time `json:"origModTime"`
}

func (ref *ImageRef) PopulateRef(ctx context.Context, replaceOrig bool) error {
	if ref == nil {
		return errors.New("image attachment missing ref")
	}
	path := strings.TrimSpace(ref.Path)
	if path == "" {
		return errors.New("image attachment missing path")
	}

	toolOut, err := llmtoolsutil.ReadImage(ctx, imagetool.ReadImageArgs{
		Path:              path,
		IncludeBase64Data: false,
	})
	if err != nil || toolOut == nil {
		return errors.Join(ErrUnreadableFile, err)
	}

	if strings.TrimSpace(ref.OrigPath) == "" || replaceOrig {
		ref.OrigPath = toolOut.Path
		ref.OrigSize = toolOut.SizeBytes
		ref.OrigModTime = *toolOut.ModTime
	}

	ref.Path = toolOut.Path
	ref.Exists = toolOut.Exists
	ref.IsDir = false
	ref.Size = toolOut.SizeBytes
	ref.ModTime = toolOut.ModTime
	ref.Width = toolOut.Width
	ref.Height = toolOut.Height
	ref.Format = toolOut.Format

	ref.Name = toolOut.Name
	ref.MIMEType = MIMEType(toolOut.MIMEType)
	return nil
}

func (ref *ImageRef) IsModified() bool {
	if ref == nil {
		return false
	}
	if strings.TrimSpace(ref.OrigPath) == "" {
		return false
	}
	if !ref.Exists {
		return true
	}
	if ref.Path != ref.OrigPath {
		return true
	}
	if ref.Size != ref.OrigSize {
		return true
	}
	if !ref.ModTime.Equal(ref.OrigModTime) {
		return true
	}
	return false
}

func (ref *ImageRef) BuildContentBlock(ctx context.Context) (*ContentBlock, error) {
	return buildImageBlockFromLocal(ctx, ref.Path)
}

func buildImageBlockFromLocal(ctx context.Context, path string) (*ContentBlock, error) {
	toolOut, err := llmtoolsutil.ReadImage(ctx, imagetool.ReadImageArgs{
		Path:              path,
		IncludeBase64Data: true,
	})
	if err != nil || toolOut == nil {
		return nil, errors.Join(ErrUnreadableFile, err)
	}

	return &ContentBlock{
		Kind:       ContentBlockImage,
		Base64Data: &toolOut.Base64Data,
		MIMEType:   &toolOut.MIMEType,
		FileName:   &toolOut.Name,
	}, nil
}
