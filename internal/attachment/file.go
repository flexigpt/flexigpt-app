package attachment

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/fstool"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"
)

// FileRef carries metadata for file attachments.
type FileRef struct {
	PathInfo

	// Snapshot of the original file state when it was first attached.
	// This lets us detect if the underlying file changed between turns.
	OrigPath    string    `json:"origPath"`
	OrigSize    int64     `json:"origSize"`
	OrigModTime time.Time `json:"origModTime"`
}

func (ref *FileRef) PopulateRef(ctx context.Context, replaceOrig bool) error {
	if ref == nil {
		return errors.New("file attachment missing ref")
	}
	path := strings.TrimSpace(ref.Path)
	if path == "" {
		return errors.New("file attachment missing path")
	}
	pathInfo, err := llmtoolsutil.StatPath(ctx, fstool.StatPathArgs{
		Path: path,
	})
	if err != nil {
		return err
	}
	if pathInfo == nil {
		return ErrUnreadableFile
	}

	// Capture original snapshot once.
	if strings.TrimSpace(ref.OrigPath) == "" || replaceOrig {
		ref.OrigPath = pathInfo.Path
		ref.OrigSize = pathInfo.SizeBytes
		ref.OrigModTime = *pathInfo.ModTime
	}

	ref.Path = pathInfo.Path
	ref.Name = pathInfo.Name
	ref.Exists = pathInfo.Exists
	ref.IsDir = pathInfo.IsDir
	ref.Size = pathInfo.SizeBytes
	ref.ModTime = pathInfo.ModTime

	return nil
}

// IsModified reports whether the current file state differs from the original
// snapshot captured when the attachment was first used.
func (ref *FileRef) IsModified() bool {
	if ref == nil {
		return false
	}
	if strings.TrimSpace(ref.OrigPath) == "" {
		// No baseline; treat as not modified.
		return false
	}
	// If the file no longer exists or any key metadata changed, mark as modified.
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

func (ref *FileRef) BuildContentBlock(
	ctx context.Context,
	attachmentContentBlockMode AttachmentContentBlockMode,
	onlyIfTextKind bool,
) (*ContentBlock, error) {
	path := strings.TrimSpace(ref.Path)
	if path == "" {
		return nil, errors.New("got invalid path")
	}
	switch attachmentContentBlockMode {
	case AttachmentContentBlockModeImage:
		if onlyIfTextKind {
			return nil, ErrNonTextContentBlock
		}
		// Treat a file in "image" mode as an image attachment.
		return buildImageBlockFromLocal(ctx, path)

	case AttachmentContentBlockModeFile:
		// File mode needs to handle all kinds of supported files appropriately.
		toolOut, err := llmtoolsutil.MIMEForPath(ctx, fstool.MIMEForPathArgs{
			Path: path,
		})
		if err != nil || toolOut == nil {
			return nil, errors.Join(ErrUnreadableFile, err)
		}

		mimeType := MIMEType(toolOut.BaseMIMEType)
		extensionMode := toolOut.Mode

		switch extensionMode {
		case fstool.MIMEModeText:
			// If this is a text type file, we cannot attach it as b64 encoded currently.
			// Right now we are making a safe fallback to send it as text block.
			// Ideally we should not reach here if UI takes care of AttachmentKind and AttachmentContentBlockMode
			// properly.
			return ref.getTextBlock(ctx, mimeType)
		case fstool.MIMEModeImage:
			if onlyIfTextKind {
				return nil, ErrNonTextContentBlock
			}
			// Images have to be attached specially.
			// Ideally we should not reach here if UI takes care of AttachmentKind and AttachmentContentBlockMode
			// properly.
			return buildImageBlockFromLocal(ctx, path)
		case fstool.MIMEModeDocument:
			if onlyIfTextKind {
				return nil, ErrNonTextContentBlock
			}
			// As of now non pdf files are not supported to be attached as b64 files in any APIs.
			// But, as the call is reached here, we are still sending the content back as b64 content.
			// As APIs start supporting other file types, and UI and fetch api handles things this will be seamless.
			c, err := ref.getBinaryFileContent(ctx, path, mimeType)
			if err != nil {
				return nil, err
			}
			return c, nil

		case fstool.MIMEModeDefault:
			return nil, ErrUnreadableFile
		default:
			return nil, ErrUnreadableFile
		}

	case AttachmentContentBlockModeText:
		toolOut, err := llmtoolsutil.MIMEForPath(ctx, fstool.MIMEForPathArgs{
			Path: path,
		})
		if err != nil || toolOut == nil {
			return nil, errors.Join(ErrUnreadableFile, err)
		}

		mimeType := MIMEType(toolOut.BaseMIMEType)
		extensionMode := toolOut.Mode

		if extensionMode != fstool.MIMEModeText && mimeType != MIMEApplicationPDF {
			// Could not detect mime or non pdf text attachment sent, render as unreadable file.
			return nil, ErrUnreadableFile
		}
		// Text mode mimes and pdf with text extraction is supported.
		return ref.getTextBlock(ctx, mimeType)

	case AttachmentContentBlockModeNotReadable,
		AttachmentContentBlockModePageContent,
		AttachmentContentBlockModeTextLink,
		AttachmentContentBlockModePRDiff,
		AttachmentContentBlockModePRPage,
		AttachmentContentBlockModeCommitDiff,
		AttachmentContentBlockModeCommitPage:
		// Provide a short note so the model knows the file exists but is not included.
		return nil, ErrUnreadableFile
	default:
		return nil, errors.New("invalid attachment mode")
	}
}

func (ref *FileRef) getTextBlock(ctx context.Context, mimetype MIMEType) (*ContentBlock, error) {
	path := strings.TrimSpace(ref.Path)
	if path == "" {
		return nil, errors.New("got invalid path")
	}

	c, err := ref.getTextFileContent(ctx, path, mimetype)
	if err != nil {
		ext := strings.ToLower(filepath.Ext(path))
		// Special handling for PDFs as fallback: attach as binary content.
		if mimetype == MIMEApplicationPDF || ext == string(ExtPDF) {
			return ref.getBinaryFileContent(ctx, path, mimetype)
		}
		return nil, err
	}
	return c, nil
}

func (ref *FileRef) getTextFileContent(
	ctx context.Context,
	path string,
	mimeType MIMEType,
) (*ContentBlock, error) {
	// Fstool supports Text extraction of pdf too.
	toolOut, err := llmtoolsutil.ReadFile(ctx, fstool.ReadFileArgs{
		Path:     path,
		Encoding: "text",
	})
	if err != nil {
		return nil, err
	}
	if len(toolOut) == 0 || toolOut[0].Kind != llmtoolsgoSpec.ToolStoreOutputKindText ||
		toolOut[0].TextItem == nil {
		return nil, ErrUnreadableFile
	}
	tItem := toolOut[0].TextItem

	mStr := string(mimeType)
	fname := filepath.Base(path)
	return &ContentBlock{
		Kind:     ContentBlockText,
		Text:     &tItem.Text,
		MIMEType: &mStr,
		FileName: &fname,
	}, nil
}

func (ref *FileRef) getBinaryFileContent(
	ctx context.Context,
	path string,
	mimeType MIMEType,
) (*ContentBlock, error) {
	toolOut, err := llmtoolsutil.ReadFile(ctx, fstool.ReadFileArgs{
		Path:     path,
		Encoding: "binary",
	})
	if err != nil {
		return nil, err
	}
	if len(toolOut) == 0 || toolOut[0].Kind != llmtoolsgoSpec.ToolStoreOutputKindFile ||
		toolOut[0].FileItem == nil {
		return nil, ErrUnreadableFile
	}
	fItem := toolOut[0].FileItem

	mStr := string(mimeType)

	return &ContentBlock{
		Kind:       ContentBlockFile,
		Base64Data: &fItem.FileData,
		MIMEType:   &mStr,
		FileName:   &fItem.FileName,
	}, nil
}
