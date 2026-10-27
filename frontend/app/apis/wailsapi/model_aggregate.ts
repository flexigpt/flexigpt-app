import type { ArtifactRef, StoreArtifact } from '@/spec/artifact';
import type {
	DefaultModelSelection,
	ManagedProviderCreateRequest,
	ManagedProviderReplaceRequest,
	ManagedProviderResult,
	ModelProviderView,
	ProviderAPIKeyStatus,
	SaveProviderSettingsRequest,
	SetProviderAPIKeyRequest,
} from '@/spec/model';

import type { IModelAggregateAPI } from '@/apis/interface';
import {
	managedProviderCreateRequestToWails,
	managedProviderReplaceRequestToWails,
} from '@/apis/wailsapi/model_request_projection';
import { optionalWailsBody, requiredObject } from '@/apis/wailsapi/transport';
import {
	ClearDefaultProvider,
	ClearProviderAPIKey,
	CreateProvider,
	DeleteProvider,
	GetDefaultModel,
	GetDefaultProvider,
	ResetProviderSettings,
	SaveProviderSettings,
	SetDefaultProvider,
	SetProviderAPIKey,
	SetProviderEnabled,
	UpdateProvider,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/modelaggregatewrapper';

export class WailsModelAggregateAPI implements IModelAggregateAPI {
	async getDefaultProvider(): Promise<ArtifactRef | undefined> {
		return optionalWailsBody<ArtifactRef>(await GetDefaultProvider(), 'GetDefaultProvider');
	}

	async getDefaultModel(): Promise<DefaultModelSelection> {
		return requiredObject<DefaultModelSelection>(await GetDefaultModel(), 'GetDefaultModel');
	}

	async setDefaultProvider(provider: ArtifactRef): Promise<void> {
		await SetDefaultProvider(provider as Parameters<typeof SetDefaultProvider>[0]);
	}

	async clearDefaultProvider(): Promise<void> {
		await ClearDefaultProvider();
	}

	async saveProviderSettings(request: SaveProviderSettingsRequest): Promise<ModelProviderView> {
		return requiredObject<ModelProviderView>(
			await SaveProviderSettings(request as Parameters<typeof SaveProviderSettings>[0]),
			'SaveProviderSettings'
		);
	}

	async resetProviderSettings(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedSettingsRevision: number
	): Promise<ModelProviderView> {
		return requiredObject<ModelProviderView>(
			await ResetProviderSettings(
				ref as Parameters<typeof ResetProviderSettings>[0],
				expectedProviderRevision,
				expectedSettingsRevision
			),
			'ResetProviderSettings'
		);
	}

	async setProviderAPIKey(request: SetProviderAPIKeyRequest): Promise<ProviderAPIKeyStatus> {
		return requiredObject<ProviderAPIKeyStatus>(
			await SetProviderAPIKey(request as Parameters<typeof SetProviderAPIKey>[0]),
			'SetProviderAPIKey'
		);
	}

	async clearProviderAPIKey(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedAPIKeyRevision: number
	): Promise<ProviderAPIKeyStatus> {
		return requiredObject<ProviderAPIKeyStatus>(
			await ClearProviderAPIKey(
				ref as Parameters<typeof ClearProviderAPIKey>[0],
				expectedProviderRevision,
				expectedAPIKeyRevision
			),
			'ClearProviderAPIKey'
		);
	}

	async createProvider(request: ManagedProviderCreateRequest): Promise<ManagedProviderResult> {
		return requiredObject<ManagedProviderResult>(
			await CreateProvider(managedProviderCreateRequestToWails(request)),
			'CreateProvider'
		);
	}

	async updateProvider(request: ManagedProviderReplaceRequest): Promise<ManagedProviderResult> {
		return requiredObject<ManagedProviderResult>(
			await UpdateProvider(managedProviderReplaceRequestToWails(request)),
			'UpdateProvider'
		);
	}

	async deleteProvider(ref: ArtifactRef, expectedProviderRevision: number): Promise<void> {
		await DeleteProvider(ref as Parameters<typeof DeleteProvider>[0], expectedProviderRevision);
	}

	async setProviderEnabled(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		enabled: boolean
	): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await SetProviderEnabled(ref as Parameters<typeof SetProviderEnabled>[0], expectedProviderRevision, enabled),
			'SetProviderEnabled'
		);
	}
}
