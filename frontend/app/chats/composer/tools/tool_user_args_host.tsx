import type { Dispatch, SetStateAction } from 'react';

import type { ToolArgsTarget } from '@/spec/tool';

import type { ToolCatalogState } from '@/hooks/use_tool';

import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';
import { ToolUserArgsModal } from '@/chats/composer/tools/tool_user_args_modal';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

interface ToolArgsModalHostProps {
	attachedToolEntries: AttachedToolEntry[];
	setAttachedToolUserArgSchemaInstance: (selectionID: string, value: string) => void;
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	toolArgsTarget: ToolArgsTarget | null;
	setToolArgsTarget: Dispatch<SetStateAction<ToolArgsTarget | null>>;
	recomputeAttachedToolArgsBlocked: () => void;
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	toolCatalog: ToolCatalogState;
}

export function ToolArgsModalHost(props: ToolArgsModalHostProps) {
	const target = props.toolArgsTarget;
	const close = () => {
		props.setToolArgsTarget(null);
	};

	if (!target) {
		return null;
	}

	if (target.kind === 'attached') {
		const entry = props.attachedToolEntries.find(value => value.selectionID === target.selectionID);
		if (!entry) {
			return null;
		}
		return (
			<ToolUserArgsModal
				isOpen
				onClose={close}
				toolLabel={entry.displayName || entry.target.name}
				schema={entry.toolSnapshot?.userArgSchema}
				existingInstance={entry.userArgSchemaInstance}
				modalIdentity={`attached:${entry.selectionID}`}
				loadError={entry.toolSnapshot ? undefined : 'The attached Tool definition is unavailable.'}
				onSave={value => {
					props.setAttachedToolUserArgSchemaInstance(entry.selectionID, value);
					props.recomputeAttachedToolArgsBlocked();
				}}
			/>
		);
	}

	if (target.kind === 'conversation') {
		const entry = props.conversationToolsState.find(value => value.key === target.key);
		if (!entry) {
			return null;
		}
		return (
			<ToolUserArgsModal
				isOpen
				onClose={close}
				toolLabel={entry.toolStoreChoice.displayName || entry.toolStoreChoice.target.name}
				schema={entry.toolDefinition?.userArgSchema}
				existingInstance={entry.toolStoreChoice.userArgSchemaInstance}
				modalIdentity={`conversation:${entry.key}`}
				isLoading={!entry.toolDefinition && !entry.toolLoadError}
				loadError={entry.toolLoadError}
				onSave={value => {
					props.setConversationToolsState(previous =>
						previous.map(candidate =>
							candidate.key === entry.key
								? {
										...candidate,
										toolStoreChoice: {
											...candidate.toolStoreChoice,
											userArgSchemaInstance: value,
										},
										argStatus: candidate.toolDefinition
											? computeToolUserArgsStatus(candidate.toolDefinition.userArgSchema, value)
											: candidate.argStatus,
									}
								: candidate
						)
					);
				}}
			/>
		);
	}

	const active = props.webSearchTemplates[0];
	if (!active) {
		return null;
	}
	const key = toolIdentityKey(active.target);
	const item = props.toolCatalog.data.find(value => toolIdentityKey(value.target) === key);
	const loading = props.toolCatalog.loading || props.toolCatalog.isRefreshing;
	const error = props.toolCatalog.error
		? 'The Tool catalog could not be loaded. Refresh the Tools picker.'
		: props.toolCatalog.hasResolved && !loading && !item
			? 'This web-search Tool is unavailable. Remove it or select an available Tool.'
			: undefined;

	return (
		<ToolUserArgsModal
			isOpen
			onClose={close}
			toolLabel={item?.toolDefinition.displayName || active.displayName || active.target.name}
			schema={item?.toolDefinition.userArgSchema}
			existingInstance={active.userArgSchemaInstance}
			modalIdentity={`web-search:${key}`}
			isLoading={loading || !props.toolCatalog.hasResolved}
			loadError={error}
			onSave={value => {
				props.setWebSearchTemplates(previous =>
					previous.map(template =>
						toolIdentityKey(template.target) === key ? { ...template, userArgSchemaInstance: value } : template
					)
				);
			}}
		/>
	);
}
