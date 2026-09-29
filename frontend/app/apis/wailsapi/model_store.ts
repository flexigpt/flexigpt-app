import type { ArtifactRef, ArtifactRootID, StoreArtifact } from '@/spec/artifact';
import type {
	ManagedModelCreateRequest,
	ManagedModelReplaceRequest,
	ManagedModelResult,
	ManagedProviderCreateRequest,
	ManagedProviderReplaceRequest,
	ManagedProviderResult,
	ModelListItem,
	ModelProviderListItem,
	ModelProviderRuntimeOverlayView,
	ModelProviderView,
	ModelView,
} from '@/spec/model';

import type { IModelStoreAPI } from '@/apis/interface';
import {
	CreateManagedModel,
	CreateModelProvider,
	DeleteManagedModel,
	DeleteModelProvider,
	GetModel,
	GetModelProvider,
	ListModelProviders,
	ListModels,
	ReplaceManagedModel,
	ReplaceModelProvider,
	SetModelEnabled,
	SetModelProviderCredential,
	SetModelProviderEnabled,
} from '@/apis/wailsjs/go/main/ModelStoreWrapper';

export class WailsModelStoreAPI implements IModelStoreAPI {
	listModelProviders(rootID?: ArtifactRootID): Promise<ModelProviderListItem[]> {
		return ListModelProviders((rootID ?? '') as never) as unknown as Promise<ModelProviderListItem[]>;
	}

	listModels(rootID?: ArtifactRootID): Promise<ModelListItem[]> {
		return ListModels((rootID ?? '') as never) as unknown as Promise<ModelListItem[]>;
	}

	getModelProvider(ref: ArtifactRef): Promise<ModelProviderView> {
		return GetModelProvider(ref as never) as unknown as Promise<ModelProviderView>;
	}

	getModel(ref: ArtifactRef): Promise<ModelView> {
		return GetModel(ref as never) as unknown as Promise<ModelView>;
	}

	createModelProvider(request: ManagedProviderCreateRequest): Promise<ManagedProviderResult> {
		return CreateModelProvider(request as never) as unknown as Promise<ManagedProviderResult>;
	}

	replaceModelProvider(request: ManagedProviderReplaceRequest): Promise<ManagedProviderResult> {
		return ReplaceModelProvider(request as never) as unknown as Promise<ManagedProviderResult>;
	}

	deleteModelProvider(ref: ArtifactRef, expectedRevision: number): Promise<void> {
		return DeleteModelProvider(ref as never, expectedRevision);
	}

	createManagedModel(request: ManagedModelCreateRequest): Promise<ManagedModelResult> {
		return CreateManagedModel(request as never) as unknown as Promise<ManagedModelResult>;
	}

	replaceManagedModel(request: ManagedModelReplaceRequest): Promise<ManagedModelResult> {
		return ReplaceManagedModel(request as never) as unknown as Promise<ManagedModelResult>;
	}

	deleteManagedModel(ref: ArtifactRef, expectedRevision: number): Promise<void> {
		return DeleteManagedModel(ref as never, expectedRevision);
	}

	setModelProviderEnabled(ref: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return SetModelProviderEnabled(ref as never, expectedRevision, enabled) as unknown as Promise<StoreArtifact>;
	}

	setModelEnabled(ref: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return SetModelEnabled(ref as never, expectedRevision, enabled) as unknown as Promise<StoreArtifact>;
	}

	setModelProviderCredential(
		ref: ArtifactRef,
		expectedOverlayRevision: number,
		secret: string
	): Promise<ModelProviderRuntimeOverlayView> {
		return SetModelProviderCredential(
			ref as never,
			expectedOverlayRevision,
			secret
		) as unknown as Promise<ModelProviderRuntimeOverlayView>;
	}
}
