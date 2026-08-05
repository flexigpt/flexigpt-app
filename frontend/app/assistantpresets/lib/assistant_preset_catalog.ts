import type { ArtifactRef } from '@/spec/artifact';
import { ArtifactState } from '@/spec/artifact';
import type { AssistantModelPresetOption, ModelPresetRef, ProviderPreset } from '@/spec/modelpreset';
import type { AssistantSkillOption } from '@/spec/skill';
import { SkillBundleAttachmentRole, SkillInsert } from '@/spec/skill';
import type { AssistantToolOption, ToolRef } from '@/spec/tool';

import { raceWithAbortSignal, withTimeout } from '@/lib/async_utils';

import { artifactStoreAPI, modelPresetStoreAPI, skillBundleAPI, toolStoreAPI } from '@/apis/baseapi';

import {
	buildModelPresetRefKey,
	buildSkillRefKey,
	buildToolRefKey,
} from '@/assistantpresets/lib/assistant_preset_utils';

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

function artifactRefKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

async function loadSkillOptionsUncached(): Promise<AssistantSkillOption[]> {
	const roots = await artifactStoreAPI.listArtifactRoots();
	const bundleGroups = await Promise.all(roots.map(root => skillBundleAPI.listSkillBundles(root.id)));
	const bundles = bundleGroups.flat();
	const artifactGroups = await Promise.all(
		bundles.map(async bundle => ({
			bundle,
			artifacts: await skillBundleAPI.listSkillBundleArtifacts(bundle.bundle),
		}))
	);
	const artifactRefs = artifactGroups.flatMap(group => group.artifacts.map(artifact => artifact.artifact));
	const runtimeSkills =
		artifactRefs.length > 0 ? await skillBundleAPI.listRuntimeSkills({ allowArtifacts: artifactRefs }) : [];
	const runtimeSkillByArtifact = new Map(runtimeSkills.map(skill => [artifactRefKey(skill.artifact), skill]));

	const options: AssistantSkillOption[] = artifactGroups.flatMap(({ bundle, artifacts }) =>
		artifacts.flatMap(artifact => {
			const runtimeSkill = runtimeSkillByArtifact.get(artifactRefKey(artifact.artifact));
			if (!runtimeSkill) {
				return [];
			}

			let availabilityReason: string | undefined;
			if (!bundle.enabled) {
				availabilityReason = 'Skill Bundle is disabled.';
			} else if (!artifact.enabled) {
				availabilityReason = 'Skill is disabled.';
			} else if (artifact.state !== ArtifactState.Available) {
				availabilityReason = `Skill is ${artifact.state}.`;
			} else if (runtimeSkill.errorMessage) {
				availabilityReason = runtimeSkill.errorMessage;
			} else if (runtimeSkill.insert !== SkillInsert.Instructions) {
				availabilityReason =
					'User-message Skills are composer templates and cannot be selected as Assistant Preset session Skills.';
			}

			const bundleDisplayName = bundle.displayName || bundle.logicalName || bundle.bundle.collectionID;
			const isBuiltIn = bundle.attachments.some(attachment => attachment.role === SkillBundleAttachmentRole.BuiltIn);

			return [
				{
					key: buildSkillRefKey(artifact.artifact),
					sel: {
						artifact: {
							rootID: artifact.artifact.rootID,
							artifactID: artifact.artifact.artifactID,
						},
						preLoadAsActive: false,
						useAsInstructions: false,
					},
					skillDefinition: runtimeSkill,
					bundleSlug: bundle.logicalName,
					bundleDisplayName,
					isBuiltIn,
					isBundleEnabled: bundle.enabled,
					isSkillEnabled: artifact.enabled,
					isSelectable: availabilityReason === undefined,
					availabilityReason,
					label: `${runtimeSkill.displayName || runtimeSkill.name || artifact.name} — ${bundleDisplayName} (${
						runtimeSkill.insert ?? SkillInsert.Instructions
					})`,
				},
			];
		})
	);

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
