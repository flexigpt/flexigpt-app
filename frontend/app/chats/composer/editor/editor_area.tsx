import type { KeyboardEvent as ReactKeyboardEvent, SubmitEventHandler } from 'react';
import {
	forwardRef,
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
import type { MCPAppModelContextUpdate, MCPConversationContext, MCPToolSelection } from '@/spec/mcp';
import { MCPExecutionMode } from '@/spec/mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolArgsTarget, ToolListItem, ToolStoreChoice } from '@/spec/tool';
import { ToolStoreChoiceType } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { formatShortcut } from '@/lib/keyboard_shortcuts';

import { useEnterSubmit } from '@/hooks/use_enter_submit';

import { HoverTip } from '@/components/hover_tip';

import type { AssistantPresetRuntimeSnapshot } from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { mapAssistantPresetWebSearchTemplatesToChoices } from '@/chats/composer/assistantpresets/assistant_preset_runtime';
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
import { buildEditorValueFromPlainText, insertPlainTextAsSingleBlock } from '@/chats/composer/platedoc/platedoc_utils';
import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import {
	getAttachedTools,
	insertToolSelectionNode,
	removeToolByKey,
	setAttachedToolUserArgSchemaInstanceBySelectionID,
	setToolAutoExecuteByKey,
} from '@/chats/composer/platedoc/tool_document_ops';
import { useComposerDocument } from '@/chats/composer/platedoc/use_composer_document';
import { useComposerSkills } from '@/chats/composer/skills/use_composer_skills';
import type { ComposerSystemPromptController } from '@/chats/composer/skills/use_composer_system_prompt';
import {
	createAutoSubmitTracker,
	getToolAutoSubmitKey,
	isAutoSubmitEligibleToolCall,
} from '@/chats/composer/toolruntime/tool_runtime_utils';
import { useComposerTools } from '@/chats/composer/toolruntime/use_composer_tools';
import { dispatchOpenToolArgs, useOpenToolArgs } from '@/chats/composer/toolruntime/use_open_toolargs_event';
import type { ToolDetailsState } from '@/chats/composer/tools/tool_details_modal';
import { ToolDetailsModal } from '@/chats/composer/tools/tool_details_modal';
import { ToolArgsModalHost } from '@/chats/composer/tools/tool_user_args_host';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import { buildWebSearchChoicesForSubmit } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';
import { conversationToolsToChoices, mergeConversationToolsWithNewChoices } from '@/tools/lib/conversation_tool_utils';
import { isRunnableComposerToolCall } from '@/tools/lib/tool_call_utils';
import { dedupeToolChoices, uiToolChoiceToToolStoreChoice } from '@/tools/lib/tool_choice_utils';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

interface SkillStateApplyOptions {
	syncSession?: 'none' | 'if-session-exists' | 'ensure-if-enabled';
	forceResetSession?: boolean;
}

export interface EditorAreaHandle {
	focus: () => void;
	openTemplateMenu: () => void;
	openToolMenu: () => void;
	openAttachmentMenu: () => void;
	openSystemPromptMenu: () => void;
	openSkillsMenu: () => void;
	openMCPMenu: () => void;
	requestStopResponse: () => void;
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
	return choices.filter(c => isAutoExecutableToolChoice(c)).length;
}

function getMCPContextToolSelections(context?: MCPConversationContext): MCPToolSelection[] {
	const out: MCPToolSelection[] = [];

	for (const server of context?.servers ?? []) {
		for (const tool of server.selectedTools ?? []) {
			out.push({
				...tool,
				bundleID: tool.bundleID || server.bundleID,
				serverID: tool.serverID || server.serverID,
			});
		}
	}

	return out;
}

function countAutoExecutableMCPTools(context?: MCPConversationContext): number {
	return getMCPContextToolSelections(context).filter(
		tool => tool.executionMode === MCPExecutionMode.MCPExecutionModeAuto
	).length;
}

function mergeMCPToolSelection(selection: MCPToolSelection, contextSelection: MCPToolSelection): MCPToolSelection {
	return {
		...contextSelection,
		...selection,
		bundleID: selection.bundleID || contextSelection.bundleID,
		serverID: selection.serverID || contextSelection.serverID,
		toolName: selection.toolName || contextSelection.toolName,
		providerToolName: selection.providerToolName || contextSelection.providerToolName,
		choiceID: selection.choiceID || contextSelection.choiceID,
		digest: selection.digest || contextSelection.digest,
		approvalRule: selection.approvalRule ?? contextSelection.approvalRule,
		executionMode: selection.executionMode ?? contextSelection.executionMode,
		appResourceUri: selection.appResourceUri || contextSelection.appResourceUri,
		visibility: selection.visibility ?? contextSelection.visibility,
	};
}

function findMCPToolSelectionForCall(
	toolCall: UIToolCall,
	context?: MCPConversationContext
): MCPToolSelection | undefined {
	const contextSelections = getMCPContextToolSelections(context);
	if (contextSelections.length === 0) {
		return undefined;
	}

	const selection = toolCall.mcpToolSelection;

	if (selection?.choiceID) {
		const bySelectionChoiceID = contextSelections.find(candidate => candidate.choiceID === selection.choiceID);
		if (bySelectionChoiceID) {
			return bySelectionChoiceID;
		}
	}

	if (toolCall.choiceID) {
		const byCallChoiceID = contextSelections.find(candidate => candidate.choiceID === toolCall.choiceID);
		if (byCallChoiceID) {
			return byCallChoiceID;
		}
	}

	if (!selection) {
		return undefined;
	}

	return contextSelections.find(candidate => {
		const bundleMatches = !selection.bundleID || selection.bundleID === candidate.bundleID;
		const serverMatches = !selection.serverID || selection.serverID === candidate.serverID;
		if (!bundleMatches || !serverMatches) {
			return false;
		}

		if (selection.toolName && selection.toolName === candidate.toolName) {
			return true;
		}
		if (selection.providerToolName && selection.providerToolName === candidate.providerToolName) {
			return true;
		}
		if (toolCall.name && toolCall.name === candidate.providerToolName) {
			return true;
		}

		return false;
	});
}

function enrichMCPToolCallsFromContext(toolCalls: UIToolCall[], context?: MCPConversationContext): UIToolCall[] {
	let changed = false;

	const next = toolCalls.map(toolCall => {
		const contextSelection = findMCPToolSelectionForCall(toolCall, context);
		if (!contextSelection) {
			return toolCall;
		}

		const mcpToolSelection = toolCall.mcpToolSelection
			? mergeMCPToolSelection(toolCall.mcpToolSelection, contextSelection)
			: contextSelection;

		changed = true;
		return {
			...toolCall,
			mcpToolSelection,
		};
	});

	return changed ? next : toolCalls;
}

function focusMenuSearchOrFirstItem(menuElement: HTMLElement | null | undefined) {
	const searchInput = menuElement?.querySelector<HTMLInputElement>('[data-searchable-menu-input="true"]');
	if (searchInput) {
		searchInput.focus({ preventScroll: true });
		searchInput.select();
		return;
	}

	menuElement
		?.querySelector<HTMLElement>(
			'[data-searchable-menu-item="true"]:not([aria-disabled="true"]):not(:disabled), [role="menuitem"]'
		)
		?.focus();
}

interface SubmitOptions {
	runPendingTools: boolean;
}

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
	const lastSubmittedMCPContextRef = useRef<MCPConversationContext | undefined>(undefined);
	const mcpApproval = useMCPApproval();
	const resetAutoSubmitTracker = useCallback(() => {
		autoSubmitTrackerRef.current = createAutoSubmitTracker();
	}, []);

	const {
		editor,
		contentRef,
		hasText,
		hasTextRef,
		attachedToolEntries,
		getAttachedToolEntriesSnapshot,
		onEditorChange,
		onEditorPaste,
		scrollSelectionIntoEditorView,
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
	const skillsMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const mcpMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const templateButtonRef = useRef<HTMLButtonElement | null>(null);
	const toolButtonRef = useRef<HTMLButtonElement | null>(null);
	const attachmentButtonRef = useRef<HTMLButtonElement | null>(null);

	const toolArgsEventTarget = useMemo<EventTarget | null>(() => {
		return typeof EventTarget !== 'undefined' ? new EventTarget() : null;
	}, []);

	// Track whether a menu was opened via shortcut so we can:
	// - force focus into the menu (arrow-key nav)
	// - optionally restore focus to editor on close (Esc)
	const menuOpenedByShortcutRef = useRef({
		templates: false,
		tools: false,
		attachments: false,
		skills: false,
		mcp: false,
	});
	const suppressNextAttachmentMenuFocusRestoreRef = useRef(false);

	const {
		attachments,
		directoryGroups,
		attachFiles: handleAttachFiles,
		attachDirectory: handleAttachDirectory,
		attachURL: handleAttachURL,
		attachPathsAsAttachments,
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
		skillsLoadError,
		enabledSkillRefs,
		activeSkillRefs,
		setEnabledSkillRefs,
		setActiveSkillRefs,
		setActiveSkillRefsFromSession,
		enableAllSkills,
		disableAllSkills,
		refreshSkills,
		applySkillSelectionState,
		ensureSkillSession,
		listActiveSkillRefs,
		getCurrentSkillSessionID,
		getCurrentEnabledSkillRefs,
		getCurrentActiveSkillRefs,
	} = useComposerSkills();

	const [autoExecStopVisible, setAutoExecStopVisible] = useState(false);
	const [autoExecStopRequested, setAutoExecStopRequested] = useState(false);
	const [autoExecBlockedByUser, setAutoExecBlockedByUser] = useState(false);

	const autoExecStopRequestedRef = useRef(false);
	const autoExecBlockedByUserRef = useRef(false);

	const clearAutoExecStopState = useCallback(() => {
		autoExecStopRequestedRef.current = false;
		autoExecBlockedByUserRef.current = false;
		setAutoExecStopRequested(false);
		setAutoExecBlockedByUser(false);
		setAutoExecStopVisible(false);
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
		setActiveSkillRefsFromSession,
		getCurrentSkillSessionID,
		getAttachedToolEntries: getAttachedToolEntriesSnapshot,
		externalExecutionBlocked: fastForwardPending || autoExecBlockedByUser,
		requestMCPApproval: mcpApproval.requestMCPApproval,
	});

	const previousProviderSDKTypeRef = useRef(currentProviderSDKType);
	const hasBlockingMCPArgs = mcp.argumentsBlocked;
	const hasBlockingToolArgs = toolArgsBlocked || webSearchArgsBlocked || hasBlockingMCPArgs;

	useLayoutEffect(() => {
		if (previousProviderSDKTypeRef.current === currentProviderSDKType) {
			return;
		}

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

	const handleInsertTemplateText = useCallback(
		(text: string) => {
			insertPlainTextAsSingleBlock(editor, text);
			focusEditorAtEnd();
		},
		[editor, focusEditorAtEnd]
	);

	const handleAttachTool = useCallback(
		(item: ToolListItem, autoExecute: boolean) => {
			const identityKey = toolIdentityKey(item.bundleID, item.bundleSlug, item.toolSlug, item.toolVersion);
			if (attachedToolIdentityKeys.has(identityKey)) {
				return;
			}

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
			if (!attachedToolIdentityKeys.has(identityKey)) {
				return;
			}

			removeToolByKey(editor, identityKey);
			handleAttachedToolsChanged();
		},
		[attachedToolIdentityKeys, editor, handleAttachedToolsChanged]
	);

	const handleSetAttachedToolAutoExecuteByKey = useCallback(
		(identityKey: string, autoExecute: boolean) => {
			if (!attachedToolIdentityKeys.has(identityKey)) {
				return;
			}

			setToolAutoExecuteByKey(editor, identityKey, autoExecute);
			handleAttachedToolsChanged();
		},
		[attachedToolIdentityKeys, editor, handleAttachedToolsChanged]
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
			if (entries.length === 0) {
				return;
			}

			const uniqueKeys = new Set<string>();

			for (const entry of entries) {
				const identityKey = toolIdentityKey(entry.bundleID, entry.bundleSlug, entry.toolSlug, entry.toolVersion);
				if (!attachedToolIdentityKeys.has(identityKey) || uniqueKeys.has(identityKey)) {
					continue;
				}
				uniqueKeys.add(identityKey);
			}

			if (uniqueKeys.size === 0) {
				return;
			}

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
	const skillsMenuOpen = useStoreState(skillsMenu, 'open');
	const mcpMenuOpen = useStoreState(mcpMenu, 'open');
	const templateMenuEl = useStoreState(templateMenu, 'contentElement');
	const toolMenuEl = useStoreState(toolMenu, 'contentElement');
	const attachmentMenuEl = useStoreState(attachmentMenu, 'contentElement');
	const skillsMenuEl = useStoreState(skillsMenu, 'contentElement');
	const mcpMenuEl = useStoreState(mcpMenu, 'contentElement');

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
		skillsMenu.hide();
		mcpMenu.hide();
	}, [attachmentMenu, mcpMenu, skillsMenu, templateMenu, toolMenu]);

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
		if (!menuOpenedByShortcutRef.current.templates) {
			return;
		}

		requestAnimationFrame(() => {
			focusMenuSearchOrFirstItem(templateMenuEl);
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
		if (!menuOpenedByShortcutRef.current.tools) {
			return;
		}

		requestAnimationFrame(() => {
			focusMenuSearchOrFirstItem(toolMenuEl);
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
		if (!menuOpenedByShortcutRef.current.attachments) {
			return;
		}

		requestAnimationFrame(() => {
			focusMenuSearchOrFirstItem(attachmentMenuEl);
		});
	}, [attachmentMenuOpen, attachmentMenuEl, focusEditorPreservingSelection]);

	useEffect(() => {
		if (!skillsMenuOpen) {
			if (menuOpenedByShortcutRef.current.skills) {
				menuOpenedByShortcutRef.current.skills = false;
				requestAnimationFrame(() => {
					focusEditorPreservingSelection();
				});
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.skills) {
			return;
		}

		requestAnimationFrame(() => {
			focusMenuSearchOrFirstItem(skillsMenuEl);
		});
	}, [focusEditorPreservingSelection, skillsMenuEl, skillsMenuOpen]);

	useEffect(() => {
		if (!mcpMenuOpen) {
			if (menuOpenedByShortcutRef.current.mcp) {
				menuOpenedByShortcutRef.current.mcp = false;
				requestAnimationFrame(() => {
					focusEditorPreservingSelection();
				});
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.mcp) {
			return;
		}

		requestAnimationFrame(() => {
			focusMenuSearchOrFirstItem(mcpMenuEl);
		});
	}, [focusEditorPreservingSelection, mcpMenuEl, mcpMenuOpen]);

	const openTemplatePicker = useCallback(() => {
		if (isInputLocked) {
			return;
		}
		menuOpenedByShortcutRef.current.templates = true;

		closeAllMenus();
		templateMenu.show();
		// Make Ariakit's "return focus" behavior deterministic on close.
		templateButtonRef.current?.focus({ preventScroll: true });
	}, [closeAllMenus, isInputLocked, templateMenu]);

	const openToolPicker = useCallback(() => {
		if (isInputLocked) {
			return;
		}
		menuOpenedByShortcutRef.current.tools = true;

		closeAllMenus();
		toolMenu.show();
		toolButtonRef.current?.focus({ preventScroll: true });
	}, [closeAllMenus, isInputLocked, toolMenu]);

	const openAttachmentPicker = useCallback(() => {
		if (isInputLocked) {
			return;
		}
		menuOpenedByShortcutRef.current.attachments = true;

		closeAllMenus();
		attachmentMenu.show();
		attachmentButtonRef.current?.focus({ preventScroll: true });
	}, [isInputLocked, closeAllMenus, attachmentMenu]);

	const openSkillsPicker = useCallback(() => {
		if (isInputLocked) {
			return;
		}
		menuOpenedByShortcutRef.current.skills = true;

		closeAllMenus();
		skillsMenu.show();
	}, [closeAllMenus, isInputLocked, skillsMenu]);

	const openMCPPicker = useCallback(() => {
		if (isInputLocked) {
			return;
		}
		menuOpenedByShortcutRef.current.mcp = true;

		closeAllMenus();
		mcpMenu.show();
	}, [closeAllMenus, isInputLocked, mcpMenu]);

	const restorePreEditContext = useCallback(() => {
		const prevConv = preEditConversationToolsRef.current;
		const prevWs = preEditWebSearchTemplatesRef.current;
		const prevMCP = preEditMCPContextRef.current;
		const prevSkills = preEditEnabledSkillRefsRef.current;
		const prevActive = preEditActiveSkillRefsRef.current;

		if (prevConv) {
			setConversationToolsState(prevConv);
		}
		if (prevWs) {
			setWebSearchTemplates(prevWs);
		}
		if (prevMCP !== null) {
			mcp.restoreContext(prevMCP ?? undefined);
		}
		if (prevSkills || prevActive) {
			void applySkillSelectionState(
				prevSkills ?? getCurrentEnabledSkillRefs(),
				prevActive ?? getCurrentActiveSkillRefs()
			);
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
		if (isInputLocked) {
			return false;
		}
		if (isSubmitting) {
			return false;
		}
		if (hasBlockingToolArgs) {
			return false;
		}

		if (hasText) {
			return true;
		}

		const hasAttachments = attachments.length > 0;
		const hasOutputs = toolOutputs.length > 0;

		return hasAttachments || hasOutputs;
	}, [isInputLocked, isSubmitting, hasBlockingToolArgs, hasText, attachments.length, toolOutputs.length]);

	const { formRef, onKeyDown } = useEnterSubmit({
		isBusy: isGenerating || isSubmitting || fastForwardPending,
		canSubmit: () => {
			if (isInputLocked) {
				return false;
			}
			if (isSubmitting) {
				return false;
			}
			if (hasBlockingToolArgs) {
				return false;
			}

			// Default Enter behavior is "run pending tools, then send", so allow
			// submission when pending tools exist even if they are currently the
			// only content source.
			if (hasPendingToolCalls && !hasRunningToolCalls) {
				return true;
			}

			if (hasText) {
				return true;
			}

			const hasAttachments = attachments.length > 0;
			const hasOutputs = toolOutputs.length > 0;

			return hasAttachments || hasOutputs;
		},
		insertSoftBreak: () => {
			editor.tf.insertSoftBreak();
		},
	});

	const handleEditorKeyDown = useCallback(
		(event: ReactKeyboardEvent<HTMLDivElement>) => {
			onKeyDown(event);
			if (event.defaultPrevented) {
				return;
			}
			if (event.key !== 'PageUp' && event.key !== 'PageDown') {
				return;
			}
			const editorEl = contentRef.current;
			if (!editorEl) {
				return;
			}
			event.preventDefault();
			event.stopPropagation();
			const direction = event.key === 'PageDown' ? 1 : -1;
			const delta = Math.max(120, editorEl.clientHeight * 0.85) * direction;
			const maxTop = Math.max(0, editorEl.scrollHeight - editorEl.clientHeight);
			editorEl.scrollTop = Math.max(0, Math.min(maxTop, editorEl.scrollTop + delta));
		},
		[contentRef, onKeyDown]
	);

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
			if (pendingRunnableToolCallIDs.length === 0) {
				return;
			}

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

			if (isSubmittingRef.current) {
				return;
			}
			if (isInputLocked) {
				return;
			}

			// If conversation sync queued enabled/active skill refs via timeout,
			// flush them now so this submit uses the latest skill selection.
			let effectiveEnabledSkillRefs = getCurrentEnabledSkillRefs();

			// Pure send path: if we're *not* running tools, bail out when we
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
						.filter(toolCall => toolCall.status === 'pending' && isRunnableComposerToolCall(toolCall))
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

				// A selection update may invalidate an in-flight session creation.
				// Re-read the refs after awaiting and never send lifecycle selections
				// without the corresponding authoritative runtime session.
				effectiveEnabledSkillRefs = getCurrentEnabledSkillRefs();
				let activeForMessage = getCurrentActiveSkillRefs();
				if (!effectiveSkillSessionID && effectiveEnabledSkillRefs.length > 0) {
					setSubmitError(
						'Selected skills are still being synchronized. Wait briefly and send again so the skill session can be created.'
					);
					return;
				}

				// Build final message content after tools have run.
				// Always read from the live tool runtime snapshot here. Auto-execute
				// can complete and trigger submit before React re-renders the latest
				// toolOutputs / hasPendingToolCalls values into this component.
				const runtimeAfterRun = getToolRuntimeSnapshot();
				const resolvedSystemPrompt = systemPrompt.resolvedSystemPrompt.trim() || undefined;

				const textToSend = editor.api.string([]);
				const finalToolOutputs: UIToolOutput[] = runtimeAfterRun.toolOutputs;
				const unfinishedRunnableToolCalls = runtimeAfterRun.toolCalls.filter(
					toolCall =>
						(toolCall.status === 'pending' || toolCall.status === 'running') && isRunnableComposerToolCall(toolCall)
				);
				const failedRunnableToolCalls = runtimeAfterRun.toolCalls.filter(
					toolCall => toolCall.status === 'failed' && isRunnableComposerToolCall(toolCall)
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
				const explicitChoices = attachedTools.map(c => uiToolChoiceToToolStoreChoice(c));
				const conversationChoices = conversationToolsToChoices(conversationToolsState);
				const webSearchChoices = buildWebSearchChoicesForSubmit(webSearchTemplates);

				const finalToolChoices = dedupeToolChoices([...explicitChoices, ...conversationChoices, ...webSearchChoices]);

				let preparedMCPContext: MCPConversationContext | undefined;
				try {
					preparedMCPContext = await mcp.prepareForSubmit();
				} catch (err) {
					setSubmitError((err as Error)?.message || 'Fill required MCP arguments before sending.');
					return;
				}
				shouldShowAutoExecStopAfterSend =
					countAutoExecutableToolChoices(finalToolChoices) + countAutoExecutableMCPTools(preparedMCPContext) >= 1;

				const payload: EditorSubmitPayload = {
					text: textToSend,
					resolvedSystemPrompt,
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

				lastSubmittedMCPContextRef.current = preparedMCPContext;
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
			conversationToolsState,
			editor,
			ensureSkillSession,
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
			setActiveSkillRefs,
			setConversationToolsState,
			setSubmitting,
			startFastForwardRun,
			systemPrompt.resolvedSystemPrompt,
			toolCalls,
			webSearchTemplates,
		]
	);

	/**
	 * Default form submit / Enter: "run pending tools, then send".
	 */
	const handleSubmit: SubmitEventHandler<HTMLFormElement> = e => {
		if (e) {
			e.preventDefault();
		}
		void doSubmit({ runPendingTools: true });
	};

	const finishFastForwardWithError = useCallback((message: string) => {
		setFastForwardPending(false);
		setSubmitError(message);
	}, []);

	const finishFastForwardAndSubmit = useCallback(() => {
		setFastForwardPending(false);
		void doSubmit({
			runPendingTools: false,
		});
	}, [doSubmit]);

	useEffect(() => {
		if (!fastForwardPending) {
			return;
		}
		if (isGenerating || isInputLocked || isSubmittingRef.current) {
			return;
		}

		if (hasBlockingToolArgs) {
			return;
		}

		const unfinishedRunnableToolCalls = toolCalls.filter(
			toolCall =>
				(toolCall.status === 'pending' || toolCall.status === 'running') && isRunnableComposerToolCall(toolCall)
		);

		if (unfinishedRunnableToolCalls.length > 0) {
			return;
		}

		const failedRunnableToolCalls = toolCalls.filter(
			toolCall => toolCall.status === 'failed' && isRunnableComposerToolCall(toolCall)
		);

		if (failedRunnableToolCalls.length > 0) {
			finishFastForwardWithError('Some tool calls failed. Retry or discard them before sending.');
			return;
		}

		if (!hasText && attachments.length === 0 && toolOutputs.length === 0) {
			finishFastForwardWithError('Tool calls did not produce any outputs, so there is nothing to send yet.');
			return;
		}

		finishFastForwardAndSubmit();
	}, [
		attachments.length,
		doSubmit,
		fastForwardPending,
		finishFastForwardAndSubmit,
		finishFastForwardWithError,
		hasBlockingToolArgs,
		hasText,
		isGenerating,
		isInputLocked,
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

		if (autoExecState.phase !== 'idle') {
			return;
		}
		if (tracker.observedCallKeys.size === 0) {
			return;
		}
		if (!tracker.allObservedCallsAreAutoExecute) {
			return;
		}
		if (autoExecBlockedByUser) {
			return;
		}
		if (isGenerating || isInputLocked || isSubmittingRef.current) {
			return;
		}

		if (hasBlockingToolArgs) {
			return;
		}

		const runtime = getToolRuntimeSnapshot();

		if (runtime.toolCalls.length === 0 && runtime.toolOutputs.length === 0) {
			resetAutoSubmitTracker();
			return;
		}

		if (runtime.toolCalls.length > 0) {
			return;
		}

		const observedCallKeys = [...tracker.observedCallKeys].toSorted();
		const batchSignature = observedCallKeys.join('::');
		if (!batchSignature) {
			return;
		}
		if (tracker.attemptedBatchSignature === batchSignature) {
			return;
		}

		const outputByCallKey = new Map(runtime.toolOutputs.map(output => [getToolAutoSubmitKey(output), output] as const));

		const allObservedCallsProducedSuccessfulOutputs = observedCallKeys.every(callKey => {
			const output = outputByCallKey.get(callKey);
			return output && !output.isError;
		});

		if (!allObservedCallsProducedSuccessfulOutputs) {
			return;
		}

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

			const hasCurrentText = hasTextRef.current;
			if (hasCurrentText) {
				return false;
			}

			setDraftText(text);
			return true;
		},
		[setDraftText, hasTextRef]
	);

	const handleEditorDocumentChange = useCallback(() => {
		const didProcessChange = onEditorChange();
		if (!didProcessChange) {
			return;
		}

		if (toolCalls.length > 0) {
			handleAttachedToolsChanged();
		}
		if (submitError) {
			setSubmitError(null);
		}
		// Skip auto-cancel while loadExternalMessage is in progress.
		// The transient state (cleared attachments / tool outputs) is not yet restored.
		if (isLoadingExternalMessageRef.current) {
			return;
		}

		// Auto-cancel editing when the editor is completely empty
		// (no text, no tools, no attachments, no tool outputs).
		const hasTextNow = hasTextRef.current;

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
			const enrichedToolCalls = enrichMCPToolCallsFromContext(
				toolCallsToLoad,
				lastSubmittedMCPContextRef.current ?? mcp.mcpContext
			);
			const autoEligibleCount = enrichedToolCalls.filter(t => isAutoSubmitEligibleToolCall(t)).length;
			const shouldSuppressAutoExec = autoExecStopRequestedRef.current || autoExecBlockedByUserRef.current;

			const preparedToolCalls = shouldSuppressAutoExec
				? enrichedToolCalls.map(toolCall =>
						isAutoSubmitEligibleToolCall(toolCall) ? { ...toolCall, suppressAutoExecute: true } : toolCall
					)
				: enrichedToolCalls;

			const tracker = autoSubmitTrackerRef.current;
			for (const toolCall of preparedToolCalls) {
				tracker.observedCallKeys.add(getToolAutoSubmitKey(toolCall));
				if (!isAutoSubmitEligibleToolCall(toolCall)) {
					tracker.allObservedCallsAreAutoExecute = false;
				}
			}

			if (shouldSuppressAutoExec || autoEligibleCount === 0) {
				autoExecStopRequestedRef.current = false;
				autoExecBlockedByUserRef.current = false;
				setAutoExecStopRequested(false);
				setAutoExecBlockedByUser(false);
				setAutoExecStopVisible(false);
			}

			loadToolCalls(preparedToolCalls);
		},
		[loadToolCalls, mcp.mcpContext, resetAutoSubmitTracker]
	);

	const finishAssistantTurn = useCallback(
		(payload: AssistantTurnFinishedPayload) => {
			if (payload.loadedRunnableToolCallCount === 0) {
				clearAutoExecStopState();
			}
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
			openSystemPromptMenu: () => {
				openSkillsPicker();
			},
			openSkillsMenu: () => {
				openSkillsPicker();
			},
			openMCPMenu: () => {
				openMCPPicker();
			},
			requestStopResponse: () => {
				if (isGenerating) {
					onRequestStop();
				}
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
			openSkillsPicker,
			openMCPPicker,
			mcp,
			applySkillSelectionState,
			isGenerating,
			onRequestStop,
		]
	);

	const handleCancelEditing = useCallback(() => {
		resetEditor();
		restorePreEditContext();
		cancelEditing();
	}, [cancelEditing, resetEditor, restorePreEditContext]);

	const stopResponseShortcut = formatShortcut(shortcutConfig.stopResponse);

	const handleRunToolsOnlyClick = useCallback(async () => {
		if (!hasPendingToolCalls || isInputLocked || isSubmitting || fastForwardPending || hasRunningToolCalls) {
			return;
		}
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
		!hasBlockingToolArgs;

	const activeAutoExecBatchCount = useMemo(
		() => toolCalls.filter(toolCall => isAutoSubmitEligibleToolCall(toolCall)).length,
		[toolCalls]
	);

	const showAutoExecStopButton = autoExecStopVisible || activeAutoExecBatchCount >= 2;
	const erroredToolOutputsReadyToSubmit =
		toolOutputs.some(output => output.isError) &&
		!hasPendingToolCalls &&
		!hasRunningToolCalls &&
		!hasBlockingToolArgs &&
		!isInputLocked;

	useEffect(() => {
		onAssistantPresetRuntimeStateChange?.({
			conversationToolChoices: conversationToolsToChoices(conversationToolsState),
			webSearchChoices: mapAssistantPresetWebSearchTemplatesToChoices(webSearchTemplates),
			enabledSkillRefs,
			activeSkillRefs,
			mcpContext: mcp.mcpContext,
		});
	}, [
		conversationToolsState,
		enabledSkillRefs,
		mcp.mcpContext,
		activeSkillRefs,
		onAssistantPresetRuntimeStateChange,
		webSearchTemplates,
	]);

	return (
		<>
			<form
				ref={formRef}
				onSubmit={handleSubmit}
				className="mx-0 flex max-h-full w-full max-w-full min-w-0 flex-col overflow-hidden"
			>
				{submitError ? (
					<div className="alert alert-error mx-4 mt-3 mb-1 flex items-start gap-2 text-sm" role="alert">
						<FiAlertTriangle size={16} className="mt-0.5" />
						<span>{submitError}</span>
					</div>
				) : null}
				{!submitError && erroredToolOutputsReadyToSubmit ? (
					<output className="alert alert-warning mx-4 mt-3 mb-1 flex items-start gap-2 text-sm">
						<FiTool size={16} className="mt-0.5" />
						<span>
							A tool returned an error result. It is ready to submit as tool output, or you can retry/discard it.
						</span>
					</output>
				) : null}

				<Plate editor={editor} onChange={handleEditorDocumentChange}>
					<div className="bg-base-100 border-base-200 flex min-h-0 w-full max-w-full min-w-0 flex-[1_1_auto] overflow-hidden rounded-2xl border">
						<div className="flex min-h-0 min-w-0 grow flex-col p-0">
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
							<div className="flex min-h-24 min-w-0 grow gap-2 overflow-hidden p-1">
								<PlateContent
									ref={contentRef}
									placeholder="Type message..."
									spellCheck={false}
									autoCorrect="off"
									autoCapitalize="off"
									readOnly={isInputLocked}
									onKeyDown={handleEditorKeyDown}
									onPaste={onEditorPaste}
									scrollSelectionIntoView={scrollSelectionIntoEditorView}
									className="max-h-96 min-w-0 flex-1 resize-none overflow-x-hidden overflow-y-auto overscroll-contain bg-transparent p-1 wrap-break-word whitespace-break-spaces tab-2 outline-none focus:outline-none"
									style={{
										fontSize: 14,
										lineHeight: 1.5,
										whiteSpace: 'break-spaces',
										tabSize: 2,
										minHeight: '4rem',
										boxSizing: 'border-box',
										overflowAnchor: 'none',
										overscrollBehavior: 'contain',
									}}
								/>
							</div>
							{/* Unified chips bar: attachments, directories, tools, tool calls & outputs (scrollable) */}
							<div className="w-full min-w-0 shrink-0 items-center overflow-x-auto overscroll-contain p-1 text-xs">
								<EditorChipsBar
									attachments={attachments}
									directoryGroups={directoryGroups}
									toolCalls={toolCalls}
									toolOutputs={toolOutputs}
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
									onOpenToolCallDetails={handleOpenToolCallDetails}
								/>
							</div>
						</div>
						{/* Primary / secondary actions anchored at bottom-right */}
						<div className="flex shrink-0 flex-col items-end justify-end gap-2 p-1">
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
								<HoverTip
									content={stopResponseShortcut ? `Stop response (${stopResponseShortcut})` : 'Stop response'}
									placement="left"
								>
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
													if (!canRunToolsOnly) {
														return;
													}
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
													if (!canRunToolsAndSend) {
														return;
													}
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
												if (!canSendOnly) {
													return;
												}
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

					{/* Bottom bar for composer pickers and keyboard shortcuts */}
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
						onInsertTemplateText={handleInsertTemplateText}
						onAttachTemplateResourcePaths={async (p: string[]) => {
							await attachPathsAsAttachments(p);
						}}
						templateMenuState={templateMenu}
						toolMenuState={toolMenu}
						attachmentMenuState={attachmentMenu}
						skillsMenuState={skillsMenu}
						mcpMenuState={mcpMenu}
						templateButtonRef={templateButtonRef}
						toolButtonRef={toolButtonRef}
						attachmentButtonRef={attachmentButtonRef}
						toolArgsEventTarget={toolArgsEventTarget}
						shortcutConfig={shortcutConfig}
						currentProviderSDKType={currentProviderSDKType}
						attachedToolEntries={attachedToolEntries}
						conversationToolsState={conversationToolsState}
						setConversationToolsState={setConversationToolsState}
						onAttachTool={handleAttachTool}
						onDetachToolByKey={handleDetachAttachedToolByKey}
						onSetAttachedToolAutoExecute={handleSetAttachedToolAutoExecuteByKey}
						webSearchTemplates={webSearchTemplates}
						setWebSearchTemplates={setWebSearchTemplates}
						onWebSearchArgsBlockedChange={nextBlocked => {
							setWebSearchArgsBlocked(nextBlocked);
						}}
						onRemoveAttachedTool={handleRemoveAttachedTool}
						onRemoveAllAttachedTools={handleRemoveAllAttachedTools}
						onEditAttachedToolOptions={handleEditAttachedToolOptions}
						onOpenAttachedToolDetails={handleOpenAttachedToolDetails}
						onOpenConversationToolDetails={handleOpenConversationToolDetails}
						allSkills={allSkills}
						skillsLoading={skillsLoading}
						enabledSkillRefs={enabledSkillRefs}
						activeSkillRefs={activeSkillRefs}
						setEnabledSkillRefs={setEnabledSkillRefs}
						setActiveSkillRefs={setActiveSkillRefs}
						onEnableAllSkills={enableAllSkills}
						onDisableAllSkills={disableAllSkills}
						onRefreshSkills={refreshSkills}
						systemPrompt={systemPrompt}
						isInputLocked={isInputLocked || fastForwardPending}
						mcpState={mcp}
						skillsLoadError={skillsLoadError}
						mcpAppContextUpdateCount={mcpAppContextUpdates.length}
						onClearMCPAppContextUpdates={() => {
							setMCPAppContextUpdates([]);
						}}
					/>
				</Plate>
			</form>

			<ToolDetailsModal
				state={toolDetailsState}
				onClose={() => {
					setToolDetailsState(null);
				}}
			/>

			<MCPApprovalModal
				approvalRequest={mcpApproval.approvalRequest}
				onResolve={r => {
					mcpApproval.resolveMCPApproval(r);
				}}
			/>

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
