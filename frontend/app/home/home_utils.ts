import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import { type AuthKeyMeta, AuthKeyTypeProvider } from '@/spec/setting';

const DEFAULT_PROVIDER_NAME_HINTS = ['openai', 'anthropic', 'gemini', 'google', 'openrouter'];

function normaliseProviderText(value: string) {
	return value.toLowerCase().replaceAll(/[^a-z0-9]/g, '');
}

function providerDisplayName(providerPresets: Record<ProviderName, ProviderPreset>, providerName: string) {
	return providerPresets[providerName]?.displayName || providerName;
}

export function getConfiguredProviderNames(authKeys: AuthKeyMeta[]) {
	return authKeys
		.filter(key => key.type === AuthKeyTypeProvider && key.nonEmpty)
		.map(key => key.keyName)
		.filter(Boolean);
}

export function pickDefaultProviderName(providerPresets: Record<ProviderName, ProviderPreset>) {
	const entries = Object.entries(providerPresets);
	if (entries.length === 0) {
		return null;
	}

	for (const hint of DEFAULT_PROVIDER_NAME_HINTS) {
		const match = entries.find(([providerName, preset]) => {
			const values = [providerName, preset.displayName ?? ''].map(v => normaliseProviderText(v));
			return values.some(value => value.includes(hint));
		});

		if (match) {
			return match[0];
		}
	}

	return entries[0][0];
}

export function formatConfiguredProviderSummary(
	configuredProviderNames: string[],
	providerPresets: Record<ProviderName, ProviderPreset>
) {
	if (configuredProviderNames.length === 0) {
		return '';
	}

	const [first, ...rest] = configuredProviderNames;
	const firstName = providerDisplayName(providerPresets, first);
	return rest.length > 0 ? `${firstName} + ${rest.length} more` : firstName;
}
