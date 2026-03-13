import {
	forwardRef,
	type SubmitEventHandler,
	useCallback,
	useDeferredValue,
	useEffect,
	useImperativeHandle,
	useMemo,
	useRef,
	useState,
} from 'react';

import { FiAlertTriangle, FiEdit2, FiFastForward, FiPlay, FiSend, FiSquare, FiX } from 'react-icons/fi';

import { useMenuStore, useStoreState } from '@ariakit/react';
import { SingleBlockPlugin, type Value } from 'platejs';
import { Plate, PlateContent, type PlateEditor, usePlateEditor } from 'platejs/react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { ProviderSDKType, UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';

import { type ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { compareEntryByPathDeepestFirst } from '@/lib/path_utils';
import { cssEscape } from '@/lib/text_utils';

import { useEnterSubmit } from '@/hooks/use_enter_submit';

import { AlignKit } from '@/components/editor/plugins/align_kit';
import { BasicBlocksKit } from '@/components/editor/plugins/basic_blocks_kit';
import { BasicMarksKit } from '@/components/editor/plugins/basic_marks_kit';
import { FloatingToolbarKit } from '@/components/editor/plugins/floating_toolbar_kit';
import { IndentKit } from '@/components/editor/plugins/indent_kit';
import { LineHeightKit } from '@/components/editor/plugins/line_height_kit';
import { ListKit } from '@/components/editor/plugins/list_kit';
import { TabbableKit } from '@/components/editor/plugins/tabbable_kit';

import { useComposerAttachments } from '@/chats/attachments/use_composer_attachments';
import { dispatchTemplateFlashEvent } from '@/chats/events/template_flash';
import { EditorBottomBar } from '@/chats/inputarea/input_editor_bottom_bar';
import { EditorChipsBar } from '@/chats/inputarea/input_editor_chips_bar';
import {
	buildEditorValueFromPlainText,
	buildSingleParagraphValue,
	buildSingleParagraphValueChunked,
	clearAllMarks,
	createEmptyEditorValue,
	type EditorExternalMessage,
	type EditorSubmitPayload,
	hasNonEmptyUserText,
	insertPlainTextAsSingleBlock,
	isCursorAtDocumentEnd,
	isSimpleEmptyParagraphDocument,
	LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_AUTODECHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_CHUNK_SIZE,
} from '@/chats/inputarea/input_editor_utils';
import { useComposerSkills } from '@/chats/skills/use_composer_skills';
import {
	getFirstTemplateNodeWithPath,
	getTemplateNodesWithPath,
	getTemplateSelections,
	toPlainTextReplacingVariables,
} from '@/chats/templates/template_editor_utils';
import { TemplateSlashKit } from '@/chats/templates/template_plugin';
import { getLastUserBlockContent } from '@/chats/templates/template_processing';
import { TemplateToolbars } from '@/chats/templates/template_toolbars';
import { buildUserInlineChildrenFromText } from '@/chats/templates/template_variables_inline';
import {
	type ConversationToolStateEntry,
	conversationToolsToChoices,
	mergeConversationToolsWithNewChoices,
} from '@/chats/tools/conversation_tool_utils';
import { ToolDetailsModal } from '@/chats/tools/tool_details_modal';
import {
	dedupeToolChoices,
	editorAttachedToolToToolChoice,
	getAttachedTools,
	getToolNodesWithPath,
} from '@/chats/tools/tool_editor_utils';
import { ToolPlusKit } from '@/chats/tools/tool_plugin';
import { ToolArgsModalHost } from '@/chats/tools/tool_user_args_host';
import { useComposerTools } from '@/chats/tools/use_composer_tools';
import { buildWebSearchChoicesForSubmit, type WebSearchChoiceTemplate } from '@/chats/tools/websearch_utils';

export interface EditorAreaHandle {
	focus: () => void;
	openTemplateMenu: () => void;
	openToolMenu: () => void;
	openAttachmentMenu: () => void;
	loadExternalMessage: (msg: EditorExternalMessage) => void;
	resetEditor: () => void;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	setConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	setWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
	applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => void;
	setEnabledSkillRefsFromMessage: (refs: SkillRef[]) => void;
	setActiveSkillRefsFromMessage: (refs: SkillRef[]) => void;
}

const isSelectionOnlyEditorChange = (editor: PlateEditor): boolean => {
	const operations = editor.operations ?? [];
	return operations.length > 0 && operations.every(op => op.type === 'set_selection');
};

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

interface EditorAreaProps {
	isBusy: boolean;
	currentProviderSDKType: ProviderSDKType;
	shortcutConfig: ShortcutConfig;
	onSubmit: (payload: EditorSubmitPayload) => Promise<void>;
	onRequestStop: () => void;
	editingMessageId: string | null;
	cancelEditing: () => void;
}

export const EditorArea = forwardRef<EditorAreaHandle, EditorAreaProps>(function EditorArea(
	{ isBusy, currentProviderSDKType, shortcutConfig, onSubmit, onRequestStop, editingMessageId, cancelEditing },
	ref
) {
	const initialEditorValue = useMemo<Value>(() => createEmptyEditorValue(), []);
	const editorPlugins = useMemo(() => createEditorPlugins(), []);
	const editor = usePlateEditor({
		plugins: editorPlugins,
		value: initialEditorValue,
	});

	const isSubmittingRef = useRef<boolean>(false);
	const contentRef = useRef<HTMLDivElement | null>(null);
	const editorRef = useRef(editor);
	editorRef.current = editor; // keep a live ref for key handlers
	const isMountedRef = useRef(true);
	useEffect(() => {
		return () => {
			isMountedRef.current = false;
		};
	}, []);

	const templateMenu = useMenuStore({ placement: 'top-start', focusLoop: true });
	const toolMenu = useMenuStore({ placement: 'top-start', focusLoop: true });
	const attachmentMenu = useMenuStore({ placement: 'top-start', focusLoop: true });
	const templateButtonRef = useRef<HTMLButtonElement | null>(null);
	const toolButtonRef = useRef<HTMLButtonElement | null>(null);
	const attachmentButtonRef = useRef<HTMLButtonElement | null>(null);
	// Track whether a menu was opened via shortcut so we can:
	// - force focus into the menu (arrow-key nav)
	// - optionally restore focus to editor on close (Esc)
	const menuOpenedByShortcutRef = useRef({ templates: false, tools: false, attachments: false });

	// doc version tick to re-run selection computations on any editor change
	const [docVersion, setDocVersion] = useState(0);
	const deferredDocVersion = useDeferredValue(docVersion);

	// Cache "has text" so we don't re-scan the editor tree multiple times per render.
	const [hasText, setHasText] = useState(false);
	const hasTextRef = useRef(false);
	const isAutoChunkingRef = useRef(false);

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
		const editor = editorRef.current;
		if (!editor || isBusy) return;

		requestAnimationFrame(() => {
			try {
				editor.tf.focus();
			} catch {
				// noop
			}
		});
	}, [isBusy]);

	const focusEditorAtEnd = useCallback(() => {
		const editor = editorRef.current;
		if (!editor) return;

		// No visible caret in readOnly mode.
		if (isBusy) return;

		requestAnimationFrame(() => {
			try {
				editor.tf.withoutNormalizing(() => {
					editor.tf.select(undefined, { edge: 'end' });
					editor.tf.collapse({ edge: 'end' });
				});

				editor.tf.focus();

				// Keep end visible if content is long
				if (contentRef.current) {
					contentRef.current.scrollTop = contentRef.current.scrollHeight;
				}
			} catch {
				editor.tf.focus();
			}
		});
	}, [isBusy]);

	const {
		attachments,
		directoryGroups,
		attachFiles: handleAttachFiles,
		attachDirectory: handleAttachDirectory,
		attachURL: handleAttachURL,
		changeAttachmentMode: handleChangeAttachmentContentBlockMode,
		removeAttachment: handleRemoveAttachment,
		removeDirectoryGroup: handleRemoveDirectoryGroup,
		removeOverflowDir: handleRemoveOverflowDir,
		applyAttachmentsDrop,
		clearAttachments,
		loadAttachmentsFromMessage,
	} = useComposerAttachments({
		isBusy,
		focusEditorAtEnd,
	});

	const [submitError, setSubmitError] = useState<string | null>(null);

	// ---- Skills (conversation-level) ----
	const {
		allSkills,
		skillsLoading,
		enabledSkillRefs,
		activeSkillRefs,
		skillSessionID,
		setEnabledSkillRefs,
		setActiveSkillRefs,
		enableAllSkills,
		disableAllSkills,
		applySkillSelectionState,
		ensureSkillSession,
		listActiveSkillRefs,
		applyEnabledSkillRefsFromMessage,
		applyActiveSkillRefsFromMessage,
		getCurrentSkillSessionID,
	} = useComposerSkills();

	const lastPopulatedSelectionKeyRef = useRef<Set<string>>(new Set());

	const selectionInfo = useMemo(() => {
		// Fast path: if the document contains no template-selection elements at all,
		// short-circuit instead of running the heavier helpers.
		const tplNodeWithPath = getFirstTemplateNodeWithPath(editor);
		if (!tplNodeWithPath) {
			return {
				tplNodeWithPath: undefined,
				hasTemplate: false,
				requiredCount: 0,
				firstPendingVar: undefined,
			};
		}

		const selections = getTemplateSelections(editor);
		const hasTemplate = selections.length > 0;

		let requiredCount = 0;
		let firstPendingVar: { name: string; selectionID?: string } | undefined = undefined;

		for (const s of selections) {
			requiredCount += s.requiredCount;

			if (!firstPendingVar) {
				if (s.requiredVariables.length > 0) {
					firstPendingVar = { name: s.requiredVariables[0], selectionID: s.selectionID };
				}
			}
		}

		return {
			tplNodeWithPath,
			hasTemplate,
			requiredCount,
			firstPendingVar,
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [editor, deferredDocVersion]);

	const templateBlocked = selectionInfo.hasTemplate && selectionInfo.requiredCount > 0;

	const submitPendingToolsAndSendRef = useRef<(() => void | Promise<void>) | null>(null);

	const {
		toolCalls,
		toolOutputs,
		setToolOutputs,
		conversationToolsState,
		setConversationToolsState,
		setConversationToolsStateAndMaybeAutoExecute,
		webSearchTemplates,
		setWebSearchTemplates,
		setWebSearchTemplatesAndMaybeAutoExecute,
		toolDetailsState,
		setToolDetailsState,
		toolArgsTarget,
		setToolArgsTarget,
		toolsDefLoading,
		toolArgsBlocked,
		hasPendingToolCalls,
		hasRunningToolCalls,
		toolEntriesVersion,
		bumpToolEntriesVersion,
		runAllPendingToolCalls,
		handleRunSingleToolCall,
		handleDiscardToolCall,
		handleRemoveToolOutput,
		handleRetryErroredOutput,
		handleAttachedToolsChanged,
		applyConversationToolsFromChoices,
		applyWebSearchFromChoices,
		loadToolCalls,
		handleOpenToolOutput,
		handleOpenToolCallDetails,
		handleOpenConversationToolDetails,
		handleOpenAttachedToolDetails,
		clearComposerToolsState,
	} = useComposerTools({
		editor,
		isBusy,
		isSubmittingRef,
		templateBlocked,
		submitPendingToolsAndSendRef,
		ensureSkillSession,
		listActiveSkillRefs,
		setActiveSkillRefs,
		getCurrentSkillSessionID,
		skillSessionID,
	});

	// eslint-disable-next-line react-hooks/exhaustive-deps
	const attachedToolEntries = useMemo(() => getToolNodesWithPath(editor), [editor, toolEntriesVersion]);

	// When editing an earlier message we temporarily override the current
	// conversation-tool + web-search config. Keep a snapshot so Cancel restores it.
	const preEditConversationToolsRef = useRef<ConversationToolStateEntry[] | null>(null);
	const preEditWebSearchTemplatesRef = useRef<WebSearchChoiceTemplate[] | null>(null);
	const preEditEnabledSkillRefsRef = useRef<SkillRef[] | null>(null);
	const preEditActiveSkillRefsRef = useRef<SkillRef[] | null>(null);

	// --- Fix: focus management for menus opened by shortcuts ---
	const templateMenuOpen = useStoreState(templateMenu, 'open');
	const toolMenuOpen = useStoreState(toolMenu, 'open');
	const attachmentMenuOpen = useStoreState(attachmentMenu, 'open');
	const templateMenuEl = useStoreState(templateMenu, 'contentElement');
	const toolMenuEl = useStoreState(toolMenu, 'contentElement');
	const attachmentMenuEl = useStoreState(attachmentMenu, 'contentElement');

	const clearPreEditSnapshot = useCallback(() => {
		preEditConversationToolsRef.current = null;
		preEditWebSearchTemplatesRef.current = null;
		preEditEnabledSkillRefsRef.current = null;
		preEditActiveSkillRefsRef.current = null;
	}, []);

	const closeAllMenus = useCallback(() => {
		templateMenu.hide();
		toolMenu.hide();
		attachmentMenu.hide();
	}, [templateMenu, toolMenu, attachmentMenu]);

	useEffect(() => {
		if (!templateMenuOpen) {
			if (menuOpenedByShortcutRef.current.templates) {
				menuOpenedByShortcutRef.current.templates = false;
				requestAnimationFrame(() => {
					focusEditorPreservingSelection();
				});
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.templates) return;

		requestAnimationFrame(() => {
			templateMenuEl?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
		});
	}, [templateMenuOpen, templateMenuEl, focusEditorPreservingSelection]);

	useEffect(() => {
		if (!toolMenuOpen) {
			if (menuOpenedByShortcutRef.current.tools) {
				menuOpenedByShortcutRef.current.tools = false;
				requestAnimationFrame(() => {
					focusEditorPreservingSelection();
				});
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.tools) return;

		requestAnimationFrame(() => {
			toolMenuEl?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
		});
	}, [toolMenuOpen, toolMenuEl, focusEditorPreservingSelection]);

	useEffect(() => {
		if (!attachmentMenuOpen) {
			if (menuOpenedByShortcutRef.current.attachments) {
				menuOpenedByShortcutRef.current.attachments = false;
				requestAnimationFrame(() => {
					focusEditorPreservingSelection();
				});
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.attachments) return;

		requestAnimationFrame(() => {
			attachmentMenuEl?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
		});
	}, [attachmentMenuOpen, attachmentMenuEl, focusEditorPreservingSelection]);

	const openTemplatePicker = useCallback(() => {
		menuOpenedByShortcutRef.current.templates = true;

		closeAllMenus();
		templateMenu.show();
		// Make Ariakit's "return focus" behavior deterministic on close.
		templateButtonRef.current?.focus({ preventScroll: true });
	}, [closeAllMenus, templateMenu]);

	const openToolPicker = useCallback(() => {
		menuOpenedByShortcutRef.current.tools = true;

		closeAllMenus();
		toolMenu.show();
		toolButtonRef.current?.focus({ preventScroll: true });
	}, [closeAllMenus, toolMenu]);

	const openAttachmentPicker = useCallback(() => {
		menuOpenedByShortcutRef.current.attachments = true;

		closeAllMenus();
		attachmentMenu.show();
		attachmentButtonRef.current?.focus({ preventScroll: true });
	}, [closeAllMenus, attachmentMenu]);

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

	const restorePreEditContext = useCallback(() => {
		const prevConv = preEditConversationToolsRef.current;
		const prevWs = preEditWebSearchTemplatesRef.current;
		const prevSkills = preEditEnabledSkillRefsRef.current;
		const prevActive = preEditActiveSkillRefsRef.current;

		if (prevConv) setConversationToolsState(prevConv);
		if (prevWs) setWebSearchTemplates(prevWs);
		if (prevSkills || prevActive) {
			applySkillSelectionState(prevSkills ?? enabledSkillRefs, prevActive ?? activeSkillRefs);
		}
		clearPreEditSnapshot();
	}, [
		activeSkillRefs,
		applySkillSelectionState,
		clearPreEditSnapshot,
		enabledSkillRefs,
		setConversationToolsState,
		setWebSearchTemplates,
	]);

	const isSendButtonEnabled = useMemo(() => {
		if (isBusy) return false;
		if (templateBlocked) return false;
		if (toolArgsBlocked) return false;
		if (toolsDefLoading) return false;

		if (hasText) return true;

		const hasAttachments = attachments.length > 0;
		const hasOutputs = toolOutputs.length > 0;

		return hasAttachments || hasOutputs;
	}, [isBusy, templateBlocked, attachments, toolOutputs, hasText, toolArgsBlocked, toolsDefLoading]);

	const { formRef, onKeyDown } = useEnterSubmit({
		isBusy,
		canSubmit: () => {
			if (toolArgsBlocked) return false;
			if (toolsDefLoading) return false;

			if (selectionInfo.hasTemplate) {
				return selectionInfo.requiredCount === 0;
			}

			if (hasTextRef.current) return true;

			const hasAttachments = attachments.length > 0;
			const hasOutputs = toolOutputs.length > 0;

			return hasAttachments || hasOutputs;
		},
		insertSoftBreak: () => {
			editor.tf.insertSoftBreak();
		},
	});

	const clearComposerTransientState = useCallback(() => {
		closeAllMenus();
		setSubmitError(null);
		lastPopulatedSelectionKeyRef.current.clear();
		isSubmittingRef.current = false;

		clearAttachments();
		clearComposerToolsState();
	}, [clearAttachments, clearComposerToolsState, closeAllMenus]);

	const replaceEditorDocument = useCallback(
		(nextValue: Value, nextHasText: boolean, focus: 'none' | 'preserve' | 'end' = 'none') => {
			const editor = editorRef.current;
			if (!editor) return;

			editor.tf.withoutNormalizing(() => {
				try {
					editor.tf.deselect();
				} catch {
					// noop
				}
				editor.tf.setValue(nextValue);
			});

			hasTextRef.current = nextHasText;
			setHasText(nextHasText);
			setDocVersion(v => v + 1);
			bumpToolEntriesVersion();
			if (focus === 'end') {
				focusEditorAtEnd();
			} else if (focus === 'preserve') {
				focusEditorPreservingSelection();
			}
		},
		[bumpToolEntriesVersion, focusEditorAtEnd, focusEditorPreservingSelection]
	);

	const resetEditor = useCallback(() => {
		clearComposerTransientState();
		replaceEditorDocument(createEmptyEditorValue(), false, 'end');
	}, [clearComposerTransientState, replaceEditorDocument]);

	/**
	 * Main submit logic, parameterized by whether to run pending tool calls
	 * before sending.
	 */
	const doSubmit = useCallback(
		async (options: { runPendingTools: boolean }) => {
			const { runPendingTools } = options;

			if (isSubmittingRef.current) return;
			if (isBusy) return;

			// 1) Templates: never allow send when required vars are missing.
			if (templateBlocked) {
				// Ask the toolbar (rendered via plugin) to flash.
				dispatchTemplateFlashEvent();

				// Focus first pending variable pill (if any).
				const fpv = selectionInfo.firstPendingVar;
				if (fpv?.name && contentRef.current) {
					const idSegment = fpv.selectionID ? `[data-selection-id="${cssEscape(fpv.selectionID)}"]` : '';
					const sel = contentRef.current.querySelector(
						`span[data-template-variable][data-var-name="${cssEscape(fpv.name)}"]${idSegment}`
					);
					if (sel && 'focus' in sel && typeof sel.focus === 'function') {
						sel.focus();
					} else {
						focusEditorAtEnd();
					}
				} else {
					focusEditorAtEnd();
				}
				return;
			}

			// 2) Pure send path: if we're *not* running tools, bail out when we
			//    don't already have something to send.
			if (!runPendingTools && !isSendButtonEnabled) {
				return;
			}

			// Guard explicitly here as well, so even programmatic calls respect it.
			if (toolArgsBlocked) {
				setSubmitError('Some attached tools require options. Fill the required tool options before sending.');
				return;
			}

			setSubmitError(null);
			isSubmittingRef.current = true;
			const hadPendingTools = runPendingTools && hasPendingToolCalls;
			let didSend = false;

			try {
				// Ensure session exists BEFORE running any pending tools (skills.* needs it)
				let effectiveSkillSessionID: string | null = null;
				if (enabledSkillRefs.length > 0) {
					try {
						effectiveSkillSessionID = await ensureSkillSession();
					} catch (err) {
						setSubmitError((err as Error)?.message || 'Failed to create skills session.');
						return;
					}
				}

				const existingOutputs = toolOutputs;
				let newlyProducedOutputs: UIToolOutput[] = [];

				// 3) Optional tool run (fast-forward path).
				if (runPendingTools && hasPendingToolCalls) {
					newlyProducedOutputs = await runAllPendingToolCalls();
				}

				// 4) Build final message content after tools have run.
				const selections = getTemplateSelections(editor);
				const hasTpl = selections.length > 0;

				const textToSend = hasTpl ? toPlainTextReplacingVariables(editor) : editor.api.string([]);
				const finalToolOutputs = [...existingOutputs, ...newlyProducedOutputs];

				const hasNonEmptyText = textToSend.trim().length > 0;
				const hasAttachmentsToSend = attachments.length > 0;
				const hasToolOutputsToSend = finalToolOutputs.length > 0;

				// Enforce the "non-empty message" invariant *after* tools have run.
				if (!hasNonEmptyText && !hasAttachmentsToSend && !hasToolOutputsToSend) {
					setSubmitError(
						hadPendingTools
							? 'Tool calls did not produce any outputs, so there is nothing to send yet.'
							: 'Nothing to send. Add text, attachments, or tool outputs first.'
					);
					return;
				}

				// Snapshot active skills right before send (authoritative persistence).
				let activeForMessage: SkillRef[] = activeSkillRefs;
				if (effectiveSkillSessionID && enabledSkillRefs.length > 0) {
					try {
						activeForMessage = await listActiveSkillRefs(effectiveSkillSessionID);
						if (getCurrentSkillSessionID() === effectiveSkillSessionID) {
							setActiveSkillRefs(activeForMessage);
						}
					} catch {
						// keep last-known
					}
				}

				// 5) Tool choices (editor-attached + conversation-level).
				const attachedTools = getAttachedTools(editor);
				const explicitChoices = attachedTools.map(editorAttachedToolToToolChoice);
				const conversationChoices = conversationToolsToChoices(conversationToolsState);
				const webSearchChoices = buildWebSearchChoicesForSubmit(webSearchTemplates);

				const finalToolChoices = dedupeToolChoices([...explicitChoices, ...conversationChoices, ...webSearchChoices]);

				const payload: EditorSubmitPayload = {
					text: textToSend,
					attachedTools,
					attachments,
					toolOutputs: finalToolOutputs,
					finalToolChoices,
					enabledSkillRefs,
					activeSkillRefs: activeForMessage,
					skillSessionID: effectiveSkillSessionID ?? undefined,
				};

				await onSubmit(payload);
				setSubmitError(null);
				setConversationToolsState(prev => mergeConversationToolsWithNewChoices(prev, finalToolChoices));
				didSend = true;
			} finally {
				isSubmittingRef.current = false;

				// Only clear the editor if we actually sent something.
				if (didSend) {
					resetEditor();
					// If we were editing, the old snapshot is no longer relevant.
					clearPreEditSnapshot();
				}
			}
		},
		[
			activeSkillRefs,
			attachments,
			clearPreEditSnapshot,
			conversationToolsState,
			editor,
			enabledSkillRefs,
			ensureSkillSession,
			focusEditorAtEnd,
			getCurrentSkillSessionID,
			hasPendingToolCalls,
			isBusy,
			isSendButtonEnabled,
			listActiveSkillRefs,
			onSubmit,
			resetEditor,
			runAllPendingToolCalls,
			selectionInfo.firstPendingVar,
			setActiveSkillRefs,
			setConversationToolsState,
			templateBlocked,
			toolArgsBlocked,
			toolOutputs,
			webSearchTemplates,
		]
	);

	const doSubmitRef = useRef(doSubmit);
	doSubmitRef.current = doSubmit;
	submitPendingToolsAndSendRef.current = () => doSubmitRef.current({ runPendingTools: true });

	/**
	 * Default form submit / Enter: "run pending tools, then send".
	 */
	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		if (e) e.preventDefault();
		void doSubmit({ runPendingTools: true });
	};

	const loadExternalMessage = useCallback(
		(incoming: EditorExternalMessage) => {
			// Snapshot current context so Cancel Editing can restore it.
			if (!preEditConversationToolsRef.current) {
				preEditConversationToolsRef.current = conversationToolsState;
			}
			if (!preEditWebSearchTemplatesRef.current) {
				preEditWebSearchTemplatesRef.current = webSearchTemplates;
			}
			if (!preEditEnabledSkillRefsRef.current) {
				preEditEnabledSkillRefsRef.current = enabledSkillRefs;
			}
			if (!preEditActiveSkillRefsRef.current) {
				preEditActiveSkillRefsRef.current = activeSkillRefs;
			}
			clearComposerTransientState();

			// 1) Reset document to plain text paragraphs.
			const plain = incoming.text ?? '';
			const value = buildEditorValueFromPlainText(plain);
			replaceEditorDocument(value, plain.trim().length > 0, 'end');

			// 2) Rebuild flat attachment chips from incoming attachment state.
			loadAttachmentsFromMessage(incoming.attachments);

			// 3) Restore tool choices into conversation-level state.
			const incomingToolChoices = incoming.toolChoices ?? [];
			applyConversationToolsFromChoices(incomingToolChoices);
			applyWebSearchFromChoices(incomingToolChoices);

			// 4) Restore enabled/active skills together so invariants hold immediately.
			applySkillSelectionState(incoming.enabledSkillRefs ?? [], incoming.activeSkillRefs ?? []);

			// 5) Restore any tool outputs that were previously attached to this message.
			setToolOutputs(incoming.toolOutputs ?? []);
		},
		[
			activeSkillRefs,
			applyConversationToolsFromChoices,
			applySkillSelectionState,
			applyWebSearchFromChoices,
			clearComposerTransientState,
			conversationToolsState,
			enabledSkillRefs,
			loadAttachmentsFromMessage,
			replaceEditorDocument,
			setToolOutputs,
			webSearchTemplates,
		]
	);

	useImperativeHandle(ref, () => ({
		focus: () => {
			focusEditorAtEnd();
		},
		openTemplateMenu: () => {
			openTemplatePicker();
		},
		openToolMenu: () => {
			openToolPicker();
		},
		openAttachmentMenu: () => {
			openAttachmentPicker();
		},
		loadExternalMessage,
		resetEditor,
		loadToolCalls,
		setConversationToolsFromChoices: applyConversationToolsFromChoices,
		setWebSearchFromChoices: applyWebSearchFromChoices,
		applyAttachmentsDrop,
		setEnabledSkillRefsFromMessage: applyEnabledSkillRefsFromMessage,
		setActiveSkillRefsFromMessage: applyActiveSkillRefsFromMessage,
	}));

	const handleCancelEditing = useCallback(() => {
		resetEditor();
		restorePreEditContext();
		cancelEditing();
	}, [cancelEditing, resetEditor, restorePreEditContext]);

	const handleRunToolsOnlyClick = useCallback(async () => {
		if (!hasPendingToolCalls || isBusy || hasRunningToolCalls) return;
		await runAllPendingToolCalls();
	}, [hasPendingToolCalls, hasRunningToolCalls, isBusy, runAllPendingToolCalls]);

	// Button-state helpers:
	// - Play: run tools only (enabled when there are pending tools and none are running).
	// - Fast-forward: run tools then send (enabled when there are pending tools and
	//   templates are satisfied; "sendability" will be re-checked after tools run).
	// - Send: send only (enabled when send is allowed and there are no pending tools).
	const canSendOnly = !hasPendingToolCalls && isSendButtonEnabled && !hasRunningToolCalls;
	const canRunToolsOnly = hasPendingToolCalls && !hasRunningToolCalls && !isBusy;
	const canRunToolsAndSend =
		hasPendingToolCalls && !hasRunningToolCalls && !isBusy && !templateBlocked && !toolArgsBlocked && !toolsDefLoading;

	return (
		<>
			<form
				ref={formRef}
				onSubmit={handleSubmit}
				className="mx-0 flex w-full max-w-full min-w-0 flex-col overflow-x-hidden overflow-y-visible"
			>
				{submitError ? (
					<div className="alert alert-error mx-4 mt-3 mb-1 flex items-start gap-2 text-sm" role="alert">
						<FiAlertTriangle size={16} className="mt-0.5" />
						<span>{submitError}</span>
					</div>
				) : null}
				<Plate
					editor={editor}
					onChange={() => {
						const currentEditor = editorRef.current;
						if (isAutoChunkingRef.current || isSelectionOnlyEditorChange(currentEditor)) {
							// Avoid feedback loops and skip selection-only updates.
							return;
						}
						scheduleDocRecompute();
						scheduleAutoChunk();
						if (toolCalls.length > 0) {
							handleAttachedToolsChanged();
						}
						if (submitError) {
							setSubmitError(null);
						}

						// Auto-cancel editing when the editor is completely empty
						// (no text, no tools, no attachments, no tool outputs).
						const hasTextNow = editingMessageId ? hasNonEmptyUserText(currentEditor) : hasTextRef.current;

						const hasAttachmentsLocal = attachments.length > 0;
						const hasToolOutputsLocal = toolOutputs.length > 0;
						// Tools alone are not considered enough to keep edit mode alive.
						const isEffectivelyEmpty = !hasTextNow && !hasAttachmentsLocal && !hasToolOutputsLocal;

						// Only do this while editing an older message.
						if (editingMessageId && isEffectivelyEmpty) {
							// IMPORTANT: do NOT call resetEditor here; we only exit edit mode.
							restorePreEditContext();
							cancelEditing();
						}
					}}
				>
					<div className="bg-base-100 border-base-200 flex w-full max-w-full min-w-0 overflow-hidden rounded-2xl border">
						<div className="flex min-w-0 grow flex-col p-0">
							<TemplateToolbars />
							{editingMessageId && (
								<div className="flex items-center justify-end gap-2 pt-1 pr-3 pb-0 text-xs">
									<div className="flex items-center gap-2">
										<FiEdit2 size={14} />
										<span>Editing an earlier message. Sending will replace it and drop all later messages.</span>
									</div>
									<button
										type="button"
										className="btn btn-circle btn-neutral btn-xs shrink-0"
										onClick={handleCancelEditing}
										title="Cancel Edit"
									>
										<FiX size={14} />
									</button>
								</div>
							)}
							{/* Row: editor with send/stop button on the right */}
							<div className="flex min-h-20 min-w-0 grow gap-2 px-1 py-0">
								<PlateContent
									ref={contentRef}
									placeholder="Type message..."
									spellCheck={false}
									readOnly={isBusy}
									onKeyDown={e => {
										onKeyDown(e); // from useEnterSubmit
									}}
									onPaste={e => {
										e.preventDefault();
										e.stopPropagation();
										const text = e.clipboardData.getData('text/plain');
										if (!text) return;
										clearAllMarks(editor);

										// PERF: if paste is huge AND the doc is truly the default empty paragraph,
										// set chunked value directly. Do not use "has text" as a proxy for
										// emptiness, or we can blow away template/tool nodes.
										if (
											isSimpleEmptyParagraphDocument(editorRef.current) &&
											text.length >= LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS
										) {
											editor.tf.withoutNormalizing(() => {
												editor.tf.setValue(buildSingleParagraphValueChunked(text, LARGE_TEXT_CHUNK_SIZE));
												editor.tf.collapse({ edge: 'end' });
											});

											hasTextRef.current = true;
											setHasText(true);
											setDocVersion(v => v + 1);
											return;
										}
										insertPlainTextAsSingleBlock(editor, text);
									}}
									className="max-h-96 min-w-0 flex-1 resize-none overflow-auto bg-transparent p-1 wrap-break-word whitespace-break-spaces outline-none [tab-size:2] focus:outline-none"
									style={{
										fontSize: '14px',
										whiteSpace: 'break-spaces',
										tabSize: 2,
										minHeight: '4rem',
									}}
								/>
							</div>
							{/* Unified chips bar: attachments, directories, tools, tool calls & outputs (scrollable) */}
							<div
								className="scrollbar-custom-thin w-full min-w-0 items-center overflow-x-auto overscroll-contain p-1 text-xs"
								style={{ scrollbarGutter: 'stable' }}
							>
								<EditorChipsBar
									attachments={attachments}
									directoryGroups={directoryGroups}
									conversationTools={conversationToolsState}
									toolCalls={toolCalls}
									toolOutputs={toolOutputs}
									toolEntries={attachedToolEntries}
									isBusy={isBusy || isSubmittingRef.current}
									onRunToolCall={handleRunSingleToolCall}
									onDiscardToolCall={handleDiscardToolCall}
									onOpenOutput={handleOpenToolOutput}
									onRemoveOutput={handleRemoveToolOutput}
									onRetryErroredOutput={handleRetryErroredOutput}
									onRemoveAttachment={handleRemoveAttachment}
									onChangeAttachmentContentBlockMode={handleChangeAttachmentContentBlockMode}
									onRemoveDirectoryGroup={handleRemoveDirectoryGroup}
									onRemoveOverflowDir={handleRemoveOverflowDir}
									onConversationToolsChange={setConversationToolsStateAndMaybeAutoExecute}
									onAttachedToolsChanged={handleAttachedToolsChanged}
									onOpenToolCallDetails={handleOpenToolCallDetails}
									onOpenConversationToolDetails={handleOpenConversationToolDetails}
									onOpenAttachedToolDetails={handleOpenAttachedToolDetails}
								/>
							</div>
						</div>
						{/* Primary / secondary actions anchored at bottom-right */}
						<div className="flex flex-col items-end justify-end gap-2 p-1">
							{isBusy ? (
								<button
									type="button"
									className="btn btn-circle btn-neutral btn-sm shrink-0"
									onClick={onRequestStop}
									title="Stop response"
									aria-label="Stop response"
								>
									<FiSquare size={20} />
								</button>
							) : (
								<>
									{/* Run tools only (Play) */}
									{hasPendingToolCalls && (
										<div className="tooltip tooltip-left" data-tip="Run tools only">
											<button
												type="button"
												className={`btn btn-circle btn-neutral btn-sm shrink-0 ${
													!canRunToolsOnly ? 'btn-disabled' : ''
												}`}
												disabled={!canRunToolsOnly}
												onClick={() => {
													if (!canRunToolsOnly) return;
													void handleRunToolsOnlyClick();
												}}
												aria-label="Run tools only"
											>
												<FiPlay size={18} />
											</button>
										</div>
									)}

									{/* Run tools and send (Fast-forward) */}
									{hasPendingToolCalls && (
										<div className="tooltip tooltip-left" data-tip="Run tools and send">
											<button
												type="button"
												className={`btn btn-circle btn-neutral btn-sm shrink-0 ${
													!canRunToolsAndSend ? 'btn-disabled' : ''
												}`}
												disabled={!canRunToolsAndSend}
												onClick={() => {
													if (!canRunToolsAndSend) return;
													void doSubmit({ runPendingTools: true });
												}}
												aria-label="Run tools and send"
											>
												<FiFastForward size={18} />
											</button>
										</div>
									)}

									{/* Send only (plane). Disabled while there are pending tools. */}
									<div
										className="tooltip tooltip-left"
										data-tip={hasPendingToolCalls ? 'Send (enabled after tools finish)' : 'Send message'}
									>
										<button
											type="button"
											className={`btn btn-circle btn-neutral btn-sm shrink-0 ${!canSendOnly ? 'btn-disabled' : ''}`}
											disabled={!canSendOnly}
											onClick={() => {
												if (!canSendOnly) return;
												void doSubmit({ runPendingTools: false });
											}}
											aria-label="Send message"
										>
											<FiSend size={18} />
										</button>
									</div>
								</>
							)}
						</div>
					</div>

					{/* Bottom bar for template/tool/attachment pickers + tips menus */}
					<EditorBottomBar
						onAttachFiles={handleAttachFiles}
						onAttachDirectory={handleAttachDirectory}
						onAttachURL={handleAttachURL}
						templateMenuState={templateMenu}
						toolMenuState={toolMenu}
						attachmentMenuState={attachmentMenu}
						templateButtonRef={templateButtonRef}
						toolButtonRef={toolButtonRef}
						attachmentButtonRef={attachmentButtonRef}
						shortcutConfig={shortcutConfig}
						currentProviderSDKType={currentProviderSDKType}
						onToolsChanged={handleAttachedToolsChanged}
						attachedToolEntries={attachedToolEntries}
						webSearchTemplates={webSearchTemplates}
						setWebSearchTemplates={setWebSearchTemplatesAndMaybeAutoExecute}
						allSkills={allSkills}
						skillsLoading={skillsLoading}
						enabledSkillRefs={enabledSkillRefs}
						setEnabledSkillRefs={setEnabledSkillRefs}
						onEnableAllSkills={enableAllSkills}
						onDisableAllSkills={disableAllSkills}
					/>
				</Plate>
			</form>

			{/* Tool choice / call inspector modal */}

			<ToolDetailsModal
				state={toolDetailsState}
				onClose={() => {
					setToolDetailsState(null);
				}}
			/>

			{/* Tool user-args editor modal host */}

			<ToolArgsModalHost
				editor={editor}
				conversationToolsState={conversationToolsState}
				setConversationToolsState={setConversationToolsStateAndMaybeAutoExecute}
				toolArgsTarget={toolArgsTarget}
				setToolArgsTarget={setToolArgsTarget}
				recomputeAttachedToolArgsBlocked={handleAttachedToolsChanged}
				webSearchTemplates={webSearchTemplates}
				setWebSearchTemplates={setWebSearchTemplatesAndMaybeAutoExecute}
			/>
		</>
	);
});
