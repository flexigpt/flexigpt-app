import { useCallback } from 'react';

import type {
	WorkspaceContextInspectionView,
	WorkspaceRecordView,
	WorkspaceSkillLoadView,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceArtifactKind } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalSection } from '@/components/modal/modal_section';

import {
	formatByteCount,
	getArtifactKindLabel,
	getErrorMessage,
	getRecordStateTone,
	workspaceLocatorToPath,
} from '@/workspaces/lib/workspace_utils';
import { WorkspaceDiagnostics } from '@/workspaces/workspace_diagnostics';

interface WorkspaceResourceDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	workspace: WorkspaceView;
	record: WorkspaceRecordView | null;
}

interface RecordInspection {
	record: WorkspaceRecordView;
	context?: WorkspaceContextInspectionView;
	skill?: WorkspaceSkillLoadView;
	previewError?: string;
}

function sourceLabel(workspace: WorkspaceView, sourceID: string): string {
	const attachment = workspace.attachments.find(item => item.sourceID === sourceID);
	return attachment?.path ?? attachment?.sourceDisplayName ?? 'Workspace source';
}

function WorkspaceResourceDetailsContent({
	onClose,
	workspace,
	record,
}: Omit<WorkspaceResourceDetailsModalProps, 'isOpen'> & { record: WorkspaceRecordView }) {
	const loadInspection = useCallback(
		async (signal: AbortSignal): Promise<RecordInspection> => {
			const freshRecord = await workspaceAPI.getWorkspaceRecord(workspace.rootID, record.id);
			throwIfAborted(signal);

			let context: WorkspaceContextInspectionView | undefined;
			let skill: WorkspaceSkillLoadView | undefined;
			let previewError: string | undefined;

			try {
				if (freshRecord.kind === WorkspaceArtifactKind.Context) {
					context = await workspaceAPI.loadWorkspaceContexts(workspace.rootID, [freshRecord.id]);
				} else if (freshRecord.kind === WorkspaceArtifactKind.Skill) {
					skill = await workspaceAPI.loadWorkspaceSkills(workspace.rootID, [freshRecord.id]);
				}
				throwIfAborted(signal);
			} catch (error) {
				previewError = getErrorMessage(error, 'The artifact content could not be loaded.');
			}

			return {
				record: freshRecord,
				context,
				skill,
				previewError,
			};
		},
		[record.id, workspace.rootID]
	);

	const {
		data: inspection,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadInspection, {
		initialData: null as RecordInspection | null,
	});

	const current = inspection?.record ?? record;
	const contribution = inspection?.context?.contributions.find(item => item.recordID === current.id);
	const skill = inspection?.skill?.skills.find(item => item.recordID === current.id);
	const source = sourceLabel(workspace, current.sourceID);
	const location = workspaceLocatorToPath(
		workspace.attachments.find(item => item.sourceID === current.sourceID)?.path,
		current.locator
	);

	return (
		<ManagementDetailsModal
			isOpen
			onClose={onClose}
			title="Workspace Resource"
			description={current.name}
			modalKey={`${workspace.rootID}:${record.id}:${record.revision}`}
			width="wide"
			height="tall"
		>
			{error ? (
				<ManagementResourceError
					title="Workspace resource could not be loaded"
					error={error}
					isRetrying={isRefreshing}
					onRetry={reloadOrThrow}
				/>
			) : null}

			{isLoading && !inspection ? <Loader text="Loading workspace resource..." /> : null}

			<ModalSection title="Resource details">
				<ManagementInfoGrid>
					<ManagementInfoRow label="Name">{current.name}</ManagementInfoRow>
					<ManagementInfoRow label="Kind">{getArtifactKindLabel(current.kind)}</ManagementInfoRow>
					<ManagementInfoRow label="State">
						<StatusBadge tone={getRecordStateTone(current.state)}>{current.state}</StatusBadge>
					</ManagementInfoRow>
					<ManagementInfoRow label="Enabled">{current.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Source">
						<span className="break-all">{source}</span>
					</ManagementInfoRow>
					<ManagementInfoRow label="Location">
						<span className="font-mono text-xs break-all">{location}</span>
					</ManagementInfoRow>
					{current.subresourceLocator ? (
						<ManagementInfoRow label="Subresource">
							<span className="font-mono text-xs break-all">{current.subresourceLocator}</span>
						</ManagementInfoRow>
					) : null}
				</ManagementInfoGrid>
			</ModalSection>

			{inspection?.previewError ? (
				<div className="alert alert-warning rounded-2xl text-sm">{inspection.previewError}</div>
			) : null}

			{contribution ? (
				<ModalSection title="Context content">
					<div className="flex flex-wrap gap-2">
						<MetadataPill label="Role">{contribution.role}</MetadataPill>
						<MetadataPill label="Original">{formatByteCount(contribution.originalBytes)}</MetadataPill>
						<MetadataPill label="Included">{formatByteCount(contribution.includedBytes)}</MetadataPill>
						{contribution.truncated ? <MetadataPill>Truncated</MetadataPill> : null}
					</div>
					<pre className="bg-base-100 max-h-[50vh] overflow-auto rounded-2xl p-4 text-xs whitespace-pre-wrap">
						{contribution.content || '(Context content is empty.)'}
					</pre>
				</ModalSection>
			) : null}

			{skill ? (
				<ModalSection title="Skill artifact">
					<ManagementInfoGrid>
						<ManagementInfoRow label="Display name">{skill.skill.displayName || skill.skill.name}</ManagementInfoRow>
						<ManagementInfoRow label="Description">{skill.skill.description || 'None'}</ManagementInfoRow>
						<ManagementInfoRow label="Insert">{skill.skill.insert}</ManagementInfoRow>
						<ManagementInfoRow label="Arguments">
							{skill.skill.arguments?.length ? (
								<div className="space-y-2">
									{skill.skill.arguments.map(argument => (
										<div key={argument.name} className="bg-base-100 rounded-xl p-3">
											<div className="font-mono text-xs">{argument.name}</div>
											{argument.description ? (
												<div className="text-base-content/70 mt-1 text-xs">{argument.description}</div>
											) : null}
											{argument.default !== undefined ? (
												<div className="text-base-content/70 mt-1 text-xs">Default: {argument.default}</div>
											) : null}
										</div>
									))}
								</div>
							) : (
								'None'
							)}
						</ManagementInfoRow>
						<ManagementInfoRow label="Tags">
							{skill.skill.tags?.length ? skill.skill.tags.join(', ') : 'None'}
						</ManagementInfoRow>
					</ManagementInfoGrid>

					<pre className="bg-base-100 max-h-[50vh] overflow-auto rounded-2xl p-4 text-xs whitespace-pre-wrap">
						{skill.markdownBody || '(Skill markdown body was not returned.)'}
					</pre>
				</ModalSection>
			) : null}

			<ModalSection title="Diagnostics">
				<WorkspaceDiagnostics diagnostics={current.diagnostics} />
			</ModalSection>
		</ManagementDetailsModal>
	);
}

export function WorkspaceResourceDetailsModal(props: WorkspaceResourceDetailsModalProps) {
	if (!props.isOpen || !props.record) {
		return null;
	}

	return (
		<WorkspaceResourceDetailsContent
			key={`${props.workspace.rootID}:${props.record.id}:${props.record.revision}`}
			{...props}
			record={props.record}
		/>
	);
}
