import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';

interface ManagementJSONSourceProps<T> {
	label: string;
	fileName: string;
	load: () => Promise<T>;
	note?: string;
}

/**
 * Exports a structured API document, not original source-file bytes.
 * Callers must pass only a safe, non-secret document or projection.
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
			note={note ?? 'This JSON is reconstructed from the API document. It is not the original source file.'}
		/>
	);
}
