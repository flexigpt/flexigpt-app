import { useCallback, useEffect, useSyncExternalStore } from 'react';

import { getUUIDv7 } from '@/lib/uuid_utils';

export type SystemPromptItem = {
	id: string;
	title: string;
	prompt: string;
	createdAt: number;
};

type StoredSystemPromptsV1 = {
	v: 1;
	prompts: SystemPromptItem[];
};

const SYSTEM_PROMPTS_STORAGE_KEY = 'app.chats.system-prompts.v1';

const listeners = new Set<() => void>();
let cachedPrompts: SystemPromptItem[] | null = null;
let storageListenerAttached = false;

function canUseDOM(): boolean {
	return typeof window !== 'undefined' && typeof localStorage !== 'undefined';
}

function buildSystemPromptTitle(prompt: string): string {
	const s = (prompt || '').trim();
	if (!s) return '(empty)';
	return s.length > 64 ? `${s.slice(0, 64)}…` : s;
}

function normalizeStoredPrompt(raw: unknown): SystemPromptItem | null {
	if (!raw || typeof raw !== 'object') return null;

	const obj = raw as Partial<SystemPromptItem>;
	const prompt = typeof obj.prompt === 'string' ? obj.prompt.trim() : '';
	if (!prompt) return null;

	const id = typeof obj.id === 'string' && obj.id.trim() ? obj.id : getUUIDv7();
	const createdAt = typeof obj.createdAt === 'number' && Number.isFinite(obj.createdAt) ? obj.createdAt : Date.now();
	const title = typeof obj.title === 'string' && obj.title.trim() ? obj.title : buildSystemPromptTitle(prompt);

	return {
		id,
		title,
		prompt,
		createdAt,
	};
}

function sortPrompts(prompts: SystemPromptItem[]): SystemPromptItem[] {
	return [...prompts].sort((a, b) => a.createdAt - b.createdAt);
}

function readPromptsFromStorage(): SystemPromptItem[] {
	if (!canUseDOM()) return [];

	try {
		const raw = localStorage.getItem(SYSTEM_PROMPTS_STORAGE_KEY);
		if (!raw) return [];

		const parsed = JSON.parse(raw) as StoredSystemPromptsV1;
		if (!parsed || parsed.v !== 1 || !Array.isArray(parsed.prompts)) return [];

		const normalized = parsed.prompts.map(normalizeStoredPrompt).filter(Boolean) as SystemPromptItem[];

		const dedupedById = new Map<string, SystemPromptItem>();
		for (const item of normalized) {
			if (!dedupedById.has(item.id)) {
				dedupedById.set(item.id, item);
			}
		}

		return sortPrompts([...dedupedById.values()]);
	} catch {
		return [];
	}
}

function writePromptsToStorage(prompts: SystemPromptItem[]) {
	cachedPrompts = sortPrompts(prompts);

	if (!canUseDOM()) return;

	try {
		const payload: StoredSystemPromptsV1 = {
			v: 1,
			prompts: cachedPrompts,
		};
		localStorage.setItem(SYSTEM_PROMPTS_STORAGE_KEY, JSON.stringify(payload));
	} catch {
		// ignore persistence failure; in-memory cache still works for current session
	}
}

function emitChange() {
	for (const listener of listeners) {
		listener();
	}
}

function ensureStorageListener() {
	if (storageListenerAttached || !canUseDOM()) return;

	window.addEventListener('storage', e => {
		if (e.key !== SYSTEM_PROMPTS_STORAGE_KEY) return;
		cachedPrompts = null;
		emitChange();
	});

	storageListenerAttached = true;
}

function addSystemPrompt(prompt: string): SystemPromptItem | undefined {
	const trimmed = prompt.trim();
	if (!trimmed) return undefined;

	const current = getSystemPromptsSnapshot();
	const nextItem = createSystemPromptItem(trimmed);
	writePromptsToStorage([...current, nextItem]);
	emitChange();
	return nextItem;
}

function getSystemPromptsSnapshot(): SystemPromptItem[] {
	if (cachedPrompts) return cachedPrompts;

	cachedPrompts = readPromptsFromStorage();
	return cachedPrompts;
}

function subscribeToSystemPrompts(listener: () => void): () => void {
	listeners.add(listener);
	return () => {
		listeners.delete(listener);
	};
}

function createSystemPromptItem(prompt: string): SystemPromptItem {
	const trimmed = prompt.trim();

	return {
		id: getUUIDv7(),
		title: buildSystemPromptTitle(trimmed),
		prompt: trimmed,
		createdAt: Date.now(),
	};
}

function ensureSystemPrompt(prompt: string): SystemPromptItem | undefined {
	const trimmed = prompt.trim();
	if (!trimmed) return undefined;

	const current = getSystemPromptsSnapshot();
	const existing = current.find(item => item.prompt.trim() === trimmed);
	if (existing) return existing;

	return addSystemPrompt(trimmed);
}

function deleteSystemPrompt(id: string) {
	const current = getSystemPromptsSnapshot();
	const next = current.filter(item => item.id !== id);

	if (next.length === current.length) return;

	writePromptsToStorage(next);
	emitChange();
}

export function useSystemPrompts() {
	useEffect(() => {
		ensureStorageListener();
	}, []);

	const prompts = useSyncExternalStore(
		subscribeToSystemPrompts,
		getSystemPromptsSnapshot,
		() => [] as SystemPromptItem[]
	);

	const addPrompt = useCallback((prompt: string) => {
		return addSystemPrompt(prompt);
	}, []);

	const ensurePrompt = useCallback((prompt: string) => {
		return ensureSystemPrompt(prompt);
	}, []);

	const removePrompt = useCallback((id: string) => {
		deleteSystemPrompt(id);
	}, []);

	return {
		prompts,
		addPrompt,
		ensurePrompt,
		deletePrompt: removePrompt,
	};
}
