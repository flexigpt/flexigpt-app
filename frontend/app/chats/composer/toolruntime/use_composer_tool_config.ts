import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { ResolvedToolView, ToolStoreChoice } from '@/spec/tool';
import { ToolStoreChoiceType } from '@/spec/tool';

import { getErrorMessage } from '@/lib/error_utils';
import { resolveStateUpdate } from '@/lib/hook_utils';

import { toolManagementAPI } from '@/apis/baseapi';
import { toolStoreChoiceFromSelection } from '@/apis/tool_management';

import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';
import { normalizeWebSearchChoiceTemplates, webSearchTemplateFromChoice } from '@/chats/composer/tools/websearch_utils';
import { toolStoreChoicesToConversationTools } from '@/tools/lib/conversation_tool_utils';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

interface UseComposerToolConfigArgs {
	getAttachedToolEntries: (uniqueByIdentity?: boolean) => AttachedToolEntry[];
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
	const mountedRef = useRef(true);
	const [conversationToolsState, setConversationToolsStateRaw] = useState<ConversationToolStateEntry[]>([]);
	const conversationRef = useRef<ConversationToolStateEntry[]>([]);
	const [webSearchTemplates, setWebSearchTemplatesRaw] = useState<WebSearchChoiceTemplate[]>([]);
	const webSearchRef = useRef<WebSearchChoiceTemplate[]>([]);
	const [attachedBlocked, setAttachedBlocked] = useState(false);

	const resolvedCacheRef = useRef(new Map<string, ResolvedToolView>());
	const inFlightRef = useRef(new Set<string>());

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
		};
	}, []);

	const commitConversation = useCallback((entries: ConversationToolStateEntry[]) => {
		conversationRef.current = entries;
		setConversationToolsStateRaw(entries);
	}, []);

	const primeFromCache = useCallback((entries: ConversationToolStateEntry[]) => {
		return entries.map(entry => {
			const key = toolIdentityKey(entry.toolStoreChoice.target);
			const resolved = resolvedCacheRef.current.get(key);
			const definition = resolved?.tool ?? entry.toolDefinition;
			if (!definition) {
				return entry;
			}

			const choice = resolved ? toolStoreChoiceFromSelection(entry.toolStoreChoice, resolved) : entry.toolStoreChoice;

			return {
				...entry,
				toolStoreChoice: choice,
				toolDefinition: definition,
				toolLoadError: undefined,
				argStatus: computeToolUserArgsStatus(definition.userArgSchema, choice.userArgSchemaInstance),
			};
		});
	}, []);

	const hydrateMissing = useCallback(
		(entries: ConversationToolStateEntry[]) => {
			for (const entry of entries) {
				const target = entry.toolStoreChoice.target;
				const key = toolIdentityKey(target);
				if (entry.toolDefinition || resolvedCacheRef.current.has(key) || inFlightRef.current.has(key)) {
					continue;
				}

				inFlightRef.current.add(key);
				void (async () => {
					try {
						const resolved = await toolManagementAPI.resolveMappedTool(target);
						if (!mountedRef.current) {
							return;
						}
						resolvedCacheRef.current.set(key, resolved);
						commitConversation(primeFromCache(conversationRef.current));
					} catch (error) {
						if (!mountedRef.current) {
							return;
						}
						const message = getErrorMessage(error, 'The selected Tool could not be resolved.');
						commitConversation(
							conversationRef.current.map(current =>
								toolIdentityKey(current.toolStoreChoice.target) === key && !current.toolDefinition
									? { ...current, toolLoadError: message }
									: current
							)
						);
					} finally {
						inFlightRef.current.delete(key);
					}
				})();
			}
		},
		[commitConversation, primeFromCache]
	);

	const setConversationToolsState = useCallback<Dispatch<SetStateAction<ConversationToolStateEntry[]>>>(
		update => {
			const next = primeFromCache(resolveStateUpdate(update, conversationRef.current));
			commitConversation(next);
			hydrateMissing(next);
		},
		[commitConversation, hydrateMissing, primeFromCache]
	);

	const setWebSearchTemplates = useCallback<Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>>(update => {
		const previous = webSearchRef.current;
		const next = normalizeWebSearchChoiceTemplates(resolveStateUpdate(update, previous));
		if (
			previous.length === next.length &&
			previous.every(
				(item, index) =>
					toolIdentityKey(item.target) === toolIdentityKey(next[index].target) &&
					item.userArgSchemaInstance === next[index].userArgSchemaInstance
			)
		) {
			return;
		}
		webSearchRef.current = next;
		setWebSearchTemplatesRaw(next);
	}, []);

	const recomputeAttachedToolArgsBlocked = useCallback(() => {
		const blocked = getAttachedToolEntries(false).some(entry => {
			if (!entry.toolSnapshot) {
				return true;
			}
			const status = computeToolUserArgsStatus(entry.toolSnapshot.userArgSchema, entry.userArgSchemaInstance);
			return status.hasSchema && !status.isSatisfied;
		});
		setAttachedBlocked(previous => (previous === blocked ? previous : blocked));
	}, [getAttachedToolEntries]);

	const clearAttachedToolValidation = useCallback(() => {
		setAttachedBlocked(false);
	}, []);

	const conversationBlocked = useMemo(
		() =>
			conversationToolsState.some(entry => {
				if (!entry.enabled) {
					return false;
				}
				if (!entry.toolDefinition || entry.toolLoadError) {
					return true;
				}
				const status =
					entry.argStatus ??
					computeToolUserArgsStatus(entry.toolDefinition.userArgSchema, entry.toolStoreChoice.userArgSchemaInstance);
				return status.hasSchema && !status.isSatisfied;
			}),
		[conversationToolsState]
	);

	const applyConversationToolsFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			setConversationToolsState(toolStoreChoicesToConversationTools(tools));
		},
		[setConversationToolsState]
	);

	const applyWebSearchFromChoices = useCallback(
		(tools: ToolStoreChoice[]) => {
			setWebSearchTemplates(
				(tools ?? [])
					.filter(tool => tool.toolType === ToolStoreChoiceType.WebSearch)
					.map(tool => webSearchTemplateFromChoice(tool))
			);
		},
		[setWebSearchTemplates]
	);

	return {
		conversationToolsState,
		setConversationToolsState,
		webSearchTemplates,
		setWebSearchTemplates,
		toolArgsBlocked: attachedBlocked || conversationBlocked,
		recomputeAttachedToolArgsBlocked,
		clearAttachedToolValidation,
		applyConversationToolsFromChoices,
		applyWebSearchFromChoices,
	};
}
