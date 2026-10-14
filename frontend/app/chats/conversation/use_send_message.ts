import type { RefObject } from 'react';
import { useCallback } from 'react';

import type { UIModelOption } from '@/spec/model';
import { SessionRunState } from '@/spec/session';

import { generateTitle } from '@/lib/title_utils';

import { sessionManagementAPI } from '@/apis/baseapi';

import type { EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import type { ChatTabState } from '@/chats/tabs/tabs_model';
import { requireAcceptedSessionCommand, sessionCommandMeta } from '@/chats/session/session_commands';
import { buildSessionSubmissionIntent } from '@/chats/session/session_intent';

interface UseSendMessageArgs {
	tabsRef: RefObject<ChatTabState[]>;
	updateTab: (tabId: string, updater: (tab: ChatTabState) => ChatTabState) => void;
	scrollTabToBottomSoon: (tabId: string) => void;
	tabExists: (tabId: string) => boolean;
}

export function useSendMessage({ tabsRef, updateTab, scrollTabToBottomSoon, tabExists }: UseSendMessageArgs) {
	const updateStreamingMessage = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIModelOption) => {
			if (!tabExists(tabId)) {
				return;
			}

			if (!options.model) {
				throw new Error('Select an available Artifact-backed model before sending.');
			}

			const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);
			if (!tab) {
				return;
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
						isBusy: true,
						runtimeError: undefined,
					}));
					scrollTabToBottomSoon(tabId);
					return;
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
					isBusy: true,
					runtimeError: undefined,
				}));
				scrollTabToBottomSoon(tabId);
			} catch (error) {
				updateTab(tabId, current => ({
					...current,
					isBusy: false,
					runtimeError: error instanceof Error ? error.message : 'Session submission failed.',
				}));
				throw error;
			}
		},
		[scrollTabToBottomSoon, tabExists, tabsRef, updateTab]
	);

	const sendMessageForTab = useCallback(
		async (tabId: string, payload: EditorSubmitPayload, options: UIModelOption) => {
			const tab = tabsRef.current.find(t => t.tabId === tabId);
			if (!tab) {
				return;
			}
			if (tab.isBusy || tab.isHydrating) {
				return;
			}

			if (tab.editingMessageId) {
				if (!tab.sessionID || !tab.sessionSnapshot) {
					throw new Error('Edit-and-resend requires an existing Session snapshot.');
				}
				const submission = await buildSessionSubmissionIntent(payload, options);
				requireAcceptedSessionCommand(
					await sessionManagementAPI.editAndResend({
						meta: sessionCommandMeta(tab.sessionSnapshot.session.revision),
						sessionID: tab.sessionID,
						replaceMessageID: tab.editingMessageId,
						input: submission.input,
						configuration: submission.configuration,
					}),
					'Edit and resend'
				);
				updateTab(tabId, current => ({ ...current, editingMessageId: null, isBusy: true }));
				return;
			}

			await updateStreamingMessage(tabId, payload, options);
		},
		[tabsRef, updateStreamingMessage, updateTab]
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

			updateTab(tabId, current => ({ ...current, editingMessageId: id }));
		},
		[tabsRef, updateTab]
	);

	const cancelEditingForTab = useCallback(
		(tabId: string) => {
			updateTab(tabId, current => ({ ...current, editingMessageId: null }));
		},
		[updateTab]
	);

	return {
		sendMessageForTab,
		beginEditMessageForTab,
		cancelEditingForTab,
	};
}
