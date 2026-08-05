import type {
	ArtifactRoot,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactSourceKind,
	ArtifactSourceSummary,
	CreateArtifactRootBody,
	CreateArtifactSourceBody,
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
	byteArrayToWails,
	rawJSONObjectToWails,
	requireWailsArray,
	requireWailsBody,
	requireWailsObject,
	requireWailsString,
	toFrontendDate,
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

function artifactRootFromWails(value: unknown): ArtifactRoot {
	const root = requireWailsObject(value, 'ArtifactRoot');

	return {
		id: root.id as ArtifactRootID,
		displayName: root.displayName as string,
		description: root.description ?? undefined,
		revision: root.revision as number,
		createdAt: toFrontendDate(root.createdAt, 'artifactRoot.createdAt'),
		modifiedAt: toFrontendDate(root.modifiedAt, 'artifactRoot.modifiedAt'),
		retiredAt: root.retiredAt ? toFrontendDate(root.retiredAt, 'artifactRoot.retiredAt') : undefined,
	} as ArtifactRoot;
}

function artifactSourceFromWails(value: unknown): ArtifactSourceSummary {
	const source = requireWailsObject(value, 'ArtifactSourceSummary');

	return {
		id: source.id as ArtifactSourceID,
		rootID: source.rootID as ArtifactRootID,
		kind: source.kind as ArtifactSourceKind,
		displayName: source.displayName as string,
		enabled: source.enabled as boolean,
		revision: source.revision as number,
		createdAt: toFrontendDate(source.createdAt, 'artifactSource.createdAt'),
		modifiedAt: toFrontendDate(source.modifiedAt, 'artifactSource.modifiedAt'),
		retiredAt: source.retiredAt ? toFrontendDate(source.retiredAt, 'artifactSource.retiredAt') : undefined,
	};
}

function createSourceBodyToWails(body: CreateArtifactSourceBody): unknown {
	return {
		id: body.id,
		kind: body.kind,
		displayName: body.displayName,
		enabled: body.enabled,
		config: rawJSONObjectToWails(body.config, 'artifact source config'),
	};
}

function updateSourceBodyToWails(body: UpdateArtifactSourceBody): unknown {
	return {
		expectedRevision: body.expectedRevision,
		displayName: body.displayName,
		enabled: body.enabled,
		...(body.config === undefined ? {} : { config: rawJSONObjectToWails(body.config, 'artifact source config') }),
	};
}

function publishManagedPackageBodyToWails(body: PublishManagedSourcePackageBody): unknown {
	return {
		expectedSourceRevision: body.expectedSourceRevision,
		directory: body.directory,
		expectedGeneration: body.expectedGeneration,
		files: body.files.map(file => ({
			locator: file.locator,
			content: byteArrayToWails(file.content),
		})),
	};
}

export class WailsArtifactStoreAPI implements IArtifactStoreAPI {
	async createArtifactRoot(body: CreateArtifactRootBody): Promise<ArtifactRoot> {
		const response = await CreateArtifactRoot({
			Body: body,
		} as Parameters<typeof CreateArtifactRoot>[0]);

		return artifactRootFromWails(requireWailsBody(response.Body, 'CreateArtifactRoot'));
	}

	async getArtifactRoot(rootID: ArtifactRootID): Promise<ArtifactRoot> {
		const response = await GetArtifactRoot({
			RootID: rootID,
		} as Parameters<typeof GetArtifactRoot>[0]);

		return artifactRootFromWails(requireWailsBody(response.Body, 'GetArtifactRoot'));
	}

	async listArtifactRoots(): Promise<ArtifactRoot[]> {
		const response = await ListArtifactRoots({} as Parameters<typeof ListArtifactRoots>[0]);
		const body = requireWailsObject(requireWailsBody(response.Body, 'ListArtifactRoots'), 'ListArtifactRoots');

		return requireWailsArray(body.roots, 'ListArtifactRoots.roots').map(r => {
			return artifactRootFromWails(r);
		});
	}

	async updateArtifactRoot(rootID: ArtifactRootID, body: UpdateArtifactRootBody): Promise<ArtifactRoot> {
		const response = await UpdateArtifactRoot({
			RootID: rootID,
			Body: body,
		} as Parameters<typeof UpdateArtifactRoot>[0]);

		return artifactRootFromWails(requireWailsBody(response.Body, 'UpdateArtifactRoot'));
	}

	async retireArtifactRoot(rootID: ArtifactRootID, expectedRevision: number): Promise<ArtifactRoot> {
		const response = await RetireArtifactRoot({
			RootID: rootID,
			expectedRevision,
		} as Parameters<typeof RetireArtifactRoot>[0]);

		return artifactRootFromWails(requireWailsBody(response.Body, 'RetireArtifactRoot'));
	}

	async purgeArtifactRoot(rootID: ArtifactRootID, expectedRevision: number): Promise<PurgeArtifactRootResult> {
		const result = requireWailsObject(
			await PurgeArtifactRoot({
				RootID: rootID,
				expectedRevision,
			} as Parameters<typeof PurgeArtifactRoot>[0]),
			'PurgeArtifactRoot'
		);

		return { rootID: requireWailsString(result.rootID, 'PurgeArtifactRoot.rootID') as ArtifactRootID };
	}

	async createArtifactSource(rootID: ArtifactRootID, body: CreateArtifactSourceBody): Promise<ArtifactSourceSummary> {
		const response = await CreateArtifactSource({
			RootID: rootID,
			Body: createSourceBodyToWails(body),
		} as Parameters<typeof CreateArtifactSource>[0]);

		return artifactSourceFromWails(requireWailsBody(response.Body, 'CreateArtifactSource'));
	}

	async getArtifactSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<ArtifactSourceSummary> {
		const response = await GetArtifactSource({
			RootID: rootID,
			SourceID: sourceID,
		} as Parameters<typeof GetArtifactSource>[0]);

		return artifactSourceFromWails(requireWailsBody(response.Body, 'GetArtifactSource'));
	}

	async listArtifactSources(rootID: ArtifactRootID): Promise<ArtifactSourceSummary[]> {
		const response = await ListArtifactSources({
			RootID: rootID,
		} as Parameters<typeof ListArtifactSources>[0]);

		const body = requireWailsObject(requireWailsBody(response.Body, 'ListArtifactSources'), 'ListArtifactSources');
		return requireWailsArray(body.sources, 'ListArtifactSources.sources').map(source =>
			artifactSourceFromWails(source)
		);
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

		return artifactSourceFromWails(requireWailsBody(response.Body, 'UpdateArtifactSource'));
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

		return artifactSourceFromWails(requireWailsBody(response.Body, 'RetireArtifactSource'));
	}

	async purgeArtifactSource(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		expectedRevision: number
	): Promise<PurgeArtifactSourceResult> {
		const result = requireWailsObject(
			await PurgeArtifactSource({
				RootID: rootID,
				SourceID: sourceID,
				expectedRevision,
			} as Parameters<typeof PurgeArtifactSource>[0]),
			'PurgeArtifactSource'
		);

		return {
			rootID: result.rootID as ArtifactRootID,
			sourceID: result.sourceID as ArtifactSourceID,
		};
	}

	async listArtifactSourceKinds(): Promise<ArtifactSourceKind[]> {
		const response = await ListArtifactSourceKinds({} as Parameters<typeof ListArtifactSourceKinds>[0]);
		const body = requireWailsObject(
			requireWailsBody(response.Body, 'ListArtifactSourceKinds'),
			'ListArtifactSourceKinds'
		);

		return requireWailsArray(body.kinds, 'ListArtifactSourceKinds.kinds').map(
			(kind, index) => requireWailsString(kind, `ListArtifactSourceKinds.kinds[${index}]`) as ArtifactSourceKind
		);
	}

	async getManagedSourceState(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<ManagedSourceState> {
		const response = await GetManagedSourceState({
			rootID,
			sourceID,
		} as Parameters<typeof GetManagedSourceState>[0]);

		const body = requireWailsObject(requireWailsBody(response.Body, 'GetManagedSourceState'), 'GetManagedSourceState');
		return {
			generation: body.generation as string,
			source: artifactSourceFromWails(body.source),
		};
	}

	async publishManagedSourcePackage(
		rootID: ArtifactRootID,
		sourceID: ArtifactSourceID,
		body: PublishManagedSourcePackageBody
	): Promise<ManagedSourcePackageResult> {
		const response = await PublishManagedSourcePackage({
			rootID,
			sourceID,
			Body: publishManagedPackageBodyToWails(body),
		} as Parameters<typeof PublishManagedSourcePackage>[0]);

		const responseBody = requireWailsObject(
			requireWailsBody(response.Body, 'PublishManagedSourcePackage'),
			'PublishManagedSourcePackage'
		);

		return {
			generation: responseBody.generation as string,
			source: artifactSourceFromWails(responseBody.source),
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
			directory: body.directory,
			expectedGeneration: body.expectedGeneration,
		} as Parameters<typeof RemoveManagedSourcePackage>[0]);

		const responseBody = requireWailsObject(
			requireWailsBody(response.Body, 'RemoveManagedSourcePackage'),
			'RemoveManagedSourcePackage'
		);

		return {
			generation: responseBody.generation as string,
			source: artifactSourceFromWails(responseBody.source),
		};
	}
}
