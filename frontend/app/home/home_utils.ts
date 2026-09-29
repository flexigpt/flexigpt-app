import type { ModelProviderListItem } from '@/spec/model';

const DEFAULT_PROVIDER_NAME_HINTS = ['openai', 'anthropic', 'gemini', 'google', 'openrouter'];

function normaliseProviderText(value: string) {
	return value.toLowerCase().replaceAll(/[^a-z0-9]/g, '');
}

function providerDisplayName(providers: ModelProviderListItem[], providerName: string) {
	return providers.find(item => item.name === providerName)?.displayName || providerName;
}

export function getConfiguredProviderNames(providers: ModelProviderListItem[]) {
	return providers.filter(item => item.credentialConfigured).map(item => item.name);
}

export function pickDefaultProviderName(providers: ModelProviderListItem[]) {
	if (providers.length === 0) {
		return null;
	}

	for (const hint of DEFAULT_PROVIDER_NAME_HINTS) {
		const match = providers.find(item => {
			const values = [item.name, item.displayName ?? ''].map(v => normaliseProviderText(v));
			return values.some(value => value.includes(hint));
		});

		if (match) {
			return match.name;
		}
	}

	return providers[0].name;
}

export function formatConfiguredProviderSummary(configuredProviderNames: string[], providers: ModelProviderListItem[]) {
	if (configuredProviderNames.length === 0) {
		return '';
	}

	const [first, ...rest] = configuredProviderNames;
	const firstName = providerDisplayName(providers, first);
	return rest.length > 0 ? `${firstName} + ${rest.length} more` : firstName;
}
