package sessionv3runtime

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
	v3AttachmentDigestPrefix    = "sha256:"
	v3AttachmentDigestSeparator = "\x00"
	v3DirectoryOverflowNotice   = "directory attachment was truncated after %d files"
)

type V3AttachmentAdmitter struct {
	maximumDirectoryFiles int
}

func NewV3AttachmentAdmitter() *V3AttachmentAdmitter {
	return &V3AttachmentAdmitter{
		maximumDirectoryFiles: attachmentRuntime.DefaultDirectoryWalkMaxFiles,
	}
}

func (a *V3AttachmentAdmitter) AdmitAttachments(
	ctx context.Context,
	request sessionv3Runtime.AttachmentAdmissionRequest,
) ([]sessionv3Runtime.Attachment, error) {
	if a == nil {
		return nil, errors.New("v3 attachment admitter is unavailable")
	}
	if len(request.Sources) == 0 {
		return nil, nil
	}

	output := make(
		[]sessionv3Runtime.Attachment,
		0,
		len(request.Sources),
	)
	for index := range request.Sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := request.Sources[index].Validate(); err != nil {
			return nil, fmt.Errorf("attachment source %d: %w", index, err)
		}

		values, err := a.admitSource(ctx, request.Sources[index])
		if err != nil {
			return nil, fmt.Errorf("admit attachment source %d: %w", index, err)
		}
		output = append(output, values...)
	}
	return output, nil
}

func (a *V3AttachmentAdmitter) admitSource(
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

func (a *V3AttachmentAdmitter) admitFile(
	ctx context.Context,
	source sessionv3Runtime.AttachmentSource,
) ([]sessionv3Runtime.Attachment, error) {
	info, err := v3AttachmentPathInfo(ctx, source.Path)
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
	if err := applyV3AttachmentSourcePresentation(value, source); err != nil {
		return nil, err
	}
	snapshot, err := v3AttachmentSnapshot(ctx, *value)
	if err != nil {
		return nil, err
	}
	return []sessionv3Runtime.Attachment{snapshot}, nil
}

func (a *V3AttachmentAdmitter) admitDirectory(
	ctx context.Context,
	source sessionv3Runtime.AttachmentSource,
) ([]sessionv3Runtime.Attachment, error) {
	maximum := a.maximumDirectoryFiles
	if maximum < 1 {
		maximum = attachmentRuntime.DefaultDirectoryWalkMaxFiles
	}

	walk, err := attachmentRuntime.WalkDirectoryWithFiles(
		ctx,
		source.Path,
		maximum,
	)
	if err != nil {
		return nil, err
	}
	if len(walk.Files) == 0 {
		return nil, errors.New("directory attachment source has no readable files")
	}

	output := make([]sessionv3Runtime.Attachment, 0, len(walk.Files)+1)
	for index := range walk.Files {
		childSource := source
		childSource.Kind = sessionv3Runtime.AttachmentSourceKindFile
		childSource.Path = walk.Files[index].Path
		childSource.URL = ""
		childSource.Label = v3DirectoryChildLabel(
			source.Label,
			walk.Files[index].Name,
		)

		values, err := a.admitFile(ctx, childSource)
		if err != nil {
			return nil, fmt.Errorf(
				"directory attachment child %q: %w",
				walk.Files[index].Path,
				err,
			)
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
			Text: fmt.Sprintf(
				v3DirectoryOverflowNotice,
				len(walk.Files),
			),
		}
		notice.Digest = v3AttachmentDigest(notice)
		if err := notice.Validate(); err != nil {
			return nil, err
		}
		output = append(output, notice)
	}
	return output, nil
}

func (a *V3AttachmentAdmitter) admitURL(
	ctx context.Context,
	source sessionv3Runtime.AttachmentSource,
) ([]sessionv3Runtime.Attachment, error) {
	value, err := attachmentRuntime.BuildAttachmentForURLWithContext(
		ctx,
		source.URL,
	)
	if err != nil {
		return nil, err
	}
	if err := applyV3AttachmentSourcePresentation(value, source); err != nil {
		return nil, err
	}

	snapshot, err := v3AttachmentSnapshot(ctx, *value)
	if err != nil {
		return nil, err
	}
	return []sessionv3Runtime.Attachment{snapshot}, nil
}

func v3AttachmentPathInfo(
	ctx context.Context,
	path string,
) (attachmentRuntime.PathInfo, error) {
	if strings.TrimSpace(path) == "" {
		return attachmentRuntime.PathInfo{}, errors.New(
			"local attachment source path is empty",
		)
	}

	stat, err := llmtoolsutil.StatPath(
		ctx,
		fstool.StatPathArgs{Path: path},
	)
	if err != nil {
		return attachmentRuntime.PathInfo{}, err
	}
	if stat == nil || !stat.Exists || stat.ModTime == nil {
		return attachmentRuntime.PathInfo{}, errors.New(
			"local attachment source is unavailable",
		)
	}

	modifiedAt := stat.ModTime.UTC()
	return attachmentRuntime.PathInfo{
		Path:    stat.Path,
		Name:    stat.Name,
		Exists:  stat.Exists,
		IsDir:   stat.IsDir,
		Size:    stat.SizeBytes,
		ModTime: &modifiedAt,
	}, nil
}

func applyV3AttachmentSourcePresentation(
	value *attachmentRuntime.Attachment,
	source sessionv3Runtime.AttachmentSource,
) error {
	if value == nil {
		return errors.New("attachment source produced no attachment")
	}
	if label := strings.TrimSpace(source.Label); label != "" {
		value.Label = label
	}

	mode, selected, err := v3AttachmentSourceMode(source.Mode)
	if err != nil {
		return err
	}
	if !selected {
		return nil
	}
	if !v3AttachmentAllowsMode(*value, mode) {
		return errors.New("attachment source mode is not available")
	}
	value.Mode = mode
	return nil
}

func v3AttachmentSourceMode(
	value sessionv3Runtime.AttachmentSourceMode,
) (attachmentRuntime.AttachmentContentBlockMode, bool, error) {
	switch value {
	case "", sessionv3Runtime.AttachmentSourceModeAuto:
		return "", false, nil

	case sessionv3Runtime.AttachmentSourceModeText:
		return attachmentRuntime.AttachmentContentBlockModeText, true, nil

	case sessionv3Runtime.AttachmentSourceModeFile:
		return attachmentRuntime.AttachmentContentBlockModeFile, true, nil

	case sessionv3Runtime.AttachmentSourceModeImage:
		return attachmentRuntime.AttachmentContentBlockModeImage, true, nil

	case sessionv3Runtime.AttachmentSourceModePage:
		return attachmentRuntime.AttachmentContentBlockModePageContent, true, nil

	case sessionv3Runtime.AttachmentSourceModeLink:
		return attachmentRuntime.AttachmentContentBlockModeTextLink, true, nil

	default:
		return "", false, errors.New("attachment source mode is unavailable")
	}
}

func v3AttachmentAllowsMode(
	value attachmentRuntime.Attachment,
	mode attachmentRuntime.AttachmentContentBlockMode,
) bool {
	return slices.Contains(value.AvailableContentBlockModes, mode)
}

func v3AttachmentSnapshot(
	ctx context.Context,
	value attachmentRuntime.Attachment,
) (sessionv3Runtime.Attachment, error) {
	sourceURL := ""
	if value.URLRef != nil {
		sourceURL = strings.TrimSpace(value.URLRef.Normalized)
		if sourceURL == "" {
			sourceURL = strings.TrimSpace(value.URLRef.URL)
		}
	}

	blocks, err := attachmentRuntime.BuildContentBlocks(
		ctx,
		[]attachmentRuntime.Attachment{value},
	)
	if err != nil {
		return sessionv3Runtime.Attachment{}, err
	}
	if len(blocks) != 1 {
		return sessionv3Runtime.Attachment{}, errors.New(
			"attachment source produced an unexpected content block count",
		)
	}

	return v3AttachmentFromContentBlock(
		blocks[0],
		value.Label,
		sourceURL,
	)
}

func v3AttachmentFromContentBlock(
	block attachmentRuntime.ContentBlock,
	label string,
	sourceURL string,
) (sessionv3Runtime.Attachment, error) {
	value := sessionv3Runtime.Attachment{
		Label:     strings.TrimSpace(label),
		SourceURL: sourceURL,
	}
	if block.MIMEType != nil {
		value.MIMEType = strings.TrimSpace(*block.MIMEType)
	}
	if value.Label == "" && block.FileName != nil {
		value.Label = strings.TrimSpace(*block.FileName)
	}

	switch block.Kind {
	case attachmentRuntime.ContentBlockText:
		if block.Text == nil {
			return sessionv3Runtime.Attachment{}, errors.New(
				"attachment text snapshot is missing content",
			)
		}
		value.Kind = sessionv3Runtime.AttachmentKindText
		value.Text = *block.Text

	case attachmentRuntime.ContentBlockImage:
		if block.Base64Data == nil ||
			strings.TrimSpace(*block.Base64Data) == "" {
			return sessionv3Runtime.Attachment{}, errors.New(
				"attachment image snapshot is not replay-safe",
			)
		}
		value.Kind = sessionv3Runtime.AttachmentKindImage
		value.Base64Data = *block.Base64Data

	case attachmentRuntime.ContentBlockFile:
		if block.Base64Data == nil ||
			strings.TrimSpace(*block.Base64Data) == "" {
			return sessionv3Runtime.Attachment{}, errors.New(
				"attachment file snapshot is not replay-safe",
			)
		}
		value.Kind = sessionv3Runtime.AttachmentKindFile
		value.Base64Data = *block.Base64Data

	default:
		return sessionv3Runtime.Attachment{}, errors.New(
			"attachment content block kind is unavailable",
		)
	}

	value.Digest = v3AttachmentDigest(value)
	if err := value.Validate(); err != nil {
		return sessionv3Runtime.Attachment{}, err
	}
	return value, nil
}

func v3AttachmentDigest(value sessionv3Runtime.Attachment) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(value.Kind),
		value.MIMEType,
		value.Text,
		value.Base64Data,
		value.SourceURL,
	}, v3AttachmentDigestSeparator)))
	return v3AttachmentDigestPrefix + hex.EncodeToString(sum[:])
}

func v3DirectoryChildLabel(
	directoryLabel string,
	childName string,
) string {
	directoryLabel = strings.TrimSpace(directoryLabel)
	childName = strings.TrimSpace(childName)
	if directoryLabel == "" {
		return childName
	}
	if childName == "" {
		return directoryLabel
	}
	return directoryLabel + "/" + childName
}
