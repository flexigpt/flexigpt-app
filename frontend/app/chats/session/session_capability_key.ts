import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import { CapabilityTargetForm } from '@/spec/artifact';

export function sessionArtifactKey(ref: Pick<ArtifactRef, 'rootID' | 'artifactID'>): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

/**
 * Session collection override keys must use the same identity regardless of
 * whether they were authored from an Agent projection or a direct Composer
 * choice. This is intentionally not the UI-only Tool identity key.
 */
export function sessionCapabilityKey(target: CapabilityTarget): string {
	if (target.form === CapabilityTargetForm.Artifact && target.artifact) {
		return sessionArtifactKey(target.artifact);
	}

	return ['direct', target.providerIdentity ?? '', target.providerLocalID ?? '', target.type, target.name].join(':');
}
