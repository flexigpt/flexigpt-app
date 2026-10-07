import type { CapabilityTarget } from '@/spec/artifact';

export function toolIdentityKey(target: CapabilityTarget): string {
	return JSON.stringify([target.providerIdentity, target.providerLocalID, target.type, target.name]);
}

/** Formats provider-facing tool names, not mapped target identifiers. */
export function getPrettyToolName(name: string): string {
	if (!name) {
		return 'Tool';
	}
	let base = name;
	if (base.includes('/')) {
		base = base.split('/').at(-1) || base;
	}
	if (base.includes('@')) {
		base = base.split('@')[0] || base;
	}
	return base.replaceAll(/[-_]/g, ' ');
}
