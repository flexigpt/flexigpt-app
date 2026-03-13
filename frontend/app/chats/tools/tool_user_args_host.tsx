import { type Dispatch, type SetStateAction, useEffect, useMemo, useState } from 'react';

import type { Tool } from '@/spec/tool';

import type { JSONSchema } from '@/lib/jsonschema_utils';

import { toolStoreAPI } from '@/apis/baseapi';

import type { AttachedToolEntry } from '@/chats/platedoc/tool_document_ops';
import type { ConversationToolStateEntry } from '@/chats/tools/conversation_tool_utils';
import { computeToolUserArgsStatus } from '@/chats/tools/tool_editor_utils';
import { type ToolArgsTarget, ToolUserArgsModal } from '@/chats/tools/tool_user_args_modal';
import { type WebSearchChoiceTemplate } from '@/chats/tools/websearch_utils';

interface ToolArgsModalHostProps {
	attachedToolEntries: AttachedToolEntry[];
	setAttachedToolUserArgSchemaInstance: (selectionID: string, newInstance: string) => void;
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	toolArgsTarget: ToolArgsTarget | null;
	setToolArgsTarget: Dispatch<SetStateAction<ToolArgsTarget | null>>;
	recomputeAttachedToolArgsBlocked: () => void;

	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
}

export function ToolArgsModalHost({
	attachedToolEntries,
	setAttachedToolUserArgSchemaInstance,
	conversationToolsState,
	setConversationToolsState,
	toolArgsTarget,
	setToolArgsTarget,
	recomputeAttachedToolArgsBlocked,
	webSearchTemplates,
	setWebSearchTemplates,
}: ToolArgsModalHostProps) {
	let isOpen = false;
	let toolLabel = '';
	let schema: JSONSchema | undefined;
	let existingInstance: string | undefined;
	let modalIdentity: string | undefined;
	let onSave: (newInstance: string) => void = () => {};

	const activeWebSearch = useMemo(
		() => (webSearchTemplates.length > 0 ? webSearchTemplates[0] : undefined),
		[webSearchTemplates]
	);

	const [loadedWebSearchToolDef, setLoadedWebSearchToolDef] = useState<{
		key: string;
		def: Tool | null;
	} | null>(null);
	const activeWebSearchBundleID = activeWebSearch?.bundleID;
	const activeWebSearchToolSlug = activeWebSearch?.toolSlug;
	const activeWebSearchToolVersion = activeWebSearch?.toolVersion;
	const activeWebSearchKey =
		activeWebSearchBundleID && activeWebSearchToolSlug && activeWebSearchToolVersion
			? `${activeWebSearchBundleID}:${activeWebSearchToolSlug}:${activeWebSearchToolVersion}`
			: null;

	const webSearchToolDef = loadedWebSearchToolDef?.key === activeWebSearchKey ? loadedWebSearchToolDef.def : null;

	useEffect(() => {
		let cancelled = false;

		if (
			toolArgsTarget?.kind !== 'webSearch' ||
			!activeWebSearchBundleID ||
			!activeWebSearchToolSlug ||
			!activeWebSearchToolVersion ||
			!activeWebSearchKey
		) {
			return;
		}

		(async () => {
			try {
				const def = await toolStoreAPI.getTool(
					activeWebSearchBundleID,
					activeWebSearchToolSlug,
					activeWebSearchToolVersion
				);
				if (!cancelled) {
					setLoadedWebSearchToolDef({
						key: activeWebSearchKey,
						def: def ?? null,
					});
				}
			} catch {
				if (!cancelled) {
					setLoadedWebSearchToolDef({
						key: activeWebSearchKey,
						def: null,
					});
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [
		toolArgsTarget?.kind,
		activeWebSearch,
		activeWebSearchBundleID,
		activeWebSearchToolSlug,
		activeWebSearchToolVersion,
		activeWebSearchKey,
	]);

	if (toolArgsTarget?.kind === 'attached') {
		const hit = attachedToolEntries.find(n => n.selectionID === toolArgsTarget.selectionID);
		if (hit) {
			isOpen = true;
			modalIdentity = `attached:${toolArgsTarget.selectionID}`;
			schema = hit.toolSnapshot?.userArgSchema;
			toolLabel =
				hit.toolSnapshot?.displayName && hit.toolSnapshot.displayName.length > 0
					? hit.toolSnapshot.displayName
					: hit.toolSlug;
			existingInstance = hit.userArgSchemaInstance;

			onSave = newInstance => {
				setAttachedToolUserArgSchemaInstance(toolArgsTarget.selectionID, newInstance);

				// Do *not* close here; the modal will call dialog.close(),
				// which triggers onClose -> setToolArgsTarget(null).
				recomputeAttachedToolArgsBlocked();
			};
		}
	} else if (toolArgsTarget?.kind === 'conversation') {
		const entry = conversationToolsState.find(e => e.key === toolArgsTarget.key);
		if (entry) {
			isOpen = true;
			modalIdentity = `conversation:${toolArgsTarget.key}`;
			const def = entry.toolDefinition;
			schema = def?.userArgSchema;
			toolLabel =
				entry.toolStoreChoice.displayName && entry.toolStoreChoice.displayName.length > 0
					? entry.toolStoreChoice.displayName
					: entry.toolStoreChoice.toolSlug;
			existingInstance = entry.toolStoreChoice.userArgSchemaInstance;

			onSave = newInstance => {
				setConversationToolsState(prev =>
					prev.map(e => {
						if (e.key !== toolArgsTarget.key) return e;

						const nextToolStoreChoice = {
							...e.toolStoreChoice,
							userArgSchemaInstance: newInstance,
						};

						const nextStatus =
							e.toolDefinition && e.toolDefinition.userArgSchema
								? computeToolUserArgsStatus(e.toolDefinition.userArgSchema, newInstance)
								: e.argStatus;

						return {
							...e,
							toolStoreChoice: nextToolStoreChoice,
							argStatus: nextStatus,
						};
					})
				);

				// This only recomputes attached-tool blocking; conversation-level
				// blocking is already handled by the useEffect in EditorArea.
				recomputeAttachedToolArgsBlocked();
			};
		}
	} else if (toolArgsTarget?.kind === 'webSearch') {
		const active = activeWebSearch;

		if (active) {
			isOpen = true;
			modalIdentity = `web-search:${active.bundleID}:${active.toolSlug}:${active.toolVersion}`;
			toolLabel =
				(webSearchToolDef?.displayName && webSearchToolDef.displayName.length > 0
					? webSearchToolDef.displayName
					: active.displayName && active.displayName.length > 0
						? active.displayName
						: active.toolSlug) ?? active.toolSlug;

			schema = webSearchToolDef?.userArgSchema;
			existingInstance = active.userArgSchemaInstance;

			onSave = newInstance => {
				setWebSearchTemplates(prev => {
					if (!prev.length) return prev;
					const next = [...prev];
					next[0] = { ...next[0], userArgSchemaInstance: newInstance };
					return next;
				});
			};
		}
	}

	const handleClose = () => {
		// Sync React state when the native <dialog> closes (ESC, backdrop, Cancel, Save).
		setToolArgsTarget(null);
	};

	return (
		<ToolUserArgsModal
			isOpen={isOpen}
			onClose={handleClose}
			toolLabel={toolLabel}
			schema={schema}
			modalIdentity={modalIdentity}
			existingInstance={existingInstance}
			onSave={onSave}
		/>
	);
}
