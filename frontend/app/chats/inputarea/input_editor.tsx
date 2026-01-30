import {
	type FormEvent,
	forwardRef,
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
import { Plate, PlateContent, usePlateEditor } from 'platejs/react';

import type {
	Attachment,
	AttachmentContentBlockMode,
	DirectoryAttachmentsResult,
	UIAttachment,
} from '@/spec/attachment';
import { AttachmentKind } from '@/spec/attachment';
import type { ProviderSDKType, UIToolCall, UIToolOutput } from '@/spec/inference';
import { type Tool, type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { type ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { compareEntryByPathDeepestFirst } from '@/lib/path_utils';
import { cssEscape } from '@/lib/text_utils';

import { useEnterSubmit } from '@/hooks/use_enter_submit';

import { backendAPI, toolStoreAPI } from '@/apis/baseapi';

import { AlignKit } from '@/components/editor/plugins/align_kit';
import { BasicBlocksKit } from '@/components/editor/plugins/basic_blocks_kit';
import { BasicMarksKit } from '@/components/editor/plugins/basic_marks_kit';
import { EmojiKit } from '@/components/editor/plugins/emoji_kit';
import { FloatingToolbarKit } from '@/components/editor/plugins/floating_toolbar_kit';
import { IndentKit } from '@/components/editor/plugins/indent_kit';
import { LineHeightKit } from '@/components/editor/plugins/line_height_kit';
import { ListKit } from '@/components/editor/plugins/list_kit';
import { TabbableKit } from '@/components/editor/plugins/tabbable_kit';

import { AttachmentBottomBar } from '@/chats/attachments/attachment_bottom_bar';
import {
	buildUIAttachmentForLocalPath,
	buildUIAttachmentForURL,
	type DirectoryAttachmentGroup,
	MAX_FILES_PER_DIRECTORY,
	uiAttachmentKey,
} from '@/chats/attachments/attachment_editor_utils';
import { useOpenToolArgs } from '@/chats/events/open_attached_toolargs';
import { dispatchTemplateFlashEvent } from '@/chats/events/template_flash';
import { EditorChipsBar } from '@/chats/inputarea/input_editor_chips_bar';
import {
	buildSingleParagraphValue,
	buildSingleParagraphValueChunked,
	clearAllMarks,
	type EditorExternalMessage,
	type EditorSubmitPayload,
	hasNonEmptyUserText,
	insertPlainTextAsSingleBlock,
	isCursorAtDocumentEnd,
	LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_AUTODECHUNK_THRESHOLD_CHARS,
	LARGE_TEXT_CHUNK_SIZE,
} from '@/chats/inputarea/input_editor_utils';
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
	initConversationToolsStateFromChoices,
	mergeConversationToolsWithNewChoices,
} from '@/chats/tools/conversation_tools_chip';
import { ToolDetailsModal, type ToolDetailsState } from '@/chats/tools/tool_details_modal';
import {
	computeToolUserArgsStatus,
	dedupeToolChoices,
	editorAttachedToolToToolChoice,
	formatToolOutputSummary,
	getAttachedTools,
	getToolNodesWithPath,
	type ToolSelectionElementNode,
} from '@/chats/tools/tool_editor_utils';
import { ToolPlusKit } from '@/chats/tools/tool_plugin';
import { ToolArgsModalHost } from '@/chats/tools/tool_user_args_host';
import { type ToolArgsTarget } from '@/chats/tools/tool_user_args_modal';
import {
	buildWebSearchChoicesForSubmit,
	type WebSearchChoiceTemplate,
	webSearchTemplateFromChoice,
} from '@/chats/tools/websearch_utils';

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
}

const EDITOR_EMPTY_VALUE: Value = [{ type: 'p', children: [{ text: '' }] }];

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
	const editor = usePlateEditor({
		plugins: [
			SingleBlockPlugin,
			...BasicBlocksKit,
			...BasicMarksKit,
			...LineHeightKit,
			...AlignKit,
			...EmojiKit,
			...IndentKit,
			...ListKit,
			// ...AutoformatKit, // Don't want any formatting on typing
			...TabbableKit,
			...TemplateSlashKit,
			...ToolPlusKit,
			...FloatingToolbarKit,
		],
		value: EDITOR_EMPTY_VALUE,
	});

	const isSubmittingRef = useRef<boolean>(false);
	const contentRef = useRef<HTMLDivElement | null>(null);
	const editorRef = useRef(editor);
	editorRef.current = editor; // keep a live ref for key handlers
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

	const [submitError, setSubmitError] = useState<string | null>(null);
	const [attachments, setAttachments] = useState<UIAttachment[]>([]);
	const [directoryGroups, setDirectoryGroups] = useState<DirectoryAttachmentGroup[]>([]);

	// Tool-call chips (assistant-suggested) + tool outputs attached to the next user message.
	const [toolCalls, setToolCalls] = useState<UIToolCall[]>([]);
	const [toolOutputs, setToolOutputs] = useState<UIToolOutput[]>([]);

	// Only recompute attached-tool entries when tools change (not on every keystroke).
	const [toolNodesVersion, setToolNodesVersion] = useState(0);
	const attachedToolEntries = useMemo(() => getToolNodesWithPath(editor), [editor, toolNodesVersion]);

	const [toolDetailsState, setToolDetailsState] = useState<ToolDetailsState>(null);

	const [conversationToolsState, setConversationToolsState] = useState<ConversationToolStateEntry[]>([]);
	// When editing an earlier message we temporarily override the current
	// conversation-tool + web-search config. Keep a snapshot so Cancel restores it.
	const preEditConversationToolsRef = useRef<ConversationToolStateEntry[] | null>(null);
	const preEditWebSearchTemplatesRef = useRef<WebSearchChoiceTemplate[] | null>(null);

	// Count of in‑flight tool‑definition hydration tasks (conversation‑level + attached).
	// Used to gate sending while schemas are still loading.
	const [toolsHydratingCount, setToolsHydratingCount] = useState(0);
	const toolsDefLoading = toolsHydratingCount > 0;
	// Arg-blocking state, split by attached-vs-conversation tools.
	const [attachedToolArgsBlocked, setAttachedToolArgsBlocked] = useState(false);
	const [conversationToolArgsBlocked, setConversationToolArgsBlocked] = useState(false);
	const toolArgsBlocked = attachedToolArgsBlocked || conversationToolArgsBlocked;

	// Single “active tool args editor” target (conversation-level or attached).
	const [toolArgsTarget, setToolArgsTarget] = useState<ToolArgsTarget | null>(null);
	const [webSearchTemplates, setWebSearchTemplates] = useState<WebSearchChoiceTemplate[]>([]);

	useOpenToolArgs(target => {
		setToolArgsTarget(target);
	});

	const closeAllMenus = useCallback(() => {
		templateMenu.hide();
		toolMenu.hide();
		attachmentMenu.hide();
	}, [templateMenu, toolMenu, attachmentMenu]);

	// --- Fix: focus management for menus opened by shortcuts ---
	const templateMenuOpen = useStoreState(templateMenu, 'open');
	const toolMenuOpen = useStoreState(toolMenu, 'open');
	const attachmentMenuOpen = useStoreState(attachmentMenu, 'open');

	const templateMenuEl = useStoreState(templateMenu, 'contentElement');
	const toolMenuEl = useStoreState(toolMenu, 'contentElement');
	const attachmentMenuEl = useStoreState(attachmentMenu, 'contentElement');

	useEffect(() => {
		if (!templateMenuOpen) {
			if (menuOpenedByShortcutRef.current.templates) {
				menuOpenedByShortcutRef.current.templates = false;
				editor.tf.focus();
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.templates) return;

		requestAnimationFrame(() => {
			templateMenuEl?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
		});
	}, [templateMenuOpen, templateMenuEl, editor]);

	useEffect(() => {
		if (!toolMenuOpen) {
			if (menuOpenedByShortcutRef.current.tools) {
				menuOpenedByShortcutRef.current.tools = false;
				editor.tf.focus();
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.tools) return;

		requestAnimationFrame(() => {
			toolMenuEl?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
		});
	}, [toolMenuOpen, toolMenuEl, editor]);

	useEffect(() => {
		if (!attachmentMenuOpen) {
			if (menuOpenedByShortcutRef.current.attachments) {
				menuOpenedByShortcutRef.current.attachments = false;
				editor.tf.focus();
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.attachments) return;

		requestAnimationFrame(() => {
			attachmentMenuEl?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
		});
	}, [attachmentMenuOpen, attachmentMenuEl, editor]);

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
	}, [editor, deferredDocVersion]);

	const hasPendingToolCalls = useMemo(() => toolCalls.some(c => c.status === 'pending'), [toolCalls]);
	const hasRunningToolCalls = useMemo(() => toolCalls.some(c => c.status === 'running'), [toolCalls]);
	const templateBlocked = selectionInfo.hasTemplate && selectionInfo.requiredCount > 0;

	// Populate editor with effective last-USER block for EACH template selection (once per selectionID)
	useEffect(() => {
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
						editor.tf.focus();
					}
				} catch {
					editor.tf.focus();
				}
			});
		}
	}, [editor, deferredDocVersion]);

	// Helper to recompute attached-tool arg blocking on demand (not tied to docVersion).
	const recomputeAttachedToolArgsBlocked = useCallback(() => {
		const toolEntries = getToolNodesWithPath(editor, false);
		for (const [node] of toolEntries) {
			const schema = node.toolSnapshot?.userArgSchema;
			const status = computeToolUserArgsStatus(schema, node.userArgSchemaInstance);
			if (status.hasSchema && !status.isSatisfied) {
				setAttachedToolArgsBlocked(true);
				return;
			}
		}
		setAttachedToolArgsBlocked(false);
	}, [editor]);

	// Single callback to call whenever attached tools change (add/remove/options),
	// so we update both arg-blocking + cached tool entries without scanning on every keystroke.
	const handleAttachedToolsChanged = useCallback(() => {
		setToolNodesVersion(v => v + 1);
		recomputeAttachedToolArgsBlocked();
	}, [recomputeAttachedToolArgsBlocked]);

	// Ensure we have full Tool definitions (and argStatus) for conversation-level tools.
	useEffect(() => {
		const missing = conversationToolsState.filter(e => !e.toolDefinition);
		if (!missing.length) return;

		let cancelled = false;
		setToolsHydratingCount(c => c + 1);

		(async () => {
			const results = await Promise.all(
				missing.map(async entry => {
					try {
						const def = await toolStoreAPI.getTool(
							entry.toolStoreChoice.bundleID,
							entry.toolStoreChoice.toolSlug,
							entry.toolStoreChoice.toolVersion
						);
						return def ? { key: entry.key, def } : null;
					} catch {
						return null;
					}
				})
			);

			if (cancelled) return;

			const loaded = results.filter((r): r is { key: string; def: Tool } => r !== null);
			if (!loaded.length) return;

			setConversationToolsState(prev =>
				prev.map(e => {
					const hit = loaded.find(l => l.key === e.key);
					if (!hit) return e;
					const argStatus = computeToolUserArgsStatus(hit.def.userArgSchema, e.toolStoreChoice.userArgSchemaInstance);
					return { ...e, toolDefinition: hit.def, argStatus };
				})
			);
		})().finally(() => {
			if (!cancelled) {
				setToolsHydratingCount(c => Math.max(0, c - 1));
			}
		});

		return () => {
			cancelled = true;
		};
	}, [conversationToolsState]);

	const restorePreEditContext = useCallback(() => {
		const prevConv = preEditConversationToolsRef.current;
		const prevWs = preEditWebSearchTemplatesRef.current;

		if (prevConv) setConversationToolsState(prevConv);
		if (prevWs) setWebSearchTemplates(prevWs);

		preEditConversationToolsRef.current = null;
		preEditWebSearchTemplatesRef.current = null;
	}, []);

	// Recompute conversation-level arg blocking whenever that state changes.
	useEffect(() => {
		let blocked = false;
		for (const entry of conversationToolsState) {
			if (!entry.enabled) continue;
			const status = entry.argStatus;
			if (status?.hasSchema && !status.isSatisfied) {
				blocked = true;
				break;
			}
		}
		setConversationToolArgsBlocked(blocked);
	}, [conversationToolsState]);

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

	const runToolCallInternal = useCallback(async (toolCall: UIToolCall): Promise<UIToolOutput | null> => {
		if (toolCall.type !== ToolStoreChoiceType.Function && toolCall.type !== ToolStoreChoiceType.Custom) {
			const errMsg = 'This tool call type cannot be executed from the composer.';
			setToolCalls(prev =>
				prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: errMsg } : c))
			);
			return null;
		}

		// Resolve identity using toolStoreChoice when available; fall back to name parsing.
		let bundleID: string | undefined;
		let toolSlug: string | undefined;
		let version: string | undefined;

		if (toolCall.toolStoreChoice) {
			bundleID = toolCall.toolStoreChoice.bundleID;
			toolSlug = toolCall.toolStoreChoice.toolSlug;
			version = toolCall.toolStoreChoice.toolVersion;
		}

		if (!bundleID || !toolSlug || !version) {
			const errMsg = 'Cannot resolve tool identity for this call.';
			setToolCalls(prev =>
				prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: errMsg } : c))
			);
			return null;
		}

		// Mark as running (allow retry after failure by overwriting previous status).
		setToolCalls(prev =>
			prev.map(c => (c.id === toolCall.id ? { ...c, status: 'running', errorMessage: undefined } : c))
		);

		try {
			const resp = await toolStoreAPI.invokeTool(bundleID, toolSlug, version, toolCall.arguments);

			const isError = !!resp.isError;
			const errorMessage =
				resp.errorMessage || (isError ? 'Tool reported an error. Inspect the output for details.' : undefined);

			const output: UIToolOutput = {
				id: toolCall.id,
				callID: toolCall.callID,
				name: toolCall.name,
				choiceID: toolCall.choiceID,
				type: toolCall.type,
				summary: isError ? `Error: ${formatToolOutputSummary(toolCall.name)}` : formatToolOutputSummary(toolCall.name),
				toolStoreOutputs: resp.outputs,
				isError,
				errorMessage,
				arguments: toolCall.arguments,
				webSearchToolCallItems: toolCall.webSearchToolCallItems,
				toolStoreChoice: toolCall.toolStoreChoice,
			};

			// Remove the call chip & append the output.
			setToolCalls(prev => prev.filter(c => c.id !== toolCall.id));
			setToolOutputs(prev => [...prev, output]);

			return output;
		} catch (err) {
			const msg = (err as Error)?.message || 'Tool invocation failed.';
			setToolCalls(prev => prev.map(c => (c.id === toolCall.id ? { ...c, status: 'failed', errorMessage: msg } : c)));
			return null;
		}
	}, []);

	/**
	 * Run all currently pending tool calls (in sequence) and return the
	 * UIToolOutput objects produced in this pass.
	 */
	const runAllPendingToolCalls = useCallback(async (): Promise<UIToolOutput[]> => {
		const pending = toolCalls.filter(c => c.status === 'pending');
		if (pending.length === 0) return [];

		const produced: UIToolOutput[] = [];
		for (const chip of pending) {
			if (chip.type === ToolStoreChoiceType.Function || chip.type === ToolStoreChoiceType.Custom) {
				const out = await runToolCallInternal(chip);
				if (out) produced.push(out);
			}
		}
		return produced;
	}, [toolCalls, runToolCallInternal]);

	const handleRunSingleToolCall = useCallback(
		async (id: string) => {
			const chip = toolCalls.find(c => c.id === id && (c.status === 'pending' || c.status === 'failed'));
			if (!chip) return;
			await runToolCallInternal(chip);
		},
		[toolCalls, runToolCallInternal]
	);

	const handleDiscardToolCall = useCallback((id: string) => {
		setToolCalls(prev => prev.filter(c => c.id !== id));
	}, []);

	const handleRemoveToolOutput = useCallback((id: string) => {
		setToolOutputs(prev => prev.filter(o => o.id !== id));
		setToolDetailsState(current => (current && current.kind === 'output' && current.output.id === id ? null : current));
	}, []);

	const handleRetryErroredOutput = useCallback((output: UIToolOutput) => {
		// Only support retry for function/custom tools where we still
		// know which tool and arguments were used.
		if (!output.isError || !output.toolStoreChoice || !output.arguments) {
			return;
		}

		if (output.type !== ToolStoreChoiceType.Function && output.type !== ToolStoreChoiceType.Custom) {
			return;
		}

		const { bundleID, toolSlug, toolVersion } = output.toolStoreChoice;
		if (!bundleID || !toolSlug || !toolVersion) {
			return;
		}

		const newId = crypto.randomUUID();

		const chip: UIToolCall = {
			id: newId,
			callID: output.callID || newId,
			name: output.name,
			arguments: output.arguments,
			webSearchToolCallItems: output.webSearchToolCallItems,
			choiceID: output.choiceID,
			type: output.type,
			status: 'pending',
			toolStoreChoice: output.toolStoreChoice,
		};

		setToolOutputs(prev => prev.filter(o => o.id !== output.id));
		setToolCalls(prev => [...prev, chip]);
	}, []);

	const handleOpenToolOutput = useCallback((output: UIToolOutput) => {
		setToolDetailsState({ kind: 'output', output });
	}, []);

	const handleOpenToolCallDetails = useCallback((call: UIToolCall) => {
		setToolDetailsState({ kind: 'call', call });
	}, []);

	const handleOpenConversationToolDetails = useCallback((entry: ConversationToolStateEntry) => {
		setToolDetailsState({ kind: 'choice', choice: entry.toolStoreChoice });
	}, []);

	const handleOpenAttachedToolDetails = useCallback((node: ToolSelectionElementNode) => {
		const choice: ToolStoreChoice = {
			choiceID: node.choiceID,
			bundleID: node.bundleID,
			bundleSlug: node.bundleSlug,
			toolSlug: node.toolSlug,
			toolVersion: node.toolVersion,
			displayName: node.overrides?.displayName ?? node.toolSnapshot?.displayName ?? node.toolSlug,
			description: node.overrides?.description ?? node.toolSnapshot?.description ?? node.toolSlug,
			toolID: node.toolSnapshot?.id,
			toolType: node.toolType,
			userArgSchemaInstance: node.userArgSchemaInstance,
		};
		setToolDetailsState({ kind: 'choice', choice });
	}, []);

	/**
	 * Main submit logic, parameterized by whether to run pending tool calls
	 * before sending.
	 */
	const doSubmit = async (options: { runPendingTools: boolean }) => {
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
					editor.tf.focus();
				}
			} else {
				editor.tf.focus();
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
			};

			await onSubmit(payload);
			setSubmitError(null);
			setConversationToolsState(prev => mergeConversationToolsWithNewChoices(prev, finalToolChoices));
			didSend = true;
		} finally {
			isSubmittingRef.current = false;

			// Only clear the editor if we actually sent something.
			if (didSend) {
				editor.tf.setValue(EDITOR_EMPTY_VALUE);
				hasTextRef.current = false;
				setHasText(false);
				setAttachments([]);
				setDirectoryGroups([]);
				setToolCalls([]);
				setToolOutputs([]);
				setToolDetailsState(null);
				// If we were editing, the old snapshot is no longer relevant.
				preEditConversationToolsRef.current = null;
				preEditWebSearchTemplatesRef.current = null;

				lastPopulatedSelectionKeyRef.current.clear();
				editor.tf.focus();
			}
		}
	};
	/**
	 * Default form submit / Enter: "run pending tools, then send".
	 */
	const handleSubmit = (e?: FormEvent) => {
		if (e) e.preventDefault();
		void doSubmit({ runPendingTools: true });
	};

	const resetEditor = useCallback(() => {
		closeAllMenus();
		setSubmitError(null);
		lastPopulatedSelectionKeyRef.current.clear();
		isSubmittingRef.current = false;

		editor.tf.setValue(EDITOR_EMPTY_VALUE);
		setAttachments([]);
		setDirectoryGroups([]);
		setToolCalls([]);
		setToolOutputs([]);
		setToolDetailsState(null);
		setToolNodesVersion(v => v + 1);
		hasTextRef.current = false;
		setHasText(false);
		// Let Plate onChange bump docVersion; no need to do it here.
		editor.tf.focus();
	}, [closeAllMenus, editor]);

	const loadExternalMessage = useCallback(
		(incoming: EditorExternalMessage) => {
			closeAllMenus();
			setSubmitError(null);
			lastPopulatedSelectionKeyRef.current.clear();
			isSubmittingRef.current = false;

			// Snapshot current context so Cancel Editing can restore it.
			if (!preEditConversationToolsRef.current) {
				preEditConversationToolsRef.current = conversationToolsState;
			}
			if (!preEditWebSearchTemplatesRef.current) {
				preEditWebSearchTemplatesRef.current = webSearchTemplates;
			}

			// 1) Reset document to plain text paragraphs.
			const plain = incoming.text ?? '';
			// PERF: Keep a single block and preserve newlines within the text node.
			// Splitting into many paragraphs makes Slate normalization + plugin passes very expensive.
			// PERF: One paragraph, but chunk large text into many leaves.
			const value: Value =
				plain.length >= LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS
					? buildSingleParagraphValueChunked(plain, LARGE_TEXT_CHUNK_SIZE)
					: buildSingleParagraphValue(plain);
			editor.tf.withoutNormalizing(() => {
				editor.tf.setValue(value);
			});
			hasTextRef.current = plain.trim().length > 0;
			setHasText(hasTextRef.current);

			// 2) Rebuild attachments as UIAttachment[]
			setAttachments(() => {
				if (!incoming.attachments || incoming.attachments.length === 0) return [];
				const next: UIAttachment[] = [];
				const seen = new Set<string>();

				for (const att of incoming.attachments) {
					let ui: UIAttachment | undefined = undefined;

					if (att.kind === AttachmentKind.url) {
						// URL attachment
						if (att.urlRef) {
							ui = buildUIAttachmentForURL(att);
						} else {
							continue;
						}
					} else if (att.kind === AttachmentKind.file || att.kind === AttachmentKind.image) {
						// File/image/etc. – same type we originally got from backend.
						ui = buildUIAttachmentForLocalPath(att);
					}

					if (!ui) continue;

					const key = uiAttachmentKey(ui);
					if (seen.has(key)) continue;
					seen.add(key);
					next.push(ui);
				}
				return next;
			});

			// We don’t attempt to reconstruct directoryGroups; show flat chips instead.
			setDirectoryGroups([]);

			// 3) Re-add tool choices as tool selection nodes.
			// Restore tools into conversation-level state (persistent semantics).
			const incomingToolChoices = incoming.toolChoices ?? [];

			setConversationToolsState(initConversationToolsStateFromChoices(incomingToolChoices));

			// Restore web-search separately (NOT as tool nodes; separate UX).
			const wsChoices = incomingToolChoices.filter(c => c.toolType === ToolStoreChoiceType.WebSearch);
			setWebSearchTemplates(wsChoices.map(webSearchTemplateFromChoice));

			// Since we no longer reconstruct attached-tool nodes from history,
			// ensure attached-tool arg blocking resets.
			setAttachedToolArgsBlocked(false);

			// 4) Restore any tool outputs that were previously attached to this message.
			setToolOutputs(incoming.toolOutputs ?? []);
			setToolCalls([]);
			setToolDetailsState(null);
			setToolNodesVersion(v => v + 1);

			editor.tf.focus();
		},
		[closeAllMenus, editor, conversationToolsState, webSearchTemplates]
	);

	const loadToolCalls = useCallback((toolCalls: UIToolCall[]) => {
		setToolCalls(toolCalls);
	}, []);

	useImperativeHandle(ref, () => ({
		focus: () => {
			editor.tf.focus();
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
		setConversationToolsFromChoices: (tools: ToolStoreChoice[]) => {
			setConversationToolsState(initConversationToolsStateFromChoices(tools));
		},

		setWebSearchFromChoices: (tools: ToolStoreChoice[]) => {
			const ws = (tools ?? []).filter(t => t.toolType === ToolStoreChoiceType.WebSearch);
			setWebSearchTemplates(ws.map(webSearchTemplateFromChoice));
		},
	}));

	const handleAttachFiles = async () => {
		let results: Attachment[];
		try {
			results = await backendAPI.openMultipleFilesAsAttachments(true);
		} catch {
			return;
		}

		if (!results || results.length === 0) return;

		setAttachments(prev => {
			const existing = new Set(prev.map(uiAttachmentKey));
			const next: UIAttachment[] = [...prev];

			for (const r of results) {
				const att = buildUIAttachmentForLocalPath(r);
				if (!att) {
					console.error('invalid attachment result');
					continue;
				}
				const key = uiAttachmentKey(att);
				if (existing.has(key)) continue;
				existing.add(key);
				next.push(att);
			}
			return next;
		});
		editor.tf.focus();
	};

	const handleAttachDirectory = async () => {
		let result: DirectoryAttachmentsResult;
		try {
			result = await backendAPI.openDirectoryAsAttachments(MAX_FILES_PER_DIRECTORY);
		} catch {
			// Backend canceled or errored; nothing to do.
			return;
		}

		if (!result || !result.dirPath) return;

		const { dirPath, attachments: dirAttachments, overflowDirs } = result;

		if ((!dirAttachments || dirAttachments.length === 0) && (!overflowDirs || overflowDirs.length === 0)) {
			// Nothing readable / allowed in this folder.
			return;
		}

		const folderLabel = dirPath.trim().split(/[\\/]/).pop() || dirPath.trim();

		const groupId = crypto.randomUUID() ?? `dir-${Date.now()}-${Math.random().toString(16).slice(2)}`;

		// First, add or reuse attachments.
		const attachmentKeysForGroup: string[] = [];
		const ownedAttachmentKeysForGroup: string[] = [];

		// Deduplicate paths/keys within this single directory attach
		const seenKeysForGroup = new Set<string>();

		setAttachments(prev => {
			const existing = new Map<string, UIAttachment>();
			for (const att of prev) {
				existing.set(uiAttachmentKey(att), att);
			}

			const added: UIAttachment[] = [];

			for (const r of dirAttachments ?? []) {
				const att = buildUIAttachmentForLocalPath(r);
				if (!att) {
					console.error('invalid attachment result');
					continue;
				}
				const key = uiAttachmentKey(att);

				// Skip duplicates within this folder selection
				if (seenKeysForGroup.has(key)) continue;
				seenKeysForGroup.add(key);

				attachmentKeysForGroup.push(key);

				if (!existing.has(key)) {
					existing.set(key, att);
					added.push(att);
					ownedAttachmentKeysForGroup.push(key);
				}
			}

			return [...prev, ...added];
		});

		// Then, record the directory group that references those attachments.
		setDirectoryGroups(prev => [
			...prev,
			{
				id: groupId,
				dirPath,
				label: folderLabel,
				attachmentKeys: attachmentKeysForGroup,
				ownedAttachmentKeys: ownedAttachmentKeysForGroup,

				overflowDirs: overflowDirs ?? [],
			},
		]);

		editor.tf.focus();
	};

	const handleAttachURL = async (rawUrl: string) => {
		const trimmed = rawUrl.trim();
		if (!trimmed) return;

		const bAtt = await backendAPI.openURLAsAttachment(trimmed);
		if (!bAtt) return;
		const att = buildUIAttachmentForURL(bAtt);
		const key = uiAttachmentKey(att);

		setAttachments(prev => {
			const existing = new Set(prev.map(uiAttachmentKey));
			if (existing.has(key)) return prev;
			return [...prev, att];
		});
	};

	const handleChangeAttachmentContentBlockMode = (att: UIAttachment, newMode: AttachmentContentBlockMode) => {
		const targetKey = uiAttachmentKey(att);
		setAttachments(prev => prev.map(a => (uiAttachmentKey(a) === targetKey ? { ...a, mode: newMode } : a)));
		editor.tf.focus();
	};

	const handleRemoveAttachment = (att: UIAttachment) => {
		const targetKey = uiAttachmentKey(att);

		setAttachments(prev => prev.filter(a => uiAttachmentKey(a) !== targetKey));

		// Also detach from any directory groups (and drop empty groups)
		setDirectoryGroups(prevGroups => {
			const updated = prevGroups.map(g => ({
				...g,
				attachmentKeys: g.attachmentKeys.filter(k => k !== targetKey),
				ownedAttachmentKeys: g.ownedAttachmentKeys.filter(k => k !== targetKey),
			}));
			return updated.filter(g => g.attachmentKeys.length > 0 || g.overflowDirs.length > 0);
		});
	};

	const handleRemoveDirectoryGroup = (groupId: string) => {
		setDirectoryGroups(prevGroups => {
			const groupToRemove = prevGroups.find(g => g.id === groupId);
			if (!groupToRemove) return prevGroups;

			const remainingGroups = prevGroups.filter(g => g.id !== groupId);

			// Keys owned by other groups (so we don't delete shared attachments).
			const keysOwnedByOtherGroups = new Set<string>();
			for (const g of remainingGroups) {
				for (const key of g.ownedAttachmentKeys) {
					keysOwnedByOtherGroups.add(key);
				}
			}

			if (groupToRemove.ownedAttachmentKeys.length > 0) {
				setAttachments(prevAttachments =>
					prevAttachments.filter(att => {
						const key = uiAttachmentKey(att);
						if (!groupToRemove.ownedAttachmentKeys.includes(key)) return true;
						// If other groups still own this attachment, keep it.
						if (keysOwnedByOtherGroups.has(key)) return true;
						// Otherwise, drop it when this folder is removed.
						return false;
					})
				);
			}

			return remainingGroups;
		});
	};

	const handleRemoveOverflowDir = (groupId: string, dirPath: string) => {
		setDirectoryGroups(prevGroups => {
			const updated = prevGroups.map(g =>
				g.id !== groupId
					? g
					: {
							...g,
							overflowDirs: g.overflowDirs.filter(od => od.dirPath !== dirPath),
						}
			);
			return updated.filter(g => g.attachmentKeys.length > 0 || g.overflowDirs.length > 0);
		});
	};

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
		hasPendingToolCalls && !hasRunningToolCalls && !isBusy && !templateBlocked && !toolsDefLoading;
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
						if (isAutoChunkingRef.current) {
							// Avoid feedback loops.
							return;
						}
						scheduleDocRecompute();
						scheduleAutoChunk();

						if (submitError) {
							setSubmitError(null);
						}

						// Auto-cancel editing when the editor is completely empty
						// (no text, no tools, no attachments, no tool outputs).
						const hasTextNow = editingMessageId ? hasNonEmptyUserText(editorRef.current) : hasTextRef.current;

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

										// PERF: if paste is huge AND editor is empty, set chunked value directly.
										if (!hasTextRef.current && text.length >= LARGE_TEXT_AUTOCHUNK_THRESHOLD_CHARS) {
											editor.tf.withoutNormalizing(() => {
												editor.tf.setValue(buildSingleParagraphValueChunked(text, LARGE_TEXT_CHUNK_SIZE));
												editor.tf.collapse({ edge: 'end' });
											});

											hasTextRef.current = true;
											setHasText(true);
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
									onConversationToolsChange={setConversationToolsState}
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
					<AttachmentBottomBar
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
						webSearchTemplates={webSearchTemplates}
						setWebSearchTemplates={setWebSearchTemplates}
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
				setConversationToolsState={setConversationToolsState}
				toolArgsTarget={toolArgsTarget}
				setToolArgsTarget={setToolArgsTarget}
				recomputeAttachedToolArgsBlocked={recomputeAttachedToolArgsBlocked}
				webSearchTemplates={webSearchTemplates}
				setWebSearchTemplates={setWebSearchTemplates}
			/>
		</>
	);
});
