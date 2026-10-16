import type { KeyboardEvent as ReactKeyboardEvent, SubmitEventHandler } from 'react';
import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import { FiAlertTriangle, FiEdit2, FiFastForward, FiPlay, FiSend, FiSquare, FiTool, FiX } from 'react-icons/fi';

import { useMenuStore, useStoreState } from '@ariakit/react';
import { Plate, PlateContent } from 'platejs/react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { ProviderSDKType, UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SessionConfiguration } from '@/spec/session';
import type {
	MCPAppModelContextUpdate,
	MCPConversationContext,
	MCPRuntimeServerID,
	MCPToolSelection,
} from '@/spec/session_mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolArgsTarget, ToolListItem, ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection } from '@/spec/workspace';
import { UIToolCallStatus } from '@/spec/inference';
import { SkillSessionSyncMode } from '@/spec/skill';
import { ToolImplType } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { formatShortcut } from '@/lib/keyboard_shortcuts';

import { useEnterSubmit } from '@/hooks/use_enter_submit';
import { useTools } from '@/hooks/use_tool';

import { HoverTip } from '@/components/hover_tip';

import type {
	AssistantTurnFinishedPayload,
	EditorExternalMessage,
	EditorSubmitPayload,
} from '@/chats/composer/editor/editor_types';
import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import type { AgentSystemPromptController } from '@/chats/composer/skills/use_agent_system_prompt';
import type { ToolDetailsState } from '@/chats/composer/tools/tool_details_modal';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';
import { useComposerAttachments } from '@/chats/composer/attachments/use_composer_attachments';
import { EditorBottomBar } from '@/chats/composer/editor/editor_bottom_bar';
import { EditorChipsBar } from '@/chats/composer/editor/editor_chips_bar';
import { useComposerMCP } from '@/chats/composer/mcp/use_composer_mcp';
import { buildEditorValueFromPlainText, insertPlainTextAsSingleBlock } from '@/chats/composer/platedoc/platedoc_utils';
import {
	getAttachedTools,
	insertToolSelectionNode,
	removeToolByKey,
	setAttachedToolUserArgSchemaInstanceBySelectionID,
	setToolAutoExecuteByKey,
} from '@/chats/composer/platedoc/tool_document_ops';
import { useComposerDocument } from '@/chats/composer/platedoc/use_composer_document';
import { useComposerSkills } from '@/chats/composer/skills/use_composer_skills';
import { useComposerTools } from '@/chats/composer/toolruntime/use_composer_tools';
import { dispatchOpenToolArgs, useOpenToolArgs } from '@/chats/composer/toolruntime/use_open_toolargs_event';
import { ToolDetailsModal } from '@/chats/composer/tools/tool_details_modal';
import { ToolArgsModalHost } from '@/chats/composer/tools/tool_user_args_host';
import { buildWebSearchChoicesForSubmit, getWebSearchConfiguration } from '@/chats/composer/tools/websearch_utils';
import { useComposerWorkspace } from '@/chats/composer/workspaces/use_composer_workspace';
import {
	effectiveActiveSkillSelections,
	effectiveArtifactSelections,
	hydrateSessionMCPOverrides,
	hydrateSessionToolOverrides,
} from '@/chats/session/session_composer_restore';
import { conversationToolsToChoices, mergeConversationToolsWithNewChoices } from '@/tools/lib/conversation_tool_utils';
import { isRunnableComposerToolCall, requiresComposerToolResponse } from '@/tools/lib/tool_call_utils';
import { dedupeToolChoices, uiToolChoiceToToolStoreChoice } from '@/tools/lib/tool_choice_utils';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

interface SkillStateApplyOptions {
	syncSession?: SkillSessionSyncMode;
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
	openWorkspaceMenu: () => void;
	clearWorkspace: () => void;
	requestStopResponse: () => void;
	loadExternalMessage: (msg: EditorExternalMessage) => void;
	setDraftText: (text: string) => void;
	setDraftTextIfEmpty: (text: string) => boolean;
	resetEditor: () => void;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	setConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	setToolSelectionIssues: (issues: ToolSelectionIssue[]) => void;
	setMCPContextFromMessage: (context?: MCPConversationContext) => void;
	setMCPAppContextUpdatesFromMessage: (updates?: MCPAppModelContextUpdate[]) => void;
	appendMCPAppContextUpdate: (update: MCPAppModelContextUpdate) => void;
	clearMCPContext: () => void;
	setWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
	applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => void;
	setSkillStateFromMessage: (enabledRefs: SkillRef[], activeRefs: SkillRef[], options?: SkillStateApplyOptions) => void;
	setInstalledSkillStateFromAgent: (enabledRefs: SkillRef[], activeRefs: SkillRef[]) => void;
	setWorkspaceSelectionFromMessage: (selection?: WorkspaceConversationSelection, syncSkills?: boolean) => void;
	finishAssistantTurn: (payload: AssistantTurnFinishedPayload) => void;
	restoreSessionConfiguration: (configuration: SessionConfiguration) => Promise<void>;
}

interface EditorAreaProps {
	isGenerating: boolean;
	isInputLocked: boolean;
	isRunActive: boolean;
	submissionMode?: 'message' | 'continuation';
	currentProviderSDKType: ProviderSDKType;
	shortcutConfig: ShortcutConfig;
	onSubmit: (payload: EditorSubmitPayload) => Promise<void>;
	onRequestStop: () => void;
	editingMessageId: string | null;
	cancelEditing: () => void;
	systemPrompt: AgentSystemPromptController;
}

function getMCPContextToolSelections(context?: MCPConversationContext): MCPToolSelection[] {
	const out: MCPToolSelection[] = [];

	for (const server of context?.servers ?? []) {
		for (const tool of server.selectedTools ?? []) {
			out.push({
				...tool,
				server: tool.server ?? server.server,
			});
		}
	}

	return out;
}

function mergeMCPToolSelection(selection: MCPToolSelection, contextSelection: MCPToolSelection): MCPToolSelection {
	return {
		...contextSelection,
		...selection,
		server: selection.server ?? contextSelection.server,
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

function sameMCPRuntimeServerID(left: MCPRuntimeServerID | undefined, right: MCPRuntimeServerID | undefined): boolean {
	return left === right;
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
		if (!sameMCPRuntimeServerID(selection.server, candidate.server)) {
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
		isRunActive,
		submissionMode = 'message',
		currentProviderSDKType,
		shortcutConfig,
		onSubmit,
		onRequestStop,
		editingMessageId,
		cancelEditing,
		systemPrompt,
	},
	ref
) {
	const lastSubmittedMCPContextRef = useRef<MCPConversationContext | undefined>(undefined);

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
	const workspaceMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const mcpMenuOpen = useStoreState(mcpMenu, 'open');
	const mcp = useComposerMCP(mcpMenuOpen);
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
		workspace: false,
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
		removeDirectoryAttachments: handleRemoveDirectoryAttachments,
		restoreDirectoryAttachments: handleRestoreDirectoryAttachments,
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
	const toolCatalog = useTools();
	const [toolSelectionIssues, setToolSelectionIssues] = useState<ToolSelectionIssue[]>([]);
	const [toolDetailsState, setToolDetailsState] = useState<ToolDetailsState>(null);
	const [toolArgsTarget, setToolArgsTarget] = useState<ToolArgsTarget | null>(null);

	useOpenToolArgs(target => {
		setToolArgsTarget(target);
	}, toolArgsEventTarget);

	const {
		allSkills,
		skillsLoading,
		templateSkills,
		skillsLoadError,
		activeSkillRefs,
		installedEnabledSkillRefs,
		installedActiveSkillRefs,
		setEnabledSkillRefs,
		setActiveSkillRefs,
		enableAllSkills,
		disableAllSkills,
		refreshSkills,
		applyInstalledSkillSelectionState,
		applySkillSelectionState,
		applyWorkspaceSkillSelectionState,
		getCurrentEnabledSkillRefs,
		getCurrentActiveSkillRefs,
		getCurrentInstalledEnabledSkillRefs,
		getCurrentInstalledActiveSkillRefs,
		getCurrentInstalledEnabledSkillSelections,
		getCurrentActiveSkillSelections,
	} = useComposerSkills();

	const workspace = useComposerWorkspace({
		applyWorkspaceSkillSelectionState,
		getCurrentActiveSkillRefs,
	});

	const workspaceSelectionPending = workspace.selectionLoading;
	const workspaceExecutionBlocked = workspaceSelectionPending || workspace.blockingError !== null;
	const workspaceControlsLocked = isInputLocked || isSubmitting || fastForwardPending || workspaceSelectionPending;

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
		handleRetryErroredOutput,
		handleAttachedToolsChanged,
		handleRemoveToolOutput,
		applyConversationToolsFromChoices,
		applyWebSearchFromChoices,
		loadToolCalls,
		clearComposerToolsState,
		getToolRuntimeSnapshot,
	} = useComposerTools({
		isBusy: isGenerating,
		isSubmitting,
		getAttachedToolEntries: getAttachedToolEntriesSnapshot,
		externalExecutionBlocked: fastForwardPending || workspaceExecutionBlocked,
	});

	const webSearchConfiguration = getWebSearchConfiguration(
		webSearchTemplates,
		toolCatalog.data,
		toolCatalog.ready,
		currentProviderSDKType
	);
	const incompatibleSDKSelection = [
		...attachedToolEntries,
		...conversationToolsToChoices(conversationToolsState),
		...webSearchTemplates,
	].some(
		choice => choice.implementationKind === ToolImplType.SDK && choice.sdkType !== currentProviderSDKType.toString()
	);
	const hasBlockingMCPArgs = mcp.argumentsBlocked;
	const hasBlockingToolArgs =
		toolArgsBlocked ||
		webSearchConfiguration.blocked ||
		hasBlockingMCPArgs ||
		incompatibleSDKSelection ||
		toolSelectionIssues.length > 0;

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
		setToolDetailsState({ kind: 'choice', choice: uiToolChoiceToToolStoreChoice(entry) });
	}, []);

	const attachedToolIdentityKeys = useMemo(() => {
		return new Set(attachedToolEntries.map(entry => toolIdentityKey(entry.target)));
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

	const attachApprovedTemplateResourcePaths = useCallback(
		(paths: string[], canApply: () => boolean) => {
			return attachPathsAsAttachments(paths, undefined, {
				canApply,
			});
		},
		[attachPathsAsAttachments]
	);

	const handleAttachTool = useCallback(
		(item: ToolListItem, autoExecute: boolean) => {
			const identityKey = toolIdentityKey(item.target);
			if (attachedToolIdentityKeys.has(identityKey)) {
				return;
			}

			insertToolSelectionNode(editor, item, autoExecute);
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
			const identityKey = toolIdentityKey(entry.target);
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
				const identityKey = toolIdentityKey(entry.target);
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
	const preEditToolSelectionIssuesRef = useRef<ToolSelectionIssue[] | null>(null);
	const preEditMCPContextRef = useRef<MCPConversationContext | undefined | null>(null);
	const preEditEnabledSkillRefsRef = useRef<SkillRef[] | null>(null);
	const preEditActiveSkillRefsRef = useRef<SkillRef[] | null>(null);
	const preEditWorkspaceSelectionRef = useRef<WorkspaceConversationSelection | undefined | null>(null);

	// --- Fix: focus management for menus opened by shortcuts ---
	const templateMenuOpen = useStoreState(templateMenu, 'open');
	const toolMenuOpen = useStoreState(toolMenu, 'open');
	const attachmentMenuOpen = useStoreState(attachmentMenu, 'open');
	const skillsMenuOpen = useStoreState(skillsMenu, 'open');
	const workspaceMenuOpen = useStoreState(workspaceMenu, 'open');
	const templateMenuEl = useStoreState(templateMenu, 'contentElement');
	const toolMenuEl = useStoreState(toolMenu, 'contentElement');
	const attachmentMenuEl = useStoreState(attachmentMenu, 'contentElement');
	const skillsMenuEl = useStoreState(skillsMenu, 'contentElement');
	const mcpMenuEl = useStoreState(mcpMenu, 'contentElement');
	const workspaceMenuEl = useStoreState(workspaceMenu, 'contentElement');

	const clearPreEditSnapshot = useCallback(() => {
		preEditConversationToolsRef.current = null;
		preEditWebSearchTemplatesRef.current = null;
		preEditToolSelectionIssuesRef.current = null;
		preEditMCPContextRef.current = null;
		preEditEnabledSkillRefsRef.current = null;
		preEditActiveSkillRefsRef.current = null;
		preEditWorkspaceSelectionRef.current = null;
	}, []);

	const closeAllMenus = useCallback(() => {
		templateMenu.hide();
		toolMenu.hide();
		attachmentMenu.hide();
		skillsMenu.hide();
		mcpMenu.hide();
		workspaceMenu.hide();
	}, [attachmentMenu, mcpMenu, skillsMenu, templateMenu, toolMenu, workspaceMenu]);

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

	useEffect(() => {
		if (!workspaceMenuOpen) {
			if (menuOpenedByShortcutRef.current.workspace) {
				menuOpenedByShortcutRef.current.workspace = false;
				requestAnimationFrame(() => {
					focusEditorPreservingSelection();
				});
			}
			return;
		}
		if (!menuOpenedByShortcutRef.current.workspace) {
			return;
		}

		requestAnimationFrame(() => {
			focusMenuSearchOrFirstItem(workspaceMenuEl);
		});
	}, [focusEditorPreservingSelection, workspaceMenuEl, workspaceMenuOpen]);

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

	const openWorkspacePicker = useCallback(() => {
		if (workspaceControlsLocked) {
			return;
		}
		menuOpenedByShortcutRef.current.workspace = true;

		closeAllMenus();
		workspaceMenu.show();
	}, [closeAllMenus, workspaceControlsLocked, workspaceMenu]);

	const restorePreEditContext = useCallback(() => {
		const prevConv = preEditConversationToolsRef.current;
		const prevWs = preEditWebSearchTemplatesRef.current;
		const prevMCP = preEditMCPContextRef.current;
		const prevSkills = preEditEnabledSkillRefsRef.current;
		const prevActive = preEditActiveSkillRefsRef.current;
		const prevWorkspace = preEditWorkspaceSelectionRef.current;

		if (prevConv) {
			setConversationToolsState(prevConv);
		}
		if (prevWs) {
			setWebSearchTemplates(prevWs);
		}
		if (preEditToolSelectionIssuesRef.current) {
			setToolSelectionIssues(preEditToolSelectionIssuesRef.current);
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
		if (prevWorkspace !== null) {
			void workspace.restoreSelection(prevWorkspace, false);
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
		workspace,
	]);

	const isSendButtonEnabled = useMemo(() => {
		// Cannot block on tools def loading here.
		if (isInputLocked) {
			return false;
		}
		if (isSubmitting) {
			return false;
		}
		if (workspaceExecutionBlocked) {
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
		const hasMCPContent = mcp.selectedServerCount > 0 || mcpAppContextUpdates.length > 0;

		return hasAttachments || hasOutputs || hasMCPContent;
	}, [
		isInputLocked,
		isSubmitting,
		hasBlockingToolArgs,
		hasText,
		attachments.length,
		toolOutputs.length,
		workspaceExecutionBlocked,
		mcp.selectedServerCount,
		mcpAppContextUpdates.length,
	]);

	const { formRef, onKeyDown } = useEnterSubmit({
		isBusy: isGenerating || isSubmitting || fastForwardPending || workspaceExecutionBlocked,
		canSubmit: () => {
			if (isInputLocked) {
				return false;
			}
			if (isSubmitting) {
				return false;
			}
			if (workspaceExecutionBlocked) {
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
			const hasMCPContent = mcp.selectedServerCount > 0 || mcpAppContextUpdates.length > 0;

			return hasAttachments || hasOutputs || hasMCPContent;
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
		clearAttachments();
		clearComposerToolsState();
		setMCPAppContextUpdates([]);
	}, [clearAttachments, clearComposerToolsState, closeAllMenus, setSubmitting]);

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

			if (workspaceExecutionBlocked) {
				setSubmitError(
					workspace.blockingError ?? 'Workspace selection is still resolving. Wait for it to finish before sending.'
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
						.filter(toolCall => toolCall.status === UIToolCallStatus.Pending && isRunnableComposerToolCall(toolCall))
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

			try {
				// Build final message content after tools have run.
				// Always read from the live tool runtime snapshot here. Auto-execute
				// can complete and trigger submit before React re-renders the latest
				// toolOutputs / hasPendingToolCalls values into this component.
				const runtimeAfterRun = getToolRuntimeSnapshot();

				const textToSend = editor.api.string([]);
				const finalToolOutputs: UIToolOutput[] = runtimeAfterRun.toolOutputs;
				const unfinishedRunnableToolCalls = runtimeAfterRun.toolCalls.filter(
					toolCall =>
						(toolCall.status === UIToolCallStatus.Pending || toolCall.status === UIToolCallStatus.Running) &&
						requiresComposerToolResponse(toolCall)
				);
				const failedRunnableToolCalls = runtimeAfterRun.toolCalls.filter(
					toolCall => toolCall.status === UIToolCallStatus.Failed && requiresComposerToolResponse(toolCall)
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
					setSubmitError('Some tool calls failed. Retry them or submit an error result before sending.');
					return;
				}

				const hasNonEmptyText = textToSend.trim().length > 0;
				const hasAttachmentsToSend = attachments.length > 0;
				const hasToolOutputsToSend = finalToolOutputs.length > 0;
				const hasMCPContentToSend = mcp.selectedServerCount > 0 || mcpAppContextUpdates.length > 0;

				// Enforce the "non-empty message" invariant *after* tools have run.
				if (!hasNonEmptyText && !hasAttachmentsToSend && !hasToolOutputsToSend && !hasMCPContentToSend) {
					setSubmitError(
						hadPendingTools
							? 'Tool calls did not produce any outputs, so there is nothing to send yet.'
							: 'Nothing to send. Add text, attachments, or tool outputs first.'
					);
					return;
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

				const payload: EditorSubmitPayload = {
					text: textToSend,
					sessionInstructionText: systemPrompt.sessionInstructionText.trim() || undefined,
					includeModelDefault: systemPrompt.includeModelDefault,
					attachedTools,
					attachments,
					toolOutputs: finalToolOutputs,
					finalToolChoices,
					mcpContext: preparedMCPContext,
					mcpAppContextUpdates:
						mcpAppContextUpdates.length > 0 ? mcpAppContextUpdates.map(update => ({ ...update })) : undefined,
					enabledSkillRefs: getCurrentInstalledEnabledSkillRefs(),
					activeSkillRefs: getCurrentInstalledActiveSkillRefs(),
					enabledSkillSelections: getCurrentInstalledEnabledSkillSelections(),
					// Workspace-owned active Skills must be serialized too. The
					// backend validates that each requested active Skill remains
					// part of the effective enabled Skill set.
					activeSkillSelections: getCurrentActiveSkillSelections(),
					workspaceSelection: workspace.getSelectionSnapshot(),
					workspacePlan: workspace.plan,
					...(editingMessageId && finalToolOutputs.length > 0
						? {
								carryForwardToolResultIDs: [...new Set(finalToolOutputs.map(output => output.id).filter(Boolean))],
							}
						: {}),
				};

				lastSubmittedMCPContextRef.current = preparedMCPContext;
				try {
					await onSubmit(payload);
				} catch (error: unknown) {
					setSubmitError(error instanceof Error ? error.message : 'The Session Runtime rejected this submission.');
					return;
				}
				setSubmitError(null);
				submittedToolChoices = finalToolChoices;

				didSend = true;
			} finally {
				setSubmitting(false);

				// Only clear the editor if we actually sent something.
				if (didSend) {
					resetEditor();

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
			systemPrompt,
			attachments,
			clearPreEditSnapshot,
			closeAllMenus,
			conversationToolsState,
			editor,
			getCurrentInstalledActiveSkillRefs,
			getCurrentInstalledEnabledSkillRefs,
			getToolRuntimeSnapshot,
			hasBlockingMCPArgs,
			editingMessageId,
			hasBlockingToolArgs,
			isInputLocked,
			isSendButtonEnabled,
			mcp,
			mcpAppContextUpdates,
			onSubmit,
			resetEditor,
			setConversationToolsState,
			setSubmitting,
			startFastForwardRun,
			toolCalls,
			webSearchTemplates,
			workspace,
			workspaceExecutionBlocked,
			getCurrentInstalledEnabledSkillSelections,
			getCurrentActiveSkillSelections,
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
				(toolCall.status === UIToolCallStatus.Pending || toolCall.status === UIToolCallStatus.Running) &&
				isRunnableComposerToolCall(toolCall)
		);

		if (unfinishedRunnableToolCalls.length > 0) {
			return;
		}

		const failedRunnableToolCalls = toolCalls.filter(
			toolCall => toolCall.status === UIToolCallStatus.Failed && isRunnableComposerToolCall(toolCall)
		);

		if (failedRunnableToolCalls.length > 0) {
			// oxlint-disable-next-line react/set-state-in-effect
			finishFastForwardWithError('Some tool calls failed. Retry them or submit an error result before sending.');
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
				if (!preEditToolSelectionIssuesRef.current) {
					preEditToolSelectionIssuesRef.current = toolSelectionIssues;
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
				if (preEditWorkspaceSelectionRef.current === null) {
					preEditWorkspaceSelectionRef.current = workspace.getSelectionSnapshot();
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
				setToolSelectionIssues(incoming.toolSelectionIssues ?? []);
				mcp.restoreContext(incoming.mcpContext);
				setMCPAppContextUpdates(incoming.mcpAppContextUpdates ?? []);
				// 4) Restore enabled/active skills together so invariants hold immediately.
				void applySkillSelectionState(incoming.enabledSkillRefs ?? [], incoming.activeSkillRefs ?? [], {
					syncSession: SkillSessionSyncMode.None,
					forceResetSession: true,
				});
				void workspace.restoreSelection(incoming.workspaceSelection, false);
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
			toolSelectionIssues,
			webSearchTemplates,
			workspace,
		]
	);

	const setDraftText = useCallback(
		(text: string) => {
			closeAllMenus();
			setSubmitError(null);
			replaceEditorDocument(buildEditorValueFromPlainText(text), 'end');
			window.requestAnimationFrame(() => {
				focusEditorAtEnd();
			});
		},
		[closeAllMenus, focusEditorAtEnd, replaceEditorDocument]
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

		handleAttachedToolsChanged();
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
		toolOutputs.length,
	]);

	const handleLoadToolCalls = useCallback(
		(toolCallsToLoad: UIToolCall[]) => {
			loadToolCalls(
				enrichMCPToolCallsFromContext(toolCallsToLoad, lastSubmittedMCPContextRef.current ?? mcp.mcpContext)
			);
		},
		[loadToolCalls, mcp.mcpContext]
	);

	const restoreSessionConfiguration = useCallback(
		async (configuration: SessionConfiguration) => {
			setSubmitError(null);
			clearComposerToolsState();
			clearAttachments();
			setMCPAppContextUpdates([]);
			applyConversationToolsFromChoices([]);
			applyWebSearchFromChoices([]);
			setToolSelectionIssues([]);
			mcp.clear();

			const restoreErrors: string[] = [];

			try {
				await workspace.detachWorkspace(false);
			} catch (error) {
				restoreErrors.push(
					`Workspace reset failed: ${
						error instanceof Error ? error.message : 'the existing Workspace selection could not be cleared'
					}`
				);
			}

			try {
				await applyInstalledSkillSelectionState([], [], {
					syncSession: SkillSessionSyncMode.None,
					forceResetSession: true,
				});
			} catch (error) {
				restoreErrors.push(
					`Skill reset failed: ${
						error instanceof Error ? error.message : 'the existing Skill selection could not be cleared'
					}`
				);
			}

			const [toolStateResult, mcpContextResult] = await Promise.allSettled([
				hydrateSessionToolOverrides(configuration.tools),
				hydrateSessionMCPOverrides(configuration.mcp),
			]);

			if (toolStateResult.status === 'fulfilled') {
				applyConversationToolsFromChoices(toolStateResult.value.choices);
				applyWebSearchFromChoices(toolStateResult.value.choices);
				setToolSelectionIssues(toolStateResult.value.issues);
			} else {
				restoreErrors.push(
					`Tools could not be restored: ${
						toolStateResult.reason instanceof Error
							? toolStateResult.reason.message
							: 'saved Tool selections are unavailable'
					}`
				);
			}

			if (mcpContextResult.status === 'fulfilled') {
				mcp.restoreContext(mcpContextResult.value);
			} else {
				restoreErrors.push(
					`MCP context could not be restored: ${
						mcpContextResult.reason instanceof Error
							? mcpContextResult.reason.message
							: 'saved MCP selections are unavailable'
					}`
				);
			}

			try {
				await workspace.restoreSessionIntent(configuration.workspace, false);
			} catch (error) {
				restoreErrors.push(
					`Workspace could not be restored: ${
						error instanceof Error ? error.message : 'saved Workspace selections are unavailable'
					}`
				);
			}

			try {
				const enabledSelections = effectiveArtifactSelections(configuration.skills);
				const activeSelections = effectiveActiveSkillSelections(configuration.activeSkills);

				await applyInstalledSkillSelectionState(
					enabledSelections.map(selection => ({
						rootID: selection.rootID,
						artifactID: selection.artifactID,
					})),
					activeSelections.map(selection => ({
						rootID: selection.rootID,
						artifactID: selection.artifactID,
					})),
					{
						syncSession: SkillSessionSyncMode.None,
						forceResetSession: true,
					}
				);
			} catch (error) {
				restoreErrors.push(
					`Skills could not be restored: ${
						error instanceof Error ? error.message : 'saved Skill selections are unavailable'
					}`
				);
			}

			if (restoreErrors.length > 0) {
				setSubmitError(`Some saved Composer settings could not be restored:\n${restoreErrors.join('\n')}`);
			}
		},
		[
			applyConversationToolsFromChoices,
			applyInstalledSkillSelectionState,
			applyWebSearchFromChoices,
			clearAttachments,
			clearComposerToolsState,
			mcp,
			setToolSelectionIssues,
			workspace,
		]
	);

	const finishAssistantTurn = useCallback((_payload: AssistantTurnFinishedPayload) => undefined, []);

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
			openWorkspaceMenu: () => {
				openWorkspacePicker();
			},
			clearWorkspace: () => {
				void workspace.detachWorkspace(false);
			},
			requestStopResponse: () => {
				if (isRunActive) {
					onRequestStop();
				}
			},
			loadExternalMessage,
			setDraftText,
			setDraftTextIfEmpty,
			resetEditor,
			loadToolCalls: handleLoadToolCalls,
			setConversationToolsFromChoices: applyConversationToolsFromChoices,
			setToolSelectionIssues,
			setWebSearchFromChoices: applyWebSearchFromChoices,
			setMCPContextFromMessage: context => {
				mcp.restoreContext(context);
			},
			setMCPAppContextUpdatesFromMessage: updates => {
				setMCPAppContextUpdates(updates ?? []);
			},
			appendMCPAppContextUpdate: update => {
				void mcp
					.ensureServerSelected(update.server)
					.then(selected => {
						if (!selected) {
							setSubmitError('The MCP App context could not be added because its server is unavailable or disabled.');
							return;
						}
						setMCPAppContextUpdates(prev => [...prev, update]);
					})
					.catch(() => {
						setSubmitError('The MCP server catalog could not be loaded.');
					});
			},
			clearMCPContext: () => {
				mcp.clear();
			},
			applyAttachmentsDrop,
			setSkillStateFromMessage: (enabledRefs, activeRefs, options) => {
				void applySkillSelectionState(enabledRefs, activeRefs, options);
			},
			setInstalledSkillStateFromAgent: (enabledRefs, activeRefs) => {
				void applyInstalledSkillSelectionState(enabledRefs, activeRefs, {
					syncSession: SkillSessionSyncMode.EnsureIfEnabled,
				});
			},
			setWorkspaceSelectionFromMessage: (selection, syncSkills = true) => {
				void workspace.restoreSelection(selection, syncSkills);
			},
			finishAssistantTurn,
			restoreSessionConfiguration,
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
			openWorkspacePicker,
			workspace,
			mcp,
			applySkillSelectionState,
			applyInstalledSkillSelectionState,
			isRunActive,
			onRequestStop,
			restoreSessionConfiguration,
		]
	);

	const handleCancelEditing = useCallback(() => {
		resetEditor();
		restorePreEditContext();
		cancelEditing();
	}, [cancelEditing, resetEditor, restorePreEditContext]);

	const stopResponseShortcut = formatShortcut(shortcutConfig.stopResponse);
	const sendActionText = submissionMode === 'continuation' ? 'Stage continuation and continue run' : 'Send message';

	const handleRunToolsOnlyClick = useCallback(async () => {
		if (workspaceExecutionBlocked) {
			setSubmitError(
				workspace.blockingError ?? 'Workspace selection is still resolving. Wait for it to finish before running tools.'
			);
			return;
		}

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
		workspace.blockingError,
		workspaceExecutionBlocked,
	]);

	// Button-state helpers:
	// - Play: run tools only (enabled when there are pending tools and none are running).
	// - Fast-forward: run tools then send (enabled when there are pending tools and
	//   templates are satisfied; "sendability" will be re-checked after tools run).
	// - Send: send only (enabled when send is allowed and there are no pending tools).
	const canSendOnly = !hasPendingToolCalls && isSendButtonEnabled && !hasRunningToolCalls && !fastForwardPending;
	const canRunToolsOnly =
		hasPendingToolCalls &&
		!hasRunningToolCalls &&
		!isInputLocked &&
		!isSubmitting &&
		!fastForwardPending &&
		!workspaceExecutionBlocked;
	const canRunToolsAndSend =
		hasPendingToolCalls &&
		!hasRunningToolCalls &&
		!isInputLocked &&
		!isSubmitting &&
		!fastForwardPending &&
		!workspaceExecutionBlocked &&
		!hasBlockingToolArgs;

	const erroredToolOutputsReadyToSubmit =
		toolOutputs.some(output => output.isError && !output.isSkipped) &&
		!hasPendingToolCalls &&
		!hasRunningToolCalls &&
		!hasBlockingToolArgs &&
		!isInputLocked;

	return (
		<div className="mx-0 flex max-h-128 w-full min-w-0 flex-col overflow-hidden">
			<form ref={formRef} noValidate onSubmit={handleSubmit} className="max-h-full">
				{submitError ? (
					<div className="alert alert-error mx-4 mt-3 mb-1 flex items-start gap-2 text-sm" role="alert">
						<FiAlertTriangle size={16} className="mt-0.5" />
						<span>{submitError}</span>
					</div>
				) : null}
				{toolSelectionIssues.length > 0 ? (
					<div className="alert alert-warning mx-4 mt-3 mb-1 block text-sm">
						<p>Some saved Tool selections are unavailable. Remove and reselect them before sending.</p>
						{toolSelectionIssues.map(issue => (
							<div key={issue.selection.choiceID} className="mt-2 flex items-start gap-2">
								<div className="min-w-0 flex-1">
									<div>{issue.selection.target.name}</div>
									<div className="text-xs">{issue.message}</div>
								</div>
								<button
									type="button"
									className="btn btn-ghost btn-xs"
									disabled={isInputLocked || isSubmitting}
									onClick={() => {
										setToolSelectionIssues(previous =>
											previous.filter(value => value.selection.choiceID !== issue.selection.choiceID)
										);
									}}
								>
									Remove selection
								</button>
							</div>
						))}
					</div>
				) : null}
				{incompatibleSDKSelection ? (
					<div className="alert alert-warning mx-4 mt-3 mb-1 text-sm">
						A selected provider Tool is incompatible with the current model provider.
					</div>
				) : null}
				{!submitError && erroredToolOutputsReadyToSubmit ? (
					<output className="alert alert-warning mx-4 mt-3 mb-1 flex items-start gap-2 text-sm">
						<FiTool size={16} className="mt-0.5" />
						<span>A tool returned an error result. Submit it to the model, or retry the tool before sending.</span>
					</output>
				) : null}

				<Plate editor={editor} onChange={handleEditorDocumentChange}>
					<div className="bg-base-100 border-base-200 flex max-h-full min-h-0 w-full max-w-full min-w-0 flex-[1_1_auto] overflow-hidden rounded-2xl border">
						<div className="flex min-h-0 min-w-0 grow flex-col p-0">
							{submissionMode === 'continuation' ? (
								<div className="border-info/30 bg-info/10 mx-3 mt-2 flex items-center gap-2 rounded-xl border px-2 py-1 text-xs">
									<FiFastForward size={14} aria-hidden="true" />
									<span>This input will be staged for the waiting Run and then continued.</span>
								</div>
							) : null}

							{editingMessageId && (
								<div className="flex min-h-0 items-center justify-end gap-2 pt-1 pr-3 pb-0 text-xs">
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
									isBusy={
										isGenerating ||
										isSubmitting ||
										isInputLocked ||
										fastForwardPending ||
										hasRunningToolCalls ||
										workspaceExecutionBlocked
									}
									onRunToolCall={handleRunSingleToolCall}
									onDiscardToolCall={handleDiscardToolCall}
									onOpenOutput={handleOpenToolOutput}
									onRetryErroredOutput={handleRetryErroredOutput}
									onRemoveToolOutput={handleRemoveToolOutput}
									onRemoveAttachment={handleRemoveAttachment}
									onChangeAttachmentContentBlockMode={handleChangeAttachmentContentBlockMode}
									onRemoveDirectoryGroup={handleRemoveDirectoryGroup}
									onRemoveDirectoryAttachments={handleRemoveDirectoryAttachments}
									onRestoreDirectoryAttachments={handleRestoreDirectoryAttachments}
									onRemoveOverflowDir={handleRemoveOverflowDir}
									onOpenToolCallDetails={handleOpenToolCallDetails}
								/>
							</div>
						</div>
						{/* Primary / secondary actions anchored at bottom-right */}
						<div className="flex shrink-0 flex-col items-end justify-end gap-2 p-1">
							{!isGenerating && isRunActive ? (
								<HoverTip content="Cancel active run" placement="left">
									<button
										type="button"
										className="btn btn-circle btn-warning btn-sm shrink-0"
										onClick={onRequestStop}
										aria-label="Cancel active run"
									>
										<FiSquare size={18} />
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
										content={hasPendingToolCalls ? 'Send (enabled after tools finish)' : sendActionText}
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
											aria-label={sendActionText}
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
						onAttachApprovedTemplateResourcePaths={attachApprovedTemplateResourcePaths}
						templateMenuState={templateMenu}
						toolMenuState={toolMenu}
						attachmentMenuState={attachmentMenu}
						skillsMenuState={skillsMenu}
						mcpMenuState={mcpMenu}
						workspaceMenuState={workspaceMenu}
						workspaceState={workspace}
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
						toolCatalog={toolCatalog}
						onRemoveAttachedTool={handleRemoveAttachedTool}
						onRemoveAllAttachedTools={handleRemoveAllAttachedTools}
						onEditAttachedToolOptions={handleEditAttachedToolOptions}
						onOpenAttachedToolDetails={handleOpenAttachedToolDetails}
						onOpenConversationToolDetails={handleOpenConversationToolDetails}
						allSkills={allSkills}
						skillsLoading={skillsLoading}
						templateSkills={templateSkills}
						enabledSkillRefs={installedEnabledSkillRefs}
						activeSkillRefs={installedActiveSkillRefs}
						setEnabledSkillRefs={setEnabledSkillRefs}
						setActiveSkillRefs={setActiveSkillRefs}
						onEnableAllSkills={enableAllSkills}
						onDisableAllSkills={disableAllSkills}
						onRefreshSkills={refreshSkills}
						systemPrompt={systemPrompt}
						workspaceActiveSkillRefs={activeSkillRefs}
						isInputLocked={workspaceControlsLocked}
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
				toolCatalog={toolCatalog}
			/>
		</div>
	);
});
