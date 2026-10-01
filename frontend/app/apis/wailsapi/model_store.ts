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
	listProviders(rootID?: ArtifactRootID): Promise<ModelProviderListItem[]> {
		return ListProviders((rootID ?? '') as never) as unknown as Promise<ModelProviderListItem[]>;
	}

	listModels(rootID?: ArtifactRootID): Promise<ModelListItem[]> {
		return ListModels((rootID ?? '') as never) as unknown as Promise<ModelListItem[]>;
	}

	getProvider(ref: ArtifactRef): Promise<ModelProviderView> {
		return GetProvider(ref as never) as unknown as Promise<ModelProviderView>;
	}

	getModel(ref: ArtifactRef): Promise<ModelView> {
		return GetModel(ref as never) as unknown as Promise<ModelView>;
	}

	getProviderAPIKeyStatus(ref: ArtifactRef): Promise<ProviderAPIKeyStatus> {
		return GetProviderAPIKeyStatus(ref as never) as unknown as Promise<ProviderAPIKeyStatus>;
	}

	createModel(request: ManagedModelCreateRequest): Promise<ManagedModelResult> {
		return CreateModel(request as never) as unknown as Promise<ManagedModelResult>;
	}

	updateModel(request: ManagedModelReplaceRequest): Promise<ManagedModelResult> {
		return UpdateModel(request as never) as unknown as Promise<ManagedModelResult>;
	}

	deleteModel(ref: ArtifactRef, expectedRevision: number): Promise<void> {
		return DeleteModel(ref as never, expectedRevision);
	}

	saveModelSettings(request: SaveModelSettingsRequest): Promise<ModelView> {
		return SaveModelSettings(request as never) as unknown as Promise<ModelView>;
	}

	resetModelSettings(
		ref: ArtifactRef,
		expectedModelRevision: number,
		expectedSettingsRevision: number
	): Promise<ModelView> {
		return ResetModelSettings(
			ref as never,
			expectedModelRevision,
			expectedSettingsRevision
		) as unknown as Promise<ModelView>;
	}

	setModelEnabled(ref: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return SetModelEnabled(ref as never, expectedRevision, enabled) as unknown as Promise<StoreArtifact>;
	}
}
