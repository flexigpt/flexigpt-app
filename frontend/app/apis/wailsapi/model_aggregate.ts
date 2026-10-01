import type { ArtifactRef, StoreArtifact } from '@/spec/artifact';
import type {
	ManagedProviderCreateRequest,
	ManagedProviderReplaceRequest,
	ManagedProviderResult,
	ModelProviderView,
	ProviderAPIKeyStatus,
	SaveProviderSettingsRequest,
	SetProviderAPIKeyRequest,
} from '@/spec/model';

import type { IModelAggregateAPI } from '@/apis/interface';
import { requiredObject, requireWailsString } from '@/apis/wailsapi/transport';
import {
	ClearDefaultProvider,
	ClearProviderAPIKey,
	CreateProvider,
	DeleteProvider,
	GetDefaultProvider,
	ResetProviderSettings,
	SaveProviderSettings,
	SetDefaultProvider,
	SetProviderAPIKey,
	SetProviderEnabled,
	UpdateProvider,
} from '@/apis/wailsjs/go/main/ModelAggregateWrapper';

function artifactRefFromWails(value: unknown, operation: string): ArtifactRef {
	const ref = requiredObject<Record<string, unknown>>(value, operation);

	return {
		rootID: requireWailsString(ref.rootID, `${operation}.rootID`),
		artifactID: requireWailsString(ref.artifactID, `${operation}.artifactID`),
	};
}

function optionalArtifactRefFromWails(value: unknown, operation: string): ArtifactRef | undefined {
	if (value === null || value === undefined) {
		return undefined;
	}
	return artifactRefFromWails(value, operation);
}

export class WailsModelAggregateAPI implements IModelAggregateAPI {
	async getDefaultProvider(): Promise<ArtifactRef | undefined> {
		const result = await GetDefaultProvider();
		return optionalArtifactRefFromWails(result, 'GetDefaultProvider');
	}

	setDefaultProvider(provider: ArtifactRef): Promise<void> {
		return SetDefaultProvider(provider as never);
	}

	clearDefaultProvider(): Promise<void> {
		return ClearDefaultProvider();
	}

	saveProviderSettings(request: SaveProviderSettingsRequest): Promise<ModelProviderView> {
		return SaveProviderSettings(request as never) as unknown as Promise<ModelProviderView>;
	}

	resetProviderSettings(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedSettingsRevision: number
	): Promise<ModelProviderView> {
		return ResetProviderSettings(
			ref as never,
			expectedProviderRevision,
			expectedSettingsRevision
		) as unknown as Promise<ModelProviderView>;
	}

	setProviderAPIKey(request: SetProviderAPIKeyRequest): Promise<ProviderAPIKeyStatus> {
		return SetProviderAPIKey(request as never) as unknown as Promise<ProviderAPIKeyStatus>;
	}

	clearProviderAPIKey(
		ref: ArtifactRef,
		expectedProviderRevision: number,
		expectedAPIKeyRevision: number
	): Promise<ProviderAPIKeyStatus> {
		return ClearProviderAPIKey(
			ref as never,
			expectedProviderRevision,
			expectedAPIKeyRevision
		) as unknown as Promise<ProviderAPIKeyStatus>;
	}

	createProvider(request: ManagedProviderCreateRequest): Promise<ManagedProviderResult> {
		return CreateProvider(request as never) as unknown as Promise<ManagedProviderResult>;
	}

	updateProvider(request: ManagedProviderReplaceRequest): Promise<ManagedProviderResult> {
		return UpdateProvider(request as never) as unknown as Promise<ManagedProviderResult>;
	}

	deleteProvider(ref: ArtifactRef, expectedProviderRevision: number): Promise<void> {
		return DeleteProvider(ref as never, expectedProviderRevision);
	}

	setProviderEnabled(ref: ArtifactRef, expectedProviderRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return SetProviderEnabled(ref as never, expectedProviderRevision, enabled) as unknown as Promise<StoreArtifact>;
	}
}
