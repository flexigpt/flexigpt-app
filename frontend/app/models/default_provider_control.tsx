import type { ArtifactRef } from '@/spec/artifact';

import type { ModelProviderManagementItem } from '@/apis/model_management';
import { isModelProviderRunnable } from '@/apis/model_management';

import { Dropdown } from '@/components/dropdown';

const DEFAULT_PROVIDER_NONE_KEY = '__no_default_provider__';
const DEFAULT_PROVIDER_UNAVAILABLE_KEY = '__default_provider_unavailable__';

interface DefaultProviderControlProps {
	providers: ModelProviderManagementItem[];
	defaultProvider?: ArtifactRef;
	pending?: boolean;
	onChange: (provider: ModelProviderManagementItem) => void;
}

function providerKey(provider: ModelProviderManagementItem): string {
	return `${provider.list.ref.rootID}\u0000${provider.list.ref.artifactID}`;
}

function artifactRefsEqual(left: ArtifactRef | undefined, right: ArtifactRef | undefined): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

export function DefaultProviderControl({
	providers,
	defaultProvider,
	pending = false,
	onChange,
}: DefaultProviderControlProps) {
	const selectedProvider = providers.find(provider => artifactRefsEqual(provider.list.ref, defaultProvider));
	const eligibleProviders = providers.filter(p => {
		return isModelProviderRunnable(p);
	});
	const defaultUnavailable = Boolean(defaultProvider) && !selectedProvider;
	const defaultNotRunnable = selectedProvider !== undefined && !isModelProviderRunnable(selectedProvider);

	const dropdownItems: Record<string, { isEnabled: boolean; displayName: string }> = {};
	const orderedKeys: string[] = [];

	if (!defaultProvider) {
		dropdownItems[DEFAULT_PROVIDER_NONE_KEY] = {
			isEnabled: false,
			displayName: 'No provider selected',
		};
		orderedKeys.push(DEFAULT_PROVIDER_NONE_KEY);
	}

	if (defaultUnavailable) {
		dropdownItems[DEFAULT_PROVIDER_UNAVAILABLE_KEY] = {
			isEnabled: false,
			displayName: 'Current backend default is unavailable',
		};
		orderedKeys.push(DEFAULT_PROVIDER_UNAVAILABLE_KEY);
	}

	for (const provider of providers) {
		const key = providerKey(provider);
		dropdownItems[key] = {
			isEnabled: isModelProviderRunnable(provider),
			displayName: provider.list.displayName || 'Provider',
		};
		orderedKeys.push(key);
	}

	const selectedKey = selectedProvider
		? providerKey(selectedProvider)
		: defaultUnavailable
			? DEFAULT_PROVIDER_UNAVAILABLE_KEY
			: DEFAULT_PROVIDER_NONE_KEY;

	const setupLabel = providers.length === 0 ? 'Add a provider' : 'Configure provider API keys';

	return (
		<div className="bg-base-100 mb-6 rounded-2xl px-4 py-3 shadow-lg">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center">
				<div className="text-sm font-medium sm:w-48">Default Provider</div>

				<div className="min-w-0 grow">
					<Dropdown<string>
						dropdownItems={dropdownItems}
						orderedKeys={orderedKeys}
						selectedKey={selectedKey}
						onChange={key => {
							const provider = providers.find(candidate => providerKey(candidate) === key);
							if (!provider || !isModelProviderRunnable(provider)) {
								return;
							}

							onChange(provider);
						}}
						filterDisabled={false}
						title="Select default provider"
						getDisplayName={key => dropdownItems[key]?.displayName ?? 'Default provider'}
						disabled={pending || eligibleProviders.length === 0}
					/>
				</div>
			</div>

			{eligibleProviders.length === 0 ? (
				<div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
					<span className="text-error">No enabled provider with a configured API key is available. {setupLabel}</span>
				</div>
			) : null}

			{defaultUnavailable ? (
				<div className="text-error mt-2 text-xs">
					The backend default is no longer available. Select an enabled provider with configured credentials to replace
					it.
				</div>
			) : null}

			{defaultNotRunnable ? (
				<div className="text-error mt-2 text-xs">
					The backend default is shown above, but it is not currently enabled and ready to send requests.
				</div>
			) : null}
		</div>
	);
}
