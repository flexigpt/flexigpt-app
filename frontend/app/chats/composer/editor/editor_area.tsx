import {
	forwardRef,
	type SubmitEventHandler,
	useCallback,
	useEffect,
	useImperativeHandle,
	useLayoutEffect,
	useMemo,
	useRef,
	useState,
} from 'react';

import {
	FiAlertTriangle,
	FiEdit2,
	FiFastForward,
	FiPlay,
	FiSend,
	FiSquare,
	FiTool,
	FiX,
	FiZapOff,
} from 'react-icons/fi';

import { useMenuStore, useStoreState } from '@ariakit/react';
import { Plate, PlateContent } from 'platejs/react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { ProviderSDKType, UIToolCall, UIToolOutput } from '@/spec/inference';
import type { MCPAppModelContextUpdate, MCPConversationContext } from '@/spec/mcp';
import type { PromptTemplate } from '@/spec/prompt';
import type { SkillRef } from '@/spec/skill';
import { type ToolArgsTarget, type ToolListItem, type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { type ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { cssEscape } from '@/lib/text_utils';

import { useEnterSubmit } from '@/hooks/use_enter_submit';

import { HoverTip } from '@/components/ariakit_hover_tip';

import {
	type AssistantPresetRuntimeSnapshot,
	mapAssistantPresetWebSearchTemplatesToChoices,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { useComposerAttachments } from '@/chats/composer/attachments/use_composer_attachments';
import { EditorBottomBar } from '@/chats/composer/editor/editor_bottom_bar';
import { EditorChipsBar } from '@/chats/composer/editor/editor_chips_bar';
import type {
	AssistantTurnFinishedPayload,
	EditorExternalMessage,
	EditorSubmitPayload,
} from '@/chats/composer/editor/editor_types';
import { MCPApprovalModal } from '@/chats/composer/mcp/mcp_approval_modal';
import { useComposerMCP } from '@/chats/composer/mcp/use_composer_mcp';
import { useMCPApproval } from '@/chats/composer/mcp/use_mcp_approval';
import { buildEditorValueFromPlainText, hasNonEmptyUserText } from '@/chats/composer/platedoc/platedoc_utils';
import {
	getInstructionPromptPartsFromSelections,
	getTemplateSelections,
	insertTemplateSelectionNode,
	toPlainTextReplacingVariables,
} from '@/chats/composer/platedoc/template_document_ops';
import {
	type AttachedToolEntry,
	getAttachedTools,
	insertToolSelectionNode,
	removeToolByKey,
	setAttachedToolUserArgSchemaInstanceBySelectionID,
	setToolAutoExecuteByKey,
} from '@/chats/composer/platedoc/tool_document_ops';
import { useComposerDocument } from '@/chats/composer/platedoc/use_composer_document';
import { useComposerSkills } from '@/chats/composer/skills/use_composer_skills';
import type { ComposerSystemPromptController } from '@/chats/composer/systemprompts/use_composer_system_prompt';
import { TemplateToolbars } from '@/chats/composer/templates/template_toolbars';
import { dispatchTemplateFlashEvent } from '@/chats/composer/templates/use_template_flash_event';
import {
	createAutoSubmitTracker,
	getToolAutoSubmitKey,
	isAutoSubmitEligibleToolCall,
} from '@/chats/composer/toolruntime/tool_runtime_utils';
import { useComposerTools } from '@/chats/composer/toolruntime/use_composer_tools';
import { dispatchOpenToolArgs, useOpenToolArgs } from '@/chats/composer/toolruntime/use_open_toolargs_event';
import { ToolDetailsModal, type ToolDetailsState } from '@/chats/composer/tools/tool_details_modal';
import { ToolArgsModalHost } from '@/chats/composer/tools/tool_user_args_host';
import { buildWebSearchChoicesForSubmit, type WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import { appendSystemPromptParts } from '@/prompts/lib/system_prompt_utils';
import {
	type ConversationToolStateEntry,
	conversationToolsToChoices,
	mergeConversationToolsWithNewChoices,
} from '@/tools/lib/conversation_tool_utils';
import { dedupeToolChoices, uiToolChoiceToToolStoreChoice } from '@/tools/lib/tool_choice_utils';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

type SkillStateApplyOptions = {
	syncSession?: 'none' | 'if-session-exists' | 'ensure-if-enabled';
	forceResetSession?: boolean;
};

export interface EditorAreaHandle {
	focus: () => void;
	openTemplateMenu: () => void;
	openToolMenu: () => void;
	openAttachmentMenu: () => void;
	loadExternalMessage: (msg: EditorExternalMessage) => void;
	setDraftText: (text: string) => void;
	setDraftTextIfEmpty: (text: string) => boolean;
	resetEditor: () => void;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	setConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	setMCPContextFromMessage: (context?: MCPConversationContext) => void;
	setMCPAppContextUpdatesFromMessage: (updates?: MCPAppModelContextUpdate[]) => void;
	appendMCPAppContextUpdate: (update: MCPAppModelContextUpdate) => void;
	clearMCPContext: () => void;
	setWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
	applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => void;
	setSkillStateFromMessage: (enabledRefs: SkillRef[], activeRefs: SkillRef[], options?: SkillStateApplyOptions) => void;
	finishAssistantTurn: (payload: AssistantTurnFinishedPayload) => void;
}

interface EditorAreaProps {
	isGenerating: boolean;
	isInputLocked: boolean;
	currentProviderSDKType: ProviderSDKType;
	shortcutConfig: ShortcutConfig;
	onSubmit: (payload: EditorSubmitPayload) => Promise<void>;
	onRequestStop: () => void;
	onAssistantPresetRuntimeStateChange?: (snapshot: AssistantPresetRuntimeSnapshot) => void;
	editingMessageId: string | null;
	cancelEditing: () => void;
	systemPrompt: ComposerSystemPromptController;
}

function isAutoExecutableToolChoice(choice: ToolStoreChoice): boolean {
	return (
		choice.autoExecute &&
		(choice.toolType === ToolStoreChoiceType.Function || choice.toolType === ToolStoreChoiceType.Custom)
	);
}

function countAutoExecutableToolChoices(choices: ToolStoreChoice[]): number {
	return choices.filter(isAutoExecutableToolChoice).length;
}

type SubmitOptions = { runPendingTools: boolean };

export const EditorArea = forwardRef<EditorAreaHandle, EditorAreaProps>(function EditorArea(
	{
		isGenerating,
		isInputLocked,
		currentProviderSDKType,
		shortcutConfig,
		onSubmit,
		onRequestStop,
		onAssistantPresetRuntimeStateChange,
		editingMessageId,
		cancelEditing,
		systemPrompt,
	},
	ref
) {
	const autoSubmitTrackerRef = useRef(createAutoSubmitTracker());
	const mcp = useComposerMCP();
	const mcpApproval = useMCPApproval();
	const resetAutoSubmitTracker = useCallback(() => {
		autoSubmitTrackerRef.current = createAutoSubmitTracker();
	}, []);

	const {
		editor,
		contentRef,
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
	} = useComposerDocument({
		isBusy: isInputLocked,
	});

	const templateMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const toolMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const attachmentMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const templateButtonRef = useRef<HTMLButtonElement | null>(null);
	const toolButtonRef = useRef<HTMLButtonElement | null>(null);
	const attachmentButtonRef = useRef<HTMLButtonElement | null>(null);

	const toolArgsEventTarget = useMemo<EventTarget | null>(() => {
		return typeof EventTarget !== 'undefined' ? new EventTarget() : null;
	}, []);

	// Track whether a menu was opened via shortcut so we can:
	// - force focus into the menu (arrow-key nav)
	// - optionally restore focus to editor on close (Esc)
	const menuOpenedByShortcutRef = useRef({ templates: false, tools: false, attachments: false });
	const suppressNextAttachmentMenuFocusRestoreRef = useRef(false);

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
		isBusy: isGenerating,
		focusEditorAtEnd,
	});

	const [submitError, setSubmitError] = useState<string | null>(null);
	const [isSubmitting, setIsSubmitting] = useState(false);
	const isSubmittingRef = useRef(false);
	const submitVisualFrameRef = useRef<number | null>(null);

	const [mcpAppContextUpdates, setMCPAppContextUpdates] = useState<MCPAppModelContextUpdate[]>([]);
	const [fastForwardPending, setFastForwardPending] = useState(false);

	// Guard: while true, handleEditorDocumentChange skips auto-cancel logic.
	// This prevents a race where Plate fires onChange after clearComposerTransientState
	// empties attachments/toolOutputs but before loadAttachmentsFromMessage restores them.
	const isLoadingExternalMessageRef = useRef(false);
	const externalMessageLoadReleaseTimerRef = useRef<number | null>(null);
	const [webSearchArgsBlocked, setWebSearchArgsBlocked] = useState(false);
	const [toolDetailsState, setToolDetailsState] = useState<ToolDetailsState>(null);
	const [toolArgsTarget, setToolArgsTarget] = useState<ToolArgsTarget | null>(null);

	useOpenToolArgs(target => {
		setToolArgsTarget(target);
	}, toolArgsEventTarget);

	const {
		allSkills,
		skillsLoading,
		enabledSkillRefs,
		activeSkillRefs,
		setEnabledSkillRefs,
		setActiveSkillRefs,
		enableAllSkills,
		disableAllSkills,
		applySkillSelectionState,
		ensureSkillSession,
		listActiveSkillRefs,
		getCurrentSkillSessionID,
		getCurrentEnabledSkillRefs,
		getCurrentActiveSkillRefs,
	} = useComposerSkills();

	const templateBlocked = selectionInfo.hasTemplate && selectionInfo.requiredCount > 0;

	const effectiveSubmitText = useMemo(() => {
		return selectionInfo.hasTemplate ? toPlainTextReplacingVariables(editor) : editor.api.string([]);
	}, [editor, selectionInfo]);
	const hasEffectiveTextForSubmit = effectiveSubmitText.trim().length > 0;

	const [autoExecStopVisible, setAutoExecStopVisible] = useState(false);
	const [autoExecStopRequested, setAutoExecStopRequested] = useState(false);
	const [autoExecBlockedByUser, setAutoExecBlockedByUser] = useState(false);
	const [activeAutoExecBatchCount, setActiveAutoExecBatchCount] = useState(0);
	const autoExecStopRequestedRef = useRef(false);
	const autoExecBlockedByUserRef = useRef(false);

	const clearAutoExecStopState = useCallback(() => {
		autoExecStopRequestedRef.current = false;
		autoExecBlockedByUserRef.current = false;
		setAutoExecStopRequested(false);
		setAutoExecBlockedByUser(false);
		setAutoExecStopVisible(false);
		setActiveAutoExecBatchCount(0);
	}, []);

	const requestBlockNextAutoExec = useCallback(() => {
		autoExecStopRequestedRef.current = true;
		autoExecBlockedByUserRef.current = true;
		setAutoExecStopRequested(true);
		setAutoExecBlockedByUser(true);
		setAutoExecStopVisible(true);
		resetAutoSubmitTracker();
	}, [resetAutoSubmitTracker]);

	const {
		toolCalls,
		toolOutputs,
		setToolOutputs,
		conversationToolsState,
		setConversationToolsState,
		webSearchTemplates,
		setWebSearchTemplates,
		toolArgsBlocked,
		hasPendingToolCalls,
		hasRunningToolCalls,
		runAllPendingToolCalls,
		handleRunSingleToolCall,
		handleDiscardToolCall,
		handleRemoveToolOutput: removeToolOutput,
		handleRetryErroredOutput,
		handleAttachedToolsChanged,
		applyConversationToolsFromChoices,
		applyWebSearchFromChoices,
		loadToolCalls,
		clearComposerToolsState,
		getToolRuntimeSnapshot,
		autoExecState,
	} = useComposerTools({
		isBusy: isGenerating,
		isSubmitting,
		ensureSkillSession,
		listActiveSkillRefs,
		setActiveSkillRefs,
		getCurrentSkillSessionID,
		getAttachedToolEntries: getAttachedToolEntriesSnapshot,
		externalExecutionBlocked: fastForwardPending || autoExecBlockedByUser,
		requestMCPApproval: mcpApproval.requestMCPApproval,
	});

	const previousProviderSDKTypeRef = useRef(currentProviderSDKType);
	const hasBlockingMCPArgs = mcp.argumentsBlocked;
	const hasBlockingToolArgs = toolArgsBlocked || webSearchArgsBlocked || hasBlockingMCPArgs;

	useLayoutEffect(() => {
		if (previousProviderSDKTypeRef.current === currentProviderSDKType) return;

		previousProviderSDKTypeRef.current = currentProviderSDKType;

		// Web-search tools are SDK-bound. Clear stale selections immediately
		// when switching SDK families so incompatible choices cannot linger
		// in UI state or slip into the next submit.
		setWebSearchTemplates([]);
		setToolArgsTarget(prev => (prev?.kind === 'webSearch' ? null : prev));
	}, [currentProviderSDKType, setToolArgsTarget, setWebSearchTemplates]);

	const handleOpenToolOutput = useCallback((output: UIToolOutput) => {
		setToolDetailsState({ kind: 'output', output });
	}, []);

	const handleOpenToolCallDetails = useCallback((call: UIToolCall) => {
		setToolDetailsState({ kind: 'call', call });
	}, []);

	const handleOpenConversationToolDetails = useCallback((entry: ConversationToolStateEntry) => {
		setToolDetailsState({ kind: 'choice', choice: entry.toolStoreChoice });
	}, []);

	const handleOpenAttachedToolDetails = useCallback((entry: AttachedToolEntry) => {
		const choice: ToolStoreChoice = {
			choiceID: entry.choiceID,
			bundleID: entry.bundleID,
			bundleSlug: entry.bundleSlug,
			toolSlug: entry.toolSlug,
			toolVersion: entry.toolVersion,
			displayName: entry.overrides?.displayName ?? entry.toolSnapshot?.displayName ?? entry.toolSlug,
			description: entry.overrides?.description ?? entry.toolSnapshot?.description ?? entry.toolSlug,
			toolID: entry.toolSnapshot?.id,
			toolType: entry.toolType,
			autoExecute: entry.autoExecute,
			userArgSchemaInstance: entry.userArgSchemaInstance,
		};

		setToolDetailsState({ kind: 'choice', choice });
	}, []);

	const handleRemoveToolOutput = useCallback(
		(id: string) => {
			removeToolOutput(id);
			setToolDetailsState(current =>
				current && current.kind === 'output' && current.output.id === id ? null : current
			);
		},
		[removeToolOutput]
	);

	const attachedToolIdentityKeys = useMemo(() => {
		return new Set(
			attachedToolEntries.map(entry =>
				toolIdentityKey(entry.bundleID, entry.bundleSlug, entry.toolSlug, entry.toolVersion)
			)
		);
	}, [attachedToolEntries]);

	useEffect(() => {
		return () => {
			if (externalMessageLoadReleaseTimerRef.current !== null) {
				window.clearTimeout(externalMessageLoadReleaseTimerRef.current);
				externalMessageLoadReleaseTimerRef.current = null;
			}
		};
	}, []);

	useEffect(() => {
		return () => {
			if (submitVisualFrameRef.current !== null) {
				window.cancelAnimationFrame(submitVisualFrameRef.current);
				submitVisualFrameRef.current = null;
			}
		};
	}, []);

	const handleInsertTemplate = useCallback(
		(args: { bundleID: string; templateSlug: string; templateVersion: string; template?: PromptTemplate }) => {
			insertTemplateSelectionNode(editor, args.bundleID, args.templateSlug, args.templateVersion, args.template);
		},
		[editor]
	);

	const handleAttachTool = useCallback(
		(item: ToolListItem, autoExecute: boolean) => {
			const identityKey = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
			if (attachedToolIdentityKeys.has(identityKey)) return;

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
		[attachedToolIdentityKeys, editor, handleAttachedToolsChanged]
	);

	const handleDetachAttachedToolByKey = useCallback(
		(identityKey: string) => {
			if (!attachedToolIdentityKeys.has(identityKey)) return;

			removeToolByKey(editor, identityKey);
			handleAttachedToolsChanged();
		},
		[attachedToolIdentityKeys, editor, handleAttachedToolsChanged]
	);

	const handleSetAttachedToolAutoExecuteByKey = useCallback(
		(identityKey: string, autoExecute: boolean) => {
			if (!attachedToolIdentityKeys.has(identityKey)) return;

			setToolAutoExecuteByKey(editor, identityKey, autoExecute);
			handleAttachedToolsChanged();
		},
		[attachedToolIdentityKeys, editor, handleAttachedToolsChanged]
	);

	const handleToggleAttachedToolAutoExecute = useCallback(
		(entry: AttachedToolEntry, autoExecute: boolean) => {
			const identityKey = toolIdentityKey(entry.bundleID, entry.bundleSlug, entry.toolSlug, entry.toolVersion);
			handleSetAttachedToolAutoExecuteByKey(identityKey, autoExecute);
		},
		[handleSetAttachedToolAutoExecuteByKey]
	);

	const handleRemoveAttachedTool = useCallback(
		(entry: AttachedToolEntry) => {
			const identityKey = toolIdentityKey(entry.bundleID, entry.bundleSlug, entry.toolSlug, entry.toolVersion);
			handleDetachAttachedToolByKey(identityKey);
		},
		[handleDetachAttachedToolByKey]
	);

	const handleRemoveAllAttachedTools = useCallback(
		(entries: AttachedToolEntry[]) => {
			if (entries.length === 0) return;

			const uniqueKeys = new Set<string>();

			for (const entry of entries) {
				const identityKey = toolIdentityKey(entry.bundleID, entry.bundleSlug, entry.toolSlug, entry.toolVersion);
				if (!attachedToolIdentityKeys.has(identityKey) || uniqueKeys.has(identityKey)) continue;
				uniqueKeys.add(identityKey);
			}

			if (uniqueKeys.size === 0) return;

			for (const identityKey of uniqueKeys) {
				removeToolByKey(editor, identityKey);
			}

			handleAttachedToolsChanged();
		},
		[attachedToolIdentityKeys, editor, handleAttachedToolsChanged]
	);

	const handleEditAttachedToolOptions = useCallback(
		(entry: AttachedToolEntry) => {
			dispatchOpenToolArgs({ kind: 'attached', selectionID: entry.selectionID }, toolArgsEventTarget);
		},
		[toolArgsEventTarget]
	);

	const handleSetAttachedToolUserArgSchemaInstance = useCallback(
		(selectionID: string, newInstance: string) => {
			setAttachedToolUserArgSchemaInstanceBySelectionID(editor, selectionID, newInstance);
		},
		[editor]
	);

	// When editing an earlier message we temporarily override the current
	// conversation-tool + web-search config. Keep a snapshot so Cancel restores it.
	const preEditConversationToolsRef = useRef<ConversationToolStateEntry[] | null>(null);
	const preEditWebSearchTemplatesRef = useRef<WebSearchChoiceTemplate[] | null>(null);
	const preEditMCPContextRef = useRef<MCPConversationContext | undefined | null>(null);
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
		preEditMCPContextRef.current = null;
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
			if (suppressNextAttachmentMenuFocusRestoreRef.current) {
				suppressNextAttachmentMenuFocusRestoreRef.current = false;
				menuOpenedByShortcutRef.current.attachments = false;
				return;
			}
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
		if (isInputLocked) return;
		menuOpenedByShortcutRef.current.templates = true;

		closeAllMenus();
		templateMenu.show();
		// Make Ariakit's "return focus" behavior deterministic on close.
		templateButtonRef.current?.focus({ preventScroll: true });
	}, [closeAllMenus, isInputLocked, templateMenu]);

	const openToolPicker = useCallback(() => {
		if (isInputLocked) return;
		menuOpenedByShortcutRef.current.tools = true;

		closeAllMenus();
		toolMenu.show();
		toolButtonRef.current?.focus({ preventScroll: true });
	}, [closeAllMenus, isInputLocked, toolMenu]);

	const openAttachmentPicker = useCallback(() => {
		if (isInputLocked) return;
		menuOpenedByShortcutRef.current.attachments = true;

		closeAllMenus();
		attachmentMenu.show();
		attachmentButtonRef.current?.focus({ preventScroll: true });
	}, [isInputLocked, closeAllMenus, attachmentMenu]);

	const restorePreEditContext = useCallback(() => {
		const prevConv = preEditConversationToolsRef.current;
		const prevWs = preEditWebSearchTemplatesRef.current;
		const prevMCP = preEditMCPContextRef.current;
		const prevSkills = preEditEnabledSkillRefsRef.current;
		const prevActive = preEditActiveSkillRefsRef.current;

		if (prevConv) setConversationToolsState(prevConv);
		if (prevWs) setWebSearchTemplates(prevWs);
		if (prevMCP !== null) mcp.restoreContext(prevMCP ?? undefined);
		if (prevSkills || prevActive) {
			applySkillSelectionState(prevSkills ?? getCurrentEnabledSkillRefs(), prevActive ?? getCurrentActiveSkillRefs());
		}
		clearPreEditSnapshot();
	}, [
		applySkillSelectionState,
		clearPreEditSnapshot,
		getCurrentActiveSkillRefs,
		getCurrentEnabledSkillRefs,
		mcp,
		setConversationToolsState,
		setWebSearchTemplates,
	]);

	const isSendButtonEnabled = useMemo(() => {
		// Cannot block on tools def loading here.
		if (isInputLocked) return false;
		if (isSubmitting) return false;
		if (templateBlocked) return false;
		if (hasBlockingToolArgs) return false;

		if (hasEffectiveTextForSubmit) return true;

		const hasAttachments = attachments.length > 0;
		const hasOutputs = toolOutputs.length > 0;

		return hasAttachments || hasOutputs;
	}, [
		isInputLocked,
		isSubmitting,
		templateBlocked,
		hasBlockingToolArgs,
		hasEffectiveTextForSubmit,
		attachments.length,
		toolOutputs.length,
	]);

	const { formRef, onKeyDown } = useEnterSubmit({
		isBusy: isGenerating || isSubmitting || fastForwardPending,
		canSubmit: () => {
			if (isInputLocked) return false;
			if (isSubmitting) return false;
			if (hasBlockingToolArgs) return false;
			if (templateBlocked) return false;

			// Default Enter behavior is "run pending tools, then send", so allow
			// submission when pending tools exist even if they are currently the
			// only content source.
			if (hasPendingToolCalls && !hasRunningToolCalls) {
				return true;
			}

			if (hasEffectiveTextForSubmit) return true;

			const hasAttachments = attachments.length > 0;
			const hasOutputs = toolOutputs.length > 0;

			return hasAttachments || hasOutputs;
		},
		insertSoftBreak: () => {
			editor.tf.insertSoftBreak();
		},
	});

	const setSubmitting = useCallback((next: boolean, options?: { deferVisual?: boolean }) => {
		isSubmittingRef.current = next;
		if (submitVisualFrameRef.current !== null) {
			window.cancelAnimationFrame(submitVisualFrameRef.current);
			submitVisualFrameRef.current = null;
		}

		// Important: submitting=true is only a visual/UI state. Triggering that
		// rerender synchronously right after an imperative Plate mutation is the
		// fragile part in this repro. Keep the ref in sync immediately for logic,
		// but defer the React render by one frame.
		if (next && options?.deferVisual) {
			submitVisualFrameRef.current = window.requestAnimationFrame(() => {
				submitVisualFrameRef.current = null;
				setIsSubmitting(current => (current === next ? current : next));
			});
			return;
		}

		setIsSubmitting(current => (current === next ? current : next));
	}, []);

	const clearComposerTransientState = useCallback(() => {
		closeAllMenus();
		setSubmitError(null);
		setSubmitting(false);
		setFastForwardPending(false);
		setToolDetailsState(null);
		setToolArgsTarget(null);
		resetAutoSubmitTracker();
		clearAutoExecStopState();
		clearAttachments();
		clearComposerToolsState();
		setMCPAppContextUpdates([]);
	}, [
		clearAttachments,
		clearAutoExecStopState,
		clearComposerToolsState,
		closeAllMenus,
		resetAutoSubmitTracker,
		setSubmitting,
	]);

	const startFastForwardRun = useCallback(
		(pendingRunnableToolCallIDs: string[]) => {
			if (pendingRunnableToolCallIDs.length === 0) return;

			setSubmitError(null);
			setFastForwardPending(true);

			void (async () => {
				try {
					for (const id of pendingRunnableToolCallIDs) {
						await handleRunSingleToolCall(id);
					}
				} catch (err) {
					setFastForwardPending(false);
					setSubmitError((err as Error)?.message || 'Failed to run pending tool calls.');
				}
			})();
		},
		[handleRunSingleToolCall]
	);

	const resetEditor = useCallback(() => {
		clearComposerTransientState();
		resetEditorDocument();
	}, [clearComposerTransientState, resetEditorDocument]);

	/**
	 * Main submit logic, parameterized by whether to run pending tool calls
	 * before sending.
	 */
	const doSubmit = useCallback(
		async (options: SubmitOptions) => {
			const { runPendingTools } = options;

			if (isSubmittingRef.current) return;
			if (isInputLocked) return;

			// If conversation sync queued enabled/active skill refs via timeout,
			// flush them now so this submit uses the latest skill selection.
			const effectiveEnabledSkillRefs = getCurrentEnabledSkillRefs();
			let activeForMessage = getCurrentActiveSkillRefs();

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
			if (hasBlockingToolArgs) {
				setSubmitError(
					hasBlockingMCPArgs
						? 'Some MCP prompts or resource templates require arguments. Fill the required arguments before sending.'
						: 'Some tools or web-search options require configuration. Fill the required options before sending.'
				);
				return;
			}

			// The tool picker can remain mounted after attaching a tool.
			// Close all menus before submit-driven state changes so no Ariakit menu
			// subtree spans the attached-tool -> conversation-tool transition.
			closeAllMenus();

			setSubmitError(null);
			const pendingRunnableToolCallIDs = runPendingTools
				? toolCalls
						.filter(
							toolCall =>
								toolCall.status === 'pending' &&
								(toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom)
						)
						.map(toolCall => toolCall.id)
				: [];
			const hadPendingTools = pendingRunnableToolCallIDs.length > 0;

			if (hadPendingTools) {
				startFastForwardRun(pendingRunnableToolCallIDs);
				return;
			}

			setSubmitting(true);

			let didSend = false;
			let submittedToolChoices: ToolStoreChoice[] | null = null;
			let shouldShowAutoExecStopAfterSend = false;

			try {
				let effectiveSkillSessionID = getCurrentSkillSessionID();
				if (!effectiveSkillSessionID && effectiveEnabledSkillRefs.length > 0) {
					try {
						effectiveSkillSessionID = await ensureSkillSession();
					} catch (err) {
						setSubmitError((err as Error)?.message || 'Failed to create skills session.');
						return;
					}
				}

				// Build final message content after tools have run.
				// Always read from the live tool runtime snapshot here. Auto-execute
				// can complete and trigger submit before React re-renders the latest
				// toolOutputs / hasPendingToolCalls values into this component.
				const runtimeAfterRun = getToolRuntimeSnapshot();
				const selections = getTemplateSelections(editor);
				const hasTpl = selections.length > 0;
				const currentTemplateSystemPrompt = appendSystemPromptParts(
					'',
					getInstructionPromptPartsFromSelections(selections)
				);
				const resolvedSystemPrompt = systemPrompt.resolvedSystemPrompt.trim() || undefined;

				const textToSend = hasTpl ? toPlainTextReplacingVariables(editor) : editor.api.string([]);
				const finalToolOutputs: UIToolOutput[] = runtimeAfterRun.toolOutputs;
				const unfinishedRunnableToolCalls = runtimeAfterRun.toolCalls.filter(
					toolCall =>
						(toolCall.status === 'pending' || toolCall.status === 'running') &&
						(toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom)
				);
				const failedRunnableToolCalls = runtimeAfterRun.toolCalls.filter(
					toolCall =>
						toolCall.status === 'failed' &&
						(toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom)
				);

				if (unfinishedRunnableToolCalls.length > 0) {
					// Fast-forward should be silent while tools are still progressing.
					// The chips already show running state, so avoid flashing an inline
					// composer alert for this transient condition.
					if (!hadPendingTools) {
						setSubmitError('Waiting for all tool calls to finish before sending.');
					}
					return;
				}
				if (failedRunnableToolCalls.length > 0) {
					setSubmitError('Some tool calls failed. Retry or discard them before sending.');
					return;
				}

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
				if (effectiveSkillSessionID && effectiveEnabledSkillRefs.length > 0) {
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
				const explicitChoices = attachedTools.map(uiToolChoiceToToolStoreChoice);
				const conversationChoices = conversationToolsToChoices(conversationToolsState);
				const webSearchChoices = buildWebSearchChoicesForSubmit(webSearchTemplates);

				const finalToolChoices = dedupeToolChoices([...explicitChoices, ...conversationChoices, ...webSearchChoices]);
				shouldShowAutoExecStopAfterSend = countAutoExecutableToolChoices(finalToolChoices) >= 1;

				let preparedMCPContext;
				try {
					preparedMCPContext = await mcp.prepareForSubmit();
				} catch (err) {
					setSubmitError((err as Error)?.message || 'Fill required MCP arguments before sending.');
					return;
				}
				const payload: EditorSubmitPayload = {
					text: textToSend,
					resolvedSystemPrompt,
					templateSystemPrompt: currentTemplateSystemPrompt.trim() || undefined,
					attachedTools,
					attachments,
					toolOutputs: finalToolOutputs,
					finalToolChoices,
					mcpContext: preparedMCPContext,
					mcpAppContextUpdates:
						mcpAppContextUpdates.length > 0 ? mcpAppContextUpdates.map(update => ({ ...update })) : undefined,
					enabledSkillRefs: effectiveEnabledSkillRefs,
					activeSkillRefs: activeForMessage,
					skillSessionID: effectiveSkillSessionID ?? undefined,
				};

				await onSubmit(payload);
				setSubmitError(null);
				submittedToolChoices = finalToolChoices;

				didSend = true;
			} finally {
				setSubmitting(false);

				// Only clear the editor if we actually sent something.
				if (didSend) {
					resetEditor();
					if (shouldShowAutoExecStopAfterSend) {
						autoExecStopRequestedRef.current = false;
						autoExecBlockedByUserRef.current = false;
						setAutoExecStopRequested(false);
						setAutoExecBlockedByUser(false);
						setAutoExecStopVisible(true);
					}

					// If we were editing, the old snapshot is no longer relevant.
					clearPreEditSnapshot();

					if (submittedToolChoices && submittedToolChoices.length > 0) {
						setConversationToolsState(prev =>
							mergeConversationToolsWithNewChoices(prev, submittedToolChoices as ToolStoreChoice[])
						);
					}
				}
			}
		},
		[
			attachments,
			clearPreEditSnapshot,
			closeAllMenus,
			contentRef,
			conversationToolsState,
			editor,
			ensureSkillSession,
			focusEditorAtEnd,
			getCurrentActiveSkillRefs,
			getCurrentEnabledSkillRefs,
			getCurrentSkillSessionID,
			getToolRuntimeSnapshot,
			hasBlockingMCPArgs,
			hasBlockingToolArgs,
			isInputLocked,
			isSendButtonEnabled,
			listActiveSkillRefs,
			mcp,
			mcpAppContextUpdates,
			onSubmit,
			resetEditor,
			selectionInfo.firstPendingVar,
			setActiveSkillRefs,
			setConversationToolsState,
			setSubmitting,
			startFastForwardRun,
			systemPrompt.resolvedSystemPrompt,
			templateBlocked,
			toolCalls,
			webSearchTemplates,
		]
	);

	/**
	 * Default form submit / Enter: "run pending tools, then send".
	 */
	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		if (e) e.preventDefault();
		void doSubmit({ runPendingTools: true });
	};

	useEffect(() => {
		if (!fastForwardPending) return;
		if (isGenerating || isInputLocked || isSubmittingRef.current) return;

		if (templateBlocked || hasBlockingToolArgs) return;

		const unfinishedRunnableToolCalls = toolCalls.filter(
			toolCall =>
				(toolCall.status === 'pending' || toolCall.status === 'running') &&
				(toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom)
		);

		if (unfinishedRunnableToolCalls.length > 0) return;

		const failedRunnableToolCalls = toolCalls.filter(
			toolCall =>
				toolCall.status === 'failed' &&
				(toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom)
		);

		if (failedRunnableToolCalls.length > 0) {
			// eslint-disable-next-line react-you-might-not-need-an-effect/no-chain-state-updates
			setFastForwardPending(false);
			// eslint-disable-next-line react-you-might-not-need-an-effect/no-chain-state-updates
			setSubmitError('Some tool calls failed. Retry or discard them before sending.');
			return;
		}

		if (!hasEffectiveTextForSubmit && attachments.length === 0 && toolOutputs.length === 0) {
			// eslint-disable-next-line react-you-might-not-need-an-effect/no-chain-state-updates
			setFastForwardPending(false);
			// eslint-disable-next-line react-you-might-not-need-an-effect/no-chain-state-updates
			setSubmitError('Tool calls did not produce any outputs, so there is nothing to send yet.');
			return;
		}

		// eslint-disable-next-line react-you-might-not-need-an-effect/no-chain-state-updates
		setFastForwardPending(false);
		void doSubmit({ runPendingTools: false });
	}, [
		attachments.length,
		doSubmit,
		fastForwardPending,
		hasBlockingToolArgs,
		hasEffectiveTextForSubmit,
		isGenerating,
		isInputLocked,
		templateBlocked,
		toolCalls,
		toolOutputs.length,
	]);

	useEffect(() => {
		const tracker = autoSubmitTrackerRef.current;

		if (toolCalls.length > 0) {
			for (const toolCall of toolCalls) {
				tracker.observedCallKeys.add(getToolAutoSubmitKey(toolCall));
				if (!isAutoSubmitEligibleToolCall(toolCall)) {
					tracker.allObservedCallsAreAutoExecute = false;
				}
			}
			return;
		}

		if (autoExecState.phase !== 'idle') return;
		if (tracker.observedCallKeys.size === 0) return;
		if (!tracker.allObservedCallsAreAutoExecute) return;
		if (autoExecBlockedByUser) return;
		if (isGenerating || isInputLocked || isSubmittingRef.current) return;

		if (templateBlocked || hasBlockingToolArgs) return;

		const runtime = getToolRuntimeSnapshot();

		if (runtime.toolCalls.length === 0 && runtime.toolOutputs.length === 0) {
			resetAutoSubmitTracker();
			return;
		}

		if (runtime.toolCalls.length > 0) return;

		const observedCallKeys = [...tracker.observedCallKeys].sort();
		const batchSignature = observedCallKeys.join('::');
		if (!batchSignature) return;
		if (tracker.attemptedBatchSignature === batchSignature) return;

		const outputByCallKey = new Map(runtime.toolOutputs.map(output => [getToolAutoSubmitKey(output), output] as const));

		const allObservedCallsProducedSuccessfulOutputs = observedCallKeys.every(callKey => {
			const output = outputByCallKey.get(callKey);
			return output && !output.isError;
		});

		if (!allObservedCallsProducedSuccessfulOutputs) return;

		tracker.attemptedBatchSignature = batchSignature;
		void doSubmit({ runPendingTools: false });
	}, [
		autoExecBlockedByUser,
		autoExecState.phase,
		doSubmit,
		getToolRuntimeSnapshot,
		hasBlockingToolArgs,
		isGenerating,
		isInputLocked,
		resetAutoSubmitTracker,
		templateBlocked,
		toolCalls,
	]);

	const loadExternalMessage = useCallback(
		(incoming: EditorExternalMessage) => {
			if (externalMessageLoadReleaseTimerRef.current !== null) {
				window.clearTimeout(externalMessageLoadReleaseTimerRef.current);
				externalMessageLoadReleaseTimerRef.current = null;
			}
			isLoadingExternalMessageRef.current = true;
			try {
				// Snapshot current context so Cancel Editing can restore it.
				if (!preEditConversationToolsRef.current) {
					preEditConversationToolsRef.current = conversationToolsState;
				}
				if (!preEditWebSearchTemplatesRef.current) {
					preEditWebSearchTemplatesRef.current = webSearchTemplates;
				}
				if (preEditMCPContextRef.current === null) {
					preEditMCPContextRef.current = mcp.mcpContext;
				}
				if (!preEditEnabledSkillRefsRef.current) {
					preEditEnabledSkillRefsRef.current = getCurrentEnabledSkillRefs();
				}
				if (!preEditActiveSkillRefsRef.current) {
					preEditActiveSkillRefsRef.current = getCurrentActiveSkillRefs();
				}
				clearComposerTransientState();

				// 1) Reset document to plain text paragraphs.
				const plain = incoming.text ?? '';
				const value = buildEditorValueFromPlainText(plain);
				replaceEditorDocument(value, 'end');

				// 2) Rebuild flat attachment chips from incoming attachment state.
				loadAttachmentsFromMessage(incoming.attachments);

				// 3) Restore tool choices into conversation-level state.
				const incomingToolChoices = incoming.toolChoices ?? [];
				applyConversationToolsFromChoices(incomingToolChoices);
				applyWebSearchFromChoices(incomingToolChoices);
				mcp.restoreContext(incoming.mcpContext);
				setMCPAppContextUpdates(incoming.mcpAppContextUpdates ?? []);
				// 4) Restore enabled/active skills together so invariants hold immediately.
				void applySkillSelectionState(incoming.enabledSkillRefs ?? [], incoming.activeSkillRefs ?? [], {
					syncSession: 'none',
					forceResetSession: true,
				});
				// 5) Restore any tool outputs that were previously attached to this message.
				setToolOutputs(incoming.toolOutputs ?? []);
			} finally {
				externalMessageLoadReleaseTimerRef.current = window.setTimeout(() => {
					externalMessageLoadReleaseTimerRef.current = null;
					isLoadingExternalMessageRef.current = false;
				}, 0);
			}
		},
		[
			applyConversationToolsFromChoices,
			applySkillSelectionState,
			applyWebSearchFromChoices,
			clearComposerTransientState,
			conversationToolsState,
			getCurrentActiveSkillRefs,
			getCurrentEnabledSkillRefs,
			loadAttachmentsFromMessage,
			mcp,
			replaceEditorDocument,
			setToolOutputs,
			webSearchTemplates,
		]
	);

	const setDraftText = useCallback(
		(text: string) => {
			closeAllMenus();
			setSubmitError(null);
			resetAutoSubmitTracker();
			replaceEditorDocument(buildEditorValueFromPlainText(text), 'end');
			window.requestAnimationFrame(() => {
				focusEditorAtEnd();
			});
		},
		[closeAllMenus, focusEditorAtEnd, replaceEditorDocument, resetAutoSubmitTracker]
	);

	const setDraftTextIfEmpty = useCallback(
		(text: string) => {
			if (text.trim().length === 0) {
				return false;
			}

			const hasCurrentText = hasNonEmptyUserText(editor) || editor.api.string([]).trim().length > 0;
			if (hasCurrentText) {
				return false;
			}

			setDraftText(text);
			return true;
		},
		[editor, setDraftText]
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
		// Skip auto-cancel while loadExternalMessage is in progress.
		// The transient state (cleared attachments / tool outputs) is not yet restored.
		if (isLoadingExternalMessageRef.current) return;

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
		attachments.length,
		cancelEditing,
		editingMessageId,
		editor,
		handleAttachedToolsChanged,
		hasTextRef,
		onEditorChange,
		restorePreEditContext,
		submitError,
		toolCalls.length,
		toolOutputs.length,
	]);

	const handleLoadToolCalls = useCallback(
		(toolCallsToLoad: UIToolCall[]) => {
			resetAutoSubmitTracker();
			const autoEligibleCount = toolCallsToLoad.filter(isAutoSubmitEligibleToolCall).length;
			const shouldSuppressAutoExec = autoExecStopRequestedRef.current || autoExecBlockedByUserRef.current;

			const preparedToolCalls = shouldSuppressAutoExec
				? toolCallsToLoad.map(toolCall =>
						isAutoSubmitEligibleToolCall(toolCall) ? { ...toolCall, suppressAutoExecute: true } : toolCall
					)
				: toolCallsToLoad;

			setActiveAutoExecBatchCount(autoEligibleCount >= 1 ? autoEligibleCount : 0);

			if (shouldSuppressAutoExec || autoEligibleCount === 0) {
				autoExecStopRequestedRef.current = false;
				autoExecBlockedByUserRef.current = false;
				setAutoExecStopRequested(false);
				setAutoExecBlockedByUser(false);
				setAutoExecStopVisible(false);
			}

			loadToolCalls(preparedToolCalls);
		},
		[loadToolCalls, resetAutoSubmitTracker]
	);

	const finishAssistantTurn = useCallback(
		(payload: AssistantTurnFinishedPayload) => {
			if (payload.loadedRunnableToolCallCount === 0) clearAutoExecStopState();
		},
		[clearAutoExecStopState]
	);

	useImperativeHandle(
		ref,
		() => ({
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
			setDraftText,
			setDraftTextIfEmpty,
			resetEditor,
			loadToolCalls: handleLoadToolCalls,
			setConversationToolsFromChoices: applyConversationToolsFromChoices,
			setWebSearchFromChoices: applyWebSearchFromChoices,
			setMCPContextFromMessage: context => {
				mcp.restoreContext(context);
			},
			setMCPAppContextUpdatesFromMessage: updates => {
				setMCPAppContextUpdates(updates ?? []);
			},
			appendMCPAppContextUpdate: update => {
				setMCPAppContextUpdates(prev => [...prev, update]);
			},
			clearMCPContext: () => {
				mcp.clear();
			},
			applyAttachmentsDrop,
			setSkillStateFromMessage: (enabledRefs, activeRefs, options) => {
				void applySkillSelectionState(enabledRefs, activeRefs, options);
			},
			finishAssistantTurn,
		}),
		[
			loadExternalMessage,
			setDraftText,
			setDraftTextIfEmpty,
			resetEditor,
			handleLoadToolCalls,
			applyConversationToolsFromChoices,
			applyWebSearchFromChoices,
			applyAttachmentsDrop,
			finishAssistantTurn,
			focusEditorAtEnd,
			openTemplatePicker,
			openToolPicker,
			openAttachmentPicker,
			mcp,
			applySkillSelectionState,
		]
	);

	const handleCancelEditing = useCallback(() => {
		resetEditor();
		restorePreEditContext();
		cancelEditing();
	}, [cancelEditing, resetEditor, restorePreEditContext]);

	const handleRunToolsOnlyClick = useCallback(async () => {
		if (!hasPendingToolCalls || isInputLocked || isSubmitting || fastForwardPending || hasRunningToolCalls) return;
		await runAllPendingToolCalls();
	}, [
		fastForwardPending,
		hasPendingToolCalls,
		hasRunningToolCalls,
		isInputLocked,
		isSubmitting,
		runAllPendingToolCalls,
	]);

	// Button-state helpers:
	// - Play: run tools only (enabled when there are pending tools and none are running).
	// - Fast-forward: run tools then send (enabled when there are pending tools and
	//   templates are satisfied; "sendability" will be re-checked after tools run).
	// - Send: send only (enabled when send is allowed and there are no pending tools).
	const canSendOnly = !hasPendingToolCalls && isSendButtonEnabled && !hasRunningToolCalls && !fastForwardPending;
	const canRunToolsOnly =
		hasPendingToolCalls && !hasRunningToolCalls && !isInputLocked && !isSubmitting && !fastForwardPending;
	const canRunToolsAndSend =
		hasPendingToolCalls &&
		!hasRunningToolCalls &&
		!isInputLocked &&
		!isSubmitting &&
		!fastForwardPending &&
		!templateBlocked &&
		!hasBlockingToolArgs;

	const showAutoExecStopButton = autoExecStopVisible || activeAutoExecBatchCount >= 2;
	const erroredToolOutputsReadyToSubmit =
		toolOutputs.some(output => output.isError) &&
		!hasPendingToolCalls &&
		!hasRunningToolCalls &&
		!hasBlockingToolArgs &&
		!templateBlocked &&
		!isInputLocked;

	useEffect(() => {
		if (toolCalls.length === 0 && autoExecState.phase === 'idle' && !isGenerating && activeAutoExecBatchCount > 0) {
			// eslint-disable-next-line react-you-might-not-need-an-effect/no-chain-state-updates
			setActiveAutoExecBatchCount(0);
		}
	}, [activeAutoExecBatchCount, autoExecState.phase, isGenerating, toolCalls.length]);

	useEffect(() => {
		onAssistantPresetRuntimeStateChange?.({
			conversationToolChoices: conversationToolsToChoices(conversationToolsState),
			webSearchChoices: mapAssistantPresetWebSearchTemplatesToChoices(webSearchTemplates),
			enabledSkillRefs,
		});
	}, [conversationToolsState, enabledSkillRefs, onAssistantPresetRuntimeStateChange, webSearchTemplates]);

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
				{!submitError && erroredToolOutputsReadyToSubmit ? (
					<div className="alert alert-warning mx-4 mt-3 mb-1 flex items-start gap-2 text-sm" role="status">
						<FiTool size={16} className="mt-0.5" />
						<span>
							A tool returned an error result. It is ready to submit as tool output, or you can retry/discard it.
						</span>
					</div>
				) : null}
				{mcpAppContextUpdates.length > 0 ? (
					<div className="alert alert-info mx-4 mt-3 mb-1 flex items-start justify-between gap-2 text-sm" role="status">
						<div className="flex items-start gap-2">
							<FiAlertTriangle size={16} className="mt-0.5" />
							<span>
								MCP App model context queued for the next send: {mcpAppContextUpdates.length} update
								{mcpAppContextUpdates.length === 1 ? '' : 's'}.
							</span>
						</div>
						<button
							type="button"
							className="btn btn-ghost btn-xs"
							onClick={() => {
								setMCPAppContextUpdates([]);
							}}
							aria-label="Clear MCP App model context"
						>
							<FiX size={14} />
						</button>
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
									autoCorrect={'off'}
									autoCapitalize={'off'}
									readOnly={isInputLocked}
									onKeyDown={e => {
										onKeyDown(e);
									}}
									onPaste={onEditorPaste}
									className="max-h-96 min-w-0 flex-1 resize-none overflow-auto bg-transparent p-1 wrap-break-word whitespace-break-spaces tab-2 outline-none focus:outline-none"
									style={{
										fontSize: 14,
										whiteSpace: 'break-spaces',
										tabSize: 2,
										minHeight: '4rem',
									}}
								/>
							</div>
							{/* Unified chips bar: attachments, directories, tools, tool calls & outputs (scrollable) */}
							<div className="w-full min-w-0 items-center overflow-x-auto overscroll-contain p-0 text-xs">
								<EditorChipsBar
									attachments={attachments}
									directoryGroups={directoryGroups}
									conversationTools={conversationToolsState}
									toolCalls={toolCalls}
									toolOutputs={toolOutputs}
									toolEntries={attachedToolEntries}
									mcpState={mcp}
									isBusy={isGenerating || isSubmitting || isInputLocked || fastForwardPending}
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
									onToggleAttachedToolAutoExecute={handleToggleAttachedToolAutoExecute}
									onRemoveAttachedTool={handleRemoveAttachedTool}
									onRemoveAllAttachedTools={handleRemoveAllAttachedTools}
									onEditAttachedToolOptions={handleEditAttachedToolOptions}
									onOpenToolCallDetails={handleOpenToolCallDetails}
									onOpenConversationToolDetails={handleOpenConversationToolDetails}
									onOpenAttachedToolDetails={handleOpenAttachedToolDetails}
									toolArgsEventTarget={toolArgsEventTarget}
								/>
							</div>
						</div>
						{/* Primary / secondary actions anchored at bottom-right */}
						<div className="flex flex-col items-end justify-end gap-2 p-1">
							{showAutoExecStopButton ? (
								<HoverTip
									content={
										autoExecStopRequested
											? 'Auto-exec stop armed for the next assistant tool-call batch'
											: 'Stop auto-exec for the next assistant tool-call batch'
									}
									placement="left"
								>
									<button
										type="button"
										className={`btn btn-circle btn-sm shrink-0 ${
											autoExecStopRequested || autoExecBlockedByUser ? 'btn-warning' : 'btn-neutral'
										}`}
										onClick={requestBlockNextAutoExec}
										aria-label="Stop auto-exec for next tool calls"
										aria-pressed={autoExecStopRequested || autoExecBlockedByUser}
									>
										<FiZapOff size={18} />
									</button>
								</HoverTip>
							) : null}
							{isGenerating ? (
								<HoverTip content="Stop response" placement="left">
									<button
										type="button"
										className="btn btn-circle btn-neutral btn-sm shrink-0"
										onClick={onRequestStop}
										aria-label="Stop response"
									>
										<FiSquare size={20} />
									</button>
								</HoverTip>
							) : (
								<>
									{/* Run tools only (Play) */}
									{hasPendingToolCalls && (
										<HoverTip content="Run tools only" placement="left">
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
										</HoverTip>
									)}

									{/* Run tools and send (Fast-forward) */}
									{hasPendingToolCalls && (
										<HoverTip content="Run tools and send" placement="left">
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
										</HoverTip>
									)}

									{/* Send only . Disabled while there are pending tools. */}
									<HoverTip
										content={hasPendingToolCalls ? 'Send (enabled after tools finish)' : 'Send message'}
										placement="left"
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
									</HoverTip>
								</>
							)}
						</div>
					</div>

					{/* Bottom bar for template/tool/attachment pickers + tips menus */}
					<EditorBottomBar
						onAttachFiles={handleAttachFiles}
						onAttachDirectory={handleAttachDirectory}
						onAttachURL={handleAttachURL}
						onOpenAttachmentUrlModal={() => {
							suppressNextAttachmentMenuFocusRestoreRef.current = true;
						}}
						onUrlAttachmentModalClose={() => {
							focusEditorAtEnd();
						}}
						onInsertTemplate={handleInsertTemplate}
						templateMenuState={templateMenu}
						toolMenuState={toolMenu}
						attachmentMenuState={attachmentMenu}
						templateButtonRef={templateButtonRef}
						toolButtonRef={toolButtonRef}
						attachmentButtonRef={attachmentButtonRef}
						toolArgsEventTarget={toolArgsEventTarget}
						shortcutConfig={shortcutConfig}
						currentProviderSDKType={currentProviderSDKType}
						attachedToolEntries={attachedToolEntries}
						onAttachTool={handleAttachTool}
						onDetachToolByKey={handleDetachAttachedToolByKey}
						onSetAttachedToolAutoExecute={handleSetAttachedToolAutoExecuteByKey}
						webSearchTemplates={webSearchTemplates}
						setWebSearchTemplates={setWebSearchTemplates}
						onWebSearchArgsBlockedChange={nextBlocked => {
							setWebSearchArgsBlocked(nextBlocked);
						}}
						allSkills={allSkills}
						skillsLoading={skillsLoading}
						enabledSkillRefs={enabledSkillRefs}
						activeSkillRefs={activeSkillRefs}
						setEnabledSkillRefs={setEnabledSkillRefs}
						onEnableAllSkills={enableAllSkills}
						onDisableAllSkills={disableAllSkills}
						isInputLocked={isInputLocked || fastForwardPending}
						systemPrompt={systemPrompt}
						mcpState={mcp}
					/>
				</Plate>
			</form>

			<ToolDetailsModal
				state={toolDetailsState}
				onClose={() => {
					setToolDetailsState(null);
				}}
			/>

			<MCPApprovalModal approvalRequest={mcpApproval.approvalRequest} onResolve={mcpApproval.resolveMCPApproval} />

			<ToolArgsModalHost
				attachedToolEntries={attachedToolEntries}
				setAttachedToolUserArgSchemaInstance={handleSetAttachedToolUserArgSchemaInstance}
				conversationToolsState={conversationToolsState}
				setConversationToolsState={setConversationToolsState}
				toolArgsTarget={toolArgsTarget}
				setToolArgsTarget={setToolArgsTarget}
				recomputeAttachedToolArgsBlocked={handleAttachedToolsChanged}
				webSearchTemplates={webSearchTemplates}
				setWebSearchTemplates={setWebSearchTemplates}
			/>
		</>
	);
});
