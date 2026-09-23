import type { ArtifactRef } from '@/spec/artifact';
import { AgentImportRelationshipStatus } from '@/spec/agent';

export function formatArtifactRef(ref?: ArtifactRef): string {
	if (!ref) {
		return '—';
	}

	return `${ref.rootID}/${ref.artifactID}`;
}

export function formatDateish(value: string | Date | undefined | null): string {
	if (!value) {
		return '—';
	}

	return value instanceof Date ? value.toISOString() : value;
}

export function textToBase64(value: string): string {
	const bytes = new TextEncoder().encode(value);
	let binary = '';

	for (let offset = 0; offset < bytes.length; offset += 0x8000) {
		binary += String.fromCodePoint(...bytes.subarray(offset, offset + 0x8000));
	}

	return globalThis.btoa(binary);
}

export function getAgentRelationshipBadgeClass(
	status: AgentImportRelationshipStatus | string
): 'badge-success' | 'badge-warning' | 'badge-error' {
	switch (status) {
		case AgentImportRelationshipStatus.Available.toString():
			return 'badge-success';
		case AgentImportRelationshipStatus.Ambiguous.toString():
			return 'badge-warning';
		default:
			return 'badge-error';
	}
}
