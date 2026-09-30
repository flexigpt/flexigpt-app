// oxlint-disable max-classes-per-file
import type { ArtifactRef } from '@/spec/artifact';
import type {
	ManagedModelCreateRequest,
	ManagedModelReplaceRequest,
	ManagedModelResult,
	ManagedProviderCreateRequest,
	ManagedProviderReplaceRequest,
	ManagedProviderResult,
	ModelListItem,
	ModelProviderDocument,
	ModelProviderListItem,
	ModelProviderView,
	ModelView,
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
				? 'No enabled provider with configured credentials is available.'
				: 'No enabled model is available for the configured providers.'
		);
		this.name = 'ModelCatalogUnavailableError';
	}
}

export function isModelCatalogUnavailableError(error: unknown): error is ModelCatalogUnavailableError {
	return error instanceof ModelCatalogUnavailableError;
}

function providerRequiresCredential(document: ModelProviderDocument): boolean {
	return (document.authentication?.mode ?? ('apiKeyHeader' as string)) !== 'none';
}

export function isModelProviderRunnable(provider: ModelProviderManagementItem): boolean {
	if (provider.list.state !== ArtifactState.Available || !provider.list.enabled) {
		return false;
	}

	if (!providerRequiresCredential(provider.view.document)) {
		return true;
	}

	return provider.list.credentialConfigured === true;
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

export function isModelRunnable(model: ModelManagementItem): boolean {
	if (!model.provider) {
		return false;
	}

	if (model.list.state !== ArtifactState.Available || !model.list.enabled) {
		return false;
	}

	return isModelProviderRunnable(model.provider);
}

function optionUsesProvider(option: UIModelOption, provider: ArtifactRef): boolean {
	return option.provider?.rootID === provider.rootID && option.provider?.artifactID === provider.artifactID;
}

function modelOptionFromItem(item: ModelManagementItem): UIModelOption {
	if (!item.provider) {
		throw new Error(`Model ${item.list.name} has no resolved provider.`);
	}

	const sourceModelParam = buildModelParamFromDefaults(
		item.view.document.providerModelID,
		item.provider.view.document.defaults,
		item.view.document.defaults
	);

	return {
		...sourceModelParam,

		model: item.list.ref,
		provider: item.provider.list.ref,

		logicalName: item.list.name,
		providerName: item.provider.view.document.name,
		providerAdapter: item.provider.view.document.adapter,
		providerSDKType: getProviderSDKType(item.provider.view.document.adapter),
		providerDisplayName: item.provider.list.displayName || item.provider.list.name,
		modelDisplayName: item.list.displayName || item.list.name,
		includePreviousMessages: 'all',
		capabilities: mergeModelCapabilities(item.provider.view.document.capabilities, item.view.document.capabilities),
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
			this.store.listModelProviders(),
			this.store.listModels(),
			this.aggregate.getDefaultModelProvider().catch(() => undefined),
		]);

		const providers = await mapWithConcurrency(providerList, 8, async list => ({
			list,
			view: await this.store.getModelProvider(list.ref),
		}));

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
		return this.store.listModelProviders();
	}

	listModels(): Promise<ModelListItem[]> {
		return this.store.listModels();
	}

	getProvider(ref: ArtifactRef): Promise<ModelProviderView> {
		return this.store.getModelProvider(ref);
	}

	getModel(ref: ArtifactRef): Promise<ModelView> {
		return this.store.getModel(ref);
	}

	getDefaultModelProvider(): Promise<ArtifactRef | undefined> {
		return this.aggregate.getDefaultModelProvider();
	}

	async setDefaultModelProvider(provider?: ArtifactRef): Promise<void> {
		await this.aggregate.setDefaultModelProvider(provider);
		this.invalidateCatalog();
	}

	getModelProviderDefaultModel(provider: ArtifactRef): Promise<ArtifactRef> {
		return this.aggregate.getModelProviderDefaultModel(provider);
	}

	private async tryGetModelProviderDefaultModel(provider: ArtifactRef): Promise<ArtifactRef | undefined> {
		try {
			return await this.aggregate.getModelProviderDefaultModel(provider);
		} catch {
			return undefined;
		}
	}

	async createProvider(request: ManagedProviderCreateRequest): Promise<ManagedProviderResult> {
		const result = await this.store.createModelProvider(request);
		this.invalidateCatalog();
		return result;
	}

	async replaceProvider(request: ManagedProviderReplaceRequest): Promise<ManagedProviderResult> {
		const result = await this.store.replaceModelProvider(request);
		this.invalidateCatalog();
		return result;
	}

	async deleteProvider(ref: ArtifactRef, expectedRevision: number): Promise<void> {
		await this.store.deleteModelProvider(ref, expectedRevision);
		this.invalidateCatalog();
	}

	async setProviderEnabled(ref: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<void> {
		await this.store.setModelProviderEnabled(ref, expectedRevision, enabled);
		this.invalidateCatalog();
	}

	async setProviderCredential(ref: ArtifactRef, expectedOverlayRevision: number, secret: string): Promise<void> {
		await this.store.setModelProviderCredential(ref, expectedOverlayRevision, secret);
		this.invalidateCatalog();
	}

	async setProviderDefaultModel(provider: ModelProviderManagementItem, model: ModelManagementItem): Promise<void> {
		if (provider.list.builtIn) {
			throw new Error(
				'Built-in provider defaults are source-owned. Select the generated default model or create a mutable provider.'
			);
		}

		const document: ModelProviderDocument = {
			...structuredClone(provider.view.document),
			defaultModel: {
				name: model.view.document.name,
				...(model.list.builtIn ? { scope: ModelLookupScope.Builtin } : {}),
			},
		};

		await this.replaceProvider({
			provider: provider.list.ref,
			expectedArtifactRevision: provider.list.revision,
			document,
			enabled: provider.list.enabled,
		});
	}

	async createModel(request: ManagedModelCreateRequest): Promise<ManagedModelResult> {
		const result = await this.store.createManagedModel(request);
		this.invalidateCatalog();
		return result;
	}

	async replaceModel(request: ManagedModelReplaceRequest): Promise<ManagedModelResult> {
		const result = await this.store.replaceManagedModel(request);
		this.invalidateCatalog();
		return result;
	}

	async deleteModel(ref: ArtifactRef, expectedRevision: number): Promise<void> {
		await this.store.deleteManagedModel(ref, expectedRevision);
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
		const defaultProvider = snapshot.defaultProvider;

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

		const defaultProviderOptions = defaultProvider
			? options.filter(option => optionUsesProvider(option, defaultProvider))
			: [];

		// This performs Model resolution only when the selected Provider has at
		// least one runnable UI option. The default-provider lookup itself is
		// metadata-only and was already started in parallel with the snapshot.
		const defaultProviderModel =
			defaultProvider && defaultProviderOptions.length > 0
				? await this.tryGetModelProviderDefaultModel(defaultProvider)
				: undefined;

		const defaultProviderOption = defaultProviderModel
			? defaultProviderOptions.find(
					option => option.model !== undefined && modelRefEqual(option.model, defaultProviderModel)
				)
			: undefined;

		return {
			options,
			defaultOption: defaultProviderOption ?? defaultProviderOptions[0] ?? fallback,
		};
	}
}
