import type { RefObject } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { useStoreState } from '@ariakit/react';
import { useTabStore } from '@ariakit/react/tab';

import type { Conversation } from '@/spec/session';
import type { SessionSearchItem } from '@/spec/sessionstore';
import { RoleEnum } from '@/spec/inference';
import { SessionSnapshotObservationKind } from '@/spec/session';
import { SessionRuntimeEventKind } from '@/spec/session_base';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';
import { defaultShortcutConfig, useChatShortcuts } from '@/lib/keyboard_shortcuts';
import { omitManyKeys } from '@/lib/obj_utils';
import { generateTitle } from '@/lib/title_utils';

import { sessionManagementAPI } from '@/apis/baseapi';

import type { ConversationAreaHandle } from '@/chats/conversation/conversation_area';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';
import type { ConversationSearchHandle } from '@/chats/search/conversation_search';
import type { ChatTabState } from '@/chats/tabs/tabs_model';
import type { InitialChatsModel } from '@/chats/tabs/tabs_persistence';
import {
	requireAcceptedSessionCommand,
	selectedSessionRun,
	sessionCommandMeta,
} from '@/chats/session/session_commands';
import { projectSessionSnapshotToConversation, sessionRunIsActive } from '@/chats/session/session_projection';
import {
	createEmptyTab,
	isScratchTab,
	MAX_TABS,
	normalizeTabsForInvariants,
	sanitizeConversationTitle,
	toTimestampMap,
} from '@/chats/tabs/tabs_model';
import { buildInitialChatsModel, writePersistedChatsPageState } from '@/chats/tabs/tabs_persistence';

interface UseChatsControllerArgs {
	conversationAreaRef: RefObject<ConversationAreaHandle | null>;
	searchRef: RefObject<ConversationSearchHandle | null>;
}

const SEARCH_REFRESH_EVENTS = new Set([
	SessionRuntimeEventKind.SessionCreated,
	SessionRuntimeEventKind.SessionRenamed,
	SessionRuntimeEventKind.SessionDeleted,
	SessionRuntimeEventKind.RunAccepted,
	SessionRuntimeEventKind.ModelOperationFinished,
	SessionRuntimeEventKind.RunCompleted,
	SessionRuntimeEventKind.RunFailed,
	SessionRuntimeEventKind.RunCancelled,
]);

function scheduleIdleWork(work: () => void, timeout = 1500): () => void {
	if (typeof window.requestIdleCallback === 'function') {
		const requestId = window.requestIdleCallback(
			() => {
				work();
			},
			{ timeout }
		);
		return () => {
			window.cancelIdleCallback(requestId);
		};
	}

	const timer = window.setTimeout(work, Math.min(timeout, 250));
	return () => {
		window.clearTimeout(timer);
	};
}

async function yieldUntilAfterNextPaint(): Promise<void> {
	if (typeof window === 'undefined' || typeof document === 'undefined' || document.visibilityState === 'hidden') {
		return;
	}

	await new Promise<void>(resolve => {
		window.requestAnimationFrame(() => {
			window.setTimeout(resolve, 0);
		});
	});
}

function buildNormalizedInitialModel(): InitialChatsModel {
	const built = buildInitialChatsModel();

	const createdFallbackTab = built.tabs.length === 0 ? createEmptyTab() : null;
	const baseTabs = built.tabs.length > 0 ? built.tabs : createdFallbackTab ? [createdFallbackTab] : [];
	const baseSelectedTabId = baseTabs.some(tab => tab.tabId === built.selectedTabId)
		? built.selectedTabId
		: (baseTabs[0]?.tabId ?? '');

	const lastActivatedAt = toTimestampMap(built.lastActivatedAtByTab);
	const normalized = normalizeTabsForInvariants(baseTabs, baseSelectedTabId, lastActivatedAt);

	for (const removedTabId of normalized.removedTabIds) {
		lastActivatedAt.delete(removedTabId);
	}

	if (createdFallbackTab) {
		lastActivatedAt.set(createdFallbackTab.tabId, Date.now());
	}

	if (normalized.addedScratchTabId) {
		lastActivatedAt.set(normalized.addedScratchTabId, Date.now());
	}

	const existingTabIds = new Set(normalized.tabs.map(tab => tab.tabId));
	const selectedTabId = normalized.tabs.some(tab => tab.tabId === baseSelectedTabId)
		? baseSelectedTabId
		: (normalized.tabs[0]?.tabId ?? '');

	const scrollTopByTab: Record<string, number> = {};
	for (const [tabId, scrollTop] of Object.entries(built.scrollTopByTab ?? {})) {
		if (existingTabIds.has(tabId) && typeof scrollTop === 'number') {
			scrollTopByTab[tabId] = scrollTop;
		}
	}
	if (createdFallbackTab) {
		scrollTopByTab[createdFallbackTab.tabId] = 0;
	}
	if (normalized.addedScratchTabId) {
		scrollTopByTab[normalized.addedScratchTabId] = 0;
	}

	const lastActivatedAtByTab: Record<string, number> = {};
	for (const [tabId, ts] of lastActivatedAt.entries()) {
		if (existingTabIds.has(tabId)) {
			lastActivatedAtByTab[tabId] = ts;
		}
	}

	return {
		...built,
		tabs: normalized.tabs,
		selectedTabId,
		scrollTopByTab,
		lastActivatedAtByTab,
	};
}

export function useChatsController({ conversationAreaRef, searchRef }: UseChatsControllerArgs) {
	const [initialModel] = useState<InitialChatsModel>(() => buildNormalizedInitialModel());
	const initialSelectedTabId = initialModel.selectedTabId;

	const controllerAliveRef = useRef(true);
	useEffect(() => {
		controllerAliveRef.current = true;
		return () => {
			controllerAliveRef.current = false;
		};
	}, []);

	const lastActivatedAtRef = useRef(toTimestampMap(initialModel.lastActivatedAtByTab));
	const touchTab = useCallback((tabId: string) => {
		lastActivatedAtRef.current.set(tabId, Date.now());
	}, []);

	const [tabs, setTabs] = useState<ChatTabState[]>(() => initialModel.tabs);
	const tabsRef = useRef(initialModel.tabs);
	const [searchRefreshKey, setSearchRefreshKey] = useState(0);
	const searchRefreshTimerRef = useRef<number | null>(null);

	const scheduleSearchRefresh = useCallback(() => {
		if (searchRefreshTimerRef.current !== null) {
			window.clearTimeout(searchRefreshTimerRef.current);
		}

		searchRefreshTimerRef.current = window.setTimeout(() => {
			searchRefreshTimerRef.current = null;
			setSearchRefreshKey(current => current + 1);
		}, 250);
	}, []);

	useEffect(() => {
		const unsubscribe = sessionManagementAPI.subscribeRuntimeEvents(event => {
			if (SEARCH_REFRESH_EVENTS.has(event.kind)) {
				scheduleSearchRefresh();
			}
		});

		return () => {
			unsubscribe();
			if (searchRefreshTimerRef.current !== null) {
				window.clearTimeout(searchRefreshTimerRef.current);
				searchRefreshTimerRef.current = null;
			}
		};
	}, [scheduleSearchRefresh]);

	const [mountedInputTabIds, setMountedInputTabIds] = useState<Set<string>>(
		() => new Set(initialSelectedTabId ? [initialSelectedTabId] : [])
	);

	const markInputMounted = useCallback((tabId: string) => {
		if (!tabId) {
			return;
		}

		setMountedInputTabIds(prev => {
			if (prev.has(tabId)) {
				return prev;
			}
			return new Set([...prev, tabId]);
		});
	}, []);

	const tabStore = useTabStore({
		defaultSelectedId: initialSelectedTabId,
	});
	const storeSelectedTabId = useStoreState(tabStore, 'selectedId') ?? initialSelectedTabId;
	const selectedTabId = useMemo(() => {
		if (tabs.some(tab => tab.tabId === storeSelectedTabId)) {
			return storeSelectedTabId;
		}
		return tabs[0]?.tabId ?? initialSelectedTabId;
	}, [initialSelectedTabId, storeSelectedTabId, tabs]);

	const selectedTabIdRef = useRef(selectedTabId);
	useEffect(() => {
		selectedTabIdRef.current = selectedTabId;
	}, [selectedTabId]);

	const pruneMountedInputTabIds = useCallback((existingTabIds: Set<string>) => {
		setMountedInputTabIds(prev => {
			const next = new Set([...prev].filter(tabId => existingTabIds.has(tabId)));
			return next.size === prev.size ? prev : next;
		});
	}, []);

	const selectTab = useCallback(
		(nextTabId: string) => {
			if (!nextTabId) {
				return;
			}

			selectedTabIdRef.current = nextTabId;
			markInputMounted(nextTabId);
			tabStore.setSelectedId(nextTabId);
		},
		[markInputMounted, tabStore]
	);

	const openConversationIds = useMemo(() => tabs.map(tab => tab.conversation.id).filter(Boolean), [tabs]);

	const scrollTopSnapshotRef = useRef(initialModel.scrollTopByTab ?? {});
	const persistNowRef = useRef<() => void>(() => {});
	const schedulePersistRef = useRef<() => void>(() => {});
	const saveQueueByTabRef = useRef(new Map<string, Promise<void>>());
	// Backend-acknowledged storage state for this controller lifetime.
	//
	// tab.isPersisted is UI state and must not decide whether the next write
	// can use PutMessagesToConversation. In particular, a failed initial
	// PutConversation must not be followed by a message-only update.
	const sessionObserverByTabRef = useRef(new Map<string, () => void>());
	const hydratePromiseByTabRef = useRef(new Map<string, Promise<void>>());
	const tabLoadVersionRef = useRef(new Map<string, number>());
	const autoTitleAttemptBySessionIDRef = useRef(new Map<string, string>());

	const beginTabLoad = useCallback((tabId: string): number => {
		const nextVersion = (tabLoadVersionRef.current.get(tabId) ?? 0) + 1;
		tabLoadVersionRef.current.set(tabId, nextVersion);
		return nextVersion;
	}, []);

	const isTabLoadCurrent = useCallback((tabId: string, version: number): boolean => {
		return (
			controllerAliveRef.current &&
			tabLoadVersionRef.current.get(tabId) === version &&
			tabsRef.current.some(tab => tab.tabId === tabId)
		);
	}, []);

	const cancelRunForDisposedTab = useCallback((tabId: string) => {
		const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);
		const snapshot = tab?.sessionSnapshot;
		const run = snapshot ? selectedSessionRun(snapshot, tab?.activeRunID) : undefined;

		if (!tab?.sessionID || !snapshot || !run || !sessionRunIsActive(run)) {
			return;
		}

		void sessionManagementAPI
			.cancelRun({
				meta: sessionCommandMeta(snapshot.session.revision, run.revision),
				sessionID: tab.sessionID,
				runID: run.id,
				reason: 'Chat tab closed',
			})
			.then(receipt => {
				requireAcceptedSessionCommand(receipt, 'Cancel Run');
			})
			.catch((error: unknown) => {
				console.error('Failed to cancel Run for a disposed chat tab:', error);
			});
	}, []);

	const disposeTabRuntime = useCallback(
		(tabId: string) => {
			cancelRunForDisposedTab(tabId);
			conversationAreaRef.current?.disposeTabRuntime(tabId);
			lastActivatedAtRef.current.delete(tabId);
			hydratePromiseByTabRef.current.delete(tabId);
			saveQueueByTabRef.current.delete(tabId);
			tabLoadVersionRef.current.delete(tabId);
		},
		[cancelRunForDisposedTab, conversationAreaRef]
	);

	const commitTabs = useCallback((nextTabs: ChatTabState[]) => {
		tabsRef.current = nextTabs;
		setTabs(nextTabs);
		schedulePersistRef.current();
	}, []);

	const normalizeAndCommitTabs = useCallback(
		(nextTabs: ChatTabState[], preferredSelectedTabId?: string) => {
			const {
				tabs: normalizedTabs,
				removedTabIds,
				addedScratchTabId,
			} = normalizeTabsForInvariants(nextTabs, selectedTabIdRef.current, lastActivatedAtRef.current);

			if (removedTabIds.length > 0 || addedScratchTabId) {
				let nextScrollSnapshot = { ...scrollTopSnapshotRef.current };

				for (const removedTabId of removedTabIds) {
					disposeTabRuntime(removedTabId);
					nextScrollSnapshot = omitManyKeys(nextScrollSnapshot, [removedTabId]);
				}

				if (addedScratchTabId) {
					touchTab(addedScratchTabId);
					conversationAreaRef.current?.setScrollTopForTab(addedScratchTabId, 0);
					nextScrollSnapshot[addedScratchTabId] = 0;
				}

				scrollTopSnapshotRef.current = nextScrollSnapshot;
			}

			const existingTabIds = new Set(normalizedTabs.map(tab => tab.tabId));
			pruneMountedInputTabIds(existingTabIds);

			commitTabs(normalizedTabs);

			const finalSelectedTabId =
				(preferredSelectedTabId && existingTabIds.has(preferredSelectedTabId) ? preferredSelectedTabId : '') ||
				(existingTabIds.has(selectedTabIdRef.current) ? selectedTabIdRef.current : '') ||
				normalizedTabs[0]?.tabId ||
				'';

			if (finalSelectedTabId && finalSelectedTabId !== selectedTabIdRef.current) {
				selectTab(finalSelectedTabId);
			}

			return normalizedTabs;
		},
		[commitTabs, conversationAreaRef, disposeTabRuntime, pruneMountedInputTabIds, selectTab, touchTab]
	);

	const updateTab = useCallback(
		(tabId: string, updater: (tab: ChatTabState) => ChatTabState) => {
			const current = tabsRef.current;
			const idx = current.findIndex(tab => tab.tabId === tabId);
			if (idx === -1) {
				return;
			}

			const updated = updater(current[idx]);
			if (updated === current[idx]) {
				return;
			}
			const next = [...current];
			next[idx] = updated;
			normalizeAndCommitTabs(next);
		},
		[normalizeAndCommitTabs]
	);

	const maybeAutomaticallyRenameSession = useCallback(
		(tabId: string, snapshot: NonNullable<ChatTabState['sessionSnapshot']>, conversation: Conversation) => {
			const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);

			if (!tab || tab.sessionID !== snapshot.session.id || tab.manualTitleLocked) {
				return;
			}

			const userMessages = conversation.messages.filter(message => message.role === RoleEnum.User);
			if (userMessages.length === 0 || userMessages.length > 2) {
				return;
			}

			const first = generateTitle(userMessages[0].uiContent);
			const proposed =
				userMessages.length === 1
					? first.title
					: (() => {
							const second = generateTitle(userMessages[1].uiContent);
							return second.score > first.score ? second.title : first.title;
						})();
			const nextTitle = sanitizeConversationTitle(proposed);
			const currentTitle = snapshot.session.title || 'Conversation';

			if (!nextTitle || nextTitle === currentTitle) {
				return;
			}

			const attemptKey = `${snapshot.session.revision}:${nextTitle}`;
			if (autoTitleAttemptBySessionIDRef.current.get(snapshot.session.id) === attemptKey) {
				return;
			}

			autoTitleAttemptBySessionIDRef.current.set(snapshot.session.id, attemptKey);

			void sessionManagementAPI
				.renameSession({
					meta: sessionCommandMeta(snapshot.session.revision),
					sessionID: snapshot.session.id,
					title: nextTitle,
				})
				.then(receipt => {
					requireAcceptedSessionCommand(receipt, 'Rename Session');
				})
				.catch((error: unknown) => {
					console.error('Failed to refine Session title:', error);
				});
		},
		[]
	);

	const applySessionSnapshot = useCallback(
		(tabId: string, snapshot: ChatTabState['sessionSnapshot']) => {
			if (!snapshot) {
				return;
			}

			const activeRun = selectedSessionRun(snapshot);
			const conversation = projectSessionSnapshotToConversation(snapshot, activeRun?.id);

			updateTab(tabId, previous => {
				if (previous.sessionID !== snapshot.session.id) {
					return previous;
				}

				const composerConfigurationRestored =
					previous.sessionID === snapshot.session.id && previous.composerConfigurationRestored;
				const acknowledgedRunNowVisible =
					previous.pendingRunID !== undefined && snapshot.runs.some(run => run.id === previous.pendingRunID);

				return {
					...previous,
					sessionSnapshot: snapshot,
					activeRunID: activeRun?.id,
					conversation,
					isLoaded: true,
					isHydrating: composerConfigurationRestored ? previous.isHydrating : true,
					composerConfigurationRestored,
					isPersisted: true,
					isBusy: sessionRunIsActive(activeRun),
					pendingRunID: acknowledgedRunNowVisible ? undefined : previous.pendingRunID,
					runtimeError: undefined,
				};
			});

			maybeAutomaticallyRenameSession(tabId, snapshot, conversation);

			if (activeRun && (activeRun.skillSession.enabled !== undefined || activeRun.skillSession.active !== undefined)) {
				conversationAreaRef.current?.syncComposerSkillSessionState(
					tabId,
					activeRun.skillSession.enabled ?? [],
					activeRun.skillSession.active ?? []
				);
			}
		},
		[conversationAreaRef, maybeAutomaticallyRenameSession, updateTab]
	);

	const ensureSessionObserver = useCallback(
		(tabId: string, sessionID: string) => {
			if (sessionObserverByTabRef.current.has(tabId)) {
				return;
			}

			const unsubscribe = sessionManagementAPI.observeSession(sessionID, observation => {
				if (observation.kind === SessionSnapshotObservationKind.Snapshot && observation.snapshot) {
					applySessionSnapshot(tabId, observation.snapshot);
				} else if (observation.kind === SessionSnapshotObservationKind.Error) {
					updateTab(tabId, tab => ({
						...tab,
						runtimeError:
							observation.error instanceof Error ? observation.error.message : 'Session snapshot refresh failed.',
					}));
				} else if (observation.kind === SessionSnapshotObservationKind.Deleted) {
					updateTab(tabId, tab => ({
						...tab,
						isBusy: false,
						isHydrating: false,
						runtimeError: 'This Session was deleted.',
					}));
				}
			});

			sessionObserverByTabRef.current.set(tabId, unsubscribe);
		},
		[applySessionSnapshot, updateTab]
	);

	const ensureTabLoaded = useCallback(
		(tabId: string): Promise<void> => {
			const tab = tabsRef.current.find(current => current.tabId === tabId);
			if (!tab?.sessionID) {
				return Promise.resolve();
			}
			if (tab.isLoaded && tab.sessionSnapshot && tab.composerConfigurationRestored) {
				return Promise.resolve();
			}

			const existing = hydratePromiseByTabRef.current.get(tabId);
			if (existing) {
				return existing;
			}

			ensureSessionObserver(tabId, tab.sessionID);
			updateTab(tabId, previous => ({
				...previous,
				isHydrating: true,
			}));

			const sessionID = tab.sessionID;
			const loadVersion = beginTabLoad(tabId);
			const load = (async () => {
				try {
					const currentTab = tabsRef.current.find(candidate => candidate.tabId === tabId);
					const snapshot =
						currentTab?.sessionID === sessionID && currentTab.sessionSnapshot
							? currentTab.sessionSnapshot
							: await sessionManagementAPI.getSessionSnapshot(sessionID);

					if (!isTabLoadCurrent(tabId, loadVersion)) {
						return;
					}

					applySessionSnapshot(tabId, snapshot);

					const area = conversationAreaRef.current;
					if (!area) {
						return;
					}

					await area.restoreComposerFromSessionConfiguration(tabId, snapshot.session.configuration);

					if (!isTabLoadCurrent(tabId, loadVersion)) {
						return;
					}

					const activeRun = selectedSessionRun(snapshot);
					updateTab(tabId, previous => {
						if (previous.sessionID !== snapshot.session.id) {
							return previous;
						}

						return {
							...previous,
							isLoaded: true,
							isHydrating: false,
							composerConfigurationRestored: true,
							isBusy: sessionRunIsActive(activeRun),
						};
					});

					if (
						activeRun &&
						(activeRun.skillSession.enabled !== undefined || activeRun.skillSession.active !== undefined)
					) {
						area.syncComposerSkillSessionState(
							tabId,
							activeRun.skillSession.enabled ?? [],
							activeRun.skillSession.active ?? []
						);
					}
				} catch (error) {
					if (!isTabLoadCurrent(tabId, loadVersion)) {
						return;
					}
					updateTab(tabId, previous => ({
						...previous,
						isLoaded: true,
						isHydrating: false,
						composerConfigurationRestored: false,
						runtimeError: error instanceof Error ? error.message : 'Failed to load Session snapshot.',
					}));
				}
			})().finally(() => {
				if (hydratePromiseByTabRef.current.get(tabId) === load) {
					hydratePromiseByTabRef.current.delete(tabId);
				}
			});

			hydratePromiseByTabRef.current.set(tabId, load);
			return load;
		},
		[applySessionSnapshot, beginTabLoad, conversationAreaRef, ensureSessionObserver, isTabLoadCurrent, updateTab]
	);

	useEffect(() => {
		const wanted = new Set(tabs.filter(tab => Boolean(tab.sessionID)).map(tab => `${tab.tabId}:${tab.sessionID}`));

		for (const tab of tabs) {
			if (tab.sessionID) {
				ensureSessionObserver(tab.tabId, tab.sessionID);
			}
		}

		for (const [tabId, unsubscribe] of sessionObserverByTabRef.current.entries()) {
			const tab = tabsRef.current.find(candidate => candidate.tabId === tabId);
			if (tab?.sessionID && wanted.has(`${tabId}:${tab.sessionID}`)) {
				continue;
			}
			unsubscribe();
			sessionObserverByTabRef.current.delete(tabId);
		}
	}, [ensureSessionObserver, tabs]);

	useEffect(() => {
		const sessionObservers = sessionObserverByTabRef.current;

		return () => {
			for (const unsubscribe of sessionObservers.values()) {
				unsubscribe();
			}
			sessionObservers.clear();
		};
	}, []);

	useEffect(() => {
		const active = tabs.find(tab => tab.tabId === selectedTabId);
		if (!active || active.isBusy || active.isHydrating || !active.isLoaded) {
			return;
		}

		const scratch = tabs.find(t => {
			return isScratchTab(t);
		});
		if (!scratch || mountedInputTabIds.has(scratch.tabId)) {
			return;
		}

		return scheduleIdleWork(() => {
			if (!controllerAliveRef.current) {
				return;
			}
			const latestScratch = tabsRef.current.find(isScratchTab);
			if (!latestScratch || latestScratch.tabId !== scratch.tabId) {
				return;
			}
			const latestActive = tabsRef.current.find(tab => tab.tabId === selectedTabIdRef.current);
			if (!latestActive || latestActive.isBusy || latestActive.isHydrating || !latestActive.isLoaded) {
				return;
			}
			markInputMounted(scratch.tabId);
		});
	}, [markInputMounted, mountedInputTabIds, selectedTabId, tabs]);

	useEffect(() => {
		touchTab(selectedTabId);
	}, [selectedTabId, touchTab]);

	const persistNow = useCallback(() => {
		const tabsSnapshot = tabsRef.current.slice(0, MAX_TABS);

		const scrollObj: Record<string, number> = {
			...scrollTopSnapshotRef.current,
			...conversationAreaRef.current?.getScrollTopByTabSnapshot(),
		};
		scrollTopSnapshotRef.current = scrollObj;

		const lruObj: Record<string, number> = {};
		for (const [key, value] of lastActivatedAtRef.current.entries()) {
			lruObj[key] = value;
		}

		writePersistedChatsPageState({
			v: 2,
			selectedTabId: selectedTabIdRef.current,
			tabs: tabsSnapshot.map(tab => ({
				tabId: tab.tabId,
				sessionID: tab.sessionID,
				title: tab.conversation.title,
				manualTitleLocked: tab.manualTitleLocked,
			})),
			scrollTopByTab: scrollObj,
			lastActivatedAtByTab: lruObj,
		});
	}, [conversationAreaRef]);

	useEffect(() => {
		persistNowRef.current = persistNow;
	}, [persistNow]);

	const persistTimerRef = useRef<number | null>(null);

	const schedulePersist = useCallback(() => {
		if (persistTimerRef.current !== null) {
			window.clearTimeout(persistTimerRef.current);
		}

		persistTimerRef.current = window.setTimeout(() => {
			persistTimerRef.current = null;
			persistNow();
		}, 250);
	}, [persistNow]);

	useEffect(() => {
		schedulePersistRef.current = schedulePersist;
	}, [schedulePersist]);

	useEffect(() => {
		schedulePersist();
	}, [schedulePersist, selectedTabId]);

	useEffect(() => {
		return () => {
			if (persistTimerRef.current !== null) {
				window.clearTimeout(persistTimerRef.current);
			}
		};
	}, []);

	useEffect(() => {
		const onPageHide = () => {
			persistNow();
		};

		const onVisibilityChange = () => {
			if (document.visibilityState === 'hidden') {
				persistNow();
			}
		};

		window.addEventListener('pagehide', onPageHide);
		document.addEventListener('visibilitychange', onVisibilityChange);

		return () => {
			window.removeEventListener('pagehide', onPageHide);
			document.removeEventListener('visibilitychange', onVisibilityChange);
		};
	}, [persistNow]);

	useEffect(() => {
		return () => {
			persistNow();
		};
	}, [persistNow]);

	const saveUpdatedConversation = useCallback(
		(tabId: string, updatedConv: Conversation, titleWasExternallyChanged = false) => {
			updateTab(tabId, current => ({
				...current,
				conversation: updatedConv,
				isLoaded: true,
				isPersisted: Boolean(current.sessionID),
				manualTitleLocked: titleWasExternallyChanged ? true : current.manualTitleLocked,
				isHydrating: false,
			}));
		},
		[updateTab]
	);

	const getOrCreateScratchTabForStarter = useCallback((): string => {
		const scratch = tabsRef.current.find(isScratchTab);
		if (scratch) {
			return scratch.tabId;
		}

		const nextScratch = createEmptyTab();
		touchTab(nextScratch.tabId);
		const normalizedTabs = normalizeAndCommitTabs([...tabsRef.current, nextScratch], nextScratch.tabId);

		return normalizedTabs.some(tab => tab.tabId === nextScratch.tabId)
			? nextScratch.tabId
			: (normalizedTabs.find(t => isScratchTab(t))?.tabId ?? '');
	}, [normalizeAndCommitTabs, touchTab]);

	const preloadWorkflowStarter = useCallback(
		async (starter: ChatWorkflowStarter): Promise<boolean> => {
			const targetId = getOrCreateScratchTabForStarter();
			if (!targetId) {
				return false;
			}

			selectTab(targetId);

			await new Promise<void>(resolve => {
				requestAnimationFrame(() => {
					resolve();
				});
			});
			if (!controllerAliveRef.current) {
				return false;
			}

			const applied = await conversationAreaRef.current?.applyWorkflowStarter(targetId, starter);
			if (applied) {
				conversationAreaRef.current?.focusInput(targetId);
			}
			return Boolean(applied);
		},
		[conversationAreaRef, getOrCreateScratchTabForStarter, selectTab]
	);

	const openNewTab = useCallback(() => {
		const scratch = tabsRef.current.find(isScratchTab) ?? null;
		const targetId = scratch?.tabId ?? selectedTabIdRef.current;
		if (!targetId) {
			return;
		}

		selectTab(targetId);

		requestAnimationFrame(() => {
			const area = conversationAreaRef.current;
			if (!area) {
				return;
			}

			const resetPromise = area.resetComposerForNewConversation(targetId);
			area.focusInput(targetId);
			void resetPromise.catch((error: unknown) => {
				console.error('Failed to reset composer for new conversation:', error);
			});
		});
	}, [conversationAreaRef, selectTab]);

	const closeTab = useCallback(
		(tabId: string) => {
			const current = tabsRef.current;
			const idx = current.findIndex(tab => tab.tabId === tabId);
			if (idx === -1) {
				return;
			}

			const wasActive = tabId === selectedTabIdRef.current;

			disposeTabRuntime(tabId);
			scrollTopSnapshotRef.current = omitManyKeys({ ...scrollTopSnapshotRef.current }, [tabId]);

			const baseNextTabs = current.filter(tab => tab.tabId !== tabId);

			const right = current[idx + 1];
			const left = idx > 0 ? current[idx - 1] : undefined;
			const preferredNextSelectedId = wasActive
				? (left && left.tabId !== tabId ? left.tabId : right && right.tabId !== tabId ? right.tabId : '') ||
					baseNextTabs[0]?.tabId ||
					''
				: undefined;

			normalizeAndCommitTabs(baseNextTabs, preferredNextSelectedId);
		},
		[disposeTabRuntime, normalizeAndCommitTabs]
	);

	const cycleTabBy = useCallback(
		(delta: number) => {
			const current = tabsRef.current;
			if (current.length < 2) {
				return;
			}

			const activeId = selectedTabIdRef.current;
			const idx = current.findIndex(tab => tab.tabId === activeId);
			const from = Math.max(idx, 0);
			const nextIndex = (from + delta + current.length) % current.length;
			const nextId = current[nextIndex]?.tabId;
			if (!nextId) {
				return;
			}

			selectTab(nextId);
			requestAnimationFrame(() => {
				conversationAreaRef.current?.focusInput(nextId);
			});
		},
		[conversationAreaRef, selectTab]
	);

	const selectNextTab = useCallback(() => {
		cycleTabBy(1);
	}, [cycleTabBy]);

	const selectPrevTab = useCallback(() => {
		cycleTabBy(-1);
	}, [cycleTabBy]);

	const renameTabTitle = useCallback((tabId: string, newTitle: string) => {
		const tab = tabsRef.current.find(t => t.tabId === tabId);
		if (!tab || !tab.sessionID || !tab.sessionSnapshot) {
			return;
		}

		const sanitized = sanitizeConversationTitle(newTitle.trim());
		if (!sanitized || sanitized === tab.conversation.title) {
			return;
		}

		void sessionManagementAPI
			.renameSession({
				meta: sessionCommandMeta(tab.sessionSnapshot.session.revision),
				sessionID: tab.sessionID,
				title: sanitized,
			})
			.then(receipt => {
				requireAcceptedSessionCommand(receipt, 'Rename Session');
			})
			.catch(console.error);
	}, []);

	const loadConversationIntoTab = useCallback(
		async (tabId: string, item: SessionSearchItem) => {
			const loadVersion = beginTabLoad(tabId);
			updateTab(tabId, tab => ({
				...tab,
				isHydrating: true,
				isBusy: false,
				editingMessageId: null,
			}));

			try {
				const selectedChat = await sessionManagementAPI.getSessionSnapshot(item.id);
				if (!isTabLoadCurrent(tabId, loadVersion)) {
					return;
				}

				const hydrated = projectSessionSnapshotToConversation(selectedChat, selectedSessionRun(selectedChat)?.id);
				conversationAreaRef.current?.resetScrollToTop(tabId);

				updateTab(tabId, tab => ({
					...tab,
					sessionID: selectedChat.session.id,
					sessionSnapshot: selectedChat,
					activeRunID: selectedSessionRun(selectedChat)?.id,
					composerConfigurationRestored: false,
					conversation: hydrated,
					isPersisted: true,
					manualTitleLocked: false,
					editingMessageId: null,
					isBusy: sessionRunIsActive(selectedSessionRun(selectedChat)),
					isLoaded: true,
					isHydrating: true,
				}));

				await yieldUntilAfterNextPaint();
				if (!isTabLoadCurrent(tabId, loadVersion)) {
					return;
				}

				await conversationAreaRef.current?.restoreComposerFromSessionConfiguration(
					tabId,
					selectedChat.session.configuration
				);

				const selectedRun = selectedSessionRun(selectedChat);
				if (
					selectedRun &&
					(selectedRun.skillSession.enabled !== undefined || selectedRun.skillSession.active !== undefined)
				) {
					conversationAreaRef.current?.syncComposerSkillSessionState(
						tabId,
						selectedRun.skillSession.enabled ?? [],
						selectedRun.skillSession.active ?? []
					);
				}

				updateTab(tabId, tab => ({
					...tab,
					isPersisted: true,
					editingMessageId: null,
					isBusy: sessionRunIsActive(selectedRun),
					isLoaded: true,
					isHydrating: false,
					composerConfigurationRestored: true,
				}));
			} catch (error) {
				if (!isTabLoadCurrent(tabId, loadVersion)) {
					return;
				}

				console.error(error);
				updateTab(tabId, tab => ({
					...tab,
					isLoaded: true,
					isHydrating: false,
					isBusy: false,
					composerConfigurationRestored: false,
					editingMessageId: null,
				}));
			}
		},
		[beginTabLoad, conversationAreaRef, isTabLoadCurrent, updateTab]
	);

	const handleSelectSession = useCallback(
		async (item: SessionSearchItem) => {
			const alreadyOpen = tabsRef.current.find(tab => tab.conversation.id === item.id);
			if (alreadyOpen) {
				selectTab(alreadyOpen.tabId);
				void ensureTabLoaded(alreadyOpen.tabId);
				return;
			}

			const scratch = tabsRef.current.find(isScratchTab);
			const targetId = scratch?.tabId ?? selectedTabIdRef.current;
			if (!targetId) {
				return;
			}

			selectTab(targetId);
			await loadConversationIntoTab(targetId, item);

			requestAnimationFrame(() => {
				conversationAreaRef.current?.focusInput(targetId);
			});
		},
		[conversationAreaRef, ensureTabLoaded, loadConversationIntoTab, selectTab]
	);

	const openSessionForMCPAppMessage = useCallback(
		async (sessionID: string): Promise<string | null> => {
			if (!sessionID.trim()) {
				return null;
			}

			const existing = tabsRef.current.find(tab => tab.sessionID === sessionID);
			if (existing) {
				selectTab(existing.tabId);
				await ensureTabLoaded(existing.tabId);
				requestAnimationFrame(() => {
					conversationAreaRef.current?.focusInput(existing.tabId);
				});
				return existing.tabId;
			}

			try {
				const snapshot = await sessionManagementAPI.getSessionSnapshot(sessionID);
				const targetId = getOrCreateScratchTabForStarter();

				if (!targetId) {
					return null;
				}

				selectTab(targetId);
				await loadConversationIntoTab(targetId, {
					id: snapshot.session.id,
					title: snapshot.session.title || 'Conversation',
					modifiedAt: snapshot.session.updatedAt,
				});

				const loaded = tabsRef.current.find(tab => tab.tabId === targetId && tab.sessionID === snapshot.session.id);
				if (!loaded) {
					return null;
				}

				requestAnimationFrame(() => {
					conversationAreaRef.current?.focusInput(targetId);
				});
				return targetId;
			} catch (error) {
				console.error('Failed to open owning Session for MCP App draft:', error);
				return null;
			}
		},
		[conversationAreaRef, ensureTabLoaded, getOrCreateScratchTabForStarter, loadConversationIntoTab, selectTab]
	);

	useEffect(() => {
		void ensureTabLoaded(selectedTabId);
	}, [ensureTabLoaded, selectedTabId]);

	const [shortcutConfig] = useState<ShortcutConfig>(defaultShortcutConfig);

	useChatShortcuts({
		config: shortcutConfig,
		isBusy: false,
		handlers: {
			newChat: () => {
				openNewTab();
			},
			closeChat: () => {
				closeTab(selectedTabIdRef.current);
			},
			nextChat: () => {
				selectNextTab();
			},
			previousChat: () => {
				selectPrevTab();
			},
			focusSearch: () => {
				searchRef.current?.focusInput();
			},
			focusInput: () => {
				conversationAreaRef.current?.focusInput(selectedTabIdRef.current);
			},
			insertTemplate: () => {
				conversationAreaRef.current?.openTemplateMenu(selectedTabIdRef.current);
			},
			insertTool: () => {
				conversationAreaRef.current?.openToolMenu(selectedTabIdRef.current);
			},
			insertAttachment: () => {
				conversationAreaRef.current?.openAttachmentMenu(selectedTabIdRef.current);
			},
			insertSystemPrompt: () => {
				conversationAreaRef.current?.openSystemPromptMenu(selectedTabIdRef.current);
			},
			attachSkills: () => {
				conversationAreaRef.current?.openSkillsMenu(selectedTabIdRef.current);
			},
			attachMCP: () => {
				conversationAreaRef.current?.openMCPMenu(selectedTabIdRef.current);
			},
			stopResponse: () => {
				conversationAreaRef.current?.requestStopResponse(selectedTabIdRef.current);
			},
		},
	});

	const getConversationForExport = useCallback(async (): Promise<string> => {
		const tab = tabsRef.current.find(current => current.tabId === selectedTabIdRef.current);
		if (!tab) {
			return JSON.stringify(null, null, 2);
		}

		if (!tab.sessionID) {
			return JSON.stringify(null, null, 2);
		}
		return sessionManagementAPI.exportSession(tab.sessionID);
	}, []);

	const tabBarItems = useMemo(
		() =>
			tabs.map(tab => ({
				tabId: tab.tabId,
				title: tab.conversation.title,
				isBusy: tab.isBusy || tab.isHydrating,
				isEmpty: tab.conversation.messages.length === 0,
				renameEnabled: tab.conversation.messages.length > 0,
			})),
		[tabs]
	);

	return {
		tabStore,
		tabs,
		selectedTabId,
		mountedInputTabIds,
		initialScrollTopByTab: initialModel.scrollTopByTab,
		shortcutConfig,
		updateTab,
		saveUpdatedConversation,
		selectTab,
		openNewTab,
		closeTab,
		renameTabTitle,
		handleSelectSession,
		getConversationForExport,
		tabBarItems,
		openConversationIds,
		searchRefreshKey,
		openSessionForMCPAppMessage,
		maxTabs: MAX_TABS,
		preloadWorkflowStarter,
	};
}
