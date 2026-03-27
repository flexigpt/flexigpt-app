import { type ProviderName } from '@/spec/inference';
import { DefaultUIChatOptions, type UIChatOption } from '@/spec/modelpreset';
import { AuthKeyTypeProvider, type SettingsSchema } from '@/spec/setting';

import { modelPresetStoreAPI, settingstoreAPI } from '@/apis/baseapi';
import { getAllProviderPresetsMap } from '@/apis/list_helper';

import {
	mergeModelCapabilitiesOverride,
	sanitizeUIChatOptionByCapabilities,
} from '@/modelpresets/lib/capabilities_override';
import { buildEffectiveModelParamFromModelPreset } from '@/modelpresets/lib/modelpreset_effective_defaults';

function hasApiKey(settings: SettingsSchema, providerName: ProviderName): boolean {
	return settings.authKeys.some(k => k.type === AuthKeyTypeProvider && k.keyName === providerName && k.nonEmpty);
}

export async function getChatInputOptions(): Promise<{
	allOptions: UIChatOption[];
	default: UIChatOption;
}> {
	try {
		/* fetch everything in parallel */
		const [allProviderPresets, settings, defaultProviderName] = await Promise.all([
			getAllProviderPresetsMap(), // contains built-ins + user presets merged
			settingstoreAPI.getSettings(),
			modelPresetStoreAPI.getDefaultProvider(),
		]);

		const allOptions: UIChatOption[] = [];
		let defaultOption: UIChatOption | undefined;

		for (const [providerName, providerPreset] of Object.entries(allProviderPresets)) {
			/* provider disabled or no key → skip */
			if (!providerPreset.isEnabled || !hasApiKey(settings, providerName)) {
				continue;
			}

			for (const [modelPresetID, modelPreset] of Object.entries(providerPreset.modelPresets)) {
				if (!modelPreset.isEnabled) continue;

				const modelParams = buildEffectiveModelParamFromModelPreset(modelPreset);
				const mergedCaps = mergeModelCapabilitiesOverride(
					providerPreset.capabilitiesOverride,
					modelPreset.capabilitiesOverride
				);

				const option: UIChatOption = {
					...modelParams,
					providerName: providerName,
					providerSDKType: providerPreset.sdkType,
					modelPresetID: modelPresetID,
					providerDisplayName: providerPreset.displayName,
					modelDisplayName: modelPreset.displayName,
					includePreviousMessages: 'all',
					capabilitiesOverride: mergedCaps,
				};

				allOptions.push(sanitizeUIChatOptionByCapabilities(option));

				if (providerName === defaultProviderName && modelPresetID === providerPreset.defaultModelPresetID) {
					defaultOption = sanitizeUIChatOptionByCapabilities(option);
				}
			}
		}

		if (!defaultOption) {
			if (allOptions.length > 0) {
				defaultOption = allOptions[0];
			} else {
				defaultOption = DefaultUIChatOptions;
				allOptions.push(DefaultUIChatOptions);
			}
		}

		return { allOptions, default: defaultOption };
	} catch (error) {
		console.error('Error while building chat input options:', error);
		return { allOptions: [DefaultUIChatOptions], default: DefaultUIChatOptions };
	}
}
