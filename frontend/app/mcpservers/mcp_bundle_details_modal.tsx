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

function formatTimestamp(value: string | Date): string {
	const date = value instanceof Date ? value : new Date(value);
	return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
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
			title="MCP Collection Details"
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
				<ManagementInfoRow label="Baseline">{bundle.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{bundle.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">{bundle.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Servers">{serversLoaded ? serverCount : 'Not loaded'}</ManagementInfoRow>
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{bundle.description || '—'}</span>
				</ManagementInfoRow>
				<ManagementInfoRow label="Created">{formatTimestamp(bundle.collection.artifact.createdAt)}</ManagementInfoRow>
				<ManagementInfoRow label="Modified">{formatTimestamp(bundle.collection.artifact.modifiedAt)}</ManagementInfoRow>
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}
