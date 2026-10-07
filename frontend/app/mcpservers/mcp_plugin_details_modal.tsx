import type { MCPPluginView } from '@/spec/mcp';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';

interface MCPPluginDetailsModalProps {
	isOpen: boolean;
	onClose: () => void;
	plugin: MCPPluginView | null;
	serverCount: number;
	serversLoaded: boolean;
}

export function MCPPluginDetailsModal({
	isOpen,
	onClose,
	plugin,
	serverCount,
	serversLoaded,
}: MCPPluginDetailsModalProps) {
	if (!isOpen || !plugin) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="MCP Plugin Details"
			description={
				serversLoaded ? `${serverCount} configured server${serverCount === 1 ? '' : 's'}` : 'Server contents not loaded'
			}
			modalKey={`mcp-plugin:${plugin.ref.rootID}:${plugin.ref.artifactID}:${plugin.plugin.revision}`}
		>
			<ManagementInfoGrid>
				<ManagementInfoRow label="Display Name">{plugin.displayName}</ManagementInfoRow>
				<ManagementInfoRow label="Logical Name" mono>
					{plugin.logicalName}
				</ManagementInfoRow>
				<ManagementInfoRow label="Baseline">{plugin.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{plugin.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">{plugin.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="State">{plugin.plugin.state}</ManagementInfoRow>
				<ManagementInfoRow label="Revision">{plugin.plugin.revision}</ManagementInfoRow>
				<ManagementInfoRow label="Servers">{serversLoaded ? serverCount : 'Not loaded'}</ManagementInfoRow>
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{plugin.description || '—'}</span>
				</ManagementInfoRow>
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}
