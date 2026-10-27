import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';

interface ManagementJSONSourceProps<T> {
	label: string;
	fileName: string;
	load: () => Promise<T>;
	note?: string;
}

/**
 * Displays caller-supplied portable JSON source. Callers must pass a real
 * source document or schema, never a management API view or store record.
 */
export function ManagementJSONSource<T>({ label, fileName, load, note }: ManagementJSONSourceProps<T>) {
	return (
		<ManagementArtifactSource
			label={label}
			load={async () => {
				const data = await load();
				return {
					data,
					text: `${JSON.stringify(data, null, 2)}\n`,
					fileName,
					extension: 'json',
				};
			}}
			note={note}
		/>
	);
}
