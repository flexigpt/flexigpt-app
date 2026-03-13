import {
	type ClipboardEvent as ReactClipboardEvent,
	type RefObject,
	useCallback,
	useDeferredValue,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import { SingleBlockPlugin, type Value } from 'platejs';
import { type PlateEditor, usePlateEditor } from 'platejs/react';

import { compareEntryByPathDeepestFirst } from '@/lib/path_utils';
import { cssEscape } from '@/lib/text_utils';

import { AlignKit } from '@/components/editor/plugins/align_kit';
import { BasicBlocksKit } from '@/components/editor/plugins/basic_blocks_kit';
import { BasicMarksKit } from '@/components/editor/plugins/basic_marks_kit';
import { FloatingToolbarKit } from '@/components/editor/plugins/floating_toolbar_kit';
import { IndentKit } from '@/components/editor/plugins/indent_kit';
import { LineHeightKit } from '@/components/editor/plugins/line_height_kit';
import { ListKit } from '@/components/editor/plugins/list_kit';
import { TabbableKit } from '@/components/editor/plugins/tabbable_kit';

import {
	buildSingleParagraphValue,
	buildSingleParagraphValueChunked,
	clearAllMarks,
	createEmptyEditorValue,
	hasNonEmptyUserText,
	insertPlainTextAsSingleBlock,
	isCursorAtDocumentEnd,
	isSimpleEmptyParagraphDocument,
	LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_AUTODECHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_CHUNK_SIZE,
} from '@/chats/inputarea/input_editor_utils';
import {
	analyzeTemplateSelectionInfo,
	type ComposerDocumentSelectionInfo,
	isSelectionOnlyEditorChange,
} from '@/chats/platedoc/document_analysis';
import { getTemplateNodesWithPath } from '@/chats/templates/template_editor_utils';
import { TemplateSlashKit } from '@/chats/templates/template_plugin';
import { getLastUserBlockContent } from '@/chats/templates/template_processing';
import { buildUserInlineChildrenFromText } from '@/chats/templates/template_variables_inline';
import { getToolNodesWithPath } from '@/chats/tools/tool_editor_utils';
import { ToolPlusKit } from '@/chats/tools/tool_plugin';

const createEditorPlugins = () => [
	SingleBlockPlugin,
	...BasicBlocksKit,
	...BasicMarksKit,
	...LineHeightKit,
	...AlignKit,
	...IndentKit,
	...ListKit,
	// ...AutoformatKit, // Don't want any formatting on typing
	...TabbableKit,
	...TemplateSlashKit,
	...ToolPlusKit,
	...FloatingToolbarKit,
];

type ReplaceEditorDocumentFocusMode = 'none' | 'preserve' | 'end';

interface UseComposerDocumentArgs {
	isBusy: boolean;
}

interface UseComposerDocumentResult {
	editor: PlateEditor;
	contentRef: RefObject<HTMLDivElement | null>;
	hasText: boolean;
	hasTextRef: RefObject<boolean>;
	selectionInfo: ComposerDocumentSelectionInfo;
	attachedToolEntries: ReturnType<typeof getToolNodesWithPath>;
	onEditorChange: () => boolean;
	onEditorPaste: (event: ReactClipboardEvent<HTMLDivElement>) => void;
	replaceEditorDocument: (nextValue: Value, nextHasText: boolean, focus?: ReplaceEditorDocumentFocusMode) => void;
	resetEditorDocument: () => void;
	focusEditorAtEnd: () => void;
	focusEditorPreservingSelection: () => void;
}

export function useComposerDocument({ isBusy }: UseComposerDocumentArgs): UseComposerDocumentResult {
	const initialEditorValue = useMemo<Value>(() => createEmptyEditorValue(), []);
	const editorPlugins = useMemo(() => createEditorPlugins(), []);
	const editor = usePlateEditor({
		plugins: editorPlugins,
		value: initialEditorValue,
	});

	const contentRef = useRef<HTMLDivElement | null>(null);
	const editorRef = useRef(editor);
	editorRef.current = editor; // keep a live ref for callbacks / async work

	// doc version tick to re-run selection computations on any editor change
	const [docVersion, setDocVersion] = useState(0);
	const deferredDocVersion = useDeferredValue(docVersion);

	// Cache "has text" so we don't re-scan the editor tree multiple times per render.
	const [hasText, setHasText] = useState(false);
	const hasTextRef = useRef(false);
	const isAutoChunkingRef = useRef(false);
	const lastPopulatedSelectionKeyRef = useRef<Set<string>>(new Set());

	// Throttle docVersion bumps to at most 1/frame to avoid re-render storms on big documents.
	const docRafRef = useRef<number | null>(null);
	const scheduleDocRecompute = useCallback(() => {
		if (docRafRef.current != null) return;
		docRafRef.current = window.requestAnimationFrame(() => {
			docRafRef.current = null;
			setDocVersion(v => v + 1);
			const nextHasText = hasNonEmptyUserText(editorRef.current);
			hasTextRef.current = nextHasText;
			setHasText(nextHasText);
		});
	}, []);

	useEffect(() => {
		return () => {
			if (docRafRef.current != null) window.cancelAnimationFrame(docRafRef.current);
		};
	}, []);

	// Auto-chunking job (runs outside the keystroke critical path).
	const idleChunkJobRef = useRef<number | null>(null);

	const autoChunkIfNeeded = useCallback(() => {
		if (isAutoChunkingRef.current) return;
		if (!contentRef.current) return;

		const domLen = (contentRef.current.textContent ?? '').length;
		const ed = editorRef.current;

		// Only operate on the simplest safe shape:
		// - single paragraph
		// - only text children
		const rootChildren = ed.children ?? [];
		if (rootChildren.length !== 1) return;
		const p = rootChildren[0];
		if (!p || p.type !== 'p' || !Array.isArray(p.children)) return;
		const pChildren = p.children;
		if (pChildren.some(c => typeof c?.text !== 'string')) return; // any inline element => skip

		// Avoid cursor jumps: only chunk/dechunk when user is typing at the end.
		if (!isCursorAtDocumentEnd(ed)) return;

		// Chunk: 1 huge leaf -> many leaves
		if (domLen >= LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS && pChildren.length === 1) {
			const text = ed.api.string([]);
			if (text.length < LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS) return;

			isAutoChunkingRef.current = true;
			try {
				ed.tf.withoutNormalizing(() => {
					ed.tf.setValue(buildSingleParagraphValueChunked(text, LARGE_TEXT_CHUNK_SIZE));
				});
				ed.tf.select(undefined, { edge: 'end' });
			} finally {
				isAutoChunkingRef.current = false;
			}
			return;
		}

		// Dechunk: many leaves -> 1 leaf (when user deletes a lot)
		if (domLen <= LARGE_TEXT_AUTODECHUNK_THRESHOLD_CHARS && pChildren.length > 1) {
			const text = ed.api.string([]);
			if (text.length > LARGE_TEXT_AUTODECHUNK_THRESHOLD_CHARS) return;

			isAutoChunkingRef.current = true;
			try {
				ed.tf.withoutNormalizing(() => {
					ed.tf.setValue(buildSingleParagraphValue(text));
				});
				ed.tf.select(undefined, { edge: 'end' });
			} finally {
				isAutoChunkingRef.current = false;
			}
		}
	}, []);

	const scheduleAutoChunk = useCallback(() => {
		if (idleChunkJobRef.current != null) return;

		// requestIdleCallback is ideal; fall back to a short timeout.
		const w = window;
		if (typeof w.requestIdleCallback === 'function') {
			idleChunkJobRef.current = w.requestIdleCallback(
				() => {
					idleChunkJobRef.current = null;
					autoChunkIfNeeded();
				},
				{ timeout: 250 }
			);
		} else {
			idleChunkJobRef.current = window.setTimeout(() => {
				idleChunkJobRef.current = null;
				autoChunkIfNeeded();
			}, 120);
		}
	}, [autoChunkIfNeeded]);

	useEffect(() => {
		return () => {
			const w = window;
			if (idleChunkJobRef.current != null && typeof w.cancelIdleCallback === 'function') {
				w.cancelIdleCallback(idleChunkJobRef.current);
			} else if (idleChunkJobRef.current != null) {
				window.clearTimeout(idleChunkJobRef.current);
			}
		};
	}, []);

	const focusEditorPreservingSelection = useCallback(() => {
		const currentEditor = editorRef.current;
		if (!currentEditor || isBusy) return;

		requestAnimationFrame(() => {
			try {
				currentEditor.tf.focus();
			} catch {
				// noop
			}
		});
	}, [isBusy]);

	const focusEditorAtEnd = useCallback(() => {
		const currentEditor = editorRef.current;
		if (!currentEditor) return;

		// No visible caret in readOnly mode.
		if (isBusy) return;

		requestAnimationFrame(() => {
			try {
				currentEditor.tf.withoutNormalizing(() => {
					currentEditor.tf.select(undefined, { edge: 'end' });
					currentEditor.tf.collapse({ edge: 'end' });
				});

				currentEditor.tf.focus();

				// Keep end visible if content is long
				if (contentRef.current) {
					contentRef.current.scrollTop = contentRef.current.scrollHeight;
				}
			} catch {
				currentEditor.tf.focus();
			}
		});
	}, [isBusy]);

	// eslint-disable-next-line react-hooks/exhaustive-deps
	const selectionInfo = useMemo(() => analyzeTemplateSelectionInfo(editor), [editor, deferredDocVersion]);

	// Intentionally not memoized only against docVersion: some editor mutations can
	// cause unrelated parent rerenders before the throttled doc tick lands, and we
	// still want attached tool chips to reflect the current editor tree on that render.
	const attachedToolEntries = getToolNodesWithPath(editor);

	// Populate editor with effective last-USER block for EACH template selection (once per selectionID)
	useEffect(() => {
		// eslint-disable-next-line react-you-might-not-need-an-effect/no-event-handler
		if (!selectionInfo.tplNodeWithPath) return;
		const populated = lastPopulatedSelectionKeyRef.current;
		const nodes = getTemplateNodesWithPath(editor);
		const insertedIds: string[] = [];

		// Process in reverse document order to keep captured paths valid
		const nodesRev = [...nodes].sort(compareEntryByPathDeepestFirst);

		for (const [tsenode, originalPath] of nodesRev) {
			if (!tsenode || !tsenode.selectionID) continue;
			const selectionID: string = tsenode.selectionID;
			if (populated.has(selectionID)) continue;

			// Build children: keep the selection chip, add parsed user text with variable pills
			const userText = getLastUserBlockContent(tsenode);
			const inlineChildren = buildUserInlineChildrenFromText(tsenode, userText);

			try {
				editor.tf.withoutNormalizing(() => {
					// Recompute a fresh path to guard against prior insertions shifting indices
					const pathArr = Array.isArray(originalPath) ? (originalPath as number[]) : [];

					if (pathArr.length >= 2) {
						const blockPath = pathArr.slice(0, pathArr.length - 1); // parent paragraph path
						const indexAfter = pathArr[pathArr.length - 1] + 1;
						const atPath = [...blockPath, indexAfter] as any;
						editor.tf.insertNodes(inlineChildren, { at: atPath });
					} else {
						// Fallback: insert at start of first paragraph
						editor.tf.insertNodes(inlineChildren, { at: [0, 0] as any });
					}
				});
			} catch {
				// Last-resort fallback: insert at selection (or end)
				editor.tf.insertNodes(inlineChildren);
			}

			populated.add(selectionID);
			insertedIds.push(selectionID);
		}

		// Focus first variable pill of the last inserted selection (if any)
		if (insertedIds.length > 0) {
			const focusId = insertedIds[insertedIds.length - 1];
			requestAnimationFrame(() => {
				try {
					const sel = contentRef.current?.querySelector(
						`span[data-template-variable][data-selection-id="${cssEscape(focusId)}"]`
					) as HTMLElement | null;
					if (sel && 'focus' in sel && typeof sel.focus === 'function') {
						sel.focus();
					} else {
						focusEditorAtEnd();
					}
				} catch {
					focusEditorAtEnd();
				}
			});
		}
	}, [editor, focusEditorAtEnd, deferredDocVersion, selectionInfo.tplNodeWithPath]);

	const replaceEditorDocument = useCallback(
		(nextValue: Value, nextHasText: boolean, focus: ReplaceEditorDocumentFocusMode = 'none') => {
			const currentEditor = editorRef.current;
			if (!currentEditor) return;

			lastPopulatedSelectionKeyRef.current.clear();

			currentEditor.tf.withoutNormalizing(() => {
				try {
					currentEditor.tf.deselect();
				} catch {
					// noop
				}
				currentEditor.tf.setValue(nextValue);
			});

			hasTextRef.current = nextHasText;
			setHasText(nextHasText);
			setDocVersion(v => v + 1);

			if (focus === 'end') {
				focusEditorAtEnd();
			} else if (focus === 'preserve') {
				focusEditorPreservingSelection();
			}
		},
		[focusEditorAtEnd, focusEditorPreservingSelection]
	);

	const resetEditorDocument = useCallback(() => {
		replaceEditorDocument(createEmptyEditorValue(), false, 'end');
	}, [replaceEditorDocument]);

	const onEditorPaste = useCallback((e: ReactClipboardEvent<HTMLDivElement>) => {
		e.preventDefault();
		e.stopPropagation();

		const text = e.clipboardData.getData('text/plain');
		if (!text) return;

		const currentEditor = editorRef.current;
		clearAllMarks(currentEditor);

		// PERF: if paste is huge AND the doc is truly the default empty paragraph,
		// set chunked value directly. Do not use "has text" as a proxy for
		// emptiness, or we can blow away template/tool nodes.
		if (isSimpleEmptyParagraphDocument(currentEditor) && text.length >= LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS) {
			currentEditor.tf.withoutNormalizing(() => {
				currentEditor.tf.setValue(buildSingleParagraphValueChunked(text, LARGE_TEXT_CHUNK_SIZE));
				currentEditor.tf.collapse({ edge: 'end' });
			});

			hasTextRef.current = true;
			setHasText(true);
			setDocVersion(v => v + 1);
			return;
		}

		insertPlainTextAsSingleBlock(currentEditor, text);
	}, []);

	const onEditorChange = useCallback(() => {
		const currentEditor = editorRef.current;
		if (isAutoChunkingRef.current || isSelectionOnlyEditorChange(currentEditor)) {
			// Avoid feedback loops and skip selection-only updates.
			return false;
		}

		scheduleDocRecompute();
		scheduleAutoChunk();
		return true;
	}, [scheduleAutoChunk, scheduleDocRecompute]);

	return {
		editor,
		contentRef,
		hasText,
		hasTextRef,
		selectionInfo,
		attachedToolEntries,
		onEditorChange,
		onEditorPaste,
		replaceEditorDocument,
		resetEditorDocument,
		focusEditorAtEnd,
		focusEditorPreservingSelection,
	};
}
