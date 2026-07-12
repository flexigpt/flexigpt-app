import type { MCPBundle } from '@/spec/mcp';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';

interface MCPBundleDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: MCPBundle | null;
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
			modalKey={`mcp-bundle:${bundle.id}:${bundle.modifiedAt}`}
		>
			<ManagementInfoGrid>
				<ManagementInfoRow label="Display Name">{bundle.displayName || '—'}</ManagementInfoRow>
				<ManagementInfoRow label="Slug" mono>
					{bundle.slug}
				</ManagementInfoRow>
				<ManagementInfoRow label="ID" mono>
					{bundle.id}
				</ManagementInfoRow>
				<ManagementInfoRow label="Configured servers">{serverCount}</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{bundle.isBuiltIn ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">{bundle.isEnabled ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{bundle.description || '—'}</span>
				</ManagementInfoRow>
				<ManagementInfoRow label="Created">{bundle.createdAt}</ManagementInfoRow>
				<ManagementInfoRow label="Modified">{bundle.modifiedAt}</ManagementInfoRow>
				{bundle.softDeletedAt ? (
					<ManagementInfoRow label="Soft deleted">{bundle.softDeletedAt}</ManagementInfoRow>
				) : null}
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}
