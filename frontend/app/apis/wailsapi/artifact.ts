import type {
	ArtifactRoot,
	ArtifactRootID,
	ArtifactSourceDraft,
	ArtifactSourceID,
	ArtifactSourceKind,
	ArtifactSourceSummary,
	CreateArtifactRootBody,
	ManagedSourcePackageResult,
	ManagedSourceState,
	PublishManagedSourcePackageBody,
	PurgeArtifactRootResult,
	PurgeArtifactSourceResult,
	RemoveManagedSourcePackageBody,
	UpdateArtifactRootBody,
	UpdateArtifactSourceBody,
} from '@/spec/artifact';

import type { IArtifactStoreAPI } from '@/apis/interface';
import {
	rawJSONObjectToWails,
	requireNonBlankString,
	requireWailsArray,
	requireWailsBody,
	requireWailsString,
} from '@/apis/wailsapi/transport';
import {
	CreateArtifactRoot,
	CreateArtifactSource,
	GetArtifactRoot,
	GetArtifactSource,
	GetManagedSourceState,
	ListArtifactRoots,
	ListArtifactSourceKinds,
	ListArtifactSources,
	PublishManagedSourcePackage,
	PurgeArtifactRoot,
	PurgeArtifactSource,
	RemoveManagedSourcePackage,
	RetireArtifactRoot,
	RetireArtifactSource,
	UpdateArtifactRoot,
	UpdateArtifactSource,
} from '@/apis/wailsjs/go/main/ArtifactStoreWrapper';
import type { artifactstore } from '@/apis/wailsjs/go/models';

function updateSourceBodyToWails(body: UpdateArtifactSourceBody): unknown {
	return {
		expectedRevision: body.expectedRevision,
		displayName: body.displayName,
		enabled: body.enabled,
		...(body.config === undefined ? {} : { config: rawJSONObjectToWails(body.config, 'artifact source config') }),
	};
}

function createSourceBodyToWails(body: ArtifactSourceDraft): unknown {
	return {
		id: body.id,
		storageKey: requireNonBlankString(body.storageKey, 'artifact source storage key'),
		kind: body.kind,
		displayName: body.displayName,
		enabled: body.enabled,
		config: rawJSONObjectToWails(body.config, 'artifact source config'),
	};
}

export class WailsArtifactStoreAPI implements IArtifactStoreAPI {
	async createArtifactRoot(body: CreateArtifactRootBody): Promise<ArtifactRoot> {
		const response = await CreateArtifactRoot({
			Body: body,
		} as Parameters<typeof CreateArtifactRoot>[0]);

		return requireWailsBody(response.Body, 'CreateArtifactRoot');
	}

	async getArtifactRoot(rootID: ArtifactRootID): Promise<ArtifactRoot> {
		const response = await GetArtifactRoot({
			RootID: rootID,
		} as Parameters<typeof GetArtifactRoot>[0]);

		return requireWailsBody(response.Body, 'GetArtifactRoot');
	}

	async listArtifactRoots(): Promise<ArtifactRoot[]> {
		const response = await ListArtifactRoots({} as Parameters<typeof ListArtifactRoots>[0]);
		const body = requireWailsBody(response.Body, 'ListArtifactRoots');

		return body.roots;
	}

	async updateArtifactRoot(rootID: ArtifactRootID, body: UpdateArtifactRootBody): Promise<ArtifactRoot> {
		const response = await UpdateArtifactRoot({
			RootID: rootID,
			Body: body,
		} as Parameters<typeof UpdateArtifactRoot>[0]);

		return requireWailsBody(response.Body, 'UpdateArtifactRoot');
	}

	async retireArtifactRoot(rootID: ArtifactRootID, expectedRevision: number): Promise<ArtifactRoot> {
		const response = await RetireArtifactRoot({
			RootID: rootID,
			expectedRevision,
		} as Parameters<typeof RetireArtifactRoot>[0]);

		return requireWailsBody(response.Body, 'RetireArtifactRoot');
	}

	async purgeArtifactRoot(rootID: ArtifactRootID, expectedRevision: number): Promise<PurgeArtifactRootResult> {
		const result = await PurgeArtifactRoot({
			RootID: rootID,
			expectedRevision,
		} as Parameters<typeof PurgeArtifactRoot>[0]);

		return { rootID: requireWailsString(result.rootID, 'PurgeArtifactRoot.rootID') as ArtifactRootID };
	}

	async createArtifactSource(rootID: ArtifactRootID, body: ArtifactSourceDraft): Promise<ArtifactSourceSummary> {
		const response = await CreateArtifactSource({
			RootID: rootID,
			Body: createSourceBodyToWails(body),
		} as Parameters<typeof CreateArtifactSource>[0]);

		return requireWailsBody(response.Body, 'CreateArtifactSource');
	}

	async getArtifactSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<ArtifactSourceSummary> {
		const response = await GetArtifactSource({
			RootID: rootID,
			SourceID: sourceID,
		} as Parameters<typeof GetArtifactSource>[0]);

		return requireWailsBody(response.Body, 'GetArtifactSource');
	}

	async listArtifactSources(rootID: ArtifactRootID): Promise<ArtifactSourceSummary[]> {
		const response = await ListArtifactSources({
			RootID: rootID,
		} as Parameters<typeof ListArtifactSources>[0]);

		const body = requireWailsBody(response.Body, 'ListArtifactSources');
		return body.sources;
	}

	async updateArtifactSource(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		body: UpdateArtifactSourceBody
	): Promise<ArtifactSourceSummary> {
		const response = await UpdateArtifactSource({
			RootID: rootID,
			SourceID: sourceID,
			Body: updateSourceBodyToWails(body),
		} as Parameters<typeof UpdateArtifactSource>[0]);

		return requireWailsBody(response.Body, 'UpdateArtifactSource');
	}

	async retireArtifactSource(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		expectedRevision: number
	): Promise<ArtifactSourceSummary> {
		const response = await RetireArtifactSource({
			RootID: rootID,
			SourceID: sourceID,
			expectedRevision,
		} as Parameters<typeof RetireArtifactSource>[0]);

		return requireWailsBody(response.Body, 'RetireArtifactSource');
	}

	async purgeArtifactSource(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		expectedRevision: number
	): Promise<PurgeArtifactSourceResult> {
		const result = await PurgeArtifactSource({
			RootID: rootID,
			SourceID: sourceID,
			expectedRevision,
		} as Parameters<typeof PurgeArtifactSource>[0]);

		return {
			rootID: result.rootID as ArtifactRootID,
			sourceID: result.sourceID as ArtifactSourceID,
		};
	}

	async listArtifactSourceKinds(): Promise<ArtifactSourceKind[]> {
		const response = await ListArtifactSourceKinds({} as Parameters<typeof ListArtifactSourceKinds>[0]);
		const body = requireWailsBody(response.Body, 'ListArtifactSourceKinds');

		return requireWailsArray(body.kinds, 'ListArtifactSourceKinds.kinds').map(
			(kind, index) => requireWailsString(kind, `ListArtifactSourceKinds.kinds[${index}]`) as ArtifactSourceKind
		);
	}

	async getManagedSourceState(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<ManagedSourceState> {
		const response = await GetManagedSourceState({
			rootID,
			sourceID,
		} as Parameters<typeof GetManagedSourceState>[0]);

		const body = requireWailsBody(response.Body, 'GetManagedSourceState');
		return {
			generation: body.generation as string,
			source: body.source,
		};
	}

	async publishManagedSourcePackage(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		body: PublishManagedSourcePackageBody
	): Promise<ManagedSourcePackageResult> {
		const req = {
			rootID: rootID,
			sourceID: sourceID,
			Body: body as unknown as artifactstore.PublishManagedSourcePackageRequestBody,
		} as artifactstore.PublishManagedSourcePackageRequest;

		const response = await PublishManagedSourcePackage(req);
		const responseBody = requireWailsBody(response.Body, 'PublishManagedSourcePackage');
		return {
			generation: responseBody.generation as string,
			source: responseBody.source,
		};
	}

	async removeManagedSourcePackage(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		body: RemoveManagedSourcePackageBody
	): Promise<ManagedSourcePackageResult> {
		const response = await RemoveManagedSourcePackage({
			rootID,
			sourceID,
			expectedSourceRevision: body.expectedSourceRevision,
			address: body.address,
			expectedGeneration: body.expectedGeneration,
		} as Parameters<typeof RemoveManagedSourcePackage>[0]);

		const responseBody = requireWailsBody(response.Body, 'RemoveManagedSourcePackage');

		return {
			generation: responseBody.generation as string,
			source: responseBody.source,
		};
	}
}
