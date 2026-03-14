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
} from '@/chats/skills/skill_utils';

interface PendingMessageSkillSelectionState {
	enabled?: SkillRef[];
	active?: SkillRef[];
	resetSession: boolean;
	timeoutID: number | null;
}

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
		nextActiveInput: SkillRef[] | null | undefined
	) => void;
	ensureSkillSession: () => Promise<string | null>;
	listActiveSkillRefs: (sid: string) => Promise<SkillRef[]>;
	applyEnabledSkillRefsFromMessage: (refs: SkillRef[]) => void;
	applyActiveSkillRefsFromMessage: (refs: SkillRef[]) => void;
	flushPendingMessageSkillSelection: () => void;
	getCurrentEnabledSkillRefs: () => SkillRef[];
	getCurrentActiveSkillRefs: () => SkillRef[];
	getCurrentSkillSessionID: () => string | null;
}

export function useComposerSkills(): UseComposerSkillsResult {
	const [allSkills, setAllSkills] = useState<SkillListItem[]>([]);
	const [enabledSkillRefs, setEnabledSkillRefsRaw] = useState<SkillRef[]>([]);
	const [activeSkillRefs, setActiveSkillRefsRaw] = useState<SkillRef[]>([]);
	const [skillSessionID, setSkillSessionID] = useState<string | null>(null);
	const [skillsLoading, setSkillsLoading] = useState(true);

	// Track the allowlist fingerprint used to create the current session.
	const sessionAllowlistKeyRef = useRef<string>('');

	const skillSessionIDRef = useRef<string | null>(null);
	const enabledSkillRefsRef = useRef<SkillRef[]>([]);
	const activeSkillRefsRef = useRef<SkillRef[]>([]);
	const skillsCatalogLoadPromiseRef = useRef<Promise<SkillListItem[]> | null>(null);
	const enableAllSkillsRequestVersionRef = useRef(0);
	const pendingMessageSkillSelectionRef = useRef<PendingMessageSkillSelectionState>({
		timeoutID: null,
		resetSession: false,
	});

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

	const applySkillSelectionStateInternal = useCallback(
		(nextEnabledInput: SkillRef[] | null | undefined, nextActiveInput: SkillRef[] | null | undefined) => {
			const nextEnabled = normalizeSkillRefs(nextEnabledInput);
			const nextActive = clampActiveSkillRefsToEnabled(nextEnabled, nextActiveInput);

			const prevSessionID = skillSessionIDRef.current;
			const prevSessionAllowlistKey = sessionAllowlistKeyRef.current;
			const nextAllowlistKey = buildSkillRefsFingerprint(nextEnabled);

			if (nextEnabled.length === 0) {
				sessionAllowlistKeyRef.current = '';
				if (prevSessionID) {
					updateSkillSessionIDState(null);
					closeSkillSessionBestEffort(prevSessionID);
				}
			} else if (!prevSessionID) {
				sessionAllowlistKeyRef.current = '';
			} else if (!prevSessionAllowlistKey) {
				// Session existed before we had a tracked allowlist; initialize tracking.
				sessionAllowlistKeyRef.current = nextAllowlistKey;
			} else if (prevSessionAllowlistKey !== nextAllowlistKey) {
				sessionAllowlistKeyRef.current = '';
				updateSkillSessionIDState(null);
				closeSkillSessionBestEffort(prevSessionID);
			}

			updateEnabledSkillRefsState(nextEnabled);
			updateActiveSkillRefsState(nextActive);
		},
		[closeSkillSessionBestEffort, updateActiveSkillRefsState, updateEnabledSkillRefsState, updateSkillSessionIDState]
	);

	const applySkillSelectionState = useCallback(
		(nextEnabledInput: SkillRef[] | null | undefined, nextActiveInput: SkillRef[] | null | undefined) => {
			cancelPendingEnableAllSkills();
			applySkillSelectionStateInternal(nextEnabledInput, nextActiveInput);
		},
		[applySkillSelectionStateInternal, cancelPendingEnableAllSkills]
	);

	const setEnabledSkillRefs = useCallback<Dispatch<SetStateAction<SkillRef[]>>>(
		update => {
			cancelPendingEnableAllSkills();
			const prevEnabled = enabledSkillRefsRef.current;
			const nextEnabled = resolveStateUpdate(update, prevEnabled);
			applySkillSelectionStateInternal(nextEnabled, activeSkillRefsRef.current);
		},
		[applySkillSelectionStateInternal, cancelPendingEnableAllSkills]
	);

	const setActiveSkillRefs = useCallback<Dispatch<SetStateAction<SkillRef[]>>>(
		update => {
			const prevActive = activeSkillRefsRef.current;
			const nextActive = resolveStateUpdate(update, prevActive);
			applySkillSelectionStateInternal(enabledSkillRefsRef.current, nextActive);
		},
		[applySkillSelectionStateInternal]
	);

	const flushPendingMessageSkillSelection = useCallback(() => {
		const pending = pendingMessageSkillSelectionRef.current;
		if (pending.timeoutID != null) {
			window.clearTimeout(pending.timeoutID);
			pending.timeoutID = null;
		}

		if (pending.enabled == null && pending.active == null) return;
		if (pending.enabled == null && pending.active == null && !pending.resetSession) return;

		const nextEnabled = pending.enabled ?? enabledSkillRefsRef.current;
		const nextActive = pending.active ?? activeSkillRefsRef.current;
		const shouldResetSession = pending.resetSession;

		pending.enabled = undefined;
		pending.active = undefined;
		pending.resetSession = false;

		if (shouldResetSession) {
			const prevSessionID = skillSessionIDRef.current;
			sessionAllowlistKeyRef.current = '';
			if (prevSessionID) {
				updateSkillSessionIDState(null);
				closeSkillSessionBestEffort(prevSessionID);
			}
		}

		cancelPendingEnableAllSkills();
		applySkillSelectionStateInternal(nextEnabled, nextActive);
	}, [
		applySkillSelectionStateInternal,
		cancelPendingEnableAllSkills,
		closeSkillSessionBestEffort,
		updateSkillSessionIDState,
	]);

	const schedulePendingMessageSkillSelectionFlush = useCallback(() => {
		const pending = pendingMessageSkillSelectionRef.current;
		if (pending.timeoutID != null) return;

		pending.timeoutID = window.setTimeout(() => {
			pending.timeoutID = null;
			flushPendingMessageSkillSelection();
		}, 0);
	}, [flushPendingMessageSkillSelection]);

	useEffect(() => {
		const pending = pendingMessageSkillSelectionRef.current;
		return () => {
			if (pending.timeoutID != null) {
				window.clearTimeout(pending.timeoutID);
				pending.timeoutID = null;
			}
		};
	}, []);

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

			applySkillSelectionStateInternal(loadedSkills.map(skillRefFromListItem), activeSkillRefsRef.current);
		})();
	}, [allSkills, applySkillSelectionStateInternal]);

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
		const currentEnabledKey = buildSkillRefsFingerprint(currentEnabled);
		const existing = skillSessionIDRef.current;

		if (existing && sessionAllowlistKeyRef.current === currentEnabledKey) return existing;

		const sess = await skillStoreAPI.createSkillSession(
			existing ?? undefined, // closeSessionID (best-effort)
			undefined, // maxActivePerSession
			currentEnabled, // allowSkillRefs (REQUIRED)
			currentActive // initial active from conversation
		);

		sessionAllowlistKeyRef.current = currentEnabledKey;
		updateSkillSessionIDState(sess.sessionID);
		updateActiveSkillRefsState(clampActiveSkillRefsToEnabled(currentEnabled, sess.activeSkillRefs ?? []));
		return sess.sessionID;
	}, [updateActiveSkillRefsState, updateSkillSessionIDState]);

	const applyEnabledSkillRefsFromMessage = useCallback(
		(refs: SkillRef[]) => {
			const pending = pendingMessageSkillSelectionRef.current;
			pending.enabled = refs ?? [];
			pending.resetSession = true;
			schedulePendingMessageSkillSelectionFlush();
		},
		[schedulePendingMessageSkillSelectionFlush]
	);

	const applyActiveSkillRefsFromMessage = useCallback(
		(refs: SkillRef[]) => {
			const pending = pendingMessageSkillSelectionRef.current;
			pending.active = refs ?? [];
			pending.resetSession = true;
			schedulePendingMessageSkillSelectionFlush();
		},
		[schedulePendingMessageSkillSelectionFlush]
	);

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
		applyEnabledSkillRefsFromMessage,
		applyActiveSkillRefsFromMessage,
		flushPendingMessageSkillSelection,
		getCurrentEnabledSkillRefs,
		getCurrentActiveSkillRefs,
		getCurrentSkillSessionID,
	};
}
