import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset, UIChatOption } from '@/spec/modelpreset';
import type { SettingsSchema } from '@/spec/setting';
import { DefaultUIChatOptions } from '@/spec/modelpreset';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import { modelPresetStoreAPI, settingstoreAPI } from '@/apis/baseapi';
import { collectAllPages } from '@/apis/wailsapi/transport';

import {
	mergeModelCapabilitiesOverride,
	sanitizeUIChatOptionByCapabilities,
} from '@/modelpresets/lib/capabilities_override';
import { buildEffectiveModelParamFromModelPreset } from '@/modelpresets/lib/modelpreset_effective_defaults';

export interface ChatInputOptionsResult {
	allOptions: UIChatOption[];
	default: UIChatOption;
}

function hasProviderAPIKey(settings: SettingsSchema, providerName: ProviderName): boolean {
	return settings.authKeys.some(
		key => key.type === AuthKeyTypeProvider && key.keyName === providerName && key.nonEmpty
	);
}

export async function getAllProviderPresetsMap(
	includeDisabled?: boolean
): Promise<Record<ProviderName, ProviderPreset>> {
	const result = Object.create(null) as Record<ProviderName, ProviderPreset>;

	const providers = await collectAllPages(async pageToken => {
		const page = await modelPresetStoreAPI.listProviderPresets(undefined, includeDisabled, undefined, pageToken);
		return {
			items: page.providers,
			nextPageToken: page.nextPageToken,
		};
	}, 20);

	for (const preset of providers) {
		if (Object.hasOwn(result, preset.name)) {
			throw new Error(`Provider preset ${preset.name} was returned more than once.`);
		}
		result[preset.name] = preset;
	}

	return result;
}

async function loadChatInputOptionsUncached(): Promise<ChatInputOptionsResult> {
	try {
		const [allProviderPresets, settings, defaultProviderName] = await Promise.all([
			getAllProviderPresetsMap(),
			settingstoreAPI.getSettings(),
			modelPresetStoreAPI.getDefaultProvider(),
		]);

		const allOptions: UIChatOption[] = [];
		let defaultOption: UIChatOption | undefined;

		for (const [providerName, providerPreset] of Object.entries(allProviderPresets)) {
			if (!providerPreset.isEnabled || !hasProviderAPIKey(settings, providerName)) {
				continue;
			}

			for (const [modelPresetID, modelPreset] of Object.entries(providerPreset.modelPresets)) {
				if (!modelPreset.isEnabled) {
					continue;
				}

				const modelParams = buildEffectiveModelParamFromModelPreset(modelPreset);
				const capabilitiesOverride = mergeModelCapabilitiesOverride(
					providerPreset.capabilitiesOverride,
					modelPreset.capabilitiesOverride
				);

				const option = sanitizeUIChatOptionByCapabilities({
					...modelParams,
					providerName,
					providerSDKType: providerPreset.sdkType,
					modelPresetID,
					providerDisplayName: providerPreset.displayName,
					modelDisplayName: modelPreset.displayName,
					includePreviousMessages: 'all',
					capabilitiesOverride,
				});

				allOptions.push(option);

				if (providerName === defaultProviderName && modelPresetID === providerPreset.defaultModelPresetID) {
					defaultOption = option;
				}
			}
		}

		if (!defaultOption) {
			defaultOption = allOptions[0] ?? DefaultUIChatOptions;
		}
		if (allOptions.length === 0) {
			allOptions.push(DefaultUIChatOptions);
		}

		return {
			allOptions,
			default: defaultOption,
		};
	} catch (error) {
		console.error('Error while building chat input options:', error);
		return {
			allOptions: [DefaultUIChatOptions],
			default: DefaultUIChatOptions,
		};
	}
}

const composerModelCatalog = createSharedAsyncCatalog(loadChatInputOptionsUncached);

export function getChatInputOptions(): Promise<ChatInputOptionsResult> {
	return composerModelCatalog.load(false);
}

export function invalidateComposerModelCatalog(): void {
	composerModelCatalog.invalidate();
}
