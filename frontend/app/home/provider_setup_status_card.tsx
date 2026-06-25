import { FiAlertTriangle, FiArrowRight, FiCheckCircle, FiKey, FiSettings } from 'react-icons/fi';

import { Link } from 'react-router';

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

export function HomeAuthKeyModalIntro({ onNavigateAway }: { onNavigateAway: () => void }) {
	return (
		<div className="space-y-2">
			<div>
				<div className="flex items-center gap-2 font-semibold">
					<FiKey size={14} />
					<span>Choose a provider and paste its API key.</span>
				</div>
				<p className="text-base-content/70 mt-1 text-xs">
					The secret is stored through the OS keyring. FlexiGPT only shows local metadata after saving.
				</p>
			</div>

			<p className="text-base-content/70 text-xs">
				Need local, Ollama, llama.cpp or a custom compatible endpoint? Configure it in{' '}
				<Link to="/modelpresets/" className="link" onClick={onNavigateAway}>
					Model Presets{' '}
				</Link>{' '}
				first.
				<br />
				<div className="flex items-center gap-1">
					<span>You can add more keys later from the sidebar:</span>
					<FiSettings className="inline font-semibold" size={12} />{' '}
					<span className="font-semibold">Settings &rarr; Auth Keys</span>
				</div>
			</p>
		</div>
	);
}

export function ProviderSetupStatus({
	settingsLoaded,
	hasUsableProviderKey,
	providerSummary,
	onAddKey,
}: {
	settingsLoaded: boolean;
	hasUsableProviderKey: boolean;
	providerSummary: string;
	onAddKey: () => void;
}) {
	if (!settingsLoaded) {
		return null;
	}

	if (hasUsableProviderKey) {
		return (
			<div className="text-base-content/60 mt-3 flex flex-wrap items-center justify-center gap-2 text-xs">
				<span className="text-success inline-flex items-center gap-1">
					<FiCheckCircle size={13} />
					API key configured:
				</span>
				<span>{providerSummary}</span>
				<span className="opacity-50">·</span>
				<Link to="/settings/#auth-keys" className="link-hover inline-flex items-center gap-1">
					Manage keys &rarr;
				</Link>
			</div>
		);
	}

	return (
		<button
			type="button"
			className="group border-warning/40 bg-warning/10 hover:border-warning/70 mt-4 block w-full max-w-lg rounded-2xl border text-left shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
			onClick={onAddKey}
		>
			<div className="flex items-center gap-3 p-4">
				<div className="flex shrink-0 items-center justify-center">
					<div className="bg-warning/20 text-warning-content rounded-xl p-2">
						<FiAlertTriangle size={18} />
					</div>
				</div>

				<div className="flex min-w-0 flex-1 flex-col">
					<div className="flex items-center justify-between gap-3">
						<span className="text-sm font-semibold">Add an API key to start</span>
						<FiArrowRight size={18} className="shrink-0 transition-transform group-hover:translate-x-1" />
					</div>

					<p className="text-base-content/70 mt-1 text-xs/relaxed">Choose a built-in provider, paste your key.</p>
				</div>
			</div>
		</button>
	);
}
