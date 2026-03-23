import type { ModelPresetRef, ProviderPreset } from '@/spec/modelpreset';
import type { PromptTemplateRef } from '@/spec/prompt';
import { PromptTemplateKind } from '@/spec/prompt';
import type { SkillRef, SkillType } from '@/spec/skill';
import type { ToolRef, ToolStoreChoiceType } from '@/spec/tool';

import { modelPresetStoreAPI, promptStoreAPI, skillStoreAPI, toolStoreAPI } from '@/apis/baseapi';

import {
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
} from '@/assistantpresets/lib/assistant_preset_utils';
import { buildPromptTemplateRefKey } from '@/prompts/lib/prompt_template_ref';

export interface ModelPresetOption {
	key: string;
	label: string;
	ref: ModelPresetRef;
	providerName: string;
	providerDisplayName: string;
	modelPresetID: string;
	modelDisplayName: string;
	isBuiltIn: boolean;
	isSelectable: boolean;
	isProviderEnabled: boolean;
	isModelEnabled: boolean;
	availabilityReason?: string;
}

export interface InstructionTemplateOption {
	key: string;
	label: string;
	ref: PromptTemplateRef;
	bundleID: string;
	bundleSlug: string;
	bundleDisplayName: string;
	displayName: string;
	templateSlug: string;
	version: string;
	isBuiltIn: boolean;
	isSelectable: boolean;
	isBundleEnabled: boolean;
	isTemplateEnabled: boolean;
	isResolved: boolean;
	availabilityReason?: string;
}

export interface ToolOption {
	key: string;
	label: string;
	toolRef: ToolRef;
	bundleID: string;
	bundleSlug: string;
	bundleDisplayName: string;
	displayName: string;
	toolSlug: string;
	version: string;
	toolType: ToolStoreChoiceType;
	autoExecReco: boolean;
	hasUserArgSchema: boolean;
	isBuiltIn: boolean;
	isSelectable: boolean;
	isBundleEnabled: boolean;
	isToolEnabled: boolean;
	availabilityReason?: string;
}

export interface SkillOption {
	key: string;
	label: string;
	ref: SkillRef;
	bundleID: string;
	bundleSlug: string;
	bundleDisplayName: string;
	displayName: string;
	skillSlug: string;
	skillID: string;
	skillType: SkillType;
	isBuiltIn: boolean;
	isSelectable: boolean;
	isBundleEnabled: boolean;
	isSkillEnabled: boolean;
	availabilityReason?: string;
}

export interface AssistantPresetEditorCatalog {
	modelPresetOptions: ModelPresetOption[];
	instructionTemplateOptions: InstructionTemplateOption[];
	toolOptions: ToolOption[];
	skillOptions: SkillOption[];
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

async function loadModelPresetOptions(): Promise<ModelPresetOption[]> {
	const providers = await collectAllPages(
		pageToken => modelPresetStoreAPI.listProviderPresets(undefined, true, 200, pageToken),
		response => response.providers,
		response => response.nextPageToken
	);

	const options: ModelPresetOption[] = [];

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
				providerName: provider.name,
				providerDisplayName: provider.displayName || provider.name,
				modelPresetID: model.id,
				modelDisplayName: model.displayName || model.name,
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

async function loadInstructionTemplateOptions(): Promise<InstructionTemplateOption[]> {
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

	const options: InstructionTemplateOption[] = [];

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
			bundleID: item.bundleID,
			bundleSlug: bundle?.slug || item.bundleSlug || item.bundleID,
			bundleDisplayName,
			displayName: template.displayName || template.slug,
			templateSlug: template.slug,
			version: template.version,
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

async function loadToolOptions(): Promise<ToolOption[]> {
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

	const options: ToolOption[] = toolListItems.map(item => {
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
			bundleID: item.bundleID,
			bundleSlug: bundle?.slug || item.bundleSlug || item.bundleID,
			bundleDisplayName,
			displayName: tool.displayName || tool.slug,
			toolSlug: tool.slug,
			version: tool.version,
			toolType: tool.llmToolType,
			autoExecReco: tool.autoExecReco,
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

async function loadSkillOptions(): Promise<SkillOption[]> {
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

	const options: SkillOption[] = skillListItems.map(item => {
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
			bundleID: item.bundleID,
			bundleSlug: bundle?.slug || item.bundleSlug || item.bundleID,
			bundleDisplayName,
			displayName: skill.displayName || skill.name || skill.slug,
			skillSlug: item.skillSlug,
			skillID: skill.id,
			skillType: skill.type,
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
