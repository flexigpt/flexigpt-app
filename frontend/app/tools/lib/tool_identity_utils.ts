import type { CapabilityTarget } from '@/spec/artifact';
import { CapabilityTargetForm } from '@/spec/artifact';

export function toolIdentityKey(target: CapabilityTarget): string {
	// Source-backed Tool targets are Artifact targets. Their durable identity
	// is the ArtifactRef; providerIdentity/providerLocalID are optional and
	// must not be fabricated by the frontend.
	if (target.form === CapabilityTargetForm.Artifact) {
		if (target.artifact) {
			return JSON.stringify(['artifact', target.artifact.rootID, target.artifact.artifactID, target.type, target.name]);
		}

		// Preserve a stable key even for malformed historical data. Runtime
		// validation decides whether that target can actually execute.
		return JSON.stringify(['artifact-missing-ref', target.type, target.name, target.provenance]);
	}

	// Direct/runtime-owned targets use their provider identity. Keep type and
	// name in the key as a defensive fallback for legacy or malformed values.
	return JSON.stringify([
		'direct',
		target.providerIdentity ?? '',
		target.providerLocalID ?? '',
		target.type,
		target.name,
	]);
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
