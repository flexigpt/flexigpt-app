import type { PluginListItem } from '@/spec/plugin';

import { toolManagementAPI } from '@/apis/baseapi';

import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementPluginSource } from '@/components/managementui/management_plugin_source';

interface ToolPluginDetailsModalProps {
	isOpen: boolean;
	plugin: PluginListItem | null;
	onClose: () => void;
}

export function ToolPluginDetailsModal({ isOpen, plugin, onClose }: ToolPluginDetailsModalProps) {
	if (!isOpen || !plugin) {
		return null;
	}

	return (
		<ManagementDetailsModal
			isOpen={isOpen}
			onClose={onClose}
			title="Tool Plugin Details"
			modalKey={`tool-plugin:${plugin.ref.rootID}:${plugin.ref.artifactID}:${plugin.revision}`}
		>
			<ManagementInfoGrid>
				<ManagementInfoRow label="Display name">{plugin.displayName || plugin.name}</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{plugin.builtIn ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">{plugin.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{plugin.description || '—'}</span>
				</ManagementInfoRow>
			</ManagementInfoGrid>

			<ManagementPluginSource
				label="tool plugin YAML"
				pluginName={plugin.name}
				load={() => toolManagementAPI.getToolPlugin(plugin.ref)}
			/>
		</ManagementDetailsModal>
	);
}
