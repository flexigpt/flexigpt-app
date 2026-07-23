import { useCallback, useMemo } from 'react';

import type { WorkspaceContextLoadPlan, WorkspaceView } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { ModalSection } from '@/components/modal/modal_section';

import { formatByteCount } from '@/workspaces/lib/workspace_utils';
import { WorkspaceDiagnostics } from '@/workspaces/workspace_diagnostics';

interface WorkspaceContextPreviewProps {
	isOpen: boolean;
	onClose: () => void;
	workspace: WorkspaceView;
}

function WorkspaceContextPreviewContent({ onClose, workspace }: Omit<WorkspaceContextPreviewProps, 'isOpen'>) {
	const loadPlan = useCallback(
		async (signal: AbortSignal): Promise<WorkspaceContextLoadPlan> => {
			const plan = await workspaceAPI.composeWorkspaceContext(workspace.rootID);
			throwIfAborted(signal);
			return plan;
		},
		[workspace.rootID]
	);

	const {
		data: plan,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadPlan, {
		initialData: null as WorkspaceContextLoadPlan | null,
	});

	const contributionLabels = useMemo(
		() =>
			new Map(
				(plan?.contributions ?? []).map(contribution => [
					contribution.recordID,
					`${contribution.name} (${contribution.locator})`,
				])
			),
		[plan?.contributions]
	);

	return (
		<ManagementDetailsModal
			isOpen
			onClose={onClose}
			title="Workspace Context Preview"
			description={`Inspect the composed Context for ${workspace.displayName}. This does not modify a conversation.`}
			modalKey={`${workspace.rootID}:${workspace.revision}:context-preview`}
			width="wide"
			height="tall"
		>
			{error ? (
				<ManagementResourceError
					title="Workspace Context could not be composed"
					error={error}
					isRetrying={isRefreshing}
					onRetry={reloadOrThrow}
				/>
			) : null}

			{isLoading && !plan ? <Loader text="Composing workspace Context..." /> : null}

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
											{contributionLabels.get(decision.recordID) ? (
												<MetadataPill label="Item">{contributionLabels.get(decision.recordID) ?? ''}</MetadataPill>
											) : null}
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

export function WorkspaceContextPreview(props: WorkspaceContextPreviewProps) {
	if (!props.isOpen) {
		return null;
	}

	return <WorkspaceContextPreviewContent key={`${props.workspace.rootID}:${props.workspace.revision}`} {...props} />;
}
