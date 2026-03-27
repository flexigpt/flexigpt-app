import {
	type ClipboardEvent as ReactClipboardEvent,
	type RefObject,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import { type Value } from 'platejs';
import { type PlateEditor, usePlateEditor } from 'platejs/react';

import { compareEntryByPathDeepestFirst } from '@/lib/path_utils';
import { cssEscape } from '@/lib/text_utils';

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
} from '@/chats/inputarea/editor/input_editor_utils';
import {
	type ComposerDocumentSelectionInfo,
	isSelectionOnlyEditorChange,
} from '@/chats/inputarea/platedoc/document_analysis';
import { createComposerEditorPlugins } from '@/chats/inputarea/platedoc/plugins';
import {
	analyzeTemplateSelectionInfo,
	getTemplateNodesWithPath,
} from '@/chats/inputarea/platedoc/templates/template_document_ops';
import { getUserBlocksContent } from '@/chats/inputarea/platedoc/templates/template_processing';
import { buildUserInlineChildrenFromText } from '@/chats/inputarea/platedoc/templates/template_variables_inline';
import { type AttachedToolEntry, getAttachedToolEntries } from '@/chats/inputarea/platedoc/tool_document_ops';

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
	attachedToolEntries: AttachedToolEntry[];
	getAttachedToolEntriesSnapshot: (uniqueByIdentity?: boolean) => AttachedToolEntry[];
	onEditorChange: () => boolean;
	onEditorPaste: (event: ReactClipboardEvent<HTMLDivElement>) => void;
	replaceEditorDocument: (nextValue: Value, focus?: ReplaceEditorDocumentFocusMode) => void;
	resetEditorDocument: () => void;
	focusEditorAtEnd: () => void;
	focusEditorPreservingSelection: () => void;
}

export function useComposerDocument({ isBusy }: UseComposerDocumentArgs): UseComposerDocumentResult {
	const initialEditorValue = useMemo<Value>(() => createEmptyEditorValue(), []);
	const editorPlugins = useMemo(() => createComposerEditorPlugins(), []);
	const editor = usePlateEditor({
		plugins: editorPlugins,
		value: initialEditorValue,
	});

	const contentRef = useRef<HTMLDivElement | null>(null);
	const editorRef = useRef(editor);
	editorRef.current = editor; // keep a live ref for callbacks / async work

	// doc version tick to re-run selection computations on any editor change
	const [docVersion, setDocVersion] = useState(0);

	// Cache "has text" so we don't re-scan the editor tree multiple times per render.
	const [hasText, setHasText] = useState(false);
	const hasTextRef = useRef(false);
	const isAutoChunkingRef = useRef(false);
	const lastPopulatedSelectionKeyRef = useRef(new Set());

	const syncDocumentDerivedState = useCallback((bumpDocVersion = true) => {
		const nextHasText = hasNonEmptyUserText(editorRef.current);
		hasTextRef.current = nextHasText;
		setHasText(prev => (prev === nextHasText ? prev : nextHasText));

		if (bumpDocVersion) {
			setDocVersion(v => v + 1);
		}
	}, []);

	// Throttle docVersion bumps to at most 1/frame to avoid re-render storms on big documents.
	const docRafRef = useRef<number | null>(null);
	const scheduleDocRecompute = useCallback(() => {
		if (docRafRef.current != null) return;
		docRafRef.current = window.requestAnimationFrame(() => {
			docRafRef.current = null;
			syncDocumentDerivedState(true);
		});
	}, [syncDocumentDerivedState]);

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

	// Recompute template selection info whenever doc changes.
	// analyzeTemplateSelectionInfo has a fast-path exit when no template nodes exist.
	// eslint-disable-next-line react-hooks/exhaustive-deps
	const selectionInfo = useMemo(() => analyzeTemplateSelectionInfo(editor), [editor, docVersion]);

	// Intentionally derived on render so any parent rerender sees current doc state.
	const attachedToolEntries = getAttachedToolEntries(editor);

	const getAttachedToolEntriesSnapshot = useCallback((uniqueByIdentity?: boolean) => {
		return getAttachedToolEntries(editorRef.current, uniqueByIdentity);
	}, []);

	// Populate editor with effective USER blocks for EACH template selection (once per selectionID)
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
			const selectionID = tsenode.selectionID;
			if (populated.has(selectionID)) continue;

			// Build children: keep the selection chip, add parsed user text with variable pills
			const userText = getUserBlocksContent(tsenode);
			const inlineChildren = buildUserInlineChildrenFromText(tsenode, userText);

			try {
				editor.tf.withoutNormalizing(() => {
					// Recompute a fresh path to guard against prior insertions shifting indices
					const pathArr = Array.isArray(originalPath) ? originalPath : [];

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
	}, [editor, focusEditorAtEnd, docVersion, selectionInfo.tplNodeWithPath, syncDocumentDerivedState]);

	const replaceEditorDocument = useCallback(
		(nextValue: Value, focus: ReplaceEditorDocumentFocusMode = 'none') => {
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

			syncDocumentDerivedState(true);

			if (focus === 'end') {
				focusEditorAtEnd();
			} else if (focus === 'preserve') {
				focusEditorPreservingSelection();
			}
		},
		[focusEditorAtEnd, focusEditorPreservingSelection, syncDocumentDerivedState]
	);

	const resetEditorDocument = useCallback(() => {
		replaceEditorDocument(createEmptyEditorValue(), 'end');
	}, [replaceEditorDocument]);

	const onEditorPaste = useCallback(
		(e: ReactClipboardEvent<HTMLDivElement>) => {
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

				syncDocumentDerivedState(true);
				return;
			}

			insertPlainTextAsSingleBlock(currentEditor, text);
		},
		[syncDocumentDerivedState]
	);

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
		getAttachedToolEntriesSnapshot,
		onEditorChange,
		onEditorPaste,
		replaceEditorDocument,
		resetEditorDocument,
		focusEditorAtEnd,
		focusEditorPreservingSelection,
	};
}
