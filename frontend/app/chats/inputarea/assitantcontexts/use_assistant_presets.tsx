import { useCallback, useEffect, useState } from 'react';

import type { AssistantPreset } from '@/spec/assistantpreset';
import type { AssistantModelPresetOption } from '@/spec/modelpreset';
import type { AssistantInstructionTemplateOption } from '@/spec/prompt';
import type { AssistantSkillOption } from '@/spec/skill';
import { type AssistantToolOption, ToolImplType } from '@/spec/tool';

import { assistantPresetStoreAPI } from '@/apis/baseapi';

import { loadAssistantPresetEditorCatalog } from '@/assistantpresets/lib/assistant_preset_catalog';
import {
	getAllAssistantPresetBundles,
	getAllAssistantPresetListItems,
} from '@/assistantpresets/lib/assistant_preset_store_list_utils';
import {
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
} from '@/assistantpresets/lib/assistant_preset_utils';
import {
	type AssistantPresetOptionItem,
	buildAssistantPresetIdentityKey,
} from '@/chats/inputarea/assitantcontexts/assistant_preset_runtime';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function getBundleDisplayName(bundle: { displayName?: string; slug: string }, fallbackID: string): string {
	return bundle.displayName || bundle.slug || fallbackID;
}

function getAssistantPresetAvailability(
	preset: AssistantPreset,
	lookups: {
		modelOptionsByKey: Map<string, AssistantModelPresetOption>;
		instructionOptionsByKey: Map<string, AssistantInstructionTemplateOption>;
		toolOptionsByKey: Map<string, AssistantToolOption>;
		skillOptionsByKey: Map<string, AssistantSkillOption>;
	}
): Pick<AssistantPresetOptionItem, 'isSelectable' | 'availabilityReason'> {
	let targetProviderSDKType: string | undefined;

	if (preset.startingModelPresetRef) {
		const key = buildModelPresetRefKey(preset.startingModelPresetRef);
		const option = lookups.modelOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Starting model preset "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Starting model preset "${key}" is not available.`,
			};
		}

		targetProviderSDKType = option.providerPreset.sdkType;
	}

	for (const ref of preset.startingInstructionTemplateRefs ?? []) {
		const key = buildPromptTemplateRefKey(ref);
		const option = lookups.instructionOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Instruction template "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Instruction template "${key}" is not available.`,
			};
		}
	}

	for (const selection of preset.startingToolSelections ?? []) {
		const key = buildToolRefKey(selection.toolRef);
		const option = lookups.toolOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Tool "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Tool "${key}" is not available.`,
			};
		}

		if (targetProviderSDKType && option.toolDefinition.type === ToolImplType.SDK) {
			const toolSDKType = option.toolDefinition.sdkImpl?.sdkType?.trim();
			if (!toolSDKType) {
				return {
					isSelectable: false,
					availabilityReason: `Tool "${option.toolDefinition.displayName || option.toolDefinition.slug}" is missing SDK metadata.`,
				};
			}

			if (toolSDKType !== targetProviderSDKType) {
				return {
					isSelectable: false,
					availabilityReason: `Tool "${option.toolDefinition.displayName || option.toolDefinition.slug}" requires "${toolSDKType}", but this preset’s starting model uses "${targetProviderSDKType}".`,
				};
			}
		}
	}

	for (const ref of preset.startingEnabledSkillRefs ?? []) {
		const key = buildSkillRefKey(ref);
		const option = lookups.skillOptionsByKey.get(key);

		if (!option) {
			return {
				isSelectable: false,
				availabilityReason: `Skill "${key}" no longer exists.`,
			};
		}

		if (!option.isSelectable) {
			return {
				isSelectable: false,
				availabilityReason: option.availabilityReason ?? `Skill "${key}" is not available.`,
			};
		}
	}

	return { isSelectable: true };
}

export function useAssistantPresets() {
	const [presetOptions, setPresetOptions] = useState<AssistantPresetOptionItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const refreshPresets = useCallback(async () => {
		setLoading(true);
		setError(null);

		try {
			const bundles = await getAllAssistantPresetBundles(undefined, false);
			const catalog = await loadAssistantPresetEditorCatalog();

			if (bundles.length === 0) {
				setPresetOptions([]);
				return;
			}
			const bundleByID = new Map(bundles.map(bundle => [bundle.id, bundle]));
			const modelOptionsByKey = new Map(catalog.modelPresetOptions.map(option => [option.key, option] as const));
			const instructionOptionsByKey = new Map(
				catalog.instructionTemplateOptions.map(option => [option.key, option] as const)
			);
			const toolOptionsByKey = new Map(catalog.toolOptions.map(option => [option.key, option] as const));
			const skillOptionsByKey = new Map(catalog.skillOptions.map(option => [option.key, option] as const));

			const listItems = await getAllAssistantPresetListItems(
				bundles.map(bundle => bundle.id),
				false
			);

			const fullResults = await Promise.all(
				listItems.map(async item => {
					const preset = await assistantPresetStoreAPI.getAssistantPreset(
						item.bundleID,
						item.assistantPresetSlug,
						item.assistantPresetVersion
					);

					return {
						item,
						preset,
					};
				})
			);

			const nextOptions: AssistantPresetOptionItem[] = fullResults.flatMap(({ item, preset }) => {
				if (!preset) {
					return [];
				}

				const bundle = bundleByID.get(item.bundleID);
				const bundleDisplayName = bundle
					? getBundleDisplayName(bundle, item.bundleID)
					: item.bundleSlug || item.bundleID;

				const displayName = preset.displayName || preset.slug;
				const label = `${displayName} — ${bundleDisplayName} (${preset.slug}@${preset.version})`;
				const availability = getAssistantPresetAvailability(preset, {
					modelOptionsByKey,
					instructionOptionsByKey,
					toolOptionsByKey,
					skillOptionsByKey,
				});

				return [
					{
						key: buildAssistantPresetIdentityKey(item.bundleID, item.assistantPresetSlug, item.assistantPresetVersion),
						bundleID: item.bundleID,
						bundleSlug: item.bundleSlug,
						bundleDisplayName,
						displayName,
						description: preset.description,
						preset,
						label,
						isSelectable: availability.isSelectable,
						availabilityReason: availability.availabilityReason,
					},
				];
			});

			setPresetOptions(nextOptions);
		} catch (refreshError) {
			console.error('Failed to load assistant presets:', refreshError);
			setError(getErrorMessage(refreshError, 'Failed to load assistant presets.'));
			setPresetOptions([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		void refreshPresets();
	}, [refreshPresets]);

	return {
		presetOptions,
		loading,
		error,
		refreshPresets,
	};
}
