import type { RefObject } from 'react';
import { forwardRef, memo, useCallback, useEffect, useImperativeHandle, useRef, useState } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { RestorableConversationContext } from '@/spec/conversation';
import type { UIToolCall } from '@/spec/inference';
import type { MCPAppModelContextUpdate } from '@/spec/mcp';
import type { UIChatOption } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import type {
	AssistantPresetPreparedApplication,
	AssistantPresetRuntimeSnapshot,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import {
	areAssistantRuntimeSnapshotsEqual,
	buildAssistantPresetIdentityKey,
	EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { useAssistantPresetManager } from '@/chats/composer/assistantpresets/use_assistant_preset_manager';
import { EditorContextBar } from '@/chats/composer/contextarea/context_bar';
import { useAssistantContextState } from '@/chats/composer/contextarea/use_context_state';
import type { EditorAreaHandle } from '@/chats/composer/editor/editor_area';
import { EditorArea } from '@/chats/composer/editor/editor_area';
import type {
	AssistantTurnFinishedPayload,
	EditorExternalMessage,
	EditorSubmitPayload,
} from '@/chats/composer/editor/editor_types';
import { useComposerSystemPrompt } from '@/chats/composer/skills/use_composer_system_prompt';
import type { ChatWorkflowStarter, ChatWorkflowStarterAssistantPresetRef } from '@/chats/conversation/starter_intent';

export interface ComposerBoxHandle {
	getUIChatOptions: () => UIChatOption;
	focus: () => void;
	resetEditor: () => void;
	resetForNewConversation: () => Promise<void>;
	openTemplateMenu: () => void;
	openToolMenu: () => void;
	openAttachmentMenu: () => void;
	openSystemPromptMenu: () => void;
	openSkillsMenu: () => void;
	openMCPMenu: () => void;
	requestStopResponse: () => void;
	loadWorkflowStarter: (starter: ChatWorkflowStarter) => Promise<boolean>;
	loadExternalMessage: (msg: EditorExternalMessage) => void;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	finishAssistantTurn: (payload: AssistantTurnFinishedPayload) => void;
	setConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	setWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
	appendMCPAppContextUpdate: (update: MCPAppModelContextUpdate) => void;
	applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => void;
	setSkillStateFromMessage: (
		enabledRefs: SkillRef[],
		activeRefs: SkillRef[],
		options?: {
			syncSession?: 'none' | 'if-session-exists' | 'ensure-if-enabled';
			forceResetSession?: boolean;
		}
	) => void;
	restoreConversationContext: (context: RestorableConversationContext) => void;
}

interface ComposerBoxProps {
	onSend: (message: EditorSubmitPayload, options: UIChatOption) => Promise<void>;
	isBusy: boolean;
	isHydrating: boolean;
	abortRef: RefObject<AbortController | null>;
	shortcutConfig: ShortcutConfig;
	editingMessageId: string | null;
	onCancelEditing: () => void;
}

const ComposerBoxImpl = forwardRef<ComposerBoxHandle, ComposerBoxProps>(function ComposerBox(
	{ onSend, isBusy, isHydrating, abortRef, shortcutConfig, editingMessageId, onCancelEditing },
	ref
) {
	const [abortConfirmationRequested, setAbortConfirmationRequested] = useState(false);
	const isGenerating = isBusy;
	const isInputLocked = isGenerating || isHydrating;

	const editorAreaRef = useRef<EditorAreaHandle>(null);
	const assistantContext = useAssistantContextState();
	const systemPrompt = useComposerSystemPrompt({
		modelDefaultPrompt: assistantContext.selectedModel.systemPrompt,
		modelOptionsLoaded: assistantContext.modelOptionsLoaded,
	});
	const chatOptions = assistantContext.chatOptions;
	const showAbortModal = isGenerating && abortConfirmationRequested;

	const [assistantRuntimeSnapshot, setAssistantRuntimeSnapshot] = useState(EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT);
	const pendingPresetResolutionModeRef = useRef<'none' | 'ensure-active' | 'track-default'>('ensure-active');
	const pendingStarterAssistantPresetRef = useRef<ChatWorkflowStarterAssistantPresetRef | null>(null);

	const replaceAssistantRuntimeSnapshot = useCallback((next: AssistantPresetRuntimeSnapshot) => {
		setAssistantRuntimeSnapshot(prev => {
			return areAssistantRuntimeSnapshotsEqual(prev, next) ? prev : next;
		});
	}, []);

	const updateAssistantRuntimeSnapshot = useCallback(
		(updater: (prev: AssistantPresetRuntimeSnapshot) => AssistantPresetRuntimeSnapshot) => {
			setAssistantRuntimeSnapshot(prev => {
				const next = updater(prev);
				return areAssistantRuntimeSnapshotsEqual(prev, next) ? prev : next;
			});
		},
		[]
	);

	const applyAssistantPresetRuntimeSelections = useCallback(
		(prepared: AssistantPresetPreparedApplication) => {
			if (prepared.hasStartingTextSelection && prepared.nextStartingText.trim().length > 0) {
				editorAreaRef.current?.setDraftTextIfEmpty(prepared.nextStartingText);
			}

			if (prepared.runtimeSelections.hasToolsSelection) {
				editorAreaRef.current?.setConversationToolsFromChoices(prepared.runtimeSelections.conversationToolChoices);
				editorAreaRef.current?.setWebSearchFromChoices(prepared.runtimeSelections.webSearchChoices);
			}

			if (prepared.runtimeSelections.hasSkillsSelection) {
				editorAreaRef.current?.setSkillStateFromMessage(
					prepared.runtimeSelections.enabledSkillRefs,
					prepared.runtimeSelections.activeSkillRefs,
					{ syncSession: 'ensure-if-enabled' }
				);
			}

			if (prepared.runtimeSelections.hasMCPSelection) {
				editorAreaRef.current?.setMCPContextFromMessage(prepared.runtimeSelections.mcpContext);
			}

			updateAssistantRuntimeSnapshot(prev => ({
				conversationToolChoices: prepared.runtimeSelections.hasToolsSelection
					? [...prepared.runtimeSelections.conversationToolChoices]
					: prev.conversationToolChoices,
				webSearchChoices: prepared.runtimeSelections.hasToolsSelection
					? [...prepared.runtimeSelections.webSearchChoices]
					: prev.webSearchChoices,
				enabledSkillRefs: prepared.runtimeSelections.hasSkillsSelection
					? [...prepared.runtimeSelections.enabledSkillRefs]
					: prev.enabledSkillRefs,
				mcpContext: prepared.runtimeSelections.hasMCPSelection
					? prepared.runtimeSelections.mcpContext
					: prev.mcpContext,
			}));
		},
		[updateAssistantRuntimeSnapshot]
	);

	const assistantPreset = useAssistantPresetManager({
		context: assistantContext,
		systemPrompt,
		runtimeSnapshot: assistantRuntimeSnapshot,
		applyRuntimeSelections: applyAssistantPresetRuntimeSelections,
	});

	const assistantPresetLayerReady =
		assistantContext.modelOptionsLoaded && !assistantPreset.loading && !assistantPreset.isApplying;
	const { ensureActivePreset, resetToBasePreset, selectPreset, trackDefaultPresetWithoutApplying } = assistantPreset;

	const queueOrApplyAssistantPresetRef = useCallback(
		async (presetRef: ChatWorkflowStarterAssistantPresetRef): Promise<boolean> => {
			if (!assistantPresetLayerReady) {
				pendingStarterAssistantPresetRef.current = presetRef;
				pendingPresetResolutionModeRef.current = 'none';
				return true;
			}

			const presetKey = buildAssistantPresetIdentityKey(
				presetRef.bundleID,
				presetRef.assistantPresetSlug,
				presetRef.assistantPresetVersion
			);
			return selectPreset(presetKey);
		},
		[assistantPresetLayerReady, selectPreset]
	);

	const flushPendingPresetResolution = useCallback(async () => {
		if (!assistantPresetLayerReady) {
			return false;
		}
		const pendingStarterAssistantPreset = pendingStarterAssistantPresetRef.current;
		if (pendingStarterAssistantPreset) {
			const ok = await queueOrApplyAssistantPresetRef(pendingStarterAssistantPreset);
			if (pendingStarterAssistantPresetRef.current === pendingStarterAssistantPreset) {
				pendingStarterAssistantPresetRef.current = null;
			}
			return ok;
		}

		if (pendingPresetResolutionModeRef.current === 'track-default') {
			const ok = await trackDefaultPresetWithoutApplying();

			if (ok) {
				pendingPresetResolutionModeRef.current = 'none';
			}
			return ok;
		}

		if (pendingPresetResolutionModeRef.current === 'ensure-active') {
			const ok = await ensureActivePreset();

			if (ok) {
				pendingPresetResolutionModeRef.current = 'none';
			}
			return ok;
		}

		return true;
	}, [
		assistantPresetLayerReady,
		ensureActivePreset,
		queueOrApplyAssistantPresetRef,
		trackDefaultPresetWithoutApplying,
	]);

	// Only re-run when readiness flips. All other triggers (workflow load,
	// restore, reset, etc.) call flushPendingPresetResolution() explicitly.
	useEffect(() => {
		if (!assistantPresetLayerReady) {
			return;
		}
		void flushPendingPresetResolution();
	}, [assistantPresetLayerReady, flushPendingPresetResolution]);

	const handleSubmitMessage = useCallback(
		(payload: EditorSubmitPayload) => {
			// Clear any stale abort confirmation request before starting a new send.
			setAbortConfirmationRequested(false);

			// Return the promise so <EditorArea /> can await it and surface
			// any synchronous errors from sendMessage.
			return onSend(payload, chatOptions);
		},
		[chatOptions, onSend]
	);

	const resetComposerStateForNewConversation = useCallback(() => {
		setAbortConfirmationRequested(false);
		pendingPresetResolutionModeRef.current = 'none';
		pendingStarterAssistantPresetRef.current = null;
		editorAreaRef.current?.resetEditor();
		editorAreaRef.current?.setConversationToolsFromChoices([]);
		editorAreaRef.current?.setWebSearchFromChoices([]);
		editorAreaRef.current?.clearMCPContext();

		editorAreaRef.current?.setSkillStateFromMessage([], [], { syncSession: 'none', forceResetSession: true });
		replaceAssistantRuntimeSnapshot(EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT);

		const nextSelectedModel = assistantContext.resetForNewConversation();
		systemPrompt.resetForNewConversation(nextSelectedModel.systemPrompt);
		return nextSelectedModel;
	}, [assistantContext, replaceAssistantRuntimeSnapshot, systemPrompt]);

	useImperativeHandle(
		ref,
		() => ({
			getUIChatOptions: () => chatOptions,
			focus: () => {
				editorAreaRef.current?.focus();
			},
			resetEditor: () => {
				editorAreaRef.current?.resetEditor();
			},
			resetForNewConversation: async () => {
				resetComposerStateForNewConversation();
				const ok = await resetToBasePreset();
				if (!ok) {
					pendingPresetResolutionModeRef.current = 'ensure-active';
					void flushPendingPresetResolution();
				}
			},

			openTemplateMenu: () => {
				editorAreaRef.current?.openTemplateMenu();
			},
			openToolMenu: () => {
				editorAreaRef.current?.openToolMenu();
			},
			openAttachmentMenu: () => {
				editorAreaRef.current?.openAttachmentMenu();
			},
			openSystemPromptMenu: () => {
				editorAreaRef.current?.openSystemPromptMenu();
			},
			openSkillsMenu: () => {
				editorAreaRef.current?.openSkillsMenu();
			},
			openMCPMenu: () => {
				editorAreaRef.current?.openMCPMenu();
			},
			requestStopResponse: () => {
				editorAreaRef.current?.requestStopResponse();
			},
			loadWorkflowStarter: async starter => {
				resetComposerStateForNewConversation();
				editorAreaRef.current?.setDraftText(starter.draft ?? '');

				if (starter.assistantPreset) {
					const ok = await queueOrApplyAssistantPresetRef(starter.assistantPreset);
					void flushPendingPresetResolution();
					editorAreaRef.current?.focus();
					return ok;
				}

				const ok = await resetToBasePreset();
				if (!ok) {
					pendingPresetResolutionModeRef.current = 'ensure-active';
				}
				void flushPendingPresetResolution();
				editorAreaRef.current?.focus();
				return true;
			},
			loadExternalMessage: msg => {
				editorAreaRef.current?.loadExternalMessage(msg);
			},
			loadToolCalls: toolCalls => {
				editorAreaRef.current?.loadToolCalls(toolCalls);
			},
			finishAssistantTurn: payload => {
				editorAreaRef.current?.finishAssistantTurn(payload);
			},
			applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => {
				editorAreaRef.current?.applyAttachmentsDrop(payload);
			},
			setConversationToolsFromChoices: tools => {
				editorAreaRef.current?.setConversationToolsFromChoices(tools);
				updateAssistantRuntimeSnapshot(prev => ({
					...prev,
					conversationToolChoices: [...tools],
				}));
			},
			setWebSearchFromChoices: choices => {
				editorAreaRef.current?.setWebSearchFromChoices(choices);
				updateAssistantRuntimeSnapshot(prev => ({
					...prev,
					webSearchChoices: [...choices],
				}));
			},
			appendMCPAppContextUpdate: update => {
				editorAreaRef.current?.appendMCPAppContextUpdate(update);
			},
			setSkillStateFromMessage: (enabledRefs, activeRefs, options) => {
				editorAreaRef.current?.setSkillStateFromMessage(enabledRefs, activeRefs, options);
				updateAssistantRuntimeSnapshot(prev => ({
					...prev,
					enabledSkillRefs: [...enabledRefs],
				}));
			},
			restoreConversationContext: context => {
				setAbortConfirmationRequested(false);
				const nextSelectedModel = assistantContext.restoreConversationContext(context);
				systemPrompt.restoreConversationContext(nextSelectedModel?.systemPrompt ?? '', context.modelParam);
				editorAreaRef.current?.setConversationToolsFromChoices(context.toolChoices);
				editorAreaRef.current?.setWebSearchFromChoices(context.webSearchChoices);
				editorAreaRef.current?.setMCPContextFromMessage(context.mcpContext);
				editorAreaRef.current?.setMCPAppContextUpdatesFromMessage(context.mcpAppContextUpdates);
				editorAreaRef.current?.setSkillStateFromMessage(context.enabledSkillRefs, context.activeSkillRefs, {
					syncSession: 'none',
					forceResetSession: true,
				});
				replaceAssistantRuntimeSnapshot({
					conversationToolChoices: [...context.toolChoices],
					webSearchChoices: [...context.webSearchChoices],
					enabledSkillRefs: [...context.enabledSkillRefs],
					mcpContext: context.mcpContext,
				});
				pendingPresetResolutionModeRef.current = 'track-default';
				void flushPendingPresetResolution();
			},
		}),
		[
			assistantContext,
			chatOptions,
			flushPendingPresetResolution,
			queueOrApplyAssistantPresetRef,
			replaceAssistantRuntimeSnapshot,
			resetToBasePreset,
			resetComposerStateForNewConversation,
			systemPrompt,
			updateAssistantRuntimeSnapshot,
		]
	);

	return (
		<div className="bg-base-200 flex max-h-128 w-full min-w-0 flex-col overflow-hidden">
			<div className="shrink-0">
				<EditorContextBar context={assistantContext} assistantPreset={assistantPreset} systemPrompt={systemPrompt} />
			</div>

			<DeleteConfirmationModal
				isOpen={showAbortModal}
				onClose={() => {
					setAbortConfirmationRequested(false);
				}}
				onConfirm={() => {
					setAbortConfirmationRequested(false);
					abortRef.current?.abort();
				}}
				title="Abort generation?"
				message="Partial answer that has already been received will stay in the chat. Do you want to stop the request?"
				confirmButtonText="Abort"
			/>

			<div className="min-h-0 flex-1 overflow-hidden">
				<EditorArea
					ref={editorAreaRef}
					isGenerating={isGenerating}
					isInputLocked={isInputLocked}
					currentProviderSDKType={chatOptions.providerSDKType}
					shortcutConfig={shortcutConfig}
					onSubmit={handleSubmitMessage}
					onRequestStop={() => {
						if (isGenerating) {
							setAbortConfirmationRequested(true);
						}
					}}
					onAssistantPresetRuntimeStateChange={replaceAssistantRuntimeSnapshot}
					editingMessageId={editingMessageId}
					cancelEditing={onCancelEditing}
					systemPrompt={systemPrompt}
				/>
			</div>
		</div>
	);
});

export const ComposerBox = memo(ComposerBoxImpl);
