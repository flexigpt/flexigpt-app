package attachment

// AttachmentReferenceKind identifies a durable semantic Attachment reference.
// Session stores these values, but Attachment owns their meaning and
// materialization contract.
type AttachmentReferenceKind string

const (
	AttachmentReferenceFile      AttachmentReferenceKind = "file"
	AttachmentReferenceDirectory AttachmentReferenceKind = "directory"
	AttachmentReferenceImage     AttachmentReferenceKind = "image"
	AttachmentReferenceURL       AttachmentReferenceKind = "url"
	AttachmentReferenceGeneric   AttachmentReferenceKind = "generic"
)

// LocalAttachmentReference records a local-file snapshot. Original fields are
// captured during admission and protect historical replay from silently using
// changed local content.
type LocalAttachmentReference struct {
	Path         string `json:"path,omitempty"`
	OriginalPath string `json:"originalPath,omitempty"`
	Name         string `json:"name,omitempty"`
	Exists       bool   `json:"exists"`
	IsDirectory  bool   `json:"isDirectory,omitempty"`
	Size         int64  `json:"size,omitempty"`
	OriginalSize int64  `json:"originalSize,omitempty"`
	ModifiedAt   string `json:"modifiedAt,omitempty"`
	OriginalMod  string `json:"originalModifiedAt,omitempty"`
	MIMEType     string `json:"mimeType,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Format       string `json:"format,omitempty"`
}

type URLAttachmentReference struct {
	URL        string `json:"url"`
	Normalized string `json:"normalized,omitempty"`
	Original   string `json:"original,omitempty"`
}

type GenericAttachmentReference struct {
	Handle         string `json:"handle"`
	OriginalHandle string `json:"originalHandle,omitempty"`
}

// DirectoryAttachmentReference stores the materialized child references that
// were selected at submission time. Replay must not walk the current
// filesystem tree again.
type DirectoryAttachmentReference struct {
	Path             string                `json:"path"`
	OriginalPath     string                `json:"originalPath,omitempty"`
	Name             string                `json:"name,omitempty"`
	ScannedFileCount int                   `json:"scannedFileCount,omitempty"`
	HasMore          bool                  `json:"hasMore,omitempty"`
	Expanded         []AttachmentReference `json:"expanded,omitempty"`
}

// AttachmentReference is the durable Attachment-domain value used by Session
// transcript and continuation state. It contains no open file, native handle,
// HTTP client, or fetched content.
type AttachmentReference struct {
	Kind           AttachmentReferenceKind       `json:"kind"`
	Label          string                        `json:"label,omitempty"`
	Mode           string                        `json:"mode,omitempty"`
	AvailableModes []string                      `json:"availableModes,omitempty"`
	File           *LocalAttachmentReference     `json:"file,omitempty"`
	Image          *LocalAttachmentReference     `json:"image,omitempty"`
	URL            *URLAttachmentReference       `json:"url,omitempty"`
	Generic        *GenericAttachmentReference   `json:"generic,omitempty"`
	Directory      *DirectoryAttachmentReference `json:"directory,omitempty"`
}
