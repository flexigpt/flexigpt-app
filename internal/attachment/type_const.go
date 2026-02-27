package attachment

import (
	"errors"
	"time"
)

var (
	ErrNonTextContentBlock             = errors.New("content block is not of kind - text")
	ErrUnreadableFile                  = errors.New("unreadable file")
	ErrExistingContentBlock            = errors.New("content block already exists")
	ErrAttachmentModifiedSinceSnapshot = errors.New("attachment modified since snapshot")

	// ErrResponseTooLarge is used internally by fetchURLBytes to signal
	// that the remote resource exceeded the configured maxBytes limit.
	ErrResponseTooLarge = errors.New("remote resource larger than the configured limit")

	// ErrPageTooLarge is returned by ExtractReadableMarkdownFromURL when
	// the HTML page exceeds maxURLPageContentBytes.
	ErrPageTooLarge = errors.New("page content too large")

	// ErrNoContentExtracted is returned when the extractor produces no
	// readable content at all (e.g., empty body, or extraction failure).
	ErrNoContentExtracted = errors.New("no readable content extracted")
)

const (
	maxAttachmentFetchBytes = 16 * 1024 * 1024 // 16MB safety limit
	maxTotalDirWalkFiles    = 256
	// Max number of bytes we will download and process as "page content"
	// for LLM consumption. If a page is larger than this, we will return
	// ErrPageTooLarge and fall back to link-only at a higher layer.
	maxURLPageContentBytes = 16 << 20 // 16 MiB
	defaultHTTPTimeout     = 30 * time.Second
)

// AttachmentKind enumerates contextual attachment categories that can be
// associated with messages sent to the inference layer.
type AttachmentKind string

const (
	AttachmentFile     AttachmentKind = "file"
	AttachmentImage    AttachmentKind = "image"
	AttachmentURL      AttachmentKind = "url"
	AttachmentDocIndex AttachmentKind = "docIndex"
	AttachmentPR       AttachmentKind = "pr"
	AttachmentCommit   AttachmentKind = "commit"
)

type PathInfo struct {
	Path    string     `json:"path"`
	Name    string     `json:"name"`
	Exists  bool       `json:"exists"`
	IsDir   bool       `json:"isDir"`
	Size    int64      `json:"size,omitempty"`
	ModTime *time.Time `json:"modTime,omitempty"`
}

// DirectoryOverflowInfo represents a directory that was *not fully walked*
// because we hit the max-files limit or had an error.
//
// Semantics:
//
//   - DirPath / RelativePath:
//     Absolute / relative-to-root path of that directory.
//   - FileCount:
//     For completely unvisited dirs (left in the BFS queue):
//   - number of direct entries (files + subdirs) from a single os.ReadDir,
//     no recursion.
//     For the single "partial" dir where we hit the limit mid-scan:
//   - number of remaining entries in that directory (files + subdirs)
//     that we did NOT process.
//     This is *approximate UI sugar*, not a full subtree count.
//   - Partial:
//     true only for the directory where we stopped in the middle of its
//     entries because maxFiles was reached. For all other overflow dirs it's
//     false.
type DirectoryOverflowInfo struct {
	DirPath      string `json:"dirPath"`
	RelativePath string `json:"relativePath"`
	FileCount    int    `json:"fileCount"`
	Partial      bool   `json:"partial"`
}

// WalkDirectoryWithFilesResult is returned when user selects a directory.
type WalkDirectoryWithFilesResult struct {
	DirPath      string                  `json:"dirPath"`
	Files        []PathInfo              `json:"files"`        // included files (flattened)
	OverflowDirs []DirectoryOverflowInfo `json:"overflowDirs"` // directories not fully included
	MaxFiles     int                     `json:"maxFiles"`     // max number of files returned (after clamping)
	TotalSize    int64                   `json:"totalSize"`    // sum of Files[i].Size
	HasMore      bool                    `json:"hasMore"`      // true if not all content included
}

type DirectoryAttachmentsResult struct {
	DirPath      string                  `json:"dirPath"`
	Attachments  []Attachment            `json:"attachments"`  // included attachments (flattened)
	OverflowDirs []DirectoryOverflowInfo `json:"overflowDirs"` // directories not fully included
	MaxFiles     int                     `json:"maxFiles"`     // max number of files returned (after clamping)
	TotalSize    int64                   `json:"totalSize"`    // sum of Files[i].Size
	HasMore      bool                    `json:"hasMore"`      // true if not all content included
}

type PathAttachmentsResult struct {
	FileAttachments []Attachment                 `json:"fileAttachments"`
	DirAttachments  []DirectoryAttachmentsResult `json:"dirAttachments"`
	Errors          []string                     `json:"errors,omitempty"`
}
