import { type CacheControl, type CacheControlKind, type CacheControlTTL } from '@/spec/inference';

import { CACHE_CONTROL_KIND_LABELS, CACHE_CONTROL_TTL_LABELS } from '@/modelpresets/lib/capabilities_override';

export const CACHE_CONTROL_TTL_PROVIDER_DEFAULT = '__provider_default__' as const;

export type CacheControlTTLSelection = CacheControlTTL | typeof CACHE_CONTROL_TTL_PROVIDER_DEFAULT;

export function resolveSupportedCacheControlKinds(
	supportedKinds: CacheControlKind[] | undefined,
	existing?: CacheControl
): CacheControlKind[] {
	const next =
		supportedKinds && supportedKinds.length > 0 ? [...supportedKinds] : existing?.kind ? [existing.kind] : [];
	return Array.from(new Set(next));
}

export function resolveSupportedCacheControlTTLs(
	supportedTTLs: CacheControlTTL[] | undefined,
	existing?: CacheControl
): CacheControlTTL[] {
	const next = supportedTTLs && supportedTTLs.length > 0 ? [...supportedTTLs] : existing?.ttl ? [existing.ttl] : [];
	return Array.from(new Set(next));
}

export function getInitialCacheControlKind(
	existing: CacheControl | undefined,
	supportedKinds: CacheControlKind[]
): CacheControlKind | '' {
	if (existing?.kind && supportedKinds.includes(existing.kind)) {
		return existing.kind;
	}
	return supportedKinds[0] ?? '';
}

export function getInitialCacheControlTTLSelection(
	existing: CacheControl | undefined,
	supportedTTLs: CacheControlTTL[]
): CacheControlTTLSelection {
	if (existing?.ttl && supportedTTLs.includes(existing.ttl)) {
		return existing.ttl;
	}
	return CACHE_CONTROL_TTL_PROVIDER_DEFAULT;
}

export function buildCacheControlFromForm(args: {
	enabled: boolean;
	kind: CacheControlKind | '';
	supportedKinds: CacheControlKind[];
	ttlSelection: CacheControlTTLSelection;
	key: string;
	supportsKey: boolean;
}): CacheControl | undefined {
	if (!args.enabled) return undefined;

	const kind = args.kind || args.supportedKinds[0];
	if (!kind) return undefined;

	const trimmedKey = args.key.trim();

	return {
		kind,
		...(args.ttlSelection !== CACHE_CONTROL_TTL_PROVIDER_DEFAULT ? { ttl: args.ttlSelection } : {}),
		...(args.supportsKey && trimmedKey ? { key: trimmedKey } : {}),
	};
}

export function buildCacheControlKindDropdownItems(
	supportedKinds: CacheControlKind[]
): Record<CacheControlKind, { isEnabled: boolean; displayName: string }> {
	return Object.fromEntries(
		supportedKinds.map(kind => [
			kind,
			{
				isEnabled: true,
				displayName: CACHE_CONTROL_KIND_LABELS[kind] ?? kind,
			},
		])
	) as Record<CacheControlKind, { isEnabled: boolean; displayName: string }>;
}

export function buildCacheControlTTLDropdownItems(
	supportedTTLs: CacheControlTTL[]
): Record<CacheControlTTLSelection, { isEnabled: boolean; displayName: string }> {
	return {
		[CACHE_CONTROL_TTL_PROVIDER_DEFAULT]: {
			isEnabled: true,
			displayName: 'Provider default',
		},
		...Object.fromEntries(
			supportedTTLs.map(ttl => [
				ttl,
				{
					isEnabled: true,
					displayName: CACHE_CONTROL_TTL_LABELS[ttl] ?? ttl,
				},
			])
		),
	} as Record<CacheControlTTLSelection, { isEnabled: boolean; displayName: string }>;
}
