import type { AgentImportRelationshipStatus } from '@/spec/agent';

export function formatDateish(value: string | Date | undefined | null): string {
	if (!value) {
		return '—';
	}

	const date = value instanceof Date ? value : new Date(value);
	return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
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
	switch (status.toLowerCase()) {
		case 'available':
			return 'badge-success';
		case 'ambiguous':
			return 'badge-warning';
		default:
			return 'badge-error';
	}
}
