import { useCallback } from 'react';

import type { ArtifactRef } from '@/spec/artifact';
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

export interface ToolDetailsSubject {
	ref: ArtifactRef;
	revision?: number;
	displayName?: string;
	name?: string;
	description?: string;
}

interface ToolDetailsModalProps {
	isOpen: boolean;
	subject: ToolDetailsSubject | null;
	onClose: () => void;
}

// oxlint-disable-next-line react/only-export-components
export function toolDetailsSubjectFromListItem(tool: ToolStoreListItem): ToolDetailsSubject {
	return {
		ref: tool.ref,
		revision: tool.revision,
		displayName: tool.displayName,
		name: tool.name,
		description: tool.description,
	};
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

function ToolDetailsModalContent({ subject, onClose }: { subject: ToolDetailsSubject; onClose: () => void }) {
	const loadTool = useCallback(
		async (signal: AbortSignal): Promise<ToolView> => {
			const value = await toolManagementAPI.getTool(subject.ref);
			throwIfAborted(signal);
			return value;
		},
		[subject]
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
			title={tool?.displayName || subject.displayName || subject.name || 'Tool'}
			description={tool?.description || subject.description}
			modalKey={`tool:${subject.ref.rootID}:${subject.ref.artifactID}:${subject.revision ?? 'current'}`}
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

export function ToolDetailsModal({ isOpen, subject, onClose }: ToolDetailsModalProps) {
	if (!isOpen || !subject) {
		return null;
	}

	return (
		<ToolDetailsModalContent
			key={`${subject.ref.rootID}:${subject.ref.artifactID}:${subject.revision ?? 'current'}`}
			subject={subject}
			onClose={onClose}
		/>
	);
}
