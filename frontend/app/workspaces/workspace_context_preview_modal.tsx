// oxlint-disable jsreact-hooks/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
import { useEffect, useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type { WorkspaceContextLoadPlan, WorkspaceView } from '@/spec/workspace';

import { workspaceAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { ModalSection } from '@/components/modal/modal_section';

import { formatByteCount, getErrorMessage } from '@/workspaces/lib/workspace_utils';
import { WorkspaceDiagnostics } from '@/workspaces/workspace_diagnostics';

interface WorkspaceContextPreviewModalProps {
	isOpen: boolean;
	onClose: () => void;
	workspace: WorkspaceView;
}

export function WorkspaceContextPreviewModal({ isOpen, onClose, workspace }: WorkspaceContextPreviewModalProps) {
	const [plan, setPlan] = useState<WorkspaceContextLoadPlan | null>(null);
	const [loadError, setLoadError] = useState('');
	const [isLoading, setIsLoading] = useState(false);

	useEffect(() => {
		if (!isOpen) {
			return;
		}

		let active = true;
		setPlan(null);
		setLoadError('');
		setIsLoading(true);

		void workspaceAPI
			.composeWorkspaceContext(workspace.rootID)
			.then(result => {
				if (active) {
					setPlan(result);
				}
			})
			.catch((error: unknown) => {
				if (active) {
					setLoadError(getErrorMessage(error, 'Failed to compose workspace context.'));
				}
			})
			.finally(() => {
				if (active) {
					setIsLoading(false);
				}
			});

		return () => {
			active = false;
		};
	}, [isOpen, workspace.rootID]);

	if (!isOpen) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="Workspace Context Preview"
			description={`Inspect the composed context for ${workspace.displayName}. This does not modify a conversation.`}
			modalKey={`${workspace.rootID}:${workspace.revision}:context-preview`}
			width="wide"
			height="tall"
		>
			{isLoading ? <Loader text="Composing workspace context..." /> : null}

			{loadError ? (
				<div className="alert alert-error rounded-2xl text-sm">
					<FiAlertCircle size={14} />
					<span>{loadError}</span>
				</div>
			) : null}

			{plan ? (
				<>
					<div className="flex flex-wrap gap-2">
						<MetadataPill label="Catalog revision">{plan.catalogRevision}</MetadataPill>
						<MetadataPill label="Contributions">{plan.contributions.length}</MetadataPill>
						<MetadataPill label="Prompt size">{formatByteCount(plan.promptBytes)}</MetadataPill>
					</div>

					<ModalSection title="Composition decisions">
						{plan.decisions.length > 0 ? (
							<div className="space-y-2">
								{plan.decisions.map(decision => (
									<div key={decision.recordID} className="border-base-content/10 rounded-2xl border p-3">
										<div className="flex flex-wrap gap-2">
											<MetadataPill label="Record">{decision.recordID}</MetadataPill>
											<MetadataPill label="Status">{decision.status}</MetadataPill>
											{decision.code ? <MetadataPill label="Code">{decision.code}</MetadataPill> : null}
											<MetadataPill label="Original">{formatByteCount(decision.originalBytes)}</MetadataPill>
											<MetadataPill label="Included">{formatByteCount(decision.includedBytes)}</MetadataPill>
										</div>
									</div>
								))}
							</div>
						) : (
							<div className="text-base-content/70 text-sm">No composition decisions were returned.</div>
						)}
					</ModalSection>

					<ModalSection title="Composed prompt">
						<pre className="bg-base-100 max-h-[55vh] overflow-auto rounded-2xl p-4 text-xs whitespace-pre-wrap">
							{plan.prompt || '(Composed prompt is empty.)'}
						</pre>
					</ModalSection>

					<ModalSection title="Diagnostics">
						<WorkspaceDiagnostics diagnostics={plan.diagnostics} />
					</ModalSection>
				</>
			) : null}
		</ManagementDetailsModal>
	);
}
