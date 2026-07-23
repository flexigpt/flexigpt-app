// oxlint-disable jsreact-hooks/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
import { useEffect, useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type { WorkspaceContextInspectionView, WorkspaceRecordView, WorkspaceSkillLoadView } from '@/spec/workspace';
import { WorkspaceArtifactKind } from '@/spec/workspace';

import { workspaceAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalSection } from '@/components/modal/modal_section';

import {
	formatByteCount,
	getArtifactKindLabel,
	getErrorMessage,
	getRecordModeLabel,
	getRecordStateTone,
} from '@/workspaces/lib/workspace_utils';
import { WorkspaceDiagnostics } from '@/workspaces/workspace_diagnostics';

interface WorkspaceRecordDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	rootID: string;
	record: WorkspaceRecordView | null;
}

interface RecordInspection {
	record: WorkspaceRecordView;
	context?: WorkspaceContextInspectionView;
	skill?: WorkspaceSkillLoadView;
	previewError?: string;
}

export function WorkspaceRecordDetailsModal({ isOpen, onClose, rootID, record }: WorkspaceRecordDetailsModalProps) {
	const [inspection, setInspection] = useState<RecordInspection | null>(null);
	const [loadError, setLoadError] = useState('');
	const [isLoading, setIsLoading] = useState(false);

	useEffect(() => {
		if (!isOpen || !record) {
			return;
		}

		let active = true;
		setInspection(null);
		setLoadError('');
		setIsLoading(true);

		void (async () => {
			try {
				const freshRecord = await workspaceAPI.getWorkspaceRecord(rootID, record.id);
				let context: WorkspaceContextInspectionView | undefined;
				let skill: WorkspaceSkillLoadView | undefined;
				let previewError: string | undefined;

				try {
					if (freshRecord.kind === WorkspaceArtifactKind.Context) {
						context = await workspaceAPI.loadWorkspaceContexts(rootID, [freshRecord.id]);
					} else if (freshRecord.kind === WorkspaceArtifactKind.Skill) {
						skill = await workspaceAPI.loadWorkspaceSkills(rootID, [freshRecord.id]);
					}
				} catch (error) {
					previewError = getErrorMessage(error, 'The artifact content could not be loaded.');
				}

				if (active) {
					setInspection({
						record: freshRecord,
						context,
						skill,
						previewError,
					});
				}
			} catch (error) {
				if (active) {
					setLoadError(getErrorMessage(error, 'Workspace record could not be loaded.'));
				}
			} finally {
				if (active) {
					setIsLoading(false);
				}
			}
		})();

		return () => {
			active = false;
		};
	}, [isOpen, record, rootID]);

	if (!isOpen || !record) {
		return null;
	}

	const current = inspection?.record ?? record;
	const contribution = inspection?.context?.contributions.find(item => item.recordID === current.id);
	const skill = inspection?.skill?.skills.find(item => item.recordID === current.id);

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="Workspace Record"
			description={current.name}
			modalKey={`${rootID}:${record.id}:${record.revision}`}
			width="wide"
			height="tall"
		>
			{isLoading ? <Loader text="Loading workspace record..." /> : null}

			{loadError ? (
				<div className="alert alert-error rounded-2xl text-sm">
					<FiAlertCircle size={14} />
					<span>{loadError}</span>
				</div>
			) : null}

			<ModalSection title="Record metadata">
				<ManagementInfoGrid>
					<ManagementInfoRow label="Name">{current.name}</ManagementInfoRow>
					<ManagementInfoRow label="Record ID" mono>
						{current.id}
					</ManagementInfoRow>
					<ManagementInfoRow label="Revision">{current.revision}</ManagementInfoRow>
					<ManagementInfoRow label="Kind">{getArtifactKindLabel(current.kind)}</ManagementInfoRow>
					<ManagementInfoRow label="State">
						<StatusBadge tone={getRecordStateTone(current.state)}>{current.state}</StatusBadge>
					</ManagementInfoRow>
					<ManagementInfoRow label="Mode">{getRecordModeLabel(current.mode)}</ManagementInfoRow>
					<ManagementInfoRow label="Enabled">{current.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Runtime allowed">{current.runtimeAllowed ? 'Yes' : 'No'}</ManagementInfoRow>
					<ManagementInfoRow label="Source ID" mono>
						{current.sourceID}
					</ManagementInfoRow>
					<ManagementInfoRow label="Locator" mono>
						{current.locator}
					</ManagementInfoRow>
					<ManagementInfoRow label="Subresource" mono>
						{current.subresourceLocator || 'None'}
					</ManagementInfoRow>
					<ManagementInfoRow label="Resolved definition" mono>
						{current.resolvedDefinition || 'None'}
					</ManagementInfoRow>
					<ManagementInfoRow label="Pinned definition" mono>
						{current.pinnedDefinition || 'None'}
					</ManagementInfoRow>
				</ManagementInfoGrid>
			</ModalSection>

			{inspection?.previewError ? (
				<div className="alert alert-warning rounded-2xl text-sm">
					<FiAlertCircle size={14} />
					<span>{inspection.previewError}</span>
				</div>
			) : null}

			{contribution ? (
				<ModalSection title="Context content">
					<div className="flex flex-wrap gap-2">
						<MetadataPill label="Role">{contribution.role}</MetadataPill>
						<MetadataPill label="Priority">{contribution.priority}</MetadataPill>
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
						<ManagementInfoRow label="Display name">{skill.skill.displayName}</ManagementInfoRow>
						<ManagementInfoRow label="Slug" mono>
							{skill.skill.slug}
						</ManagementInfoRow>
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
