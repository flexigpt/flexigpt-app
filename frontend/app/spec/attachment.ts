// AttachmentKind enumerates contextual attachment categories that can be
// associated with messages sent to the inference layer.
/**
 * @public
 */
export enum AttachmentKind {
	file = 'file',
	image = 'image',
	url = 'url',

	docIndex = 'docIndex',
	pr = 'pr',
	commit = 'commit',
}

export enum AttachmentContentBlockMode {
	text = 'text',
	file = 'file',
	image = 'image',

	page = 'page',
	textlink = 'textlink',
	imageurl = 'imageurl',
	fileurl = 'fileurl',

	notReadable = 'not-readable',

	prDiff = 'pr-diff',
	prPage = 'pr-page',
	commitDiff = 'commit-diff',
	commitPage = 'commit-page',
}

// Mapping from mode value → user-facing label (for the mode dropdown/badge).
export const ATTACHMENT_MODE_LABELS: Record<AttachmentContentBlockMode, string> = {
	[AttachmentContentBlockMode.text]: 'Text',
	[AttachmentContentBlockMode.file]: 'File',
	[AttachmentContentBlockMode.image]: 'Image',
	[AttachmentContentBlockMode.page]: 'Page content',
	[AttachmentContentBlockMode.textlink]: 'Link as text',
	[AttachmentContentBlockMode.imageurl]: 'Image as URL',
	[AttachmentContentBlockMode.fileurl]: 'File as URL',
	[AttachmentContentBlockMode.notReadable]: 'Not readable',
	[AttachmentContentBlockMode.prDiff]: 'PR diff only',
	[AttachmentContentBlockMode.prPage]: 'PR page',
	[AttachmentContentBlockMode.commitDiff]: 'Diff only',
	[AttachmentContentBlockMode.commitPage]: 'Commit page',
};

export const ATTACHMENT_MODE_DESC: Record<AttachmentContentBlockMode, string> = {
	[AttachmentContentBlockMode.text]: 'Inline text extracted into the message.',
	[AttachmentContentBlockMode.file]: 'Send the file as a binary attachment (requires API support).',
	[AttachmentContentBlockMode.image]: 'Send the image as a binary attachment (requires API support).',
	[AttachmentContentBlockMode.page]: 'Text content extracted from the HTML page.',
	[AttachmentContentBlockMode.textlink]: 'Only send the link as text; do not fetch or parse content.',
	[AttachmentContentBlockMode.imageurl]: 'Send the link as Image URL attachment; do not fetch or parse content.',
	[AttachmentContentBlockMode.fileurl]: 'Send the link as File URL attachment; do not fetch or parse content.',
	[AttachmentContentBlockMode.notReadable]: 'This file could not be read.',
	[AttachmentContentBlockMode.prDiff]: 'Only the pull request diff is sent.',
	[AttachmentContentBlockMode.prPage]: 'Send the pull request page content.',
	[AttachmentContentBlockMode.commitDiff]: 'Only the commit diff is sent.',
	[AttachmentContentBlockMode.commitPage]: 'Send the commit page content.',
};

interface AttachmentFileRef {
	path: string;
	name: string;
	exists: boolean;
	isDir: boolean;
	size?: number;
	// Go type: time
	modTime?: Date;

	origPath: string;
	origSize: number;
	origModTime: Date;
}

interface AttachmentImageRef {
	path: string;
	name: string;
	exists: boolean;
	isDir: boolean;
	size?: number;
	// Go type: time
	modTime?: Date;

	width?: number;
	height?: number;
	format?: string;
	mimeType?: string;

	origPath: string;
	origSize: number;
	origModTime: Date;
}

interface AttachmentURLRef {
	url: string;
	normalized?: string;
	origNormalized: string;
}

interface AttachmentGenericRef {
	handle: string;
	origHandle: string;
}

enum AttachmentContentBlockKind {
	text = 'text',
	image = 'image',
	file = 'file',
}

interface ContentBlock {
	kind: AttachmentContentBlockKind;
	text?: string;

	mimeType?: string;
	fileName?: string;
	base64Data?: string;
	url?: string;
}

// Attachment references contextual artifacts (files, images, doc handles, etc.).
export interface Attachment {
	kind: AttachmentKind;
	label: string;

	mode?: AttachmentContentBlockMode;
	availableContentBlockModes?: AttachmentContentBlockMode[];

	fileRef?: AttachmentFileRef;
	imageRef?: AttachmentImageRef;
	urlRef?: AttachmentURLRef;
	genericRef?: AttachmentGenericRef;

	contentBlock?: ContentBlock;
}

export interface FileFilter {
	DisplayName: string;
	Extensions: string[];
}

export interface DirectoryOverflowInfo {
	dirPath: string;
	relativePath: string;
	fileCount: number;
	partial: boolean;
}

export interface DirectoryAttachmentsResult {
	dirPath: string;
	attachments: Attachment[];
	overflowDirs: DirectoryOverflowInfo[];
	maxFiles: number;
	totalSize: number;
	hasMore: boolean;
}

export interface PathAttachmentsResult {
	fileAttachments: Attachment[];
	dirAttachments: DirectoryAttachmentsResult[];
	errors?: string[];
}

export enum AttachmentErrorReason {
	TooLargeSingle = 'too-large-single',
	TooLargeTotal = 'too-large-total',
	Unreadable = 'unreadable',
}

export type AttachmentsDroppedPayload = {
	dropID: string;
	x: number;
	y: number;

	// Built by backend (Go) already:
	files: Attachment[];
	directories: DirectoryAttachmentsResult[];

	errors?: string[];
	maxFilesPerDirectory: number;
};

/**
 * UIAttachment is the composer-side shape; it extends the backend
 * Attachment with required mode/availableContentBlockModes and error flags.
 */
export interface UIAttachment extends Attachment {
	mode: AttachmentContentBlockMode;
	availableContentBlockModes: AttachmentContentBlockMode[];
	isError?: boolean;
	errorReason?: AttachmentErrorReason;
}
