import type { ArtifactRef } from '@/spec/artifact';
import type {
	ManagedModelCreateRequest,
	ManagedModelReplaceRequest,
	ManagedProviderCreateRequest,
	ManagedProviderReplaceRequest,
	ModelArtifactNameReference,
	ModelAuthentication,
	ModelDocument,
	ModelProviderDocument,
} from '@/spec/model';
import { ModelArtifactType, ModelAuthenticationMode, ModelLookupScope } from '@/spec/model';

import type { ArtifactRef as WailsArtifactRef } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model/models';
import type {
	ArtifactNameReference as WailsArtifactNameReference,
	Authentication as WailsAuthentication,
	ModelDocument as WailsModelDocument,
	ProviderDocument as WailsProviderDocument,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain/models';
import type {
	ManagedModelCreateRequest as WailsManagedModelCreateRequest,
	ManagedModelReplaceRequest as WailsManagedModelReplaceRequest,
	ManagedProviderCreateRequest as WailsManagedProviderCreateRequest,
	ManagedProviderReplaceRequest as WailsManagedProviderReplaceRequest,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/models';
import { enumToWails } from '@/apis/wailsapi/transport';

function artifactRefToWails(ref: ArtifactRef): WailsArtifactRef {
	return {
		rootID: ref.rootID,
		artifactID: ref.artifactID,
	};
}

function artifactNameReferenceToWails(
	reference: ModelArtifactNameReference,
	field: string
): WailsArtifactNameReference {
	return {
		name: reference.name,
		...(reference.scope === undefined
			? {}
			: {
					scope: enumToWails<ModelLookupScope, NonNullable<WailsArtifactNameReference['scope']>>(
						reference.scope,
						ModelLookupScope,
						`${field}.scope`
					),
				}),
	};
}

function authenticationToWails(
	authentication: ModelAuthentication | undefined,
	field: string
): WailsAuthentication | undefined {
	if (authentication === undefined) {
		return undefined;
	}

	const { mode, ...rest } = authentication;

	return {
		...rest,
		mode: enumToWails<ModelAuthenticationMode, WailsAuthentication['mode']>(
			mode,
			ModelAuthenticationMode,
			`${field}.mode`
		),
	};
}

function modelProviderDocumentToWails(document: ModelProviderDocument): WailsProviderDocument {
	if (document.type !== ModelArtifactType.Provider) {
		throw new Error('Provider document type must be "model.provider".');
	}

	const { type, authentication, defaultModel, ...rest } = document;

	return {
		...rest,
		type: enumToWails<ModelArtifactType, WailsProviderDocument['type']>(
			type,
			ModelArtifactType,
			'ProviderDocument.type'
		),
		...(authentication === undefined
			? {}
			: {
					authentication: authenticationToWails(authentication, 'ProviderDocument.authentication'),
				}),
		...(defaultModel === undefined
			? {}
			: {
					defaultModel: artifactNameReferenceToWails(defaultModel, 'ProviderDocument.defaultModel'),
				}),
	};
}

function modelDocumentToWails(document: ModelDocument): WailsModelDocument {
	if (document.type !== ModelArtifactType.Model) {
		throw new Error('Model document type must be "model".');
	}

	const { type, provider, ...rest } = document;

	return {
		...rest,
		type: enumToWails<ModelArtifactType, WailsModelDocument['type']>(type, ModelArtifactType, 'ModelDocument.type'),
		provider: artifactNameReferenceToWails(provider, 'ModelDocument.provider'),
	};
}

export function managedProviderCreateRequestToWails(
	request: ManagedProviderCreateRequest
): WailsManagedProviderCreateRequest {
	return {
		rootID: request.rootID,
		document: modelProviderDocumentToWails(request.document),
		enabled: request.enabled,
	};
}

export function managedProviderReplaceRequestToWails(
	request: ManagedProviderReplaceRequest
): WailsManagedProviderReplaceRequest {
	return {
		provider: artifactRefToWails(request.provider),
		expectedArtifactRevision: request.expectedArtifactRevision,
		document: modelProviderDocumentToWails(request.document),
		enabled: request.enabled,
	};
}

export function managedModelCreateRequestToWails(request: ManagedModelCreateRequest): WailsManagedModelCreateRequest {
	return {
		rootID: request.rootID,
		document: modelDocumentToWails(request.document),
		enabled: request.enabled,
	};
}

export function managedModelReplaceRequestToWails(
	request: ManagedModelReplaceRequest
): WailsManagedModelReplaceRequest {
	return {
		model: artifactRefToWails(request.model),
		expectedArtifactRevision: request.expectedArtifactRevision,
		document: modelDocumentToWails(request.document),
		enabled: request.enabled,
	};
}
