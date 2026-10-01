// oxlint-disable max-classes-per-file
import type { ArtifactRef } from '@/spec/artifact';
import type { CacheControl } from '@/spec/inference';
import type {
	ManagedModelCreateRequest,
	ManagedModelReplaceRequest,
	ManagedModelResult,
	ManagedProviderCreateRequest,
	ManagedProviderReplaceRequest,
	ManagedProviderResult,
	ModelCapabilities,
	ModelDefaults,
	ModelListItem,
	ModelProviderDocument,
	ModelProviderListItem,
	ModelProviderView,
	ModelView,
	ProviderAPIKeyStatus,
	UIModelOption,
} from '@/spec/model';
import { ArtifactState } from '@/spec/artifact';
import { ModelLookupScope } from '@/spec/model';

import { mapWithConcurrency } from '@/lib/async_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type { IModelAggregateAPI, IModelStoreAPI } from '@/apis/interface';

import { mergeModelCapabilities } from '@/models/lib/capabilities';
import { modelRefEqual } from '@/models/lib/document';
import { buildModelParamFromDefaults } from '@/models/lib/model_defaults';
import { getProviderSDKType } from '@/models/lib/provider_sdk';

export interface ModelProviderManagementItem {
	list: ModelProviderListItem;
	view: ModelProviderView;
	apiKey: ProviderAPIKeyStatus;
}

export interface ModelManagementItem {
	list: ModelListItem;
	view: ModelView;
	provider?: ModelProviderManagementItem;
}

export interface ModelManagementSnapshot {
	providers: ModelProviderManagementItem[];
	models: ModelManagementItem[];
	defaultProvider?: ArtifactRef;
}

export interface ModelCatalog {
	options: UIModelOption[];
	defaultOption: UIModelOption;
}

export type ModelCatalogUnavailableReason = 'no-runnable-provider' | 'no-runnable-model';

export class ModelCatalogUnavailableError extends Error {
	// oxlint-disable-next-line typescript/parameter-properties
	constructor(readonly reason: ModelCatalogUnavailableReason) {
		super(
			reason === 'no-runnable-provider'
				? 'No enabled provider with an API key is available.'
				: 'No enabled model is available for the configured providers.'
		);
		this.name = 'ModelCatalogUnavailableError';
	}
}

export function isModelCatalogUnavailableError(error: unknown): error is ModelCatalogUnavailableError {
	return error instanceof ModelCatalogUnavailableError;
}

function providerRequiresAPIKey(document: ModelProviderDocument): boolean {
	return (document.authentication?.mode ?? ('apiKeyHeader' as string)) !== 'none';
}

export function isModelProviderRunnable(provider: ModelProviderManagementItem): boolean {
	if (provider.list.state !== ArtifactState.Available || !provider.list.enabled) {
		return false;
	}

	return !providerRequiresAPIKey(provider.view.document) || provider.apiKey.configured;
}

function resolveProvider(
	model: ModelManagementItem,
	providers: ModelProviderManagementItem[]
): ModelProviderManagementItem | undefined {
	const reference = model.view.document.provider;

	if (reference.scope === ModelLookupScope.Builtin) {
		return providers.find(provider => provider.list.builtIn && provider.view.document.name === reference.name);
	}

	return (
		providers.find(
			provider => provider.list.ref.rootID === model.list.ref.rootID && provider.view.document.name === reference.name
		) ?? providers.find(provider => provider.list.builtIn && provider.view.document.name === reference.name)
	);
}

function providerDefaultModelReference(provider: ModelProviderManagementItem): ModelProviderView['defaultModel'] {
	return provider.view.defaultModel ?? provider.view.settings.defaultModel ?? provider.view.document.defaultModel;
}

export function isProviderDefaultModel(provider: ModelProviderManagementItem, model: ModelManagementItem): boolean {
	const reference = providerDefaultModelReference(provider);

	if (!reference || reference.name !== model.view.document.name) {
		return false;
	}

	if (reference.scope === ModelLookupScope.Builtin) {
		return model.list.builtIn;
	}

	return !model.list.builtIn && model.list.ref.rootID === provider.list.ref.rootID;
}

function isModelProviderDefaultCandidate(provider: ModelProviderManagementItem, model: ModelManagementItem): boolean {
	if (
		provider.list.state !== ArtifactState.Available ||
		!provider.list.enabled ||
		model.list.state !== ArtifactState.Available ||
		!model.list.enabled
	) {
		return false;
	}

	// A built-in Provider can only store a built-in-scoped default Model.
	return !provider.list.builtIn || model.list.builtIn;
}

export function isModelRunnable(model: ModelManagementItem): boolean {
	return (
		model.list.state === ArtifactState.Available &&
		model.list.enabled &&
		model.provider !== undefined &&
		isModelProviderRunnable(model.provider)
	);
}

function mergeDefaults(base?: ModelDefaults, settings?: ModelDefaults): ModelDefaults | undefined {
	if (!base && !settings) {
		return undefined;
	}

	return {
		...base,
		...settings,
		...(base?.reasoning || settings?.reasoning
			? {
					reasoning: {
						...base?.reasoning,
						...settings?.reasoning,
					},
				}
			: {}),
		...(base?.cacheControl || settings?.cacheControl
			? {
					cacheControl: {
						...base?.cacheControl,
						...settings?.cacheControl,
					} as CacheControl,
				}
			: {}),
		...(base?.output || settings?.output
			? {
					output: {
						...base?.output,
						...settings?.output,
						...(base?.output?.format || settings?.output?.format
							? {
									format: {
										...base?.output?.format,
										...settings?.output?.format,
									},
								}
							: {}),
					},
				}
			: {}),
	};
}

function effectiveProviderDefaults(provider: ModelProviderManagementItem): ModelDefaults | undefined {
	return mergeDefaults(provider.view.document.defaults, provider.view.settings.defaults);
}

function effectiveModelDefaults(model: ModelManagementItem): ModelDefaults | undefined {
	return mergeDefaults(model.view.document.defaults, model.view.settings.defaults);
}

function effectiveProviderCapabilities(provider: ModelProviderManagementItem): ModelCapabilities | undefined {
	return mergeModelCapabilities(provider.view.document.capabilities, provider.view.settings.capabilities);
}

function effectiveModelCapabilities(model: ModelManagementItem): ModelCapabilities | undefined {
	return mergeModelCapabilities(model.view.document.capabilities, model.view.settings.capabilities);
}

function modelOptionFromItem(item: ModelManagementItem): UIModelOption {
	if (!item.provider) {
		throw new Error(`Model ${item.list.name} has no resolved provider.`);
	}

	const providerDefaults = effectiveProviderDefaults(item.provider);
	const modelDefaults = effectiveModelDefaults(item);

	return {
		...buildModelParamFromDefaults(item.view.document.providerModelID, providerDefaults, modelDefaults),
		model: item.list.ref,
		provider: item.provider.list.ref,
		logicalName: item.list.name,
		providerName: item.provider.view.document.name,
		providerAdapter: item.provider.view.document.adapter,
		providerSDKType: getProviderSDKType(item.provider.view.document.adapter),
		providerDisplayName: item.provider.list.displayName || item.provider.list.name,
		modelDisplayName: item.list.displayName || item.list.name,
		includePreviousMessages: 'all',
		capabilities: mergeModelCapabilities(
			effectiveProviderCapabilities(item.provider),
			effectiveModelCapabilities(item)
		),
	};
}

export class ModelManagementAPI {
	private readonly catalog = createSharedAsyncCatalog<ModelCatalog>(() => this.loadCatalog());

	constructor(
		// oxlint-disable-next-line typescript/parameter-properties
		private readonly store: IModelStoreAPI,
		// oxlint-disable-next-line typescript/parameter-properties
		private readonly aggregate: IModelAggregateAPI
	) {}

	invalidateCatalog(): void {
		this.catalog.invalidate();
	}

	getCatalog(): Promise<ModelCatalog> {
		return this.catalog.load(false);
	}

	async loadSnapshot(): Promise<ModelManagementSnapshot> {
		const [providerList, modelList, defaultProvider] = await Promise.all([
			this.store.listProviders(),
			this.store.listModels(),
			this.aggregate.getDefaultProvider().catch(() => undefined),
		]);

		const providers = await mapWithConcurrency(providerList, 8, async list => {
			const [view, apiKey] = await Promise.all([
				this.store.getProvider(list.ref),
				this.store.getProviderAPIKeyStatus(list.ref),
			]);

			return {
				list,
				view,
				apiKey,
			};
		});

		const modelsWithoutProvider = await mapWithConcurrency(modelList, 8, async list => ({
			list,
			view: await this.store.getModel(list.ref),
		}));

		const models = modelsWithoutProvider.map(model => ({
			...model,
			provider: resolveProvider(model, providers),
		}));

		return {
			providers,
			models,
			defaultProvider,
		};
	}

	listProviders(): Promise<ModelProviderListItem[]> {
		return this.store.listProviders();
	}

	listModels(): Promise<ModelListItem[]> {
		return this.store.listModels();
	}

	getProvider(ref: ArtifactRef): Promise<ModelProviderView> {
		return this.store.getProvider(ref);
	}

	getModel(ref: ArtifactRef): Promise<ModelView> {
		return this.store.getModel(ref);
	}

	getDefaultProvider(): Promise<ArtifactRef | undefined> {
		return this.aggregate.getDefaultProvider();
	}

	async setDefaultProvider(provider?: ArtifactRef): Promise<void> {
		if (provider) {
			await this.aggregate.setDefaultProvider(provider);
		} else {
			await this.aggregate.clearDefaultProvider();
		}
		this.invalidateCatalog();
	}

	async createProvider(request: ManagedProviderCreateRequest): Promise<ManagedProviderResult> {
		const result = await this.aggregate.createProvider(request);
		this.invalidateCatalog();
		return result;
	}

	async updateProvider(request: ManagedProviderReplaceRequest): Promise<ManagedProviderResult> {
		const result = await this.aggregate.updateProvider(request);
		this.invalidateCatalog();
		return result;
	}

	async deleteProvider(ref: ArtifactRef, expectedProviderRevision: number): Promise<void> {
		await this.aggregate.deleteProvider(ref, expectedProviderRevision);
		this.invalidateCatalog();
	}

	async setProviderEnabled(ref: ArtifactRef, expectedProviderRevision: number, enabled: boolean): Promise<void> {
		await this.aggregate.setProviderEnabled(ref, expectedProviderRevision, enabled);
		this.invalidateCatalog();
	}

	async setProviderAPIKey(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedAPIKeyRevision: number,
		apiKey: string
	): Promise<void> {
		await this.aggregate.setProviderAPIKey({
			provider: ref,
			expectedProviderRevision,
			expectedAPIKeyRevision,
			apiKey,
		});
		this.invalidateCatalog();
	}

	async clearProviderAPIKey(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedAPIKeyRevision: number
	): Promise<void> {
		await this.aggregate.clearProviderAPIKey(ref, expectedProviderRevision, expectedAPIKeyRevision);
		this.invalidateCatalog();
	}

	async setProviderDefaultModel(provider: ModelProviderManagementItem, model: ModelManagementItem): Promise<void> {
		if (!isModelProviderDefaultCandidate(provider, model)) {
			throw new Error('Enable compatible Provider and Model records before selecting a default model.');
		}

		const settings = provider.view.settings;
		await this.aggregate.saveProviderSettings({
			provider: provider.list.ref,
			expectedProviderRevision: provider.view.artifact.revision,
			expectedSettingsRevision: settings.revision,
			connection: settings.connection,
			defaults: settings.defaults,
			capabilities: settings.capabilities,
			adapterParameters: settings.adapterParameters,
			defaultModel: {
				name: model.view.document.name,
				...(model.list.builtIn ? { scope: ModelLookupScope.Builtin } : {}),
			},
		});
		this.invalidateCatalog();
	}

	async createModel(request: ManagedModelCreateRequest): Promise<ManagedModelResult> {
		const result = await this.store.createModel(request);
		this.invalidateCatalog();
		return result;
	}

	async updateModel(request: ManagedModelReplaceRequest): Promise<ManagedModelResult> {
		const result = await this.store.updateModel(request);
		this.invalidateCatalog();
		return result;
	}

	async deleteModel(ref: ArtifactRef, expectedRevision: number): Promise<void> {
		await this.store.deleteModel(ref, expectedRevision);
		this.invalidateCatalog();
	}

	async setModelEnabled(ref: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<void> {
		await this.store.setModelEnabled(ref, expectedRevision, enabled);
		this.invalidateCatalog();
	}

	/*
	 * This resolves UI display data only. It never decodes an opaque mapped
	 * target identifier and never executes a mapped target.
	 */
	async resolveMappedModelTarget(target: {
		type: string;
		name: string;
		builtin: boolean;
	}): Promise<ModelManagementItem> {
		if (target.type !== 'model') {
			throw new Error(`Target ${target.name} is not a model target.`);
		}

		const snapshot = await this.loadSnapshot();
		const matches = snapshot.models.filter(
			model => model.list.name === target.name && model.list.builtIn === target.builtin
		);

		if (matches.length === 0) {
			throw new Error(`Model target ${target.name} is unavailable.`);
		}
		if (matches.length > 1) {
			throw new Error(`Model target ${target.name} is ambiguous.`);
		}
		return matches[0];
	}

	private async loadCatalog(): Promise<ModelCatalog> {
		const snapshot = await this.loadSnapshot();
		const options = snapshot.models
			.filter(isModelRunnable)
			.map(m => {
				return modelOptionFromItem(m);
			})
			.toSorted((left, right) => {
				const providerComparison = left.providerDisplayName.localeCompare(right.providerDisplayName, undefined, {
					numeric: true,
					sensitivity: 'base',
				});

				if (providerComparison !== 0) {
					return providerComparison;
				}

				return left.modelDisplayName.localeCompare(right.modelDisplayName, undefined, {
					numeric: true,
					sensitivity: 'base',
				});
			});

		const fallback = options[0];
		if (!fallback) {
			const hasRunnableProvider = snapshot.providers.some(isModelProviderRunnable);
			throw new ModelCatalogUnavailableError(hasRunnableProvider ? 'no-runnable-model' : 'no-runnable-provider');
		}

		const selectedProvider = snapshot.defaultProvider
			? snapshot.providers.find(
					provider =>
						provider.list.ref.rootID === snapshot.defaultProvider?.rootID &&
						provider.list.ref.artifactID === snapshot.defaultProvider?.artifactID
				)
			: undefined;
		const selectedModel = selectedProvider
			? snapshot.models.find(model => isProviderDefaultModel(selectedProvider, model))
			: undefined;
		const selectedOption = selectedModel
			? options.find(option => modelRefEqual(option.model, selectedModel.list.ref))
			: undefined;

		return {
			options,
			defaultOption: selectedOption ?? fallback,
		};
	}
}
