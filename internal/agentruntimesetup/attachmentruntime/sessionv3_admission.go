package attachmentruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	attachmentRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/attachment"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/fstool"
)

const (
	sessionV3AttachmentDigestPrefix    = "sha256:"
	sessionV3AttachmentDigestSeparator = "\x00"
	sessionV3DirectoryOverflowNotice   = "directory attachment was truncated after %d files"
)

// SessionV3Admitter is the Attachment-domain V3 admission boundary. It reads
// command-only paths and URLs once, then returns immutable Session facts.
type SessionV3Admitter struct {
	maximumDirectoryFiles int
}

func NewSessionV3Admitter() *SessionV3Admitter {
	return &SessionV3Admitter{
		maximumDirectoryFiles: attachmentRuntime.DefaultDirectoryWalkMaxFiles,
	}
}

func (a *SessionV3Admitter) AdmitAttachments(
	ctx context.Context,
	request sessionv3Runtime.AttachmentAdmissionRequest,
) ([]sessionv3Runtime.Attachment, error) {
	if a == nil {
		return nil, errors.New("v3 attachment admitter is unavailable")
	}
	output := make([]sessionv3Runtime.Attachment, 0, len(request.Sources))
	for index := range request.Sources {
		values, err := a.admitSource(ctx, request.Sources[index])
		if err != nil {
			return nil, fmt.Errorf("admit attachment source %d: %w", index, err)
		}
		output = append(output, values...)
	}
	return output, nil
}

func (a *SessionV3Admitter) admitSource(
	ctx context.Context,
	source sessionv3Runtime.AttachmentSource,
) ([]sessionv3Runtime.Attachment, error) {
	switch source.Kind {
	case sessionv3Runtime.AttachmentSourceKindFile:
		return a.admitFile(ctx, source)
	case sessionv3Runtime.AttachmentSourceKindDirectory:
		return a.admitDirectory(ctx, source)
	case sessionv3Runtime.AttachmentSourceKindURL:
		return a.admitURL(ctx, source)
	default:
		return nil, errors.New("attachment source kind is unavailable")
	}
}

func (a *SessionV3Admitter) admitFile(
	ctx context.Context,
	source sessionv3Runtime.AttachmentSource,
) ([]sessionv3Runtime.Attachment, error) {
	info, err := sessionV3AttachmentPathInfo(ctx, source.Path)
	if err != nil {
		return nil, err
	}
	if info.IsDir {
		return nil, errors.New("file attachment source is a directory")
	}
	value, err := attachmentRuntime.BuildAttachmentForFile(ctx, &info)
	if err != nil {
		return nil, err
	}
	if err := applySessionV3AttachmentPresentation(value, source); err != nil {
		return nil, err
	}
	snapshot, err := sessionV3AttachmentSnapshot(ctx, *value)
	if err != nil {
		return nil, err
	}
	return []sessionv3Runtime.Attachment{snapshot}, nil
}

func (a *SessionV3Admitter) admitDirectory(
	ctx context.Context,
	source sessionv3Runtime.AttachmentSource,
) ([]sessionv3Runtime.Attachment, error) {
	maximum := a.maximumDirectoryFiles
	if maximum < 1 {
		maximum = attachmentRuntime.DefaultDirectoryWalkMaxFiles
	}
	walk, err := attachmentRuntime.WalkDirectoryWithFiles(ctx, source.Path, maximum)
	if err != nil {
		return nil, err
	}
	if len(walk.Files) == 0 {
		return nil, errors.New("directory attachment source has no readable files")
	}
	output := make([]sessionv3Runtime.Attachment, 0, len(walk.Files)+1)
	for index := range walk.Files {
		child := source
		child.Kind = sessionv3Runtime.AttachmentSourceKindFile
		child.Path = walk.Files[index].Path
		child.URL = ""
		relative, relativeErr := filepath.Rel(walk.DirPath, walk.Files[index].Path)
		if relativeErr != nil || strings.HasPrefix(relative, "..") {
			relative = walk.Files[index].Name
		}
		child.Label = sessionV3DirectoryChildLabel(source.Label, filepath.ToSlash(relative))
		values, childErr := a.admitFile(ctx, child)
		if childErr != nil {
			return nil, fmt.Errorf("directory attachment child %q: %w", child.Path, childErr)
		}
		output = append(output, values...)
	}
	if walk.HasMore {
		label := strings.TrimSpace(source.Label)
		if label == "" {
			label = filepath.Base(walk.DirPath)
		}
		notice := sessionv3Runtime.Attachment{
			Kind:  sessionv3Runtime.AttachmentKindText,
			Label: label,
			Text:  fmt.Sprintf(sessionV3DirectoryOverflowNotice, len(walk.Files)),
		}
		notice.Digest = sessionV3AttachmentDigest(notice)
		output = append(output, notice)
	}
	return output, nil
}

func (a *SessionV3Admitter) admitURL(
	ctx context.Context,
	source sessionv3Runtime.AttachmentSource,
) ([]sessionv3Runtime.Attachment, error) {
	value, err := attachmentRuntime.BuildAttachmentForURLWithContext(ctx, source.URL)
	if err != nil {
		return nil, err
	}
	if err := applySessionV3AttachmentPresentation(value, source); err != nil {
		return nil, err
	}
	snapshot, err := sessionV3AttachmentSnapshot(ctx, *value)
	if err != nil {
		return nil, err
	}
	return []sessionv3Runtime.Attachment{snapshot}, nil
}

func sessionV3AttachmentPathInfo(
	ctx context.Context,
	path string,
) (attachmentRuntime.PathInfo, error) {
	stat, err := llmtoolsutil.StatPath(ctx, fstool.StatPathArgs{Path: path})
	if err != nil {
		return attachmentRuntime.PathInfo{}, err
	}
	if stat == nil || !stat.Exists || stat.ModTime == nil {
		return attachmentRuntime.PathInfo{}, errors.New("local attachment source is unavailable")
	}
	modified := stat.ModTime.UTC()
	return attachmentRuntime.PathInfo{
		Path: stat.Path, Name: stat.Name, Exists: stat.Exists, IsDir: stat.IsDir,
		Size: stat.SizeBytes, ModTime: &modified,
	}, nil
}

func applySessionV3AttachmentPresentation(
	value *attachmentRuntime.Attachment,
	source sessionv3Runtime.AttachmentSource,
) error {
	if value == nil {
		return errors.New("attachment source produced no attachment")
	}
	if label := strings.TrimSpace(source.Label); label != "" {
		value.Label = label
	}
	mode, selected := sessionV3AttachmentMode(source.Mode)
	if !selected {
		return nil
	}
	if !slices.Contains(value.AvailableContentBlockModes, mode) {
		return errors.New("attachment source mode is not available")
	}
	value.Mode = mode
	return nil
}

func sessionV3AttachmentMode(
	value sessionv3Runtime.AttachmentSourceMode,
) (attachmentRuntime.AttachmentContentBlockMode, bool) {
	switch value {
	case "", sessionv3Runtime.AttachmentSourceModeAuto:
		return "", false
	case sessionv3Runtime.AttachmentSourceModeText:
		return attachmentRuntime.AttachmentContentBlockModeText, true
	case sessionv3Runtime.AttachmentSourceModeFile:
		return attachmentRuntime.AttachmentContentBlockModeFile, true
	case sessionv3Runtime.AttachmentSourceModeImage:
		return attachmentRuntime.AttachmentContentBlockModeImage, true
	case sessionv3Runtime.AttachmentSourceModePage:
		return attachmentRuntime.AttachmentContentBlockModePageContent, true
	case sessionv3Runtime.AttachmentSourceModeLink:
		return attachmentRuntime.AttachmentContentBlockModeTextLink, true
	default:
		return "", false
	}
}

func sessionV3AttachmentSnapshot(
	ctx context.Context,
	value attachmentRuntime.Attachment,
) (sessionv3Runtime.Attachment, error) {
	blocks, err := attachmentRuntime.BuildContentBlocks(ctx, []attachmentRuntime.Attachment{value})
	if err != nil || len(blocks) != 1 {
		return sessionv3Runtime.Attachment{}, errors.Join(
			err,
			errors.New("attachment did not produce one immutable content block"),
		)
	}
	output := sessionv3Runtime.Attachment{Label: value.Label}
	if blocks[0].MIMEType != nil {
		output.MIMEType = *blocks[0].MIMEType
	}
	switch blocks[0].Kind {
	case attachmentRuntime.ContentBlockText:
		output.Kind = sessionv3Runtime.AttachmentKindText
		if blocks[0].Text != nil {
			output.Text = *blocks[0].Text
		}
	case attachmentRuntime.ContentBlockImage:
		output.Kind = sessionv3Runtime.AttachmentKindImage
		if blocks[0].Base64Data != nil {
			output.Base64Data = *blocks[0].Base64Data
		}
	case attachmentRuntime.ContentBlockFile:
		output.Kind = sessionv3Runtime.AttachmentKindFile
		if blocks[0].Base64Data != nil {
			output.Base64Data = *blocks[0].Base64Data
		}
	default:
		return sessionv3Runtime.Attachment{}, errors.New("attachment content block kind is unavailable")
	}
	output.Digest = sessionV3AttachmentDigest(output)
	return output, output.Validate()
}

func sessionV3AttachmentDigest(value sessionv3Runtime.Attachment) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(value.Kind), value.MIMEType, value.Text, value.Base64Data, value.URL, value.SourceURL,
	}, sessionV3AttachmentDigestSeparator)))
	return sessionV3AttachmentDigestPrefix + hex.EncodeToString(sum[:])
}

func sessionV3DirectoryChildLabel(directory, child string) string {
	directory = strings.TrimSpace(directory)
	child = strings.TrimSpace(child)
	if directory == "" {
		return child
	}
	if child == "" {
		return directory
	}
	return directory + "/" + child
}
