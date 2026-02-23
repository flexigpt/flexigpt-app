// ---- Large-text perf tuning ----
// If you type huge prompts (10k+ words), a single giant text leaf becomes slow.
// We keep ONE paragraph, but chunk its text into multiple text nodes ("leaves").
import { NodeApi, type Value } from 'platejs';
import type { PlateEditor, usePlateEditor } from 'platejs/react';

import type { Attachment, UIAttachment } from '@/spec/attachment';
import type { UIToolOutput } from '@/spec/inference';
import type { SkillDef } from '@/spec/skill';
import type { ToolStoreChoice, UIToolStoreChoice } from '@/spec/tool';

import { expandTabsToSpaces } from '@/lib/text_utils';

// We add a per-chunk prop so Slate doesn't merge them back together.
export const LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS = 10000; // start chunking once draft grows beyond this
export const LARGE_TEXT_AUTODECHUNK_THRESHOLD_CHARS = 5000; // merge back once it shrinks below this
export const LARGE_TEXT_CHUNK_SIZE = 1600; // each leaf ~1.2k chars; adjust 800–2000 based on taste

export interface EditorExternalMessage {
	text: string;
	attachments?: Attachment[];
	toolChoices?: ToolStoreChoice[];
	toolOutputs?: UIToolOutput[];
	enabledSkills?: SkillDef[];
}

export interface EditorSubmitPayload {
	text: string;
	attachedTools: UIToolStoreChoice[];
	attachments: UIAttachment[];
	toolOutputs: UIToolOutput[];
	finalToolChoices: ToolStoreChoice[];
	enabledSkills?: SkillDef[];
}

type ChunkedTextNode = { text: string; __chunk: number };

function buildChunkedTextChildren(text: string, chunkSize: number): ChunkedTextNode[] {
	if (!text) return [{ text: '', __chunk: 0 }];
	const out: ChunkedTextNode[] = [];
	let i = 0;
	let idx = 0;
	while (i < text.length) {
		out.push({ text: text.slice(i, i + chunkSize), __chunk: idx++ });
		i += chunkSize;
	}
	return out;
}

export function buildSingleParagraphValue(text: string): Value {
	return [{ type: 'p', children: [{ text }] }];
}

export function buildSingleParagraphValueChunked(text: string, chunkSize: number): Value {
	return [{ type: 'p', children: buildChunkedTextChildren(text, chunkSize) }];
}

export function isCursorAtDocumentEnd(editor: PlateEditor): boolean {
	try {
		const sel = editor.selection;
		if (!sel) return false;
		const end = editor.api.end([]);
		return JSON.stringify(sel.anchor.path) === JSON.stringify(end?.path) && sel.anchor.offset === end?.offset;
	} catch {
		return false;
	}
}

export const clearAllMarks = (ed: PlateEditor) => {
	const marks = ['bold', 'italic', 'underline', 'strikethrough', 'code', 'sub', 'sup', 'highlight'];
	ed.tf.withoutNormalizing(() => {
		for (const m of marks) {
			try {
				ed.tf.removeMark(m);
			} catch {
				// Ok.
			}
		}
	});
};

export function insertPlainTextAsSingleBlock(ed: ReturnType<typeof usePlateEditor>, text: string, tabSize = 2) {
	if (!ed) return;
	const editor = ed as PlateEditor;

	// Normalize line endings
	const normalized = text.replace(/\r\n?/g, '\n');

	// Expand tabs, but keep everything as one string
	const expanded = normalized
		.split('\n')
		.map(line => expandTabsToSpaces(line, tabSize))
		.join('\n');

	// Single transform instead of O(number of lines)
	editor.tf.withoutNormalizing(() => {
		editor.tf.insertText(expanded);
		// optional: only useful if you might be replacing a range selection
		editor.tf.collapse({ edge: 'end' });
	});
}

export function hasNonEmptyUserText(ed: PlateEditor | null | undefined): boolean {
	if (!ed) return false;
	// If NodeApi.texts exists:
	for (const [t] of NodeApi.texts(ed)) {
		if (t.text.trim().length > 0) return true;
	}
	return false;
}
