import {
	forwardRef,
	type SubmitEventHandler,
	useCallback,
	useEffect,
	useImperativeHandle,
	useMemo,
	useRef,
	useState,
} from 'react';

import { FiAlertTriangle, FiEdit2, FiFastForward, FiPlay, FiSend, FiSquare, FiX } from 'react-icons/fi';

import { useMenuStore, useStoreState } from '@ariakit/react';
import { Plate, PlateContent } from 'platejs/react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { ProviderSDKType, UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SkillRef } from '@/spec/skill';
import type { ToolListItem, ToolStoreChoice } from '@/spec/tool';

import { type ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { cssEscape } from '@/lib/text_utils';

import { useEnterSubmit } from '@/hooks/use_enter_submit';

import { useComposerAttachments } from '@/chats/attachments/use_composer_attachments';
import { dispatchOpenToolArgs } from '@/chats/events/open_attached_toolargs';
import { dispatchTemplateFlashEvent } from '@/chats/events/template_flash';
import { EditorBottomBar } from '@/chats/inputarea/input_editor_bottom_bar';
import { EditorChipsBar } from '@/chats/inputarea/input_editor_chips_bar';
import {
	buildEditorValueFromPlainText,
	type EditorExternalMessage,
	type EditorSubmitPayload,
	hasNonEmptyUserText,
} from '@/chats/inputarea/input_editor_utils';
import { useComposerDocument } from '@/chats/platedoc/use_composer_document';
import { useComposerSkills } from '@/chats/skills/use_composer_skills';
import { getTemplateSelections, toPlainTextReplacingVariables } from '@/chats/templates/template_editor_utils';
import { TemplateToolbars } from '@/chats/templates/template_toolbars';
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
	insertToolSelectionNode,
	removeToolByKey,
	setToolAutoExecuteByKey,
	toolIdentityKey,
	type ToolSelectionElementNode,
} from '@/chats/tools/tool_editor_utils';
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
	const isSubmittingRef = useRef<boolean>(false);

	const {
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
	} = useComposerDocument({
		isBusy,
	});

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

	const getAttachedToolIdentityKeys = useCallback(() => {
		return new Set(
			getToolNodesWithPath(editor, false).map(([node]) =>
				toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion)
			)
		);
	}, [editor]);

	const handleAttachTool = useCallback(
		(item: ToolListItem, autoExecute: boolean) => {
			const identityKey = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
			if (getAttachedToolIdentityKeys().has(identityKey)) return;

			insertToolSelectionNode(
				editor,
				{
					bundleID: item.bundleID,
					bundleSlug: item.bundleSlug,
					toolSlug: item.toolSlug,
					toolVersion: item.toolVersion,
				},
				item.toolDefinition,
				{ autoExecute }
			);
			handleAttachedToolsChanged();
		},
		[editor, getAttachedToolIdentityKeys, handleAttachedToolsChanged]
	);

	const handleDetachAttachedToolByKey = useCallback(
		(identityKey: string) => {
			if (!getAttachedToolIdentityKeys().has(identityKey)) return;

			removeToolByKey(editor, identityKey);
			handleAttachedToolsChanged();
		},
		[editor, getAttachedToolIdentityKeys, handleAttachedToolsChanged]
	);

	const handleSetAttachedToolAutoExecuteByKey = useCallback(
		(identityKey: string, autoExecute: boolean) => {
			if (!getAttachedToolIdentityKeys().has(identityKey)) return;

			setToolAutoExecuteByKey(editor, identityKey, autoExecute);
			handleAttachedToolsChanged();
		},
		[editor, getAttachedToolIdentityKeys, handleAttachedToolsChanged]
	);

	const handleToggleAttachedToolAutoExecute = useCallback(
		(node: ToolSelectionElementNode, autoExecute: boolean) => {
			const identityKey = toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion);
			handleSetAttachedToolAutoExecuteByKey(identityKey, autoExecute);
		},
		[handleSetAttachedToolAutoExecuteByKey]
	);

	const handleRemoveAttachedTool = useCallback(
		(node: ToolSelectionElementNode) => {
			const identityKey = toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion);
			handleDetachAttachedToolByKey(identityKey);
		},
		[handleDetachAttachedToolByKey]
	);

	const handleRemoveAllAttachedTools = useCallback(
		(nodes: ToolSelectionElementNode[]) => {
			if (nodes.length === 0) return;

			const attachedKeys = getAttachedToolIdentityKeys();
			const uniqueKeys = new Set<string>();

			for (const node of nodes) {
				const identityKey = toolIdentityKey(node.bundleID, node.bundleSlug, node.toolSlug, node.toolVersion);
				if (!attachedKeys.has(identityKey) || uniqueKeys.has(identityKey)) continue;
				uniqueKeys.add(identityKey);
			}

			if (uniqueKeys.size === 0) return;

			for (const identityKey of uniqueKeys) {
				removeToolByKey(editor, identityKey);
			}

			handleAttachedToolsChanged();
		},
		[editor, getAttachedToolIdentityKeys, handleAttachedToolsChanged]
	);

	const handleEditAttachedToolOptions = useCallback((node: ToolSelectionElementNode) => {
		dispatchOpenToolArgs({ kind: 'attached', selectionID: node.selectionID });
	}, []);

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
		isSubmittingRef.current = false;

		clearAttachments();
		clearComposerToolsState();
	}, [clearAttachments, clearComposerToolsState, closeAllMenus]);

	const resetEditor = useCallback(() => {
		clearComposerTransientState();
		resetEditorDocument();
	}, [clearComposerTransientState, resetEditorDocument]);

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
			contentRef,
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

	const handleEditorDocumentChange = useCallback(() => {
		const didProcessChange = onEditorChange();
		if (!didProcessChange) return;

		if (toolCalls.length > 0) {
			handleAttachedToolsChanged();
		}
		if (submitError) {
			setSubmitError(null);
		}

		// Auto-cancel editing when the editor is completely empty
		// (no text, no tools, no attachments, no tool outputs).
		const hasTextNow = editingMessageId ? hasNonEmptyUserText(editor) : hasTextRef.current;

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
	}, [
		attachments,
		cancelEditing,
		editingMessageId,
		editor,
		handleAttachedToolsChanged,
		hasTextRef,
		onEditorChange,
		restorePreEditContext,
		submitError,
		toolCalls,
		toolOutputs,
	]);

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
				<Plate editor={editor} onChange={handleEditorDocumentChange}>
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
									onPaste={onEditorPaste}
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
									onToggleAttachedToolAutoExecute={handleToggleAttachedToolAutoExecute}
									onRemoveAttachedTool={handleRemoveAttachedTool}
									onRemoveAllAttachedTools={handleRemoveAllAttachedTools}
									onEditAttachedToolOptions={handleEditAttachedToolOptions}
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
						attachedToolEntries={attachedToolEntries}
						onAttachTool={handleAttachTool}
						onDetachToolByKey={handleDetachAttachedToolByKey}
						onSetAttachedToolAutoExecute={handleSetAttachedToolAutoExecuteByKey}
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
