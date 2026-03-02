import type { ProviderName } from '@/spec/inference';
import type {
	ModelCapabilitiesOverride,
	ModelPresetID,
	ProviderPreset,
	PutModelPresetPayload,
} from '@/spec/modelpreset';

import type { IModelPresetStoreAPI } from '@/apis/interface';
import {
	DeleteModelPreset,
	GetDefaultProvider,
	ListProviderPresets,
	PatchDefaultProvider,
	PatchModelPreset,
	PatchProviderPreset,
	PutModelPreset,
} from '@/apis/wailsjs/go/main/ModelPresetStoreWrapper';
import type { spec } from '@/apis/wailsjs/go/models';

/**
 * @public
 */
export class WailsModelPresetStoreAPI implements IModelPresetStoreAPI {
	async getDefaultProvider(): Promise<ProviderName> {
		const resp = await GetDefaultProvider({});
		return resp.Body?.DefaultProvider ?? '';
	}

	async patchDefaultProvider(providerName: ProviderName): Promise<void> {
		if (!providerName) throw new Error('Missing providerName or payload');
		const r = {
			Body: { defaultProvider: providerName } as spec.PatchDefaultProviderRequestBody,
		};
		await PatchDefaultProvider(r as spec.PatchDefaultProviderRequest);
	}

	async patchProviderPreset(
		providerName: ProviderName,
		isEnabled?: boolean,
		defaultModelPresetID?: ModelPresetID
	): Promise<void> {
		if (!providerName) throw new Error('Missing providerName');
		const r = {
			ProviderName: providerName,
			Body: {
				isEnabled: isEnabled ?? undefined,
				defaultModelPresetID: defaultModelPresetID ?? undefined,
			},
		};
		await PatchProviderPreset(r as spec.PatchProviderPresetRequest);
	}

	async putModelPreset(
		providerName: ProviderName,
		modelPresetID: ModelPresetID,
		payload: PutModelPresetPayload
	): Promise<void> {
		if (!providerName || !modelPresetID) throw new Error('Missing arguments');
		const r = {
			ProviderName: providerName,
			ModelPresetID: modelPresetID,
			Body: payload,
		};
		await PutModelPreset(r as spec.PutModelPresetRequest);
	}

	async patchModelPreset(
		providerName: ProviderName,
		modelPresetID: ModelPresetID,
		isEnabled: boolean,
		capabilitiesOverride?: ModelCapabilitiesOverride,
		clearCapabilitiesOverride?: boolean
	): Promise<void> {
		if (!providerName || !modelPresetID) throw new Error('Missing arguments');
		const r = {
			ProviderName: providerName,
			ModelPresetID: modelPresetID,
			Body: {
				isEnabled: isEnabled,
				capabilitiesOverride: capabilitiesOverride,
				clearCapabilitiesOverride: clearCapabilitiesOverride,
			} as spec.PatchModelPresetRequestBody,
		} as spec.PatchModelPresetRequest;
		await PatchModelPreset(r);
	}

	async deleteModelPreset(providerName: ProviderName, modelPresetID: ModelPresetID): Promise<void> {
		if (!providerName || !modelPresetID) throw new Error('Missing arguments');
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
		return {
			providers: (resp.Body?.providers ?? []) as ProviderPreset[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}
}
