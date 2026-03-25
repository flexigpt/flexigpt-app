import type { AssistantModelPresetOption, ModelPresetRef, ProviderPreset } from '@/spec/modelpreset';
import type { AssistantInstructionTemplateOption, PromptTemplateRef } from '@/spec/prompt';
import { PromptTemplateKind } from '@/spec/prompt';
import type { AssistantSkillOption, SkillRef } from '@/spec/skill';
import type { AssistantToolOption, ToolRef } from '@/spec/tool';

import { modelPresetStoreAPI, promptStoreAPI, skillStoreAPI, toolStoreAPI } from '@/apis/baseapi';

import {
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
} from '@/assistantpresets/lib/assistant_preset_utils';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';

export interface AssistantPresetEditorCatalog {
	modelPresetOptions: AssistantModelPresetOption[];
	instructionTemplateOptions: AssistantInstructionTemplateOption[];
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
	return [...items].sort((a, b) => {
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

export async function loadInstructionTemplateOptions(): Promise<AssistantInstructionTemplateOption[]> {
	const [promptBundles, listItems] = await Promise.all([
		collectAllPages(
			pageToken => promptStoreAPI.listPromptBundles(undefined, true, 200, pageToken),
			response => response.promptBundles,
			response => response.nextPageToken
		),
		collectAllPages(
			pageToken =>
				promptStoreAPI.listPromptTemplates(
					undefined,
					undefined,
					true,
					[PromptTemplateKind.InstructionsOnly],
					undefined,
					200,
					pageToken
				),
			response => response.promptTemplateListItems,
			response => response.nextPageToken
		),
	]);

	const bundleByID = new Map(promptBundles.map(bundle => [bundle.id, bundle]));

	const fullTemplates = await Promise.all(
		listItems.map(item => promptStoreAPI.getPromptTemplate(item.bundleID, item.templateSlug, item.templateVersion))
	);

	const options: AssistantInstructionTemplateOption[] = [];

	fullTemplates.forEach((template, index) => {
		if (!template) {
			return;
		}

		const item = listItems[index];
		const bundle = bundleByID.get(item.bundleID);

		const isBundleEnabled = bundle?.isEnabled ?? true;
		const isTemplateEnabled = template.isEnabled;
		const isResolved = template.isResolved;

		let availabilityReason: string | undefined;

		if (!isBundleEnabled) {
			availabilityReason = 'Prompt bundle is disabled.';
		} else if (!isTemplateEnabled) {
			availabilityReason = 'Prompt template is disabled.';
		} else if (template.kind !== PromptTemplateKind.InstructionsOnly) {
			availabilityReason = 'Only instructions-only prompt templates are allowed.';
		} else if (!isResolved) {
			availabilityReason = 'Prompt template must already be fully resolved.';
		}

		const ref: PromptTemplateRef = {
			bundleID: item.bundleID,
			templateSlug: template.slug,
			templateVersion: template.version,
		};

		const bundleDisplayName = bundle ? getBundleDisplayName(bundle, item.bundleID) : item.bundleSlug || item.bundleID;

		options.push({
			key: buildPromptTemplateRefKey(ref),
			ref,
			template,

			bundleSlug: bundle?.slug || item.bundleSlug || item.bundleID,
			bundleDisplayName,

			isBuiltIn: template.isBuiltIn,
			isBundleEnabled,
			isTemplateEnabled,
			isResolved,
			isSelectable: availabilityReason === undefined,
			availabilityReason,
			label: `${template.displayName || template.slug} — ${bundleDisplayName} (${template.slug}@${template.version})`,
		});
	});

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
			pageToken => skillStoreAPI.listSkills(undefined, undefined, true, true, 200, pageToken),
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
		}

		const ref: SkillRef = {
			bundleID: item.bundleID,
			skillSlug: item.skillSlug,
			skillID: skill.id,
		};

		const bundleDisplayName = bundle ? getBundleDisplayName(bundle, item.bundleID) : item.bundleSlug || item.bundleID;

		return {
			key: buildSkillRefKey(ref),
			ref,
			skillDefinition: skill,

			bundleSlug: bundle?.slug || item.bundleSlug || item.bundleID,
			bundleDisplayName,

			isBuiltIn: skill.isBuiltIn,
			isBundleEnabled,
			isSkillEnabled,
			isSelectable: availabilityReason === undefined,
			availabilityReason,
			label: `${skill.displayName || skill.name || skill.slug} — ${bundleDisplayName} (${item.skillSlug})`,
		};
	});

	return sortByBuiltInThenLabel(options);
}

export async function loadAssistantPresetEditorCatalog(): Promise<AssistantPresetEditorCatalog> {
	const [modelPresetOptions, instructionTemplateOptions, toolOptions, skillOptions] = await Promise.all([
		loadModelPresetOptions(),
		loadInstructionTemplateOptions(),
		loadToolOptions(),
		loadSkillOptions(),
	]);

	return {
		modelPresetOptions,
		instructionTemplateOptions,
		toolOptions,
		skillOptions,
	};
}
