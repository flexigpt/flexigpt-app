import { getProviderSDKOptions } from '@/models/lib/provider_sdk';

export function providerSDKDropdownItems(
	currentAdapter?: string
): Record<string, { isEnabled: boolean; displayName: string }> {
	const items = Object.fromEntries(
		getProviderSDKOptions().map(option => [
			option.adapter,
			{
				isEnabled: true,
				displayName: option.displayName,
			},
		])
	) as Record<string, { isEnabled: boolean; displayName: string }>;

	if (currentAdapter && !Object.hasOwn(items, currentAdapter)) {
		items[currentAdapter] = {
			isEnabled: false,
			displayName: 'Unsupported compatibility mode',
		};
	}

	return items;
}

export function makeUniqueLogicalName(seed: string, existingNames: Iterable<string>, fallback: string): string {
	const normalized = seed
		.trim()
		.toLowerCase()
		.replaceAll(/[^a-z0-9]+/g, '-')
		.replaceAll(/^-+|-+$/g, '');

	let base = normalized || fallback;
	if (!/^[a-z]/.test(base)) {
		base = `${fallback}-${base}`;
	}

	base = base.slice(0, 120).replaceAll(/-+$/g, '') || fallback;

	const existing = new Set([...existingNames].map(name => name.toLowerCase()));
	let candidate = base;
	let suffix = 2;

	while (existing.has(candidate.toLowerCase())) {
		const suffixText = `-${suffix}`;
		candidate = `${base.slice(0, 128 - suffixText.length)}${suffixText}`;
		suffix += 1;
	}

	return candidate;
}

export function displayModelName(displayName?: string, fallback = 'Model'): string {
	return displayName?.trim() || fallback;
}
