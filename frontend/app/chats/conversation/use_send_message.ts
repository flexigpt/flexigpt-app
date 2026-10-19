import type { RefObject } from 'react';
import { useCallback } from 'react';

import type { UIModelOption } from '@/spec/model';
import type { SessionConfiguration } from '@/spec/session';
import { RoleEnum } from '@/spec/inference';
import { SessionRunState } from '@/spec/session_base';

import { generateTitle } from '@/lib/title_utils';

import { sessionManagementAPI } from '@/apis/baseapi';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import type { ChatTabState } from '@/chats/tabs/tabs_model';
import {
	requireAcceptedSessionCommand,
	selectedSessionRun,
	sessionCommandMeta,
} from '@/chats/session/session_commands';
import { buildSessionSubmissionIntent } from '@/chats/session/session_intent';

interface UseSendMessageArgs {
	tabsRef: RefObject<ChatTabState[]>;
	updateTab: (tabId: string, updater: (tab: ChatTabState) => ChatTabState) => void;
	scrollTabToBottomSoon: (tabId: string) => void;
	tabExists: (tabId: string) => boolean;
	inputRefs: RefObject<Map<string, ComposerBoxHandle | null>>;
}

type SessionSubmissionResult = SessionConfiguration | undefined;

export function useSendMessage({
	tabsRef,
	updateTab,
	scrollTabToBottomSoon,
	tabExists,
	inputRefs,
}: UseSendMessageArgs) {
	const updateStreamingMessage = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIModelOption): Promise<SessionSubmissionResult> => {
			if (!tabExists(tabId)) {
				return undefined;
			}

			if (!options.model) {
				throw new Error('Select an available Artifact-backed model before sending.');
			}

			const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);
			if (!tab) {
				return undefined;
			}

			try {
				const submission = await buildSessionSubmissionIntent(payload, options);

				if (!tab.sessionID) {
					const title = tab.manualTitleLocked
						? tab.conversation.title
						: generateTitle(payload.text || tab.conversation.title).title;
					const created = requireAcceptedSessionCommand(
						await sessionManagementAPI.createSession({
							meta: sessionCommandMeta(),
							title,
							configuration: submission.configuration,
						}),
						'Create Session'
					);

					if (!created.sessionID || created.sessionRevision === undefined) {
						throw new Error('Session Runtime accepted creation without returning a Session identity.');
					}

					updateTab(tabId, current => ({
						...current,
						sessionID: created.sessionID,
						composerConfigurationRestored: true,
						isPersisted: true,
					}));

					const started = requireAcceptedSessionCommand(
						await sessionManagementAPI.startRun({
							meta: sessionCommandMeta(created.sessionRevision),
							sessionID: created.sessionID,
							input: submission.input,
						}),
						'Start Run'
					);
					updateTab(tabId, current => ({
						...current,
						activeRunID: started.runID,
						pendingRunID: started.runID,
						isBusy: true,
						runtimeError: undefined,
					}));
					scrollTabToBottomSoon(tabId);
					return submission.configuration;
				}

				const snapshot = await sessionManagementAPI.getSessionSnapshot(tab.sessionID);

				if (
					snapshot.runs.some(
						run =>
							run.state !== SessionRunState.Completed &&
							run.state !== SessionRunState.Failed &&
							run.state !== SessionRunState.Cancelled
					)
				) {
					throw new Error('This Session already has an active Run.');
				}

				const started = requireAcceptedSessionCommand(
					await sessionManagementAPI.startRun({
						meta: sessionCommandMeta(snapshot.session.revision),
						sessionID: tab.sessionID,
						input: submission.input,
						configuration: submission.configuration,
					}),
					'Start Run'
				);
				updateTab(tabId, current => ({
					...current,
					activeRunID: started.runID,
					pendingRunID: started.runID,
					isBusy: true,
					runtimeError: undefined,
				}));
				scrollTabToBottomSoon(tabId);
				return submission.configuration;
			} catch (error) {
				updateTab(tabId, current => ({
					...current,
					isBusy: false,
					pendingRunID: undefined,
					runtimeError: error instanceof Error ? error.message : 'Session submission failed.',
				}));
				throw error;
			}
		},
		[scrollTabToBottomSoon, tabExists, tabsRef, updateTab]
	);

	const stageContinuationForTab = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIModelOption): Promise<SessionSubmissionResult> => {
			const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);

			if (!tab) {
				return undefined;
			}
			if (!options.model) {
				throw new Error('Select an available Artifact-backed model before continuing this Run.');
			}
			if (!tab.sessionID) {
				throw new Error('A Session is required before continuation input can be staged.');
			}
			if (payload.toolOutputs.length > 0) {
				throw new Error(
					'Tool results are owned by the active Session Run and cannot be staged from the Composer continuation input.'
				);
			}

			try {
				const snapshot = tab.sessionSnapshot ?? (await sessionManagementAPI.getSessionSnapshot(tab.sessionID));
				const run = selectedSessionRun(snapshot, tab.activeRunID);

				if (!run || run.state !== SessionRunState.Waiting) {
					throw new Error('The active Run is not waiting for continuation input.');
				}

				const submission = await buildSessionSubmissionIntent(payload, options);
				const stagedAttachments = [...(run.continuation.attachments ?? []), ...(submission.input.attachments ?? [])];
				const stagedAppContext = [...(run.continuation.mcpAppContext ?? []), ...(submission.input.mcpAppContext ?? [])];

				const continuation = {
					...run.continuation,
					...(submission.input.text?.trim() ? { text: submission.input.text } : {}),
					...(stagedAttachments.length > 0 ? { attachments: stagedAttachments } : {}),
					...(stagedAppContext.length > 0 ? { mcpAppContext: stagedAppContext } : {}),
					configuration: submission.configuration,
				};

				const staged = requireAcceptedSessionCommand(
					await sessionManagementAPI.updateContinuationInput({
						meta: sessionCommandMeta(snapshot.session.revision, run.revision),
						sessionID: tab.sessionID,
						runID: run.id,
						expectedInputRevision: run.continuation.revision,
						input: continuation,
					}),
					'Stage continuation input'
				);

				requireAcceptedSessionCommand(
					await sessionManagementAPI.continueRun({
						meta: sessionCommandMeta(
							staged.sessionRevision ?? snapshot.session.revision,
							staged.runRevision ?? run.revision
						),
						sessionID: tab.sessionID,
						runID: run.id,
					}),
					'Continue Run'
				);

				updateTab(tabId, current => ({
					...current,
					isBusy: true,
					runtimeError: undefined,
				}));
				scrollTabToBottomSoon(tabId);
				return submission.configuration;
			} catch (error) {
				updateTab(tabId, current => ({
					...current,
					runtimeError:
						error instanceof Error
							? error.message
							: 'The continuation input could not be accepted by the Session Runtime.',
				}));
				throw error;
			}
		},
		[scrollTabToBottomSoon, tabsRef, updateTab]
	);

	const sendMessageForTab = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIModelOption): Promise<SessionSubmissionResult> => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) {
				return undefined;
			}
			if (tab.isBusy || tab.isHydrating) {
				return undefined;
			}

			if (tab.editingMessageId) {
				if (!tab.sessionID || !tab.sessionSnapshot) {
					throw new Error('Edit-and-resend requires an existing Session snapshot.');
				}
				const submission = await buildSessionSubmissionIntent(payload, options);
				const resent = requireAcceptedSessionCommand(
					await sessionManagementAPI.editAndResend({
						meta: sessionCommandMeta(tab.sessionSnapshot.session.revision),
						sessionID: tab.sessionID,
						replaceMessageID: tab.editingMessageId,
						input: submission.input,
						configuration: submission.configuration,
						...(payload.carryForwardToolResultIDs?.length
							? {
									carryForwardToolResultIDs: payload.carryForwardToolResultIDs,
								}
							: {}),
					}),
					'Edit and resend'
				);
				updateTab(tabId, current => ({
					...current,
					editingMessageId: null,
					activeRunID: resent.runID,
					pendingRunID: resent.runID,
					isBusy: true,
					runtimeError: undefined,
				}));
				scrollTabToBottomSoon(tabId);
				return submission.configuration;
			}

			return updateStreamingMessage(tabId, payload, options);
		},
		[tabsRef, updateStreamingMessage, updateTab, scrollTabToBottomSoon]
	);

	const beginEditMessageForTab = useCallback(
		(tabId: string, id: string) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) {
				return;
			}
			if (tab.isBusy || tab.isHydrating) {
				return;
			}

			const message = tab.conversation.messages.find(m => m.id === id);
			if (!message) {
				return;
			}
			if (message.role !== RoleEnum.User) {
				return;
			}

			const external: EditorExternalMessage = {
				text: message.uiContent ?? '',
				attachments: message.attachments,
				toolChoices: message.uiToolChoices,
				toolSelectionIssues: message.uiToolSelectionIssues,
				mcpContext: message.mcpContext,
				mcpAppContextUpdates: message.mcpAppContextUpdates,
				toolOutputs: message.uiToolOutputs,
				enabledSkillRefs: message.enabledSkillRefs,
				activeSkillRefs: message.activeSkillRefs,
				workspaceSelection: message.workspaceSelection,
			};

			inputRefs.current.get(tabId)?.loadExternalMessage(external);
			updateTab(tabId, current => ({ ...current, editingMessageId: id }));
		},
		[inputRefs, tabsRef, updateTab]
	);

	const cancelEditingForTab = useCallback(
		(tabId: string) => {
			updateTab(tabId, current => ({ ...current, editingMessageId: null }));
		},
		[updateTab]
	);

	return {
		sendMessageForTab,
		stageContinuationForTab,
		beginEditMessageForTab,
		cancelEditingForTab,
	};
}
