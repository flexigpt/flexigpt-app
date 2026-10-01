import type { ArtifactRef, ArtifactRootID, StoreArtifact } from '@/spec/artifact';
import type {
	ManagedModelCreateRequest,
	ManagedModelReplaceRequest,
	ManagedModelResult,
	ModelListItem,
	ModelProviderListItem,
	ModelProviderView,
	ModelView,
	ProviderAPIKeyStatus,
	SaveModelSettingsRequest,
} from '@/spec/model';

import type { IModelStoreAPI } from '@/apis/interface';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	CreateModel,
	DeleteModel,
	GetModel,
	GetProvider,
	GetProviderAPIKeyStatus,
	ListModels,
	ListProviders,
	ResetModelSettings,
	SaveModelSettings,
	SetModelEnabled,
	UpdateModel,
} from '@/apis/wailsjs/go/main/ModelStoreWrapper';

export class WailsModelStoreAPI implements IModelStoreAPI {
	async listProviders(rootID?: ArtifactRootID): Promise<ModelProviderListItem[]> {
		return wailsObjectArrayOrEmpty<ModelProviderListItem>(
			await ListProviders((rootID ?? '') as Parameters<typeof ListProviders>[0]),
			'ListProviders'
		);
	}

	async listModels(rootID?: ArtifactRootID): Promise<ModelListItem[]> {
		return wailsObjectArrayOrEmpty<ModelListItem>(
			await ListModels((rootID ?? '') as Parameters<typeof ListModels>[0]),
			'ListModels'
		);
	}

	async getProvider(ref: ArtifactRef): Promise<ModelProviderView> {
		return requiredObject<ModelProviderView>(
			await GetProvider(ref as Parameters<typeof GetProvider>[0]),
			'GetProvider'
		);
	}

	async getModel(ref: ArtifactRef): Promise<ModelView> {
		return requiredObject<ModelView>(await GetModel(ref as Parameters<typeof GetModel>[0]), 'GetModel');
	}

	async getProviderAPIKeyStatus(ref: ArtifactRef): Promise<ProviderAPIKeyStatus> {
		return requiredObject<ProviderAPIKeyStatus>(
			await GetProviderAPIKeyStatus(ref as Parameters<typeof GetProviderAPIKeyStatus>[0]),
			'GetProviderAPIKeyStatus'
		);
	}

	async createModel(request: ManagedModelCreateRequest): Promise<ManagedModelResult> {
		return requiredObject<ManagedModelResult>(
			await CreateModel(request as Parameters<typeof CreateModel>[0]),
			'CreateModel'
		);
	}

	async updateModel(request: ManagedModelReplaceRequest): Promise<ManagedModelResult> {
		return requiredObject<ManagedModelResult>(
			await UpdateModel(request as Parameters<typeof UpdateModel>[0]),
			'UpdateModel'
		);
	}

	async deleteModel(ref: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteModel(ref as Parameters<typeof DeleteModel>[0], expectedRevision);
	}

	async saveModelSettings(request: SaveModelSettingsRequest): Promise<ModelView> {
		return requiredObject<ModelView>(
			await SaveModelSettings(request as Parameters<typeof SaveModelSettings>[0]),
			'SaveModelSettings'
		);
	}

	async resetModelSettings(
		ref: ArtifactRef,
		expectedModelRevision: number,
		expectedSettingsRevision: number
	): Promise<ModelView> {
		return requiredObject<ModelView>(
			await ResetModelSettings(
				ref as Parameters<typeof ResetModelSettings>[0],
				expectedModelRevision,
				expectedSettingsRevision
			),
			'ResetModelSettings'
		);
	}

	async setModelEnabled(ref: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await SetModelEnabled(ref as Parameters<typeof SetModelEnabled>[0], expectedRevision, enabled),
			'SetModelEnabled'
		);
	}
}
