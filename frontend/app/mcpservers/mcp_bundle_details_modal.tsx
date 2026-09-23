import type { MCPBundleView } from '@/spec/mcp';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';

interface MCPBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: MCPBundleView | null;
	serverCount: number;
	serversLoaded: boolean;
}

export function MCPBundleDetailsModal({
	isOpen,
	onClose,
	bundle,
	serverCount,
	serversLoaded,
}: MCPBundleDetailsModalProps) {
	if (!isOpen || !bundle) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="MCP Bundle Details"
			description={
				serversLoaded ? `${serverCount} configured server${serverCount === 1 ? '' : 's'}` : 'Server contents not loaded'
			}
			modalKey={`mcp-bundle:${bundle.ref.rootID}:${bundle.ref.artifactID}:${bundle.collection.artifact.revision}`}
		>
			<ManagementInfoGrid>
				<ManagementInfoRow label="Display Name">{bundle.displayName}</ManagementInfoRow>
				<ManagementInfoRow label="Logical Name" mono>
					{bundle.logicalName}
				</ManagementInfoRow>
				<ManagementInfoRow label="Collection ID" mono>
					{bundle.ref.artifactID}
				</ManagementInfoRow>
				<ManagementInfoRow label="Root ID" mono>
					{bundle.ref.rootID}
				</ManagementInfoRow>
				<ManagementInfoRow label="Collection Revision">{bundle.collection.artifact.revision}</ManagementInfoRow>
				<ManagementInfoRow label="Managed Source ID" mono>
					{bundle.collection.artifact.binding.sourceID}
				</ManagementInfoRow>
				<ManagementInfoRow label="Editable">{bundle.editable ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Deletable">{bundle.deletable ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Baseline">{bundle.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{bundle.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">{bundle.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{bundle.description || '—'}</span>
				</ManagementInfoRow>
				<ManagementInfoRow label="Created">{bundle.collection.artifact.createdAt.toLocaleString()}</ManagementInfoRow>
				<ManagementInfoRow label="Modified">{bundle.collection.artifact.modifiedAt.toLocaleString()}</ManagementInfoRow>
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}
