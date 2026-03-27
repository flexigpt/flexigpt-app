import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { type Tool, type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { resolveStateUpdate } from '@/lib/hook_utils';

import { toolStoreAPI } from '@/apis/baseapi';

import type { AttachedToolEntry } from '@/chats/inputarea/platedoc/tool_document_ops';
import {
	normalizeWebSearchChoiceTemplates,
	type WebSearchChoiceTemplate,
	webSearchTemplateFromChoice,
} from '@/chats/inputarea/tools/websearch_utils';
import {
	type ConversationToolStateEntry,
	toolStoreChoicesToConversationTools,
} from '@/tools/lib/conversation_tool_utils';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

interface UseComposerToolConfigArgs {
	getAttachedToolEntries: (uniqueByIdentity?: boolean) => AttachedToolEntry[];
}

function conversationToolHydrationKey(entry: ConversationToolStateEntry): string {
	return `${entry.toolStoreChoice.bundleID}::${entry.toolStoreChoice.toolSlug}::${entry.toolStoreChoice.toolVersion}`;
}

function areStringArraysEqual(a: string[] | undefined, b: string[] | undefined): boolean {
	const left = a ?? [];
	const right = b ?? [];
	if (left.length !== right.length) return false;
	return left.every((item, idx) => item === right[idx]);
}

function areArgStatusesEqual(
	a: ConversationToolStateEntry['argStatus'],
	b: ConversationToolStateEntry['argStatus']
): boolean {
	if (!a && !b) return true;
	if (!a || !b) return false;

	return (
		a.hasSchema === b.hasSchema &&
		a.isInstancePresent === b.isInstancePresent &&
		a.isInstanceJSONValid === b.isInstanceJSONValid &&
		a.isSatisfied === b.isSatisfied &&
		areStringArraysEqual(a.requiredKeys, b.requiredKeys) &&
		areStringArraysEqual(a.missingRequired, b.missingRequired)
	);
}

function getConversationToolArgsBlocked(entries: ConversationToolStateEntry[]): boolean {
	for (const entry of entries) {
		if (!entry.enabled) continue;

		const status = entry.argStatus;
		if (status?.hasSchema && !status.isSatisfied) {
			return true;
		}
	}

	return false;
}

export function useComposerToolConfig({ getAttachedToolEntries }: UseComposerToolConfigArgs): {
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	toolArgsBlocked: boolean;
	recomputeAttachedToolArgsBlocked: () => void;
	clearAttachedToolValidation: () => void;
	applyConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	applyWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
} {
	const isMountedRef = useRef(true);

	const [conversationToolsState, setConversationToolsStateRaw] = useState<ConversationToolStateEntry[]>([]);
	const conversationToolsStateRef = useRef<ConversationToolStateEntry[]>([]);

	const [webSearchTemplates, setWebSearchTemplatesRaw] = useState<WebSearchChoiceTemplate[]>([]);
	const webSearchTemplatesRef = useRef<WebSearchChoiceTemplate[]>([]);

	const [attachedToolArgsBlocked, setAttachedToolArgsBlocked] = useState(false);

	// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-arguments
	const conversationToolDefsCacheRef = useRef<Map<string, Tool>>(new Map());
	const hydratingConversationToolKeysRef = useRef(new Set<string>());

	useEffect(() => {
		return () => {
			isMountedRef.current = false;
		};
	}, []);

	useEffect(() => {
		conversationToolsStateRef.current = conversationToolsState;
	}, [conversationToolsState]);

	const primeConversationToolsFromCache = useCallback((entries: ConversationToolStateEntry[]) => {
		let changed = false;

		const next = entries.map(entry => {
			const cacheKey = conversationToolHydrationKey(entry);
			const def = entry.toolDefinition ?? conversationToolDefsCacheRef.current.get(cacheKey);
			if (!def) return entry;

			const argStatus = computeToolUserArgsStatus(def.userArgSchema, entry.toolStoreChoice.userArgSchemaInstance);

			if (entry.toolDefinition === def && areArgStatusesEqual(entry.argStatus, argStatus)) {
				return entry;
			}

			changed = true;
			return { ...entry, toolDefinition: def, argStatus };
		});

		return changed ? next : entries;
	}, []);

	const hydrateConversationToolsIfNeeded = useCallback(
		(entries: ConversationToolStateEntry[]) => {
			const inFlight = hydratingConversationToolKeysRef.current;
			const cache = conversationToolDefsCacheRef.current;

			const missing = entries.filter(entry => {
				const cacheKey = conversationToolHydrationKey(entry);
				return !entry.toolDefinition && !cache.has(cacheKey) && !inFlight.has(cacheKey);
			});

			if (!missing.length) return;

			const requestedKeys = new Set<string>();
			for (const entry of missing) {
				const cacheKey = conversationToolHydrationKey(entry);
				inFlight.add(cacheKey);
				requestedKeys.add(cacheKey);
			}

			void Promise.all(
				missing.map(async entry => {
					const cacheKey = conversationToolHydrationKey(entry);
					try {
						const def = await toolStoreAPI.getTool(
							entry.toolStoreChoice.bundleID,
							entry.toolStoreChoice.toolSlug,
							entry.toolStoreChoice.toolVersion
						);
						return def ? { cacheKey, def } : null;
					} catch {
						return null;
					}
				})
			)
				.then(results => {
					if (!isMountedRef.current) return;

					let loadedAny = false;
					for (const result of results) {
						if (!result) continue;
						cache.set(result.cacheKey, result.def);
						loadedAny = true;
					}

					if (!loadedAny) return;

					setConversationToolsStateRaw(prev => {
						const next = primeConversationToolsFromCache(prev);
						conversationToolsStateRef.current = next;
						return next;
					});
				})
				.finally(() => {
					for (const key of requestedKeys) {
						inFlight.delete(key);
					}
				});
		},
		[primeConversationToolsFromCache]
	);

	const setConversationToolsState = useCallback<Dispatch<SetStateAction<ConversationToolStateEntry[]>>>(
		update => {
			const prev = conversationToolsStateRef.current;
			const requested = resolveStateUpdate(update, prev);
			const next = primeConversationToolsFromCache(requested);

			conversationToolsStateRef.current = next;
			setConversationToolsStateRaw(next);
			hydrateConversationToolsIfNeeded(next);
		},
		[hydrateConversationToolsIfNeeded, primeConversationToolsFromCache]
	);

	const setWebSearchTemplates = useCallback<Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>>(update => {
		const prev = webSearchTemplatesRef.current;
		const requested = resolveStateUpdate(update, prev);
		const next = normalizeWebSearchChoiceTemplates(requested);

		if (
			prev.length === next.length &&
			prev.every(
				(item, idx) =>
					item.bundleID === next[idx]?.bundleID &&
					item.toolSlug === next[idx]?.toolSlug &&
					item.toolVersion === next[idx]?.toolVersion &&
					item.userArgSchemaInstance === next[idx]?.userArgSchemaInstance
			)
		) {
			return;
		}

		webSearchTemplatesRef.current = next;
		setWebSearchTemplatesRaw(next);
	}, []);

	const recomputeAttachedToolArgsBlocked = useCallback(() => {
		const toolEntries = getAttachedToolEntries(false);
		let nextBlocked = false;

		for (const entry of toolEntries) {
			const schema = entry.toolSnapshot?.userArgSchema;
			const status = computeToolUserArgsStatus(schema, entry.userArgSchemaInstance);
			if (status.hasSchema && !status.isSatisfied) {
				nextBlocked = true;
				break;
			}
		}

		setAttachedToolArgsBlocked(prev => (prev === nextBlocked ? prev : nextBlocked));
	}, [getAttachedToolEntries]);

	const clearAttachedToolValidation = useCallback(() => {
		setAttachedToolArgsBlocked(false);
	}, []);

	const conversationToolArgsBlocked = useMemo(
		() => getConversationToolArgsBlocked(conversationToolsState),
		[conversationToolsState]
	);

	const toolArgsBlocked = attachedToolArgsBlocked || conversationToolArgsBlocked;

	const applyConversationToolsFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			setConversationToolsState(toolStoreChoicesToConversationTools(tools));
		},
		[setConversationToolsState]
	);

	const applyWebSearchFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			const next = normalizeWebSearchChoiceTemplates(
				(tools ?? []).filter(tool => tool.toolType === ToolStoreChoiceType.WebSearch).map(webSearchTemplateFromChoice)
			);
			setWebSearchTemplates(next);
		},
		[setWebSearchTemplates]
	);

	return {
		conversationToolsState,
		setConversationToolsState,
		webSearchTemplates,
		setWebSearchTemplates,
		toolArgsBlocked,
		recomputeAttachedToolArgsBlocked,
		clearAttachedToolValidation,
		applyConversationToolsFromChoices,
		applyWebSearchFromChoices,
	};
}
