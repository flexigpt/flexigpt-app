import type { PluginView } from '@/spec/plugin';

import { pluginShareableSourceDocument, portableSourceFileName } from '@/lib/shareable_source';

import { ManagementYAMLSource } from '@/components/managementui/management_yaml_source';

interface ManagementPluginSourceProps {
	label: string;
	pluginName: string;
	load: () => Promise<PluginView>;
	note?: string;
}

/**
 * Renders a Plugin declaration as portable YAML. Runtime state, source
 * metadata, revisions, and resolved Artifact targets are deliberately absent.
 */
export function ManagementPluginSource({
	label,
	pluginName,
	load,
	note = 'This portable Plugin declaration excludes local state, source metadata, revisions, and resolved targets.',
}: ManagementPluginSourceProps) {
	return (
		<ManagementYAMLSource
			label={label}
			fileName={portableSourceFileName(pluginName, 'yaml', 'plugin')}
			load={async () => pluginShareableSourceDocument(await load())}
			note={note}
		/>
	);
}
