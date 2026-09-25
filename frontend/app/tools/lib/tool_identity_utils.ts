import type { MappedTarget } from '@/spec/artifact';

export function toolIdentityKey(target: MappedTarget): string {
	return JSON.stringify([target.provider, target.identifier, target.type, target.name, target.builtin]);
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
