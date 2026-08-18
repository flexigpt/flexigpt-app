import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';

import type { MCPBundleView } from '@/mcpservers/lib/mcp_management';

interface MCPBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: MCPBundleView | null;
	serverCount: number;
}

export function MCPBundleDetailsModal({ isOpen, onClose, bundle, serverCount }: MCPBundleDetailsModalProps) {
	if (!isOpen || !bundle) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="MCP Bundle Details"
			description={`${serverCount} configured server${serverCount === 1 ? '' : 's'}`}
			modalKey={`mcp-bundle:${bundle.ref.rootID}:${bundle.ref.collectionID}:${bundle.bundle.collection.modifiedAt}`}
		>
			<ManagementInfoGrid>
				<ManagementInfoRow label="Display Name">{bundle.displayName}</ManagementInfoRow>
				<ManagementInfoRow label="Logical Name" mono>
					{bundle.logicalName}
				</ManagementInfoRow>
				<ManagementInfoRow label="Collection ID" mono>
					{bundle.ref.collectionID}
				</ManagementInfoRow>
				<ManagementInfoRow label="Root ID" mono>
					{bundle.ref.rootID}
				</ManagementInfoRow>
				<ManagementInfoRow label="Collection Revision">{bundle.bundle.collection.revision}</ManagementInfoRow>
				<ManagementInfoRow label="Overlay Revision">{bundle.installation.overlayRevision || '—'}</ManagementInfoRow>
				<ManagementInfoRow label="Managed Source ID" mono>
					{bundle.bundle.data.managedSourceID || '—'}
				</ManagementInfoRow>
				<ManagementInfoRow label="Attached Sources">
					<div key={bundle.bundle.attachment.sourceID} className="bg-base-100 rounded-xl p-2 text-xs">
						<div className="font-medium">{bundle.bundle.source?.displayName || bundle.bundle.attachment.sourceID}</div>
						<div className="text-base-content/60 mt-1">
							{bundle.bundle.attachment.role}
							{bundle.bundle.source?.kind ? ` · ${bundle.bundle.source.kind}` : ''}
						</div>
						<div className="text-base-content/60 mt-1 font-mono break-all">{bundle.bundle.attachment.sourceID}</div>
					</div>
				</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{bundle.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Runtime Enabled">{bundle.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{bundle.description || '—'}</span>
				</ManagementInfoRow>
				<ManagementInfoRow label="Created">{bundle.bundle.collection.createdAt.toLocaleString()}</ManagementInfoRow>
				<ManagementInfoRow label="Modified">{bundle.bundle.collection.modifiedAt.toLocaleString()}</ManagementInfoRow>
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}
