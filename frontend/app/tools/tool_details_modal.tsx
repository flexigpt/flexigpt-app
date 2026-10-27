import { useCallback } from 'react';

import type { ToolStoreListItem, ToolView } from '@/spec/tool';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { toolManagementAPI } from '@/apis/baseapi';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementJSONSource } from '@/components/managementui/management_json_source';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';

import type { ToolSchemaKind } from '@/tools/tool_schema_source';
import { getToolSchema, toolSchemaFileName } from '@/tools/tool_schema_source';

interface ToolDetailsModalProps {
	isOpen: boolean;
	tool: ToolStoreListItem | null;
	onClose: () => void;
}

function ToolSchemaSection({ tool, kind }: { tool: ToolView; kind: ToolSchemaKind }) {
	const schema = getToolSchema(tool, kind);
	const label = kind === 'input' ? 'input schema JSON' : 'output schema JSON';

	if (schema === undefined) {
		return (
			<p className="text-base-content/60 text-sm">
				This tool does not declare an {kind === 'input' ? 'input' : 'output'} schema.
			</p>
		);
	}
	return (
		<ManagementJSONSource
			label={label}
			fileName={toolSchemaFileName(tool, kind)}
			load={async () => schema}
			note={`This is the shareable ${kind} schema. Runtime and management metadata are not included.`}
		/>
	);
}

function ToolDetailsModalContent({ listedTool, onClose }: { listedTool: ToolStoreListItem; onClose: () => void }) {
	const loadTool = useCallback(
		async (signal: AbortSignal): Promise<ToolView> => {
			const value = await toolManagementAPI.getTool(listedTool.ref);
			throwIfAborted(signal);
			return value;
		},
		[listedTool]
	);

	const {
		data: tool,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadTool, {
		initialData: null as ToolView | null,
	});

	return (
		<ManagementDetailsModal
			isOpen={true}
			onClose={onClose}
			title={listedTool.displayName || listedTool.name}
			description={listedTool.description}
			modalKey={`tool:${listedTool.ref.rootID}:${listedTool.ref.artifactID}:${listedTool.revision}`}
		>
			{error ? (
				<ManagementResourceError
					title="Tool source could not be loaded"
					error={error}
					isRetrying={isRefreshing}
					onRetry={reloadOrThrow}
				/>
			) : null}

			{isLoading && !tool ? <p className="text-base-content/60 text-sm">Loading tool schemas...</p> : null}

			{tool ? (
				<>
					<ManagementInfoGrid>
						<ManagementInfoRow label="Tool name">{tool.name}</ManagementInfoRow>
						<ManagementInfoRow label="Description">{tool.description || 'No description'}</ManagementInfoRow>
					</ManagementInfoGrid>

					<ToolSchemaSection tool={tool} kind="input" />
					<ToolSchemaSection tool={tool} kind="output" />
				</>
			) : null}
		</ManagementDetailsModal>
	);
}

export function ToolDetailsModal({ isOpen, tool, onClose }: ToolDetailsModalProps) {
	if (!isOpen || !tool) {
		return null;
	}

	return (
		<ToolDetailsModalContent
			key={`${tool.ref.rootID}:${tool.ref.artifactID}:${tool.revision}`}
			listedTool={tool}
			onClose={onClose}
		/>
	);
}
