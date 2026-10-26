import type { AgentImportRelationshipStatus } from '@/spec/agent';

export function formatDateish(value: string | Date | undefined | null): string {
	if (!value) {
		return '—';
	}

	const date = value instanceof Date ? value : new Date(value);
	return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
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
