import type { ArtifactRef } from '@/spec/artifact';
import type { StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ProviderSDKType } from '@/spec/inference';
import type { MCPConversationContext } from '@/spec/mcp';
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
	ModelRequestPatch,
	ModelView,
	UIModelOption,
} from '@/spec/model';
import type { ToolSelection } from '@/spec/tool';
import { ArtifactState } from '@/spec/artifact';
import { ProviderSDKType as ProviderSDKTypeValue } from '@/spec/inference';
import { ModelLookupScope } from '@/spec/model';

import { mapWithConcurrency } from '@/lib/async_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type { IModelAggregateAPI, IModelStoreAPI } from '@/apis/interface';

import { mergeModelCapabilities, sanitizeUIModelOptionByCapabilities } from '@/models/lib/capabilities';
import { modelRefEqual } from '@/models/lib/document';
import { buildModelParamFromDefaults } from '@/models/lib/model_defaults';

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
}

export interface ModelCatalog {
	options: UIModelOption[];
	defaultOption: UIModelOption;
}

function adapterSDKType(adapter: string): ProviderSDKType {
	switch (adapter) {
		case 'anthropic.messages':
			return ProviderSDKTypeValue.ProviderSDKTypeAnthropic;
		case 'openai.chatCompletions':
			return ProviderSDKTypeValue.ProviderSDKTypeOpenAIChatCompletions;
		case 'google.generateContent':
			return ProviderSDKTypeValue.ProviderSDKTypeGoogleGenerateContent;
		default:
			return ProviderSDKTypeValue.ProviderSDKTypeOpenAIResponses;
	}
}

function providerRequiresCredential(document: ModelProviderDocument): boolean {
	return (document.authentication?.mode ?? ('apiKeyHeader' as string)) !== 'none';
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

function modelIsRunnable(model: ModelManagementItem): boolean {
	if (!model.provider) {
		return false;
	}

	if (model.list.state !== ArtifactState.Available || !model.list.enabled) {
		return false;
	}

	if (model.provider.list.state !== ArtifactState.Available || !model.provider.list.enabled) {
		return false;
	}

	if (!providerRequiresCredential(model.provider.view.document)) {
		return true;
	}

	return model.provider.list.credentialConfigured === true;
}

function providerDefaultMatches(provider: ModelProviderManagementItem, model: ModelManagementItem): boolean {
	const reference = provider.view.document.defaultModel;
	if (!reference || reference.name !== model.view.document.name) {
		return false;
	}

	if (reference.scope === ModelLookupScope.Builtin) {
		return model.list.builtIn;
	}

	return model.list.ref.rootID === provider.list.ref.rootID;
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

	return sanitizeUIModelOptionByCapabilities({
		...sourceModelParam,

		model: item.list.ref,
		provider: item.provider.list.ref,

		logicalName: item.list.name,
		providerName: item.provider.view.document.name,
		providerAdapter: item.provider.view.document.adapter,
		providerSDKType: adapterSDKType(item.provider.view.document.adapter),
		providerDisplayName: item.provider.list.displayName || item.provider.list.name,
		modelDisplayName: item.list.displayName || item.list.name,
		includePreviousMessages: 'all',
		capabilities: mergeModelCapabilities(item.provider.view.document.capabilities, item.view.document.capabilities),
		sourceModelParam,
	});
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
		const [providerList, modelList] = await Promise.all([this.store.listModelProviders(), this.store.listModels()]);

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

	fetchCompletion(
		model: ArtifactRef,
		requestPatch: ModelRequestPatch | undefined,
		current: StoreConversationMessage,
		history?: StoreConversationMessage[],
		toolSelections?: ToolSelection[],
		mcpContext?: MCPConversationContext,
		skillSessionID?: string,
		requestID?: string,
		signal?: AbortSignal,
		onStreamTextData?: (text: string) => void,
		onStreamThinkingData?: (thinking: string) => void
	): Promise<CompletionResponseBody | undefined> {
		return this.aggregate.fetchCompletion(
			model,
			requestPatch,
			current,
			history,
			toolSelections,
			mcpContext,
			skillSessionID,
			requestID,
			signal,
			onStreamTextData,
			onStreamThinkingData
		);
	}

	cancelCompletion(requestID: string): Promise<void> {
		return this.aggregate.cancelCompletion(requestID);
	}

	private async loadCatalog(): Promise<ModelCatalog> {
		const snapshot = await this.loadSnapshot();

		const options = snapshot.models
			.filter(modelIsRunnable)
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

		const providerDefault = options.find(option => {
			const model = snapshot.models.find(candidate => modelRefEqual(candidate.list.ref, option.model));

			return model?.provider ? providerDefaultMatches(model.provider, model) : false;
		});

		const fallback = options[0];

		if (!fallback) {
			throw new Error('No enabled runnable Model is configured. Configure a provider credential and enable a model.');
		}

		return {
			options,
			defaultOption: providerDefault ?? fallback,
		};
	}
}
