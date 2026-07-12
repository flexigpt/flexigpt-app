import type { AssistantModelPresetOption, ModelPresetRef, ProviderPreset } from '@/spec/modelpreset';
import type { AssistantSkillOption, SkillSelection } from '@/spec/skill';
import { SkillPresenceStatus } from '@/spec/skill';
import type { AssistantToolOption, ToolRef } from '@/spec/tool';

import { raceWithAbortSignal, withTimeout } from '@/lib/async_utils';

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
	loadErrors?: AssistantPresetCatalogLoadErrors;
}

type AssistantPresetCatalogSection = 'models' | 'tools' | 'skills';
export type AssistantPresetCatalogLoadErrors = Partial<Record<AssistantPresetCatalogSection, string>>;

export interface AssistantPresetCatalogLoadOptions {
	force?: boolean;
	signal?: AbortSignal;
}

interface AsyncCatalogCache<T> {
	value?: T;
	promise?: Promise<T>;
	generation: number;
	updatedAt?: number;
}

const modelOptionsCache: AsyncCatalogCache<AssistantModelPresetOption[]> = { generation: 0 };
const toolOptionsCache: AsyncCatalogCache<AssistantToolOption[]> = { generation: 0 };
const skillOptionsCache: AsyncCatalogCache<AssistantSkillOption[]> = { generation: 0 };
const editorCatalogCache: AsyncCatalogCache<AssistantPresetEditorCatalog> = { generation: 0 };

const MAX_CATALOG_PAGE_COUNT = 1_000;
const CATALOG_CACHE_TTL_MS = 30_000;
const CATALOG_SECTION_TIMEOUT_MS = 20_000;

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return fallback;
}

function loadWithCache<T>(cache: AsyncCatalogCache<T>, loader: () => Promise<T>, force: boolean): Promise<T> {
	if (!force && cache.promise) {
		return cache.promise;
	}
	if (
		!force &&
		cache.value !== undefined &&
		cache.updatedAt !== undefined &&
		Date.now() - cache.updatedAt < CATALOG_CACHE_TTL_MS
	) {
		return Promise.resolve(cache.value);
	}

	if (force) {
		cache.generation += 1;
	}
	const requestGeneration = cache.generation;
	const request = loader().then(value => {
		if (cache.generation === requestGeneration) {
			cache.value = value;
			cache.updatedAt = Date.now();
		}
		return value;
	});
	cache.promise = request;

	void request
		.finally(() => {
			if (cache.promise === request) {
				cache.promise = undefined;
			}
		})
		.catch(() => undefined);

	return request;
}

async function collectAllPages<TResponse, TItem>(
	fetchPage: (pageToken?: string) => Promise<TResponse>,
	pickItems: (response: TResponse) => TItem[],
	pickNextToken: (response: TResponse) => string | undefined
): Promise<TItem[]> {
	const items: TItem[] = [];
	const seenPageTokens = new Set<string>();
	let nextPageToken: string | undefined;

	for (let page = 0; page < MAX_CATALOG_PAGE_COUNT; page += 1) {
		if (nextPageToken) {
			if (seenPageTokens.has(nextPageToken)) {
				throw new Error('Catalog pagination returned a repeated page token.');
			}
			seenPageTokens.add(nextPageToken);
		}

		const response = await fetchPage(nextPageToken);
		items.push(...pickItems(response));

		nextPageToken = pickNextToken(response);
		if (!nextPageToken) {
			return items;
		}
	}

	throw new Error(`Catalog pagination exceeded ${MAX_CATALOG_PAGE_COUNT} pages.`);
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

async function loadModelPresetOptionsUncached(): Promise<AssistantModelPresetOption[]> {
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

function loadModelPresetOptions(options: AssistantPresetCatalogLoadOptions = {}) {
	return loadWithCache(modelOptionsCache, loadModelPresetOptionsUncached, options.force === true);
}

async function loadToolOptionsUncached(): Promise<AssistantToolOption[]> {
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

		const isBundleKnown = bundle !== undefined;
		const isBundleEnabled = bundle?.isEnabled ?? false;
		const isToolEnabled = tool.isEnabled;

		let availabilityReason: string | undefined;
		if (!isBundleKnown) {
			availabilityReason = 'Tool bundle no longer exists.';
		} else if (!isBundleEnabled) {
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

export function loadToolOptions(options: AssistantPresetCatalogLoadOptions = {}) {
	return loadWithCache(toolOptionsCache, loadToolOptionsUncached, options.force === true);
}

async function loadSkillOptionsUncached(): Promise<AssistantSkillOption[]> {
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

		const isBundleKnown = bundle !== undefined;
		const isBundleEnabled = bundle?.isEnabled ?? false;
		const isSkillEnabled = skill.isEnabled;

		let availabilityReason: string | undefined;
		if (!isBundleKnown) {
			availabilityReason = 'Skill bundle no longer exists.';
		} else if (!isBundleEnabled) {
			availabilityReason = 'Skill bundle is disabled.';
		} else if (!isSkillEnabled) {
			availabilityReason = 'Skill is disabled.';
		} else if (skill.presence?.status === SkillPresenceStatus.Missing) {
			availabilityReason = 'Skill files are missing from the configured location.';
		} else if (skill.presence?.status === SkillPresenceStatus.Error) {
			availabilityReason = skill.presence.lastCheckError || 'Skill files could not be verified.';
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

export function loadSkillOptions(options: AssistantPresetCatalogLoadOptions = {}) {
	return loadWithCache(skillOptionsCache, loadSkillOptionsUncached, options.force === true);
}

export function loadAssistantPresetEditorCatalog(
	options: AssistantPresetCatalogLoadOptions = {}
): Promise<AssistantPresetEditorCatalog> {
	const force = options.force === true;

	const catalogPromise = loadWithCache(
		editorCatalogCache,
		async () => {
			const [modelsResult, toolsResult, skillsResult] = await Promise.allSettled([
				withTimeout(
					loadModelPresetOptions({ force }),
					CATALOG_SECTION_TIMEOUT_MS,
					'Model preset catalog loading timed out.'
				),
				withTimeout(loadToolOptions({ force }), CATALOG_SECTION_TIMEOUT_MS, 'Tool catalog loading timed out.'),
				withTimeout(loadSkillOptions({ force }), CATALOG_SECTION_TIMEOUT_MS, 'Skill catalog loading timed out.'),
			]);

			const loadErrors: AssistantPresetCatalogLoadErrors = {};
			if (modelsResult.status === 'rejected') {
				loadErrors.models = getErrorMessage(modelsResult.reason, 'Failed to load model presets.');
			}
			if (toolsResult.status === 'rejected') {
				loadErrors.tools = getErrorMessage(toolsResult.reason, 'Failed to load tools.');
			}
			if (skillsResult.status === 'rejected') {
				loadErrors.skills = getErrorMessage(skillsResult.reason, 'Failed to load skills.');
			}

			return {
				modelPresetOptions: modelsResult.status === 'fulfilled' ? modelsResult.value : [],
				toolOptions: toolsResult.status === 'fulfilled' ? toolsResult.value : [],
				skillOptions: skillsResult.status === 'fulfilled' ? skillsResult.value : [],
				...(Object.keys(loadErrors).length > 0 ? { loadErrors } : {}),
			};
		},
		force
	);

	return raceWithAbortSignal(catalogPromise, options.signal);
}
