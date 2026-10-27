import { createPortableYAMLSource } from '@/lib/shareable_source';

import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';

interface ManagementYAMLSourceProps<T> {
	label: string;
	fileName: string;
	load: () => Promise<T>;
	note?: string;
}

/**
 * Displays a portable YAML source document through the common lazy
 * show/copy/export interaction.
 */
export function ManagementYAMLSource<T>({ label, fileName, load, note }: ManagementYAMLSourceProps<T>) {
	return (
		<ManagementArtifactSource
			label={label}
			load={async () => {
				const data = await load();

				return createPortableYAMLSource(data, fileName);
			}}
			note={note}
		/>
	);
}
