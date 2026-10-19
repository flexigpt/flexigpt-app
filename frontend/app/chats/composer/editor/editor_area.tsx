import type { KeyboardEvent as ReactKeyboardEvent, SubmitEventHandler } from 'react';
import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import {
	FiAlertTriangle,
	FiEdit2,
	FiFastForward,
	FiPlay,
	FiSend,
	FiSquare,
	FiTool,
	FiTrash2,
	FiX,
	FiZapOff,
} from 'react-icons/fi';

import { useMenuStore, useStoreState } from '@ariakit/react';
import { Plate, PlateContent } from 'platejs/react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { ProviderSDKType } from '@/spec/provider_sdk';
import type { SessionConfiguration } from '@/spec/session';
import type { MCPAppModelContextUpdate, MCPConversationContext } from '@/spec/session_mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolArgsTarget, ToolListItem, ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection } from '@/spec/workspace';
import { SkillSessionSyncMode } from '@/spec/skill';
import { ToolImplType } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { formatShortcut } from '@/lib/keyboard_shortcuts';

import { useEnterSubmit } from '@/hooks/use_enter_submit';
import { useTools } from '@/hooks/use_tool';

import { normalizeSkillRefs } from '@/apis/skill_management';
import { toolIdentityKey } from '@/apis/tool_management';

import { HoverTip } from '@/components/hover_tip';

import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import type { AgentSystemPromptController } from '@/chats/composer/skills/use_agent_system_prompt';
import type { ConversationToolStateEntry } from '@/chats/composer/tools/lib/conversation_tool_utils';
import type { ToolDetailsState } from '@/chats/composer/tools/tool_details_modal';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import { DirectoryChip } from '@/chats/composer/attachments/attachment_directory_chip';
import { uiAttachmentKey } from '@/chats/composer/attachments/attachment_editor_utils';
import { StandaloneAttachmentsChip } from '@/chats/composer/attachments/attachment_standalone_chips';
import { useComposerAttachments } from '@/chats/composer/attachments/use_composer_attachments';
import { EditorBottomBar } from '@/chats/composer/editor/editor_bottom_bar';
import { SessionToolActivityChips } from '@/chats/composer/editor/session_tool_activity_chips';
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
import { useComposerToolConfig } from '@/chats/composer/toolruntime/use_composer_tool_config';
import { dispatchOpenToolArgs, useOpenToolArgs } from '@/chats/composer/toolruntime/use_open_toolargs_event';
import {
	conversationToolsToChoices,
	mergeConversationToolsWithNewChoices,
} from '@/chats/composer/tools/lib/conversation_tool_utils';
import { dedupeToolChoices, uiToolChoiceToToolStoreChoice } from '@/chats/composer/tools/lib/tool_choice_utils';
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
	showAutomationControl: boolean;
	automationHoldEnabled: boolean;
	onToggleAutomationHold: () => void;
	showPendingToolControls: boolean;
	canContinueWithoutDraft: boolean;
	onRunPendingTools: (continueRun: boolean) => void;
	sessionToolCalls: UIToolCall[];
	sessionToolOutputs: UIToolOutput[];
	onRunSessionToolCall: (call: UIToolCall) => void;
	onRetrySessionToolCall: (call: UIToolCall) => void;
	onSkipSessionToolCall: (call: UIToolCall) => void;
	stagedAppContextUpdateCount: number;
	onDiscardStagedAppContext: () => void;
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

export const EditorArea = forwardRef<EditorAreaHandle, EditorAreaProps>(function EditorArea(
	{
		isGenerating,
		isInputLocked: inputLocked,
		isRunActive,
		submissionMode = 'message',
		currentProviderSDKType,
		shortcutConfig,
		onSubmit,
		onRequestStop,
		editingMessageId,
		cancelEditing,
		systemPrompt,
		showAutomationControl,
		automationHoldEnabled,
		onToggleAutomationHold,
		showPendingToolControls,
		canContinueWithoutDraft,
		onRunPendingTools,
		sessionToolCalls,
		sessionToolOutputs,
		onRunSessionToolCall,
		onRetrySessionToolCall,
		onSkipSessionToolCall,
		stagedAppContextUpdateCount,
		onDiscardStagedAppContext,
	},
	ref
) {
	const [externalMessageRestorePending, setExternalMessageRestorePending] = useState(false);
	const [externalMessageRestoreFailed, setExternalMessageRestoreFailed] = useState(false);
	const externalMessageRestoreVersionRef = useRef(0);
	const isInputLocked = inputLocked || externalMessageRestorePending;
	const hasContinuationToSend = submissionMode === 'continuation' && canContinueWithoutDraft && !editingMessageId;

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
		removedStandaloneAttachments,
		attachFiles: handleAttachFiles,
		attachDirectory: handleAttachDirectory,
		attachURL: handleAttachURL,
		attachPathsAsAttachments,
		changeAttachmentMode: handleChangeAttachmentContentBlockMode,
		removeAttachment: handleRemoveAttachment,
		removeDirectoryGroup: handleRemoveDirectoryGroup,
		removeDirectoryAttachments: handleRemoveDirectoryAttachments,
		restoreStandaloneAttachments: handleRestoreStandaloneAttachments,
		restoreDirectoryAttachments: handleRestoreDirectoryAttachments,
		removeOverflowDir: handleRemoveOverflowDir,
		applyAttachmentsDrop,
		clearAttachments,
		clearStandaloneAttachments: handleClearStandaloneAttachments,
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
	const [toolOutputs, setToolOutputs] = useState<UIToolOutput[]>([]);
	const [renderOnlySkillRefs, setRenderOnlySkillRefs] = useState<SkillRef[]>([]);

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
	const workspaceControlsLocked = isInputLocked || isSubmitting || workspaceSelectionPending;

	const {
		conversationToolsState,
		setConversationToolsState,
		webSearchTemplates,
		setWebSearchTemplates,
		toolArgsBlocked,
		recomputeAttachedToolArgsBlocked: handleAttachedToolsChanged,
		clearAttachedToolValidation,
		applyConversationToolsFromChoices,
		applyWebSearchFromChoices,
	} = useComposerToolConfig({
		getAttachedToolEntries: getAttachedToolEntriesSnapshot,
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
		toolSelectionIssues.length > 0 ||
		externalMessageRestoreFailed;

	const handleOpenConversationToolDetails = useCallback((entry: ConversationToolStateEntry) => {
		setToolDetailsState({ kind: 'choice', choice: entry.toolStoreChoice });
	}, []);

	const handleOpenAttachedToolDetails = useCallback((entry: AttachedToolEntry) => {
		setToolDetailsState({ kind: 'choice', choice: uiToolChoiceToToolStoreChoice(entry) });
	}, []);

	const attachedToolIdentityKeys = useMemo(() => {
		return new Set(attachedToolEntries.map(entry => toolIdentityKey(entry.target)));
	}, [attachedToolEntries]);

	const standaloneAttachments = useMemo(() => {
		const ownedAttachmentKeys = new Set<string>();
		for (const group of directoryGroups) {
			for (const attachmentKey of group.ownedAttachmentKeys) {
				ownedAttachmentKeys.add(attachmentKey);
			}
		}

		return attachments.filter(attachment => !ownedAttachmentKeys.has(uiAttachmentKey(attachment)));
	}, [attachments, directoryGroups]);
	const hasEditorChips =
		attachments.length > 0 ||
		removedStandaloneAttachments.length > 0 ||
		directoryGroups.length > 0 ||
		toolOutputs.length > 0 ||
		sessionToolCalls.length > 0 ||
		sessionToolOutputs.length > 0;

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
			handleDetachAttachedToolByKey(toolIdentityKey(entry.target));
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
			handleAttachedToolsChanged();
		},
		[editor, handleAttachedToolsChanged]
	);

	const preEditConversationToolsRef = useRef<ConversationToolStateEntry[] | null>(null);
	const preEditWebSearchTemplatesRef = useRef<WebSearchChoiceTemplate[] | null>(null);
	const preEditToolSelectionIssuesRef = useRef<ToolSelectionIssue[] | null>(null);
	const preEditMCPContextRef = useRef<MCPConversationContext | undefined | null>(null);
	const preEditEnabledSkillRefsRef = useRef<SkillRef[] | null>(null);
	const preEditActiveSkillRefsRef = useRef<SkillRef[] | null>(null);
	const preEditWorkspaceSelectionRef = useRef<WorkspaceConversationSelection | undefined | null>(null);

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
	}, [focusEditorPreservingSelection, templateMenuEl, templateMenuOpen]);

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
	}, [focusEditorPreservingSelection, toolMenuEl, toolMenuOpen]);

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
	}, [attachmentMenuEl, attachmentMenuOpen, focusEditorPreservingSelection]);

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
	}, [attachmentMenu, closeAllMenus, isInputLocked]);

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
		const previousConversationTools = preEditConversationToolsRef.current;
		const previousWebSearchTemplates = preEditWebSearchTemplatesRef.current;
		const previousMCPContext = preEditMCPContextRef.current;
		const previousEnabledSkills = preEditEnabledSkillRefsRef.current;
		const previousActiveSkills = preEditActiveSkillRefsRef.current;
		const previousWorkspace = preEditWorkspaceSelectionRef.current;

		if (previousConversationTools) {
			setConversationToolsState(previousConversationTools);
		}
		if (previousWebSearchTemplates) {
			setWebSearchTemplates(previousWebSearchTemplates);
		}
		if (preEditToolSelectionIssuesRef.current) {
			setToolSelectionIssues(preEditToolSelectionIssuesRef.current);
		}
		if (previousMCPContext !== null) {
			mcp.restoreContext(previousMCPContext ?? undefined);
		}
		if (previousEnabledSkills || previousActiveSkills) {
			void applySkillSelectionState(
				previousEnabledSkills ?? getCurrentEnabledSkillRefs(),
				previousActiveSkills ?? getCurrentActiveSkillRefs()
			);
		}
		if (previousWorkspace !== null) {
			void workspace.restoreSelection(previousWorkspace, false);
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
		if (isInputLocked || isSubmitting || workspaceExecutionBlocked || hasBlockingToolArgs) {
			return false;
		}

		if (hasText || hasContinuationToSend) {
			return true;
		}

		const hasAttachments = attachments.length > 0;
		const hasOutputs = toolOutputs.length > 0;
		const hasMCPContent = mcp.selectedServerCount > 0 || mcpAppContextUpdates.length > 0;

		return hasAttachments || hasOutputs || hasMCPContent;
	}, [
		attachments.length,
		hasBlockingToolArgs,
		hasContinuationToSend,
		hasText,
		isInputLocked,
		isSubmitting,
		mcp.selectedServerCount,
		mcpAppContextUpdates.length,
		toolOutputs.length,
		workspaceExecutionBlocked,
	]);

	const { formRef, onKeyDown } = useEnterSubmit({
		isBusy: isGenerating || isSubmitting || workspaceExecutionBlocked,
		canSubmit: () => isSendButtonEnabled && !isSubmittingRef.current,
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

			const editorElement = contentRef.current;
			if (!editorElement) {
				return;
			}

			event.preventDefault();
			event.stopPropagation();

			const direction = event.key === 'PageDown' ? 1 : -1;
			const delta = Math.max(120, editorElement.clientHeight * 0.85) * direction;
			const maxTop = Math.max(0, editorElement.scrollHeight - editorElement.clientHeight);
			editorElement.scrollTop = Math.max(0, Math.min(maxTop, editorElement.scrollTop + delta));
		},
		[contentRef, onKeyDown]
	);

	const setSubmitting = useCallback((next: boolean, options?: { deferVisual?: boolean }) => {
		isSubmittingRef.current = next;

		if (submitVisualFrameRef.current !== null) {
			window.cancelAnimationFrame(submitVisualFrameRef.current);
			submitVisualFrameRef.current = null;
		}

		if (next && options?.deferVisual) {
			submitVisualFrameRef.current = window.requestAnimationFrame(() => {
				submitVisualFrameRef.current = null;
				setIsSubmitting(current => (current === next ? current : next));
			});
			return;
		}

		setIsSubmitting(current => (current === next ? current : next));
	}, []);

	const handleTemplateRendered = useCallback((skillRef: SkillRef) => {
		setRenderOnlySkillRefs(previous =>
			normalizeSkillRefs([...previous, { rootID: skillRef.rootID, artifactID: skillRef.artifactID }])
		);
	}, []);

	const clearComposerTransientState = useCallback(() => {
		externalMessageRestoreVersionRef.current += 1;
		setExternalMessageRestorePending(false);
		setExternalMessageRestoreFailed(false);
		closeAllMenus();
		setSubmitError(null);
		setSubmitting(false);
		setToolDetailsState(null);
		setToolArgsTarget(null);
		clearAttachments();
		setToolOutputs([]);
		clearAttachedToolValidation();
		setMCPAppContextUpdates([]);
		setRenderOnlySkillRefs([]);
	}, [clearAttachedToolValidation, clearAttachments, closeAllMenus, setSubmitting]);

	const resetEditor = useCallback(() => {
		clearComposerTransientState();
		resetEditorDocument();
	}, [clearComposerTransientState, resetEditorDocument]);

	const doSubmit = useCallback(async () => {
		if (isSubmittingRef.current || isInputLocked || !isSendButtonEnabled) {
			return;
		}

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

		closeAllMenus();
		setSubmitError(null);
		setSubmitting(true);

		let didSend = false;
		let submittedToolChoices: ToolStoreChoice[] | null = null;

		try {
			const textToSend = editor.api.string([]);
			const finalToolOutputs = toolOutputs;
			const hasNonEmptyText = textToSend.trim().length > 0;
			const hasAttachmentsToSend = attachments.length > 0;
			const hasToolOutputsToSend = finalToolOutputs.length > 0;
			const hasMCPContentToSend = mcp.selectedServerCount > 0 || mcpAppContextUpdates.length > 0;

			if (
				!hasNonEmptyText &&
				!hasAttachmentsToSend &&
				!hasToolOutputsToSend &&
				!hasMCPContentToSend &&
				!hasContinuationToSend
			) {
				setSubmitError('Nothing to send. Add text, attachments, tool outputs, or MCP context first.');
				return;
			}

			const attachedTools = getAttachedTools(editor);
			const explicitChoices = attachedTools.map(choice => uiToolChoiceToToolStoreChoice(choice));
			const conversationChoices = conversationToolsToChoices(conversationToolsState);
			const webSearchChoices = buildWebSearchChoicesForSubmit(webSearchTemplates);
			const finalToolChoices = dedupeToolChoices([...explicitChoices, ...conversationChoices, ...webSearchChoices]);

			let preparedMCPContext: MCPConversationContext | undefined;
			try {
				preparedMCPContext = await mcp.prepareForSubmit();
			} catch (error) {
				setSubmitError(error instanceof Error ? error.message : 'Fill required MCP arguments before sending.');
				return;
			}

			const nonSessionSkillRefs = normalizeSkillRefs([
				...systemPrompt.selectedInstructionSkillRefs,
				...renderOnlySkillRefs,
			]);

			const payload: EditorSubmitPayload = {
				text: textToSend,
				additionalSessionInstructionText: systemPrompt.additionalSessionInstructionText.trim() || undefined,
				selectedSessionInstructionOverrides:
					systemPrompt.selectedSessionInstructionOverrides.length > 0
						? structuredClone(systemPrompt.selectedSessionInstructionOverrides)
						: undefined,
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
				activeSkillSelections: getCurrentActiveSkillSelections(),
				workspaceSelection: workspace.getSelectionSnapshot(),
				workspacePlan: workspace.plan,
				...(nonSessionSkillRefs.length > 0 ? { nonSessionSkillRefs } : {}),
				...(toolSelectionIssues.length > 0
					? {
							unresolvedToolSelections: structuredClone(toolSelectionIssues),
						}
					: {}),
				...(editingMessageId && finalToolOutputs.length > 0
					? {
							carryForwardToolResultIDs: [...new Set(finalToolOutputs.map(output => output.id).filter(Boolean))],
						}
					: {}),
			};

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

			if (didSend) {
				resetEditor();
				clearPreEditSnapshot();

				if (submittedToolChoices && submittedToolChoices.length > 0) {
					setConversationToolsState(previous =>
						mergeConversationToolsWithNewChoices(previous, submittedToolChoices as ToolStoreChoice[])
					);
				}
			}
		}
	}, [
		attachments,
		clearPreEditSnapshot,
		closeAllMenus,
		conversationToolsState,
		editingMessageId,
		editor,
		getCurrentActiveSkillSelections,
		getCurrentInstalledActiveSkillRefs,
		getCurrentInstalledEnabledSkillRefs,
		getCurrentInstalledEnabledSkillSelections,
		hasBlockingMCPArgs,
		hasBlockingToolArgs,
		hasContinuationToSend,
		isInputLocked,
		isSendButtonEnabled,
		mcp,
		mcpAppContextUpdates,
		onSubmit,
		renderOnlySkillRefs,
		resetEditor,
		setConversationToolsState,
		setSubmitting,
		systemPrompt,
		toolOutputs,
		toolSelectionIssues,
		webSearchTemplates,
		workspace,
		workspaceExecutionBlocked,
	]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		void doSubmit();
	};

	const loadExternalMessage = useCallback(
		(incoming: EditorExternalMessage) => {
			if (externalMessageLoadReleaseTimerRef.current !== null) {
				window.clearTimeout(externalMessageLoadReleaseTimerRef.current);
				externalMessageLoadReleaseTimerRef.current = null;
			}

			isLoadingExternalMessageRef.current = true;

			try {
				// External MCP/App drafts are not message edits.
				if (incoming.isEditing && !preEditConversationToolsRef.current) {
					preEditConversationToolsRef.current = conversationToolsState;
				}
				if (incoming.isEditing && !preEditWebSearchTemplatesRef.current) {
					preEditWebSearchTemplatesRef.current = webSearchTemplates;
				}
				if (incoming.isEditing && !preEditToolSelectionIssuesRef.current) {
					preEditToolSelectionIssuesRef.current = toolSelectionIssues;
				}
				if (incoming.isEditing && preEditMCPContextRef.current === null) {
					preEditMCPContextRef.current = mcp.mcpContext;
				}
				if (incoming.isEditing && !preEditEnabledSkillRefsRef.current) {
					preEditEnabledSkillRefsRef.current = getCurrentEnabledSkillRefs();
				}
				if (incoming.isEditing && !preEditActiveSkillRefsRef.current) {
					preEditActiveSkillRefsRef.current = getCurrentActiveSkillRefs();
				}
				if (incoming.isEditing && preEditWorkspaceSelectionRef.current === null) {
					preEditWorkspaceSelectionRef.current = workspace.getSelectionSnapshot();
				}

				clearComposerTransientState();

				replaceEditorDocument(buildEditorValueFromPlainText(incoming.text ?? ''), 'end');
				loadAttachmentsFromMessage(incoming.attachments);

				const replaceContext = incoming.isEditing === true;
				if (replaceContext || incoming.toolChoices !== undefined) {
					const choices = incoming.toolChoices ?? [];
					applyConversationToolsFromChoices(choices);
					applyWebSearchFromChoices(choices);
					setToolSelectionIssues(incoming.toolSelectionIssues ?? []);
				}
				if (replaceContext || incoming.mcpContext !== undefined) {
					mcp.restoreContext(incoming.mcpContext);
				}
				setMCPAppContextUpdates(incoming.mcpAppContextUpdates ?? []);

				if (replaceContext || incoming.enabledSkillRefs !== undefined || incoming.activeSkillRefs !== undefined) {
					void applySkillSelectionState(incoming.enabledSkillRefs ?? [], incoming.activeSkillRefs ?? [], {
						syncSession: SkillSessionSyncMode.None,
						forceResetSession: true,
					});
				}
				if (replaceContext || incoming.workspaceSelection !== undefined) {
					void workspace.restoreSelection(incoming.workspaceSelection, false);
				}
				setToolOutputs(incoming.toolOutputs ?? []);

				const configuration = incoming.sessionConfiguration;
				if (configuration) {
					const restoreVersion = externalMessageRestoreVersionRef.current;
					setExternalMessageRestorePending(true);

					void Promise.all([
						incoming.toolChoices === undefined
							? hydrateSessionToolOverrides(configuration.tools)
							: Promise.resolve({
									choices: incoming.toolChoices,
									issues: incoming.toolSelectionIssues ?? [],
								}),
						hydrateSessionMCPOverrides(configuration.mcp),
					])
						.then(([tools, context]) => {
							if (externalMessageRestoreVersionRef.current !== restoreVersion) {
								return;
							}

							applyConversationToolsFromChoices(tools.choices);
							applyWebSearchFromChoices(tools.choices);
							setToolSelectionIssues(tools.issues);
							mcp.restoreContext(context ?? incoming.mcpContext);
						})
						.catch((error: unknown) => {
							if (externalMessageRestoreVersionRef.current !== restoreVersion) {
								return;
							}

							setExternalMessageRestoreFailed(true);
							setSubmitError(
								`The original message settings could not be restored. Cancel editing and reopen the message. ${
									error instanceof Error ? error.message : ''
								}`.trim()
							);
						})
						.finally(() => {
							if (externalMessageRestoreVersionRef.current === restoreVersion) {
								setExternalMessageRestorePending(false);
							}
						});
				}
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
			if (text.trim().length === 0 || hasTextRef.current) {
				return false;
			}

			setDraftText(text);
			return true;
		},
		[hasTextRef, setDraftText]
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

		if (isLoadingExternalMessageRef.current) {
			return;
		}

		const hasTextNow = hasTextRef.current;
		const hasAttachmentCandidates =
			attachments.length > 0 || removedStandaloneAttachments.length > 0 || directoryGroups.length > 0;
		const hasToolOutputsLocal = toolOutputs.length > 0;
		const isEffectivelyEmpty = !hasTextNow && !hasAttachmentCandidates && !hasToolOutputsLocal;

		if (editingMessageId && isEffectivelyEmpty) {
			restorePreEditContext();
			cancelEditing();
		}
	}, [
		attachments.length,
		cancelEditing,
		directoryGroups.length,
		editingMessageId,
		handleAttachedToolsChanged,
		hasTextRef,
		onEditorChange,
		removedStandaloneAttachments.length,
		restorePreEditContext,
		submitError,
		toolOutputs.length,
	]);

	const restoreSessionConfiguration = useCallback(
		async (configuration: SessionConfiguration) => {
			// Invalidate an older edit hydration before restoring a Session.
			externalMessageRestoreVersionRef.current += 1;
			setExternalMessageRestorePending(false);
			setExternalMessageRestoreFailed(false);
			setSubmitError(null);
			setToolOutputs([]);
			clearAttachedToolValidation();
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
			clearAttachedToolValidation,
			clearAttachments,
			mcp,
			workspace,
		]
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
			resetEditor: () => {
				resetEditor();
				clearPreEditSnapshot();
			},
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

						setMCPAppContextUpdates(previous => [...previous, update]);
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
			restoreSessionConfiguration,
		}),
		[
			applyConversationToolsFromChoices,
			applyAttachmentsDrop,
			applyInstalledSkillSelectionState,
			applySkillSelectionState,
			applyWebSearchFromChoices,
			clearPreEditSnapshot,
			focusEditorAtEnd,
			loadExternalMessage,
			mcp,
			onRequestStop,
			isRunActive,
			openAttachmentPicker,
			openMCPPicker,
			openSkillsPicker,
			openTemplatePicker,
			openToolPicker,
			openWorkspacePicker,
			resetEditor,
			restoreSessionConfiguration,
			setDraftText,
			setDraftTextIfEmpty,
			workspace,
		]
	);

	const handleCancelEditing = useCallback(() => {
		resetEditor();
		restorePreEditContext();
		cancelEditing();
	}, [cancelEditing, resetEditor, restorePreEditContext]);

	const stopResponseShortcut = formatShortcut(shortcutConfig.stopResponse);
	const sendActionText = submissionMode === 'continuation' ? 'Send and continue Run' : 'Send message';

	const erroredToolOutputsReadyToSubmit =
		toolOutputs.some(output => output.isError && !output.isSkipped) && !hasBlockingToolArgs && !isInputLocked;

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
						<span>A carried Tool result contains an error. Submit it to the model or remove it before sending.</span>
					</output>
				) : null}

				<Plate editor={editor} onChange={handleEditorDocumentChange}>
					<div className="bg-base-100 border-base-200 flex max-h-full min-h-0 w-full max-w-full min-w-0 flex-[1_1_auto] overflow-hidden rounded-2xl border">
						<div className="flex min-h-0 min-w-0 grow flex-col p-0">
							{editingMessageId ? (
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
							) : null}

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

							<div className="w-full min-w-0 shrink-0 items-center overflow-x-auto overscroll-contain p-1 text-xs">
								{hasEditorChips ? (
									<div
										className="app-scrollbar-thin flex w-max shrink-0 flex-nowrap items-center gap-1 p-1"
										style={{ scrollbarGutter: 'stable' }}
									>
										{/* Aggregated chip for standalone attachments */}
										<StandaloneAttachmentsChip
											attachments={standaloneAttachments}
											removedAttachments={removedStandaloneAttachments}
											onRemoveAttachment={handleRemoveAttachment}
											onRestoreAttachments={handleRestoreStandaloneAttachments}
											onClearAttachments={handleClearStandaloneAttachments}
											onChangeAttachmentContentBlockMode={handleChangeAttachmentContentBlockMode}
										/>

										{/* Folder groups */}
										{directoryGroups.map(group => (
											<DirectoryChip
												key={group.id}
												group={group}
												attachments={attachments}
												onChangeAttachmentContentBlockMode={handleChangeAttachmentContentBlockMode}
												onRemoveDirectoryGroup={handleRemoveDirectoryGroup}
												onRemoveDirectoryAttachments={handleRemoveDirectoryAttachments}
												onRestoreDirectoryAttachments={handleRestoreDirectoryAttachments}
												onRemoveOverflowDir={handleRemoveOverflowDir}
											/>
										))}

										<SessionToolActivityChips
											runtimeCalls={sessionToolCalls}
											runtimeOutputs={sessionToolOutputs}
											draftOutputs={toolOutputs}
											onRunCall={onRunSessionToolCall}
											onRetryCall={onRetrySessionToolCall}
											onSkipCall={onSkipSessionToolCall}
											onInspectCall={call => {
												setToolDetailsState({ kind: 'call', call });
											}}
											onInspectOutput={output => {
												setToolDetailsState({ kind: 'output', output });
											}}
											onRemoveDraftOutput={output => {
												setToolOutputs(previous => previous.filter(candidate => candidate.id !== output.id));
											}}
										/>
									</div>
								) : null}
							</div>
						</div>

						<div className="flex shrink-0 flex-col items-end justify-end gap-2 p-1">
							{showPendingToolControls ? (
								<>
									<HoverTip content="Run pending Session tools" placement="left">
										<button
											type="button"
											className="btn btn-circle btn-neutral btn-sm shrink-0"
											onClick={() => {
												onRunPendingTools(false);
											}}
											aria-label="Run pending Session tools"
										>
											<FiPlay size={18} />
										</button>
									</HoverTip>

									<HoverTip content="Run pending Session tools and continue the Run" placement="left">
										<button
											type="button"
											className="btn btn-circle btn-primary btn-sm shrink-0"
											onClick={() => {
												onRunPendingTools(true);
											}}
											aria-label="Run pending Session tools and continue"
										>
											<FiFastForward size={18} />
										</button>
									</HoverTip>
								</>
							) : null}

							{showAutomationControl ? (
								<HoverTip
									content={
										automationHoldEnabled
											? 'Resume automatic tools and continuation'
											: 'Hold the next automatic tool and continuation batch'
									}
									placement="left"
								>
									<button
										type="button"
										className={`btn btn-circle btn-sm shrink-0 ${
											automationHoldEnabled ? 'btn-warning' : 'btn-neutral'
										}`}
										onClick={onToggleAutomationHold}
										aria-label={automationHoldEnabled ? 'Resume automatic tools' : 'Hold automatic tools'}
									>
										<FiZapOff size={18} />
									</button>
								</HoverTip>
							) : null}

							{stagedAppContextUpdateCount > 0 ? (
								<HoverTip
									content={`Discard ${stagedAppContextUpdateCount} staged MCP App context update${
										stagedAppContextUpdateCount === 1 ? '' : 's'
									}`}
									placement="left"
								>
									<button
										type="button"
										className="btn btn-circle btn-info btn-sm shrink-0"
										onClick={onDiscardStagedAppContext}
										aria-label="Discard staged MCP App context"
									>
										<FiTrash2 size={16} />
									</button>
								</HoverTip>
							) : null}

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
								<HoverTip content={sendActionText} placement="left">
									<button
										type="button"
										className={`btn btn-circle btn-neutral btn-sm shrink-0 ${
											!isSendButtonEnabled ? 'btn-disabled' : ''
										}`}
										disabled={!isSendButtonEnabled}
										onClick={() => {
											if (!isSendButtonEnabled) {
												return;
											}
											void doSubmit();
										}}
										aria-label={sendActionText}
									>
										<FiSend size={18} />
									</button>
								</HoverTip>
							)}
						</div>
					</div>

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
						onTemplateRendered={handleTemplateRendered}
						attachments={attachments}
						directoryGroups={directoryGroups}
						removedStandaloneAttachments={removedStandaloneAttachments}
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
