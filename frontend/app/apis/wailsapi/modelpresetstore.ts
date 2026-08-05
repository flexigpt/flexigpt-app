import type { ProviderName } from '@/spec/inference';
import { ProviderSDKType } from '@/spec/inference';
import type {
	ModelPresetID,
	PatchModelPresetPayload,
	PatchProviderPresetPayload,
	PostModelPresetPayload,
	ProviderPreset,
} from '@/spec/modelpreset';

import type { IModelPresetStoreAPI } from '@/apis/interface';
import { enumFromWails, omitUndefined, requireWailsBody } from '@/apis/wailsapi/transport';
import {
	DeleteModelPreset,
	GetDefaultProvider,
	ListProviderPresets,
	PatchDefaultProvider,
	PatchModelPreset,
	PatchProviderPreset,
	PostModelPreset,
} from '@/apis/wailsjs/go/main/ModelPresetStoreWrapper';
import type { spec } from '@/apis/wailsjs/go/models';

function normalizeProviderPreset(provider: ProviderPreset): ProviderPreset {
	return {
		...provider,
		sdkType: enumFromWails(provider.sdkType, ProviderSDKType, 'providerPreset.sdkType'),
		defaultHeaders: provider.defaultHeaders ?? {},
		modelPresets: provider.modelPresets ?? {},
	};
}

export class WailsModelPresetStoreAPI implements IModelPresetStoreAPI {
	async getDefaultProvider(): Promise<ProviderName> {
		const resp = await GetDefaultProvider({});
		const body = requireWailsBody(resp.Body, 'GetDefaultProvider');
		return body.defaultProvider;
	}

	async patchDefaultProvider(providerName: ProviderName): Promise<void> {
		if (!providerName) {
			throw new Error('Missing providerName or payload');
		}
		const r = {
			Body: { defaultProvider: providerName } as spec.PatchDefaultProviderRequestBody,
		};
		await PatchDefaultProvider(r as spec.PatchDefaultProviderRequest);
	}

	async patchProviderPreset(providerName: ProviderName, payload: PatchProviderPresetPayload): Promise<void> {
		if (!providerName) {
			throw new Error('Missing providerName');
		}
		const body = omitUndefined(payload as Record<string, unknown>);
		if (Object.keys(body).length === 0) {
			throw new Error('Provider patch payload is empty');
		}
		const r = {
			ProviderName: providerName,
			Body: body as unknown as spec.PatchProviderPresetRequestBody,
		};
		await PatchProviderPreset(r as spec.PatchProviderPresetRequest);
	}

	async postModelPreset(
		providerName: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PostModelPresetPayload
	): Promise<void> {
		if (!providerName || !modelPresetID) {
			throw new Error('Missing arguments');
		}
		const r = {
			ProviderName: providerName,
			ModelPresetID: modelPresetID,
			Body: payload,
		};
		await PostModelPreset(r as spec.PostModelPresetRequest);
	}

	async patchModelPreset(
		providerName: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PatchModelPresetPayload
	): Promise<void> {
		if (!providerName || !modelPresetID) {
			throw new Error('Missing arguments');
		}
		const body = omitUndefined(payload as Record<string, unknown>);
		if (Object.keys(body).length === 0) {
			throw new Error('Model patch payload is empty');
		}
		const r = {
			ProviderName: providerName,
			ModelPresetID: modelPresetID,
			Body: body as unknown as spec.PatchModelPresetRequestBody,
		} as spec.PatchModelPresetRequest;
		await PatchModelPreset(r);
	}

	async deleteModelPreset(providerName: ProviderName, modelPresetID: ModelPresetID): Promise<void> {
		if (!providerName || !modelPresetID) {
			throw new Error('Missing arguments');
		}
		const r = {
			ProviderName: providerName,
			ModelPresetID: modelPresetID,
		};
		await DeleteModelPreset(r as spec.DeleteModelPresetRequest);
	}

	async listProviderPresets(
		names?: ProviderName[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ providers: ProviderPreset[]; nextPageToken?: string }> {
		const r: spec.ListProviderPresetsRequest = {
			Names: names ?? [],
			IncludeDisabled: includeDisabled ?? false,
			PageSize: pageSize ?? 256,
			PageToken: pageToken ?? '',
		};
		const resp = await ListProviderPresets(r);
		const body = requireWailsBody(resp.Body, 'ListProviderPresets');
		const providers = (body.providers ?? []).map(p => normalizeProviderPreset(p as ProviderPreset));
		return {
			providers: providers,
			nextPageToken: body.nextPageToken || undefined,
		};
	}
}
