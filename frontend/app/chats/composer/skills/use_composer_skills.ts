import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';

import type { SkillListItem, SkillRef } from '@/spec/skill';

import { resolveStateUpdate } from '@/lib/hook_utils';

import { skillStoreAPI } from '@/apis/baseapi';

import {
	areSkillRefListsEqual,
	buildSkillRefsFingerprint,
	clampActiveSkillRefsToEnabled,
	normalizeSkillRefs,
	skillRefFromListItem,
	skillRefKey,
} from '@/skills/lib/skill_identity_utils';

type SkillSessionSyncMode = 'none' | 'if-session-exists' | 'ensure-if-enabled';
interface ApplySkillSelectionStateOptions {
	syncSession?: SkillSessionSyncMode;
	forceResetSession?: boolean;
}

interface UseComposerSkillsResult {
	allSkills: SkillListItem[];
	skillsLoading: boolean;
	skillsLoadError: string | null;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	skillSessionID: string | null;
	setEnabledSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	setActiveSkillRefsFromSession: Dispatch<SetStateAction<SkillRef[]>>;
	enableAllSkills: () => void;
	disableAllSkills: () => void;
	refreshSkills: () => Promise<void>;
	applySkillSelectionState: (
		nextEnabledInput: SkillRef[] | null | undefined,
		nextActiveInput: SkillRef[] | null | undefined,
		options?: ApplySkillSelectionStateOptions
	) => Promise<void>;
	ensureSkillSession: () => Promise<string | null>;
	listActiveSkillRefs: (sid: string) => Promise<SkillRef[]>;
	getCurrentEnabledSkillRefs: () => SkillRef[];
	getCurrentActiveSkillRefs: () => SkillRef[];
	getCurrentSkillSessionID: () => string | null;
}

function buildSkillSessionStateKey(enabled: SkillRef[], active: SkillRef[]): string {
	return `${buildSkillRefsFingerprint(enabled)}::${buildSkillRefsFingerprint(active)}`;
}

function isInstructionSkillListItem(item: SkillListItem): boolean {
	return (item.skillDefinition.insert || 'instructions') === 'instructions';
}

let composerSkillsCatalogCache: SkillListItem[] | undefined;
let composerSkillsCatalogPromise: Promise<SkillListItem[]> | undefined;

async function fetchComposerSkillsCatalog(): Promise<SkillListItem[]> {
	const out: SkillListItem[] = [];
	let token: string | undefined;

	for (let guard = 0; guard < 50; guard += 1) {
		const resp = await skillStoreAPI.listSkills({
			bundleIDs: [],
			types: [],
			inserts: ['instructions'],
			includeDisabled: false,
			includeMissing: false,
			recommendedPageSize: 200,
			pageToken: token,
		});

		out.push(...(resp.skillListItems ?? []));
		token = resp.nextPageToken;
		if (!token) {
			break;
		}
	}

	return out;
}

function loadComposerSkillsCatalog(force = false): Promise<SkillListItem[]> {
	if (composerSkillsCatalogPromise) {
		return composerSkillsCatalogPromise;
	}
	if (!force && composerSkillsCatalogCache !== undefined) {
		return Promise.resolve(composerSkillsCatalogCache);
	}

	const request = fetchComposerSkillsCatalog().then(items => {
		composerSkillsCatalogCache = items;
		return items;
	});
	composerSkillsCatalogPromise = request;

	void request.then(
		() => {
			if (composerSkillsCatalogPromise === request) {
				composerSkillsCatalogPromise = undefined;
			}
		},
		() => {
			if (composerSkillsCatalogPromise === request) {
				composerSkillsCatalogPromise = undefined;
			}
		}
	);

	return request;
}

export function useComposerSkills(): UseComposerSkillsResult {
	const [allSkills, setAllSkills] = useState<SkillListItem[]>([]);
	const [enabledSkillRefs, setEnabledSkillRefsRaw] = useState<SkillRef[]>([]);
	const [activeSkillRefs, setActiveSkillRefsRaw] = useState<SkillRef[]>([]);
	const [skillSessionID, setSkillSessionID] = useState<string | null>(null);
	const [skillsLoading, setSkillsLoading] = useState(true);
	const [skillsLoadError, setSkillsLoadError] = useState<string | null>(null);

	const sessionStateKeyRef = useRef('');
	const skillSessionSyncVersionRef = useRef(0);

	const availableSkillKeySetRef = useRef<Set<string>>(new Set());
	const skillsLoadingRef = useRef(true);
	const skillsCatalogReadyRef = useRef(false);

	const skillSessionIDRef = useRef<string | null>(null);
	const enabledSkillRefsRef = useRef<SkillRef[]>([]);
	const activeSkillRefsRef = useRef<SkillRef[]>([]);
	const skillsCatalogLoadPromiseRef = useRef<Promise<SkillListItem[]> | null>(null);
	const enableAllSkillsRequestVersionRef = useRef(0);

	const cancelPendingEnableAllSkills = useCallback(() => {
		enableAllSkillsRequestVersionRef.current += 1;
	}, []);

	const advanceSkillSessionSyncVersion = useCallback(() => {
		const next = skillSessionSyncVersionRef.current + 1;
		skillSessionSyncVersionRef.current = next;
		return next;
	}, []);

	const updateEnabledSkillRefsState = useCallback((next: SkillRef[]) => {
		enabledSkillRefsRef.current = next;
		setEnabledSkillRefsRaw(prev => (areSkillRefListsEqual(prev, next) ? prev : next));
	}, []);

	const updateActiveSkillRefsState = useCallback((next: SkillRef[]) => {
		const clampedNext = clampActiveSkillRefsToEnabled(enabledSkillRefsRef.current, next);
		activeSkillRefsRef.current = clampedNext;
		setActiveSkillRefsRaw(prev => (areSkillRefListsEqual(prev, clampedNext) ? prev : clampedNext));
	}, []);

	const updateSkillSessionIDState = useCallback((next: string | null) => {
		skillSessionIDRef.current = next;
		setSkillSessionID(prev => (prev === next ? prev : next));
	}, []);

	const getCurrentSkillSessionID = useCallback(() => {
		return skillSessionIDRef.current;
	}, []);

	const getCurrentEnabledSkillRefs = useCallback(() => {
		return enabledSkillRefsRef.current;
	}, []);

	const getCurrentActiveSkillRefs = useCallback(() => {
		return activeSkillRefsRef.current;
	}, []);

	useEffect(() => {
		availableSkillKeySetRef.current = new Set(
			allSkills
				.filter(item => {
					return isInstructionSkillListItem(item);
				})
				.map(item => {
					return skillRefKey(skillRefFromListItem(item));
				})
		);
	}, [allSkills]);

	useEffect(() => {
		skillsLoadingRef.current = skillsLoading;
	}, [skillsLoading]);

	const filterSkillRefsToLoadedCatalog = useCallback((refs: SkillRef[] | null | undefined): SkillRef[] => {
		const normalized = normalizeSkillRefs(refs);
		if (skillsLoadingRef.current || !skillsCatalogReadyRef.current) {
			return normalized;
		}

		return normalized.filter(ref => availableSkillKeySetRef.current.has(skillRefKey(ref)));
	}, []);

	const closeSkillSessionBestEffort = useCallback((sid: string | null | undefined) => {
		if (!sid) {
			return;
		}
		void skillStoreAPI.closeSkillSession(sid).catch(() => {});
	}, []);

	const applySkillSelectionState = useCallback(
		async (
			nextEnabledInput: SkillRef[] | null | undefined,
			nextActiveInput: SkillRef[] | null | undefined,
			options?: ApplySkillSelectionStateOptions
		) => {
			const syncSession = options?.syncSession ?? 'if-session-exists';
			const forceResetSession = options?.forceResetSession ?? false;

			const nextEnabled = filterSkillRefsToLoadedCatalog(nextEnabledInput);
			const nextActive = clampActiveSkillRefsToEnabled(nextEnabled, filterSkillRefsToLoadedCatalog(nextActiveInput));
			const prevEnabled = enabledSkillRefsRef.current;
			const prevActive = activeSkillRefsRef.current;
			const prevSelectionStateKey = buildSkillSessionStateKey(prevEnabled, prevActive);

			const prevSessionID = skillSessionIDRef.current;
			const prevSessionStateKey = sessionStateKeyRef.current;
			const nextSessionStateKey = buildSkillSessionStateKey(nextEnabled, nextActive);
			const hadSession = Boolean(prevSessionID);
			const selectionChanged = prevSelectionStateKey !== nextSessionStateKey;

			const stateChanged = hadSession ? prevSessionStateKey !== nextSessionStateKey || !prevSessionStateKey : false;
			let syncVersion = skillSessionSyncVersionRef.current;

			updateEnabledSkillRefsState(nextEnabled);
			updateActiveSkillRefsState(nextActive);
			if (selectionChanged || forceResetSession) {
				syncVersion = advanceSkillSessionSyncVersion();
			}
			const shouldEnsureSession =
				nextEnabled.length > 0 &&
				((syncSession === 'if-session-exists' && hadSession && (stateChanged || forceResetSession)) ||
					(syncSession === 'ensure-if-enabled' && (!hadSession || stateChanged || forceResetSession)));

			const shouldCloseExistingSession =
				hadSession &&
				(nextEnabled.length === 0 ||
					forceResetSession ||
					(syncSession !== 'none' && stateChanged && !shouldEnsureSession));

			const shouldKeepExistingSession =
				hadSession && nextEnabled.length > 0 && !forceResetSession && (syncSession === 'none' || !stateChanged);

			if (shouldKeepExistingSession) {
				sessionStateKeyRef.current = nextSessionStateKey;
				return;
			}

			if (!hadSession && !shouldEnsureSession) {
				sessionStateKeyRef.current = '';
				return;
			}

			if (!(selectionChanged || forceResetSession)) {
				syncVersion = advanceSkillSessionSyncVersion();
			}
			updateSkillSessionIDState(null);
			sessionStateKeyRef.current = '';

			if (!shouldEnsureSession) {
				if (prevSessionID && shouldCloseExistingSession) {
					closeSkillSessionBestEffort(prevSessionID);
				}
				return;
			}

			try {
				const sess = await skillStoreAPI.createSkillSession(
					prevSessionID ?? undefined,
					undefined,
					nextEnabled,
					nextActive
				);

				if (skillSessionSyncVersionRef.current !== syncVersion) {
					closeSkillSessionBestEffort(sess.sessionID);
					return;
				}

				const latestEnabled = enabledSkillRefsRef.current;
				const latestActive = activeSkillRefsRef.current;
				const latestSelectionStateKey = buildSkillSessionStateKey(latestEnabled, latestActive);

				if (latestSelectionStateKey !== nextSessionStateKey) {
					closeSkillSessionBestEffort(sess.sessionID);
					return;
				}

				const resolvedActive = clampActiveSkillRefsToEnabled(latestEnabled, sess.activeSkillRefs ?? latestActive);
				updateSkillSessionIDState(sess.sessionID);
				updateActiveSkillRefsState(resolvedActive);
				sessionStateKeyRef.current = buildSkillSessionStateKey(latestEnabled, resolvedActive);
			} catch {
				if (skillSessionSyncVersionRef.current !== syncVersion) {
					return;
				}
				if (prevSessionID) {
					closeSkillSessionBestEffort(prevSessionID);
				}
			}
		},
		[
			advanceSkillSessionSyncVersion,
			closeSkillSessionBestEffort,
			filterSkillRefsToLoadedCatalog,
			updateActiveSkillRefsState,
			updateEnabledSkillRefsState,
			updateSkillSessionIDState,
		]
	);

	const setEnabledSkillRefs = useCallback<Dispatch<SetStateAction<SkillRef[]>>>(
		update => {
			cancelPendingEnableAllSkills();
			const prevEnabled = enabledSkillRefsRef.current;
			const nextEnabled = resolveStateUpdate(update, prevEnabled);
			void applySkillSelectionState(nextEnabled, activeSkillRefsRef.current, {
				syncSession: 'if-session-exists',
			});
		},
		[applySkillSelectionState, cancelPendingEnableAllSkills]
	);

	const setActiveSkillRefs = useCallback<Dispatch<SetStateAction<SkillRef[]>>>(
		update => {
			const prevActive = activeSkillRefsRef.current;
			const nextActive = resolveStateUpdate(update, prevActive);
			void applySkillSelectionState(enabledSkillRefsRef.current, nextActive, {
				syncSession: 'if-session-exists',
			});
		},
		[applySkillSelectionState]
	);

	const setActiveSkillRefsFromSession = useCallback<Dispatch<SetStateAction<SkillRef[]>>>(
		update => {
			const prevActive = activeSkillRefsRef.current;
			const nextActive = resolveStateUpdate(update, prevActive);
			void applySkillSelectionState(enabledSkillRefsRef.current, nextActive, {
				syncSession: 'none',
			});
		},
		[applySkillSelectionState]
	);

	useEffect(() => {
		if (skillsLoading) {
			return;
		}

		const nextEnabled = filterSkillRefsToLoadedCatalog(enabledSkillRefsRef.current);
		const nextActive = clampActiveSkillRefsToEnabled(
			nextEnabled,
			filterSkillRefsToLoadedCatalog(activeSkillRefsRef.current)
		);

		if (
			areSkillRefListsEqual(enabledSkillRefsRef.current, nextEnabled) &&
			areSkillRefListsEqual(activeSkillRefsRef.current, nextActive)
		) {
			return;
		}

		void applySkillSelectionState(nextEnabled, nextActive, { syncSession: 'if-session-exists' });
	}, [allSkills, applySkillSelectionState, filterSkillRefsToLoadedCatalog, skillsLoading]);

	useEffect(() => {
		return () => {
			const sid = skillSessionIDRef.current;
			if (!sid) {
				return;
			}
			void skillStoreAPI.closeSkillSession(sid).catch(() => {});
		};
	}, []);

	const refreshSkills = useCallback(async () => {
		setSkillsLoading(true);
		setSkillsLoadError(null);
		try {
			const out = await loadComposerSkillsCatalog(true);
			setAllSkills(out);
			skillsCatalogReadyRef.current = true;
		} catch (error) {
			console.error('Failed to refresh skills catalog:', error);
			setSkillsLoadError(
				error instanceof Error && error.message.trim()
					? error.message
					: 'Failed to refresh the skills catalog. Existing skill selections were kept.'
			);
		} finally {
			setSkillsLoading(false);
			if (skillsCatalogLoadPromiseRef.current) {
				skillsCatalogLoadPromiseRef.current = null;
			}
		}
	}, []);

	// Fetch skills catalog (store listSkills; NOT runtime listRuntimeSkills).
	useEffect(() => {
		let cancelled = false;
		const loadPromise = loadComposerSkillsCatalog();
		skillsCatalogLoadPromiseRef.current = loadPromise;

		loadPromise
			.then(out => {
				if (cancelled) {
					return;
				}
				setAllSkills(out);
				setSkillsLoadError(null);
				skillsCatalogReadyRef.current = true;
			})
			.catch((error: unknown) => {
				if (!cancelled) {
					setSkillsLoadError(
						error instanceof Error && error.message.trim()
							? error.message
							: 'Failed to load the skills catalog. Existing skill selections were kept.'
					);
				}
			})
			.finally(() => {
				if (!cancelled) {
					setSkillsLoading(false);
				}
				if (skillsCatalogLoadPromiseRef.current === loadPromise) {
					skillsCatalogLoadPromiseRef.current = null;
				}
			});

		return () => {
			cancelled = true;
			if (skillsCatalogLoadPromiseRef.current === loadPromise) {
				skillsCatalogLoadPromiseRef.current = null;
			}
		};
	}, []);

	const enableAllSkills = useCallback(() => {
		const requestVersion = enableAllSkillsRequestVersionRef.current + 1;
		enableAllSkillsRequestVersionRef.current = requestVersion;

		void (async () => {
			const pendingLoad = skillsCatalogLoadPromiseRef.current;
			const loadedSkills =
				allSkills.length > 0 ? allSkills : pendingLoad ? await pendingLoad.catch(() => []) : ([] as SkillListItem[]);

			if (enableAllSkillsRequestVersionRef.current !== requestVersion) {
				return;
			}
			if (loadedSkills.length === 0) {
				return;
			}

			void applySkillSelectionState(
				loadedSkills
					.filter(s => {
						return isInstructionSkillListItem(s);
					})
					.map(i => skillRefFromListItem(i)),
				activeSkillRefsRef.current,
				{
					syncSession: 'if-session-exists',
				}
			);
		})();
	}, [allSkills, applySkillSelectionState]);

	const disableAllSkills = useCallback(() => {
		setEnabledSkillRefs([]);
	}, [setEnabledSkillRefs]);

	const listActiveSkillRefs = useCallback(async (sid: string): Promise<SkillRef[]> => {
		const allowSkillRefs = enabledSkillRefsRef.current;
		if (!sid || allowSkillRefs.length === 0) {
			return [];
		}

		const items = await skillStoreAPI.listRuntimeSkills({
			sessionID: sid,
			activity: 'active',
			allowSkillRefs,
		});

		return clampActiveSkillRefsToEnabled(
			allowSkillRefs,
			(items ?? []).map(it => it.skillRef)
		);
	}, []);

	const ensureSkillSession = useCallback(async (): Promise<string | null> => {
		const currentEnabled = enabledSkillRefsRef.current;
		if (currentEnabled.length === 0) {
			return null;
		}

		const currentActive = activeSkillRefsRef.current;
		const currentStateKey = buildSkillSessionStateKey(currentEnabled, currentActive);
		const existing = skillSessionIDRef.current;

		if (existing && sessionStateKeyRef.current === currentStateKey) {
			return existing;
		}
		const syncVersion = advanceSkillSessionSyncVersion();

		const sess = await skillStoreAPI.createSkillSession(
			existing ?? undefined, // closeSessionID (best-effort)
			undefined, // maxActivePerSession
			currentEnabled, // allowSkillRefs (REQUIRED)
			currentActive // initial active from conversation
		);

		if (skillSessionSyncVersionRef.current !== syncVersion) {
			closeSkillSessionBestEffort(sess.sessionID);
			return null;
		}

		const latestEnabled = enabledSkillRefsRef.current;
		const latestActive = activeSkillRefsRef.current;
		const latestStateKey = buildSkillSessionStateKey(latestEnabled, latestActive);

		if (latestStateKey !== currentStateKey) {
			closeSkillSessionBestEffort(sess.sessionID);
			return null;
		}

		const resolvedActive = clampActiveSkillRefsToEnabled(latestEnabled, sess.activeSkillRefs ?? latestActive);
		sessionStateKeyRef.current = buildSkillSessionStateKey(latestEnabled, resolvedActive);
		updateSkillSessionIDState(sess.sessionID);

		return sess.sessionID;
	}, [advanceSkillSessionSyncVersion, closeSkillSessionBestEffort, updateSkillSessionIDState]);

	return {
		allSkills,
		skillsLoading,
		skillsLoadError,
		enabledSkillRefs,
		activeSkillRefs,
		skillSessionID,
		setEnabledSkillRefs,
		setActiveSkillRefs,
		setActiveSkillRefsFromSession,
		enableAllSkills,
		disableAllSkills,
		refreshSkills,
		applySkillSelectionState,
		ensureSkillSession,
		listActiveSkillRefs,
		getCurrentEnabledSkillRefs,
		getCurrentActiveSkillRefs,
		getCurrentSkillSessionID,
	};
}
