package attachment

// AttachmentContentBlockMode describes how a given attachment should be used
// for the current turn.
type AttachmentContentBlockMode string

const (
	AttachmentContentBlockModeText  AttachmentContentBlockMode = "text"  // "Text content"
	AttachmentContentBlockModeFile  AttachmentContentBlockMode = "file"  // "File (original format)"
	AttachmentContentBlockModeImage AttachmentContentBlockMode = "image" // Image rendering

	AttachmentContentBlockModePageContent AttachmentContentBlockMode = "page"     // "Page content" for HTML/URLs
	AttachmentContentBlockModeTextLink    AttachmentContentBlockMode = "textlink" // "Link as text block" – no fetch
	AttachmentContentBlockModeImageURL    AttachmentContentBlockMode = "imageurl"
	AttachmentContentBlockModeFileURL     AttachmentContentBlockMode = "fileurl"

	AttachmentContentBlockModeNotReadable AttachmentContentBlockMode = "not-readable" // Binary/unknown – cannot process

	AttachmentContentBlockModePRDiff     AttachmentContentBlockMode = "pr-diff"
	AttachmentContentBlockModePRPage     AttachmentContentBlockMode = "pr-page"
	AttachmentContentBlockModeCommitDiff AttachmentContentBlockMode = "commit-diff"
	AttachmentContentBlockModeCommitPage AttachmentContentBlockMode = "commit-page"
)

type AttachmentContentBlockKind string

const (
	ContentBlockText  AttachmentContentBlockKind = "text"
	ContentBlockImage AttachmentContentBlockKind = "image"
	ContentBlockFile  AttachmentContentBlockKind = "file"
)

// ContentBlock represents a provider-agnostic chunk of content derived from an Attachment.
// Providers (OpenAI, Anthropic, etc.) adapt this into their own message/part formats.
type ContentBlock struct {
	Kind AttachmentContentBlockKind `json:"kind"`

	// For Kind == text: Text, MIMEType, FileName and FilePath are populated when read from local file.
	// For URL based text attachments, URL will be present and MIMEType _may_ be present.
	Text     *string `json:"text,omitempty"`
	MIMEType *string `json:"mimeType,omitempty"`
	FileName *string `json:"fileName,omitempty"`
	FilePath *string `json:"filePath,omitempty"`

	// For Kind == image or file: Base64Data is populated for local attachments.
	// FileName and FilePath are populated for local attachments when available.
	// For URL based attachments, Base64Data may or may not be present depending on AttachmentMode.
	Base64Data *string `json:"base64Data,omitempty"`

	// URL is populated for URL-based attachments.
	URL *string `json:"url,omitempty"`
}
