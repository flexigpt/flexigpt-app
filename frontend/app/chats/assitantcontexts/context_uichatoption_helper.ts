import { DefaultModelParams, type ModelParam, type ProviderName } from '@/spec/inference';
import { DefaultUIChatOptions, type ModelPreset, type UIChatOption } from '@/spec/modelpreset';
import { AuthKeyTypeProvider, type SettingsSchema } from '@/spec/setting';

import { modelPresetStoreAPI, settingstoreAPI } from '@/apis/baseapi';
import { getAllProviderPresetsMap } from '@/apis/list_helper';

function hasApiKey(settings: SettingsSchema, providerName: ProviderName): boolean {
	return settings.authKeys.some(k => k.type === AuthKeyTypeProvider && k.keyName === providerName && k.nonEmpty);
}

// Ensure every optional field of `ModelParam` is filled.
// Falls back to hard-coded defaults when the model preset does not specify a value.
function buildModelParams(modelPreset: ModelPreset): ModelParam {
	const o: ModelParam = {
		name: modelPreset.name,
		stream: modelPreset.stream ?? DefaultModelParams.stream,
		maxPromptLength: modelPreset.maxPromptLength ?? DefaultModelParams.maxPromptLength,
		maxOutputLength: modelPreset.maxOutputLength ?? DefaultModelParams.maxOutputLength,
		systemPrompt: modelPreset.systemPrompt ?? DefaultModelParams.systemPrompt,
		timeout: modelPreset.timeout ?? DefaultModelParams.timeout,

		// Optional fields in modelparams
		temperature: modelPreset.temperature,
		reasoning: modelPreset.reasoning,

		outputParam: modelPreset.outputParam ?? DefaultModelParams.outputParam,
		stopSequences: modelPreset.stopSequences ?? DefaultModelParams.stopSequences,

		additionalParametersRawJSON: modelPreset.additionalParametersRawJSON,
	};

	if (o.temperature === undefined && o.reasoning === undefined) {
		o.temperature = DefaultModelParams.temperature;
	}
	return o;
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

				const modelParams = buildModelParams(modelPreset);

				const option: UIChatOption = {
					...modelParams,
					providerName: providerName,
					providerSDKType: providerPreset.sdkType,
					modelPresetID: modelPresetID,
					providerDisplayName: providerPreset.displayName,
					modelDisplayName: modelPreset.displayName,
					disablePreviousMessages: false,
				};

				allOptions.push(option);

				if (providerName === defaultProviderName && modelPresetID === providerPreset.defaultModelPresetID) {
					defaultOption = option;
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
