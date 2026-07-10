import type { AssistantModelPresetOption, ModelPresetRef, ProviderPreset } from '@/spec/modelpreset';
import type { AssistantSkillOption, SkillSelection } from '@/spec/skill';
import type { AssistantToolOption, ToolRef } from '@/spec/tool';

import { modelPresetStoreAPI, skillStoreAPI, toolStoreAPI } from '@/apis/baseapi';

import {
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
} from '@/assistantpresets/lib/assistant_preset_utils';
import { isInstructionInsertSkill } from '@/skills/lib/skill_artifact_utils';

export interface AssistantPresetEditorCatalog {
	modelPresetOptions: AssistantModelPresetOption[];
	toolOptions: AssistantToolOption[];
	skillOptions: AssistantSkillOption[];
}

async function collectAllPages<TResponse, TItem>(
	fetchPage: (pageToken?: string) => Promise<TResponse>,
	pickItems: (response: TResponse) => TItem[],
	pickNextToken: (response: TResponse) => string | undefined
): Promise<TItem[]> {
	const items: TItem[] = [];
	let nextPageToken: string | undefined = undefined;

	while (true) {
		const response = await fetchPage(nextPageToken);
		items.push(...pickItems(response));

		nextPageToken = pickNextToken(response);
		if (!nextPageToken) {
			break;
		}
	}

	return items;
}

function sortByBuiltInThenLabel<T extends { isBuiltIn: boolean; label: string; key: string }>(items: T[]): T[] {
	return [...items].toSorted((a, b) => {
		if (a.isBuiltIn !== b.isBuiltIn) {
			return a.isBuiltIn ? 1 : -1;
		}

		const byLabel = a.label.localeCompare(b.label);
		if (byLabel !== 0) {
			return byLabel;
		}

		return a.key.localeCompare(b.key);
	});
}

function getBundleDisplayName(bundle: { displayName?: string; slug: string }, fallbackID: string): string {
	return bundle.displayName || bundle.slug || fallbackID;
}

function getModelAvailabilityReason(provider: ProviderPreset, modelEnabled: boolean): string | undefined {
	if (!provider.isEnabled) {
		return 'Provider is disabled.';
	}
	if (!modelEnabled) {
		return 'Model preset is disabled.';
	}
	return undefined;
}

async function loadModelPresetOptions(): Promise<AssistantModelPresetOption[]> {
	const providers = await collectAllPages(
		pageToken => modelPresetStoreAPI.listProviderPresets(undefined, true, 200, pageToken),
		response => response.providers,
		response => response.nextPageToken
	);

	const options: AssistantModelPresetOption[] = [];

	for (const provider of providers) {
		for (const model of Object.values(provider.modelPresets ?? {})) {
			const ref: ModelPresetRef = {
				providerName: provider.name,
				modelPresetID: model.id,
			};

			const availabilityReason = getModelAvailabilityReason(provider, model.isEnabled);

			options.push({
				key: buildModelPresetRefKey(ref),
				ref,
				providerPreset: provider,
				modelPreset: model,

				isBuiltIn: model.isBuiltIn,
				isProviderEnabled: provider.isEnabled,
				isModelEnabled: model.isEnabled,
				isSelectable: availabilityReason === undefined,
				availabilityReason,
				label: `${model.displayName || model.name} — ${provider.displayName || provider.name} (${provider.name}/${model.id})`,
			});
		}
	}

	return sortByBuiltInThenLabel(options);
}

export async function loadToolOptions(): Promise<AssistantToolOption[]> {
	const [toolBundles, toolListItems] = await Promise.all([
		collectAllPages(
			pageToken => toolStoreAPI.listToolBundles(undefined, true, 200, pageToken),
			response => response.toolBundles,
			response => response.nextPageToken
		),
		collectAllPages(
			pageToken => toolStoreAPI.listTools(undefined, undefined, true, 200, pageToken),
			response => response.toolListItems,
			response => response.nextPageToken
		),
	]);

	const bundleByID = new Map(toolBundles.map(bundle => [bundle.id, bundle]));

	const options: AssistantToolOption[] = toolListItems.map(item => {
		const bundle = bundleByID.get(item.bundleID);
		const tool = item.toolDefinition;

		const isBundleEnabled = bundle?.isEnabled ?? true;
		const isToolEnabled = tool.isEnabled;

		let availabilityReason: string | undefined;
		if (!isBundleEnabled) {
			availabilityReason = 'Tool bundle is disabled.';
		} else if (!isToolEnabled) {
			availabilityReason = 'Tool is disabled.';
		}

		const toolRef: ToolRef = {
			bundleID: item.bundleID,
			toolSlug: item.toolSlug,
			toolVersion: item.toolVersion,
		};

		const bundleDisplayName = bundle ? getBundleDisplayName(bundle, item.bundleID) : item.bundleSlug || item.bundleID;

		return {
			key: buildToolRefKey(toolRef),
			toolRef,
			toolDefinition: tool,

			bundleSlug: bundle?.slug || item.bundleSlug || item.bundleID,
			bundleDisplayName,

			hasUserArgSchema: Boolean(tool.userArgSchema),
			isBuiltIn: tool.isBuiltIn,
			isBundleEnabled,
			isToolEnabled,
			isSelectable: availabilityReason === undefined,
			availabilityReason,
			label: `${tool.displayName || tool.slug} — ${bundleDisplayName} (${tool.slug}@${tool.version})`,
		};
	});

	return sortByBuiltInThenLabel(options);
}

export async function loadSkillOptions(): Promise<AssistantSkillOption[]> {
	const [skillBundles, skillListItems] = await Promise.all([
		collectAllPages(
			pageToken => skillStoreAPI.listSkillBundles(undefined, true, 200, pageToken),
			response => response.skillBundles,
			response => response.nextPageToken
		),
		collectAllPages(
			pageToken =>
				skillStoreAPI.listSkills({
					bundleIDs: [],
					types: [],
					includeDisabled: true,
					includeMissing: true,
					recommendedPageSize: 200,
					pageToken: pageToken,
				}),
			response => response.skillListItems,
			response => response.nextPageToken
		),
	]);

	const bundleByID = new Map(skillBundles.map(bundle => [bundle.id, bundle]));

	const options: AssistantSkillOption[] = skillListItems.map(item => {
		const bundle = bundleByID.get(item.bundleID);
		const skill = item.skillDefinition;

		const isBundleEnabled = bundle?.isEnabled ?? true;
		const isSkillEnabled = skill.isEnabled;

		let availabilityReason: string | undefined;
		if (!isBundleEnabled) {
			availabilityReason = 'Skill bundle is disabled.';
		} else if (!isSkillEnabled) {
			availabilityReason = 'Skill is disabled.';
		} else if (!isInstructionInsertSkill(skill)) {
			availabilityReason =
				'User-message skills are composer templates and cannot be assistant preset skill-session selections.';
		}

		const sel: SkillSelection = {
			skillRef: {
				bundleID: item.bundleID,
				skillSlug: item.skillSlug,
				skillID: skill.id,
			},
			preLoadAsActive: false,
			useAsInstructions: false,
		};

		const bundleDisplayName = bundle ? getBundleDisplayName(bundle, item.bundleID) : item.bundleSlug || item.bundleID;

		return {
			key: buildSkillRefKey(sel.skillRef),
			sel,
			skillDefinition: skill,

			bundleSlug: bundle?.slug || item.bundleSlug || item.bundleID,
			bundleDisplayName,

			isBuiltIn: skill.isBuiltIn,
			isBundleEnabled,
			isSkillEnabled,
			isSelectable: availabilityReason === undefined,
			availabilityReason,
			label: `${skill.displayName || skill.name || skill.slug} — ${bundleDisplayName} (${item.skillSlug} · ${
				skill.insert || 'instructions'
			})`,
		};
	});

	return sortByBuiltInThenLabel(options);
}

export async function loadAssistantPresetEditorCatalog(): Promise<AssistantPresetEditorCatalog> {
	const [modelPresetOptions, toolOptions, skillOptions] = await Promise.all([
		loadModelPresetOptions(),
		loadToolOptions(),
		loadSkillOptions(),
	]);

	return {
		modelPresetOptions,
		toolOptions,
		skillOptions,
	};
}
