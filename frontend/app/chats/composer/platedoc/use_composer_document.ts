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
	type ComposerDocumentSelectionInfo,
	createEmptyEditorValue,
	hasNonEmptyUserText,
	insertPlainTextAsSingleBlock,
	isCursorAtDocumentEnd,
	isSelectionOnlyEditorChange,
	isSimpleEmptyParagraphDocument,
	LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_AUTODECHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_CHUNK_SIZE,
} from '@/chats/composer/platedoc/platedoc_utils';
import { createComposerEditorPlugins } from '@/chats/composer/platedoc/plugins';
import {
	analyzeTemplateSelectionInfo,
	getTemplateNodesWithPath,
	getUserBlocksContent,
} from '@/chats/composer/platedoc/template_document_ops';
import { type AttachedToolEntry, getAttachedToolEntries } from '@/chats/composer/platedoc/tool_document_ops';
import { buildUserInlineChildrenFromText } from '@/chats/composer/templates/template_variables_inline';

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
	editorRef.current = editor;

	// doc version tick to re-run selection computations on any editor change
	const [docVersion, setDocVersion] = useState(0);

	// Cache "has text" so we don't re-scan the editor tree multiple times per render.
	const [hasText, setHasText] = useState(false);
	const hasTextRef = useRef(false);
	const isAutoChunkingRef = useRef(false);
	const lastPopulatedSelectionKeyRef = useRef(new Set());
	const idleChunkJobKindRef = useRef<'idle' | 'timeout' | null>(null);

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

	const cancelScheduledAutoChunk = useCallback(() => {
		const jobId = idleChunkJobRef.current;
		if (jobId === null) return;

		const w = window;
		if (idleChunkJobKindRef.current === 'idle' && typeof w.cancelIdleCallback === 'function') {
			w.cancelIdleCallback(jobId);
		} else {
			window.clearTimeout(jobId);
		}

		idleChunkJobRef.current = null;
		idleChunkJobKindRef.current = null;
	}, []);

	const runAfterNextPaint = useCallback((fn: () => void) => {
		window.requestAnimationFrame(() => {
			window.requestAnimationFrame(() => {
				fn();
			});
		});
	}, []);

	const selectEditorEnd = useCallback(() => {
		const currentEditor = editorRef.current;
		if (!currentEditor) return false;

		const end = currentEditor.api.end([]);
		if (!end) return false;

		try {
			currentEditor.tf.withoutNormalizing(() => {
				currentEditor.tf.select(end);
				currentEditor.tf.collapse({ edge: 'end' });
			});
			return true;
		} catch {
			return false;
		}
	}, []);

	const scrollEditorSelectionIntoView = useCallback(() => {
		const container = contentRef.current;
		if (!container) return;

		const sel = window.getSelection();
		if (sel && sel.rangeCount > 0) {
			try {
				const range = sel.getRangeAt(0).cloneRange();
				range.collapse(false);

				const rects = range.getClientRects();
				const rect = (rects.length > 0 ? rects[rects.length - 1] : null) ?? range.getBoundingClientRect();

				if (rect && (rect.height > 0 || rect.width > 0)) {
					const containerRect = container.getBoundingClientRect();
					const padding = 12;

					if (rect.bottom > containerRect.bottom - padding) {
						container.scrollTop += rect.bottom - containerRect.bottom + padding;
						return;
					}

					if (rect.top < containerRect.top + padding) {
						container.scrollTop -= containerRect.top - rect.top + padding;
						return;
					}
				}
			} catch {
				// noop
			}
		}

		container.scrollTop = container.scrollHeight;
	}, []);

	const focusEditorPreservingSelection = useCallback(() => {
		const currentEditor = editorRef.current;
		if (!currentEditor || isBusy) return;

		runAfterNextPaint(() => {
			try {
				currentEditor.tf.focus();
			} catch {
				return;
			}

			runAfterNextPaint(() => {
				scrollEditorSelectionIntoView();
			});
		});
	}, [isBusy, runAfterNextPaint, scrollEditorSelectionIntoView]);

	const focusEditorAtEnd = useCallback(() => {
		const currentEditor = editorRef.current;
		if (!currentEditor || isBusy) return;

		runAfterNextPaint(() => {
			try {
				currentEditor.tf.focus();
			} catch {
				return;
			}

			selectEditorEnd();

			runAfterNextPaint(() => {
				selectEditorEnd();
				scrollEditorSelectionIntoView();
			});
		});
	}, [isBusy, runAfterNextPaint, scrollEditorSelectionIntoView, selectEditorEnd]);

	const autoChunkIfNeeded = useCallback(() => {
		if (isAutoChunkingRef.current) return;
		if (!contentRef.current) return;

		const domLen = (contentRef.current.textContent ?? '').length;
		const ed = editorRef.current;
		const activeEl = document.activeElement;

		if (activeEl && !contentRef.current.contains(activeEl)) {
			return;
		}

		// Only operate on the simplest safe shape:
		// - single paragraph
		// - only text children
		const rootChildren = ed.children ?? [];
		if (rootChildren.length !== 1) return;
		const p = rootChildren[0];
		if (!p || p.type !== 'p' || !Array.isArray(p.children)) return;
		const pChildren = p.children;
		if (pChildren.some(c => typeof c?.text !== 'string')) return;

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
				focusEditorAtEnd();
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
				focusEditorAtEnd();
			} finally {
				isAutoChunkingRef.current = false;
			}
		}
	}, [focusEditorAtEnd]);

	const scheduleAutoChunk = useCallback(() => {
		cancelScheduledAutoChunk();

		const w = window;
		if (typeof w.requestIdleCallback === 'function') {
			idleChunkJobKindRef.current = 'idle';
			idleChunkJobRef.current = w.requestIdleCallback(
				() => {
					idleChunkJobRef.current = null;
					idleChunkJobKindRef.current = null;
					autoChunkIfNeeded();
				},
				{ timeout: 250 }
			);
		} else {
			idleChunkJobKindRef.current = 'timeout';
			idleChunkJobRef.current = window.setTimeout(() => {
				idleChunkJobRef.current = null;
				idleChunkJobKindRef.current = null;
				autoChunkIfNeeded();
			}, 120);
		}
	}, [autoChunkIfNeeded, cancelScheduledAutoChunk]);

	useEffect(() => {
		return cancelScheduledAutoChunk;
	}, [cancelScheduledAutoChunk]);

	// Recompute template selection info whenever doc changes.
	// analyzeTemplateSelectionInfo has a fast-path exit when no template nodes exist.
	// we need docVersion, it is not unnecessary dep.
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
					const pathArr = Array.isArray(originalPath) ? originalPath : [];

					if (pathArr.length >= 2) {
						const blockPath = pathArr.slice(0, pathArr.length - 1);
						const indexAfter = pathArr[pathArr.length - 1] + 1;
						const atPath = [...blockPath, indexAfter] as any;
						editor.tf.insertNodes(inlineChildren, { at: atPath });
					} else {
						editor.tf.insertNodes(inlineChildren, { at: [0, 0] as any });
					}
				});
			} catch {
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
				});

				syncDocumentDerivedState(true);
				focusEditorAtEnd();
				cancelScheduledAutoChunk();
				return;
			}

			insertPlainTextAsSingleBlock(currentEditor, text);
			focusEditorPreservingSelection();
		},
		[cancelScheduledAutoChunk, focusEditorAtEnd, focusEditorPreservingSelection, syncDocumentDerivedState]
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
