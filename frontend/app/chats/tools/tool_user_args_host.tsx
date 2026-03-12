import { type Dispatch, type SetStateAction, useEffect, useMemo, useState } from 'react';

import type { PlateEditor } from 'platejs/react';

import type { Tool } from '@/spec/tool';

import type { JSONSchema } from '@/lib/jsonschema_utils';

import { toolStoreAPI } from '@/apis/baseapi';

import type { ConversationToolStateEntry } from '@/chats/tools/conversation_tools_chip';
import { computeToolUserArgsStatus, getToolNodesWithPath } from '@/chats/tools/tool_editor_utils';
import { type ToolArgsTarget, ToolUserArgsModal } from '@/chats/tools/tool_user_args_modal';
import { type WebSearchChoiceTemplate, webSearchIdentityKey } from '@/chats/tools/websearch_utils';

interface ToolArgsModalHostProps {
	editor: PlateEditor;
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	toolArgsTarget: ToolArgsTarget | null;
	setToolArgsTarget: Dispatch<SetStateAction<ToolArgsTarget | null>>;
	recomputeAttachedToolArgsBlocked: () => void;

	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
}

export function ToolArgsModalHost({
	editor,
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
	let onSave: (newInstance: string) => void = () => {};

	const activeWebSearch = useMemo(
		() => (webSearchTemplates.length > 0 ? webSearchTemplates[0] : undefined),
		[webSearchTemplates]
	);
	const activeWebSearchKey = useMemo(
		() => (activeWebSearch ? webSearchIdentityKey(activeWebSearch) : undefined),
		[activeWebSearch]
	);
	const [webSearchToolDef, setWebSearchToolDef] = useState<Tool | null>(null);

	useEffect(() => {
		let cancelled = false;
		if (toolArgsTarget?.kind !== 'webSearch' || !activeWebSearch) return;

		(async () => {
			try {
				const def = await toolStoreAPI.getTool(
					activeWebSearch.bundleID,
					activeWebSearch.toolSlug,
					activeWebSearch.toolVersion
				);
				if (!cancelled) setWebSearchToolDef(def ?? null);
			} catch {
				if (!cancelled) setWebSearchToolDef(null);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [toolArgsTarget?.kind, activeWebSearchKey]);

	if (toolArgsTarget?.kind === 'attached') {
		const all = getToolNodesWithPath(editor, false);
		const hit = all.find(([n]) => n.selectionID === toolArgsTarget.selectionID);
		if (hit) {
			isOpen = true;
			const [node] = hit;
			schema = node.toolSnapshot?.userArgSchema;
			toolLabel =
				node.toolSnapshot?.displayName && node.toolSnapshot.displayName.length > 0
					? node.toolSnapshot.displayName
					: node.toolSlug;
			existingInstance = node.userArgSchemaInstance;

			onSave = newInstance => {
				// Re-locate the node by selectionID to avoid stale paths.
				const allNow = getToolNodesWithPath(editor, false);
				const again = allNow.find(([n]) => n.selectionID === toolArgsTarget.selectionID);
				if (again) {
					const [, path] = again;
					editor.tf.setNodes({ userArgSchemaInstance: newInstance }, { at: path as any });
				}

				// Do *not* close here; the modal will call dialog.close(),
				// which triggers onClose -> setToolArgsTarget(null).
				recomputeAttachedToolArgsBlocked();
			};
		}
	} else if (toolArgsTarget?.kind === 'conversation') {
		const entry = conversationToolsState.find(e => e.key === toolArgsTarget.key);
		if (entry) {
			isOpen = true;
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
			existingInstance={existingInstance}
			onSave={onSave}
		/>
	);
}
