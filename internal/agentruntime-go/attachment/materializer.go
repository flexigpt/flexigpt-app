package attachment

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/fstool"
)

const defaultDirectoryAttachmentMaxFiles = 1024

type Materializer interface {
	Materialize(
		ctx context.Context,
		references []sessionRuntime.AttachmentReference,
	) ([]Attachment, error)
}

// JSONReferenceMaterializer converts durable Session references into the
// attachment domain. It deliberately does not place hydrated bytes in Session
// state. The name is retained for compatibility with existing construction
// sites; conversion is now typed rather than marshal/unmarshal based.
type JSONReferenceMaterializer struct{}

func NewJSONReferenceMaterializer() *JSONReferenceMaterializer {
	return &JSONReferenceMaterializer{}
}

func (m *JSONReferenceMaterializer) Materialize(
	ctx context.Context,
	references []sessionRuntime.AttachmentReference,
) ([]Attachment, error) {
	if len(references) == 0 {
		return nil, nil
	}

	output := make([]Attachment, 0, len(references))
	for index := range references {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		values, err := m.materializeReference(references[index])
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}

	return output, nil
}

// PrepareAttachments is the admission-time normalization boundary. Local files
// receive an immutable path/size/modtime snapshot here. Directories are walked
// once and persist their flattened child references, so historical replay does
// not silently gain or lose files after the message was accepted.
func (m *JSONReferenceMaterializer) PrepareAttachments(
	ctx context.Context,
	references []sessionRuntime.AttachmentReference,
) ([]sessionRuntime.AttachmentReference, error) {
	if len(references) == 0 {
		return nil, nil
	}

	output := make([]sessionRuntime.AttachmentReference, 0, len(references))
	for index := range references {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		reference, err := m.prepareReference(ctx, references[index])
		if err != nil {
			return nil, fmt.Errorf("prepare attachment %d: %w", index, err)
		}
		output = append(output, reference)
	}
	return output, nil
}

func (m *JSONReferenceMaterializer) prepareReference(
	ctx context.Context,
	reference sessionRuntime.AttachmentReference,
) (sessionRuntime.AttachmentReference, error) {
	switch reference.Kind {
	case sessionRuntime.AttachmentReferenceFile:
		if reference.File == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("file attachment reference has no file reference")
		}
		return prepareLocalReference(ctx, reference, reference.File.Path)

	case sessionRuntime.AttachmentReferenceImage:
		if reference.Image == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("image attachment reference has no image reference")
		}
		return prepareLocalReference(ctx, reference, reference.Image.Path)

	case sessionRuntime.AttachmentReferenceURL:
		if reference.URL == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("URL attachment reference has no URL reference")
		}
		attachment, err := BuildAttachmentForURLWithContext(
			ctx,
			reference.URL.URL,
		)
		if err != nil {
			return sessionRuntime.AttachmentReference{}, err
		}
		output, err := attachmentReferenceFromAttachment(attachment)
		if err != nil {
			return sessionRuntime.AttachmentReference{}, err
		}
		copyAttachmentReferencePresentation(&output, reference)
		return output, nil

	case sessionRuntime.AttachmentReferenceGeneric:
		if reference.Generic == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("generic attachment reference has no handle")
		}
		generic := *reference.Generic
		generic.Handle = strings.TrimSpace(generic.Handle)
		if generic.Handle == "" {
			return sessionRuntime.AttachmentReference{},
				errors.New("generic attachment reference has an empty handle")
		}
		if strings.TrimSpace(generic.OriginalHandle) == "" {
			generic.OriginalHandle = generic.Handle
		}
		reference.Generic = &generic
		if strings.TrimSpace(reference.Mode) == "" {
			reference.Mode = string(AttachmentContentBlockModeTextLink)
		}
		return reference, nil

	case sessionRuntime.AttachmentReferenceDirectory:
		if reference.Directory == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("directory attachment reference has no directory reference")
		}

		directory := *reference.Directory
		directoryPath := strings.TrimSpace(directory.Path)
		if directoryPath == "" {
			return sessionRuntime.AttachmentReference{},
				errors.New("directory attachment reference has an empty path")
		}

		// Expanded is durable, server-owned admission output. Never trust an
		// inbound Expanded list: otherwise a client can bypass walking limits,
		// file snapshots, and file-kind normalization.
		walk, err := WalkDirectoryWithFiles(
			ctx,
			directoryPath,
			defaultDirectoryAttachmentMaxFiles,
		)
		if err != nil {
			return sessionRuntime.AttachmentReference{}, err
		}
		if len(walk.Files) == 0 {
			return sessionRuntime.AttachmentReference{},
				fmt.Errorf(
					"directory attachment %q contains no readable files",
					walk.DirPath,
				)
		}

		directory.Path = walk.DirPath
		directory.OriginalPath = walk.DirPath
		if strings.TrimSpace(directory.Name) == "" {
			directory.Name = filepath.Base(walk.DirPath)
		}
		directory.Expanded = make(
			[]sessionRuntime.AttachmentReference,
			0,
			len(walk.Files),
		)
		for index := range walk.Files {
			attachment, err := BuildAttachmentForFile(ctx, &walk.Files[index])
			if err != nil {
				return sessionRuntime.AttachmentReference{},
					fmt.Errorf(
						"build directory attachment %q: %w",
						walk.Files[index].Path,
						err,
					)
			}
			ensureAttachmentSnapshot(attachment, walk.Files[index])

			child, err := attachmentReferenceFromAttachment(attachment)
			if err != nil {
				return sessionRuntime.AttachmentReference{}, err
			}
			directory.Expanded = append(directory.Expanded, child)
		}
		directory.ScannedFileCount = len(directory.Expanded)
		directory.HasMore = walk.HasMore
		reference.Directory = &directory
		return reference, nil

	default:
		return sessionRuntime.AttachmentReference{},
			fmt.Errorf("unsupported attachment reference kind %q", reference.Kind)
	}
}

func prepareLocalReference(
	ctx context.Context,
	reference sessionRuntime.AttachmentReference,
	path string,
) (sessionRuntime.AttachmentReference, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return sessionRuntime.AttachmentReference{},
			errors.New("local attachment reference has an empty path")
	}

	stat, err := llmtoolsutil.StatPath(ctx, fstool.StatPathArgs{Path: path})
	if err != nil {
		return sessionRuntime.AttachmentReference{}, err
	}
	if stat == nil || !stat.Exists || stat.IsDir || stat.ModTime == nil {
		return sessionRuntime.AttachmentReference{},
			fmt.Errorf("local attachment %q is not a readable regular file", path)
	}

	pathInfo := PathInfo{
		Path:    stat.Path,
		Name:    stat.Name,
		Exists:  stat.Exists,
		IsDir:   stat.IsDir,
		Size:    stat.SizeBytes,
		ModTime: cloneAttachmentTime(stat.ModTime),
	}
	attachment, err := BuildAttachmentForFile(ctx, &pathInfo)
	if err != nil {
		return sessionRuntime.AttachmentReference{}, err
	}
	ensureAttachmentSnapshot(attachment, pathInfo)

	output, err := attachmentReferenceFromAttachment(attachment)
	if err != nil {
		return sessionRuntime.AttachmentReference{}, err
	}
	copyAttachmentReferencePresentation(&output, reference)
	return output, nil
}

func copyAttachmentReferencePresentation(
	output *sessionRuntime.AttachmentReference,
	input sessionRuntime.AttachmentReference,
) {
	if output == nil {
		return
	}
	if strings.TrimSpace(input.Label) != "" {
		output.Label = input.Label
	}
	if strings.TrimSpace(input.Mode) != "" {
		output.Mode = input.Mode
	}
	if len(input.AvailableModes) != 0 {
		output.AvailableModes = append(
			[]string(nil),
			input.AvailableModes...,
		)
	}
}

func ensureAttachmentSnapshot(attachment *Attachment, info PathInfo) {
	if attachment == nil || info.ModTime == nil {
		return
	}
	if attachment.FileRef != nil &&
		strings.TrimSpace(attachment.FileRef.OrigPath) == "" {
		attachment.FileRef.OrigPath = info.Path
		attachment.FileRef.OrigSize = info.Size
		attachment.FileRef.OrigModTime = *info.ModTime
	}
	if attachment.ImageRef != nil &&
		strings.TrimSpace(attachment.ImageRef.OrigPath) == "" {
		attachment.ImageRef.OrigPath = info.Path
		attachment.ImageRef.OrigSize = info.Size
		attachment.ImageRef.OrigModTime = *info.ModTime
	}
}

func (m *JSONReferenceMaterializer) materializeReference(
	reference sessionRuntime.AttachmentReference,
) ([]Attachment, error) {
	switch reference.Kind {
	case sessionRuntime.AttachmentReferenceDirectory:
		if reference.Directory == nil || len(reference.Directory.Expanded) == 0 {
			return nil, errors.New("directory attachment was not expanded at submission time")
		}
		output := make([]Attachment, 0, len(reference.Directory.Expanded))
		for index := range reference.Directory.Expanded {
			children, err := m.materializeReference(
				reference.Directory.Expanded[index],
			)
			if err != nil {
				return nil, fmt.Errorf(
					"materialize directory child %d: %w",
					index,
					err,
				)
			}
			output = append(output, children...)
		}
		return output, nil

	case sessionRuntime.AttachmentReferenceFile:
		if reference.File == nil || reference.File.IsDirectory {
			return nil, errors.New("invalid file attachment reference")
		}
		modifiedAt, err := parseReferenceTime(reference.File.ModifiedAt)
		if err != nil {
			return nil, err
		}
		originalModTime, err := parseReferenceTime(reference.File.OriginalMod)
		if err != nil {
			return nil, err
		}
		originalMod := time.Time{}
		if originalModTime != nil {
			originalMod = *originalModTime
		}
		return []Attachment{{
			Kind:                       AttachmentFile,
			Label:                      reference.Label,
			Mode:                       referenceMode(reference.Mode, AttachmentContentBlockModeText),
			AvailableContentBlockModes: referenceModes(reference.AvailableModes),
			FileRef: &FileRef{
				Path:        reference.File.Path,
				Name:        reference.File.Name,
				Exists:      reference.File.Exists,
				IsDir:       false,
				Size:        reference.File.Size,
				ModTime:     modifiedAt,
				OrigPath:    reference.File.OriginalPath,
				OrigSize:    reference.File.OriginalSize,
				OrigModTime: originalMod,
			},
		}}, nil

	case sessionRuntime.AttachmentReferenceImage:
		if reference.Image == nil || reference.Image.IsDirectory {
			return nil, errors.New("invalid image attachment reference")
		}
		modifiedAt, err := parseReferenceTime(reference.Image.ModifiedAt)
		if err != nil {
			return nil, err
		}
		originalModTime, err := parseReferenceTime(reference.Image.OriginalMod)
		if err != nil {
			return nil, err
		}
		originalMod := time.Time{}
		if originalModTime != nil {
			originalMod = *originalModTime
		}
		return []Attachment{{
			Kind:                       AttachmentImage,
			Label:                      reference.Label,
			Mode:                       referenceMode(reference.Mode, AttachmentContentBlockModeImage),
			AvailableContentBlockModes: referenceModes(reference.AvailableModes),
			ImageRef: &ImageRef{
				Path:        reference.Image.Path,
				Name:        reference.Image.Name,
				Exists:      reference.Image.Exists,
				IsDir:       false,
				Size:        reference.Image.Size,
				ModTime:     modifiedAt,
				Width:       reference.Image.Width,
				Height:      reference.Image.Height,
				Format:      reference.Image.Format,
				MIMEType:    MIMEType(reference.Image.MIMEType),
				OrigPath:    reference.Image.OriginalPath,
				OrigSize:    reference.Image.OriginalSize,
				OrigModTime: originalMod,
			},
		}}, nil

	case sessionRuntime.AttachmentReferenceURL:
		if reference.URL == nil {
			return nil, errors.New("invalid URL attachment reference")
		}
		return []Attachment{{
			Kind:                       AttachmentURL,
			Label:                      reference.Label,
			Mode:                       referenceMode(reference.Mode, AttachmentContentBlockModePageContent),
			AvailableContentBlockModes: referenceModes(reference.AvailableModes),
			URLRef: &URLRef{
				URL:            reference.URL.URL,
				Normalized:     reference.URL.Normalized,
				OrigNormalized: reference.URL.Original,
			},
		}}, nil

	case sessionRuntime.AttachmentReferenceGeneric:
		if reference.Generic == nil {
			return nil, errors.New("invalid generic attachment reference")
		}
		return []Attachment{{
			Kind:  AttachmentDocIndex,
			Label: reference.Label,
			Mode:  AttachmentContentBlockModeTextLink,
			GenericRef: &GenericRef{
				Handle:     reference.Generic.Handle,
				OrigHandle: reference.Generic.OriginalHandle,
			},
		}}, nil

	default:
		return nil, fmt.Errorf(
			"unsupported attachment reference kind %q",
			reference.Kind,
		)
	}
}

func attachmentReferenceFromAttachment(
	attachment *Attachment,
) (sessionRuntime.AttachmentReference, error) {
	if attachment == nil {
		return sessionRuntime.AttachmentReference{},
			errors.New("cannot persist a nil attachment")
	}

	output := sessionRuntime.AttachmentReference{
		Label:          attachment.Label,
		Mode:           string(attachment.Mode),
		AvailableModes: attachmentModeStrings(attachment.AvailableContentBlockModes),
	}

	switch attachment.Kind {
	case AttachmentFile:
		if attachment.FileRef == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("file attachment has no file reference")
		}
		ref := attachment.FileRef
		output.Kind = sessionRuntime.AttachmentReferenceFile
		output.File = &sessionRuntime.LocalAttachmentReference{
			Path:         ref.Path,
			OriginalPath: ref.OrigPath,
			Name:         ref.Name,
			Exists:       ref.Exists,
			IsDirectory:  ref.IsDir,
			Size:         ref.Size,
			OriginalSize: ref.OrigSize,
			ModifiedAt:   formatReferenceTime(ref.ModTime),
			OriginalMod:  formatReferenceTimeValue(ref.OrigModTime),
		}
		return output, nil

	case AttachmentImage:
		if attachment.ImageRef == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("image attachment has no image reference")
		}
		ref := attachment.ImageRef
		output.Kind = sessionRuntime.AttachmentReferenceImage
		output.Image = &sessionRuntime.LocalAttachmentReference{
			Path:         ref.Path,
			OriginalPath: ref.OrigPath,
			Name:         ref.Name,
			Exists:       ref.Exists,
			IsDirectory:  ref.IsDir,
			Size:         ref.Size,
			OriginalSize: ref.OrigSize,
			ModifiedAt:   formatReferenceTime(ref.ModTime),
			OriginalMod:  formatReferenceTimeValue(ref.OrigModTime),
			MIMEType:     string(ref.MIMEType),
			Width:        ref.Width,
			Height:       ref.Height,
			Format:       ref.Format,
		}
		return output, nil

	case AttachmentURL:
		if attachment.URLRef == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("URL attachment has no URL reference")
		}
		ref := attachment.URLRef
		output.Kind = sessionRuntime.AttachmentReferenceURL
		output.URL = &sessionRuntime.URLAttachmentReference{
			URL:        ref.URL,
			Normalized: ref.Normalized,
			Original:   ref.OrigNormalized,
		}
		return output, nil

	case AttachmentDocIndex, AttachmentPR, AttachmentCommit:
		if attachment.GenericRef == nil {
			return sessionRuntime.AttachmentReference{},
				errors.New("generic attachment has no generic reference")
		}
		ref := attachment.GenericRef
		output.Kind = sessionRuntime.AttachmentReferenceGeneric
		output.Generic = &sessionRuntime.GenericAttachmentReference{
			Handle:         ref.Handle,
			OriginalHandle: ref.OrigHandle,
		}
		return output, nil

	default:
		return sessionRuntime.AttachmentReference{},
			fmt.Errorf("attachment kind %q cannot be persisted as a local reference",
				attachment.Kind,
			)
	}
}

func parseReferenceTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		//nolint:nilnil // Ok.
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("invalid attachment timestamp %q: %w", value, err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func cloneAttachmentTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func formatReferenceTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatReferenceTimeValue(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func referenceMode(
	value string,
	fallback AttachmentContentBlockMode,
) AttachmentContentBlockMode {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return AttachmentContentBlockMode(value)
}

func referenceModes(values []string) []AttachmentContentBlockMode {
	if len(values) == 0 {
		return nil
	}
	output := make([]AttachmentContentBlockMode, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			output = append(output, AttachmentContentBlockMode(value))
		}
	}
	return output
}

func attachmentModeStrings(
	values []AttachmentContentBlockMode,
) []string {
	if len(values) == 0 {
		return nil
	}
	output := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			output = append(output, string(value))
		}
	}
	return output
}
