import { type Dispatch, type SetStateAction, useCallback, useEffect, useRef, useState } from 'react';

import type { SkillListItem, SkillRef } from '@/spec/skill';

import { resolveStateUpdate } from '@/lib/hook_utils';

import { skillStoreAPI } from '@/apis/baseapi';

import {
	areSkillRefListsEqual,
	buildSkillRefsFingerprint,
	clampActiveSkillRefsToEnabled,
	normalizeSkillRefs,
	skillRefFromListItem,
} from '@/skills/lib/skill_identity_utils';

type SkillSessionSyncMode = 'none' | 'if-session-exists' | 'ensure-if-enabled';
type ApplySkillSelectionStateOptions = {
	syncSession?: SkillSessionSyncMode;
	forceResetSession?: boolean;
};

interface UseComposerSkillsResult {
	allSkills: SkillListItem[];
	skillsLoading: boolean;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	skillSessionID: string | null;
	setEnabledSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	enableAllSkills: () => void;
	disableAllSkills: () => void;
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

export function useComposerSkills(): UseComposerSkillsResult {
	const [allSkills, setAllSkills] = useState<SkillListItem[]>([]);
	const [enabledSkillRefs, setEnabledSkillRefsRaw] = useState<SkillRef[]>([]);
	const [activeSkillRefs, setActiveSkillRefsRaw] = useState<SkillRef[]>([]);
	const [skillSessionID, setSkillSessionID] = useState<string | null>(null);
	const [skillsLoading, setSkillsLoading] = useState(true);

	const sessionStateKeyRef = useRef('');
	const skillSessionSyncVersionRef = useRef(0);

	const skillSessionIDRef = useRef<string | null>(null);
	const enabledSkillRefsRef = useRef<SkillRef[]>([]);
	const activeSkillRefsRef = useRef<SkillRef[]>([]);
	const skillsCatalogLoadPromiseRef = useRef<Promise<SkillListItem[]> | null>(null);
	const enableAllSkillsRequestVersionRef = useRef(0);

	const cancelPendingEnableAllSkills = useCallback(() => {
		enableAllSkillsRequestVersionRef.current += 1;
	}, []);

	const updateEnabledSkillRefsState = useCallback((next: SkillRef[]) => {
		enabledSkillRefsRef.current = next;
		setEnabledSkillRefsRaw(prev => (areSkillRefListsEqual(prev, next) ? prev : next));
	}, []);

	const updateActiveSkillRefsState = useCallback((next: SkillRef[]) => {
		activeSkillRefsRef.current = next;
		setActiveSkillRefsRaw(prev => (areSkillRefListsEqual(prev, next) ? prev : next));
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

	const closeSkillSessionBestEffort = useCallback((sid: string | null | undefined) => {
		if (!sid) return;
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

			const nextEnabled = normalizeSkillRefs(nextEnabledInput);
			const nextActive = clampActiveSkillRefsToEnabled(nextEnabled, nextActiveInput);

			const prevSessionID = skillSessionIDRef.current;
			const prevSessionStateKey = sessionStateKeyRef.current;
			const nextSessionStateKey = buildSkillSessionStateKey(nextEnabled, nextActive);
			const hadSession = Boolean(prevSessionID);
			const stateChanged = hadSession ? prevSessionStateKey !== nextSessionStateKey || !prevSessionStateKey : false;

			updateEnabledSkillRefsState(nextEnabled);
			updateActiveSkillRefsState(nextActive);

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

			const syncVersion = skillSessionSyncVersionRef.current + 1;
			skillSessionSyncVersionRef.current = syncVersion;
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

				const resolvedActive = clampActiveSkillRefsToEnabled(nextEnabled, sess.activeSkillRefs ?? nextActive);
				updateSkillSessionIDState(sess.sessionID);
				updateActiveSkillRefsState(resolvedActive);
				sessionStateKeyRef.current = buildSkillSessionStateKey(nextEnabled, resolvedActive);
			} catch {
				if (skillSessionSyncVersionRef.current !== syncVersion) return;
				if (prevSessionID) {
					closeSkillSessionBestEffort(prevSessionID);
				}
			}
		},
		[closeSkillSessionBestEffort, updateActiveSkillRefsState, updateEnabledSkillRefsState, updateSkillSessionIDState]
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
				syncSession: 'none',
			});
		},
		[applySkillSelectionState]
	);

	useEffect(() => {
		return () => {
			const sid = skillSessionIDRef.current;
			if (!sid) return;
			void skillStoreAPI.closeSkillSession(sid).catch(() => {});
		};
	}, []);

	const fetchAllSkills = useCallback(async (): Promise<SkillListItem[]> => {
		const out: SkillListItem[] = [];
		let token: string | undefined = undefined;

		for (let guard = 0; guard < 50; guard += 1) {
			const resp = await skillStoreAPI.listSkills(
				undefined,
				undefined,
				false, // includeDisabled
				false, // includeMissing
				200, // recommendedPageSize
				token
			);

			out.push(...(resp.skillListItems ?? []));
			token = resp.nextPageToken;
			if (!token) break;
		}

		return out;
	}, []);

	// Fetch skills catalog (store listSkills; NOT runtime listRuntimeSkills).
	useEffect(() => {
		let cancelled = false;
		const loadPromise = fetchAllSkills();
		skillsCatalogLoadPromiseRef.current = loadPromise;

		loadPromise
			.then(out => {
				if (cancelled) return;
				setAllSkills(out);
			})
			.catch(() => {
				if (!cancelled) setAllSkills([]);
			})
			.finally(() => {
				if (!cancelled) setSkillsLoading(false);
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
	}, [fetchAllSkills]);

	const enableAllSkills = useCallback(() => {
		const requestVersion = enableAllSkillsRequestVersionRef.current + 1;
		enableAllSkillsRequestVersionRef.current = requestVersion;

		void (async () => {
			const pendingLoad = skillsCatalogLoadPromiseRef.current;
			const loadedSkills =
				allSkills.length > 0 ? allSkills : pendingLoad ? await pendingLoad.catch(() => []) : ([] as SkillListItem[]);

			if (enableAllSkillsRequestVersionRef.current !== requestVersion) return;
			if (loadedSkills.length === 0) return;

			void applySkillSelectionState(loadedSkills.map(skillRefFromListItem), activeSkillRefsRef.current, {
				syncSession: 'if-session-exists',
			});
		})();
	}, [allSkills, applySkillSelectionState]);

	const disableAllSkills = useCallback(() => {
		setEnabledSkillRefs([]);
	}, [setEnabledSkillRefs]);

	const listActiveSkillRefs = useCallback(async (sid: string): Promise<SkillRef[]> => {
		const allowSkillRefs = enabledSkillRefsRef.current;
		if (!sid || allowSkillRefs.length === 0) return [];

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
		if (currentEnabled.length === 0) return null;

		const currentActive = activeSkillRefsRef.current;
		const currentStateKey = buildSkillSessionStateKey(currentEnabled, currentActive);
		const existing = skillSessionIDRef.current;

		if (existing && sessionStateKeyRef.current === currentStateKey) return existing;

		const sess = await skillStoreAPI.createSkillSession(
			existing ?? undefined, // closeSessionID (best-effort)
			undefined, // maxActivePerSession
			currentEnabled, // allowSkillRefs (REQUIRED)
			currentActive // initial active from conversation
		);

		sessionStateKeyRef.current = buildSkillSessionStateKey(currentEnabled, sess.activeSkillRefs ?? currentActive);
		updateSkillSessionIDState(sess.sessionID);
		updateActiveSkillRefsState(clampActiveSkillRefsToEnabled(currentEnabled, sess.activeSkillRefs ?? []));
		return sess.sessionID;
	}, [updateActiveSkillRefsState, updateSkillSessionIDState]);

	return {
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
		getCurrentEnabledSkillRefs,
		getCurrentActiveSkillRefs,
		getCurrentSkillSessionID,
	};
}
