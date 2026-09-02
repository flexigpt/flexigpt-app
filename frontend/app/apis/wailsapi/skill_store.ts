import type { ArtifactAddress, ArtifactDiagnostic, ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import { ArtifactAdoptionMode, ArtifactState } from '@/spec/artifact';
import type {
	AdoptSkillBody,
	AttachSkillBundleSourceBody,
	CreateManagedSkillBody,
	CreateManagedSkillResult,
	CreateSkillBundleBody,
	ManagedSkillDocumentView,
	PinSkillBody,
	ResolvedSkillRuntime,
	RetireSkillBundleResult,
	SetSkillEnabledBody,
	SkillArtifactView,
	SkillBundleRef,
	SkillBundleView,
	SkillDocumentInput,
	SkillRuntimeCatalogID,
	UpdateSkillBundleBody,
} from '@/spec/skill';
import { SkillBundleAttachmentRole } from '@/spec/skill';

import type { ISkillStoreAPI } from '@/apis/interface';
import {
	byteArrayToWails,
	enumFromWails,
	requireNonBlankString,
	requireWailsBody,
	requireWailsString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	AdoptSkill,
	AttachSkillSource,
	CreateManagedSkill,
	CreateSkillBundle,
	GetManagedSkillDocument,
	GetSkillBundle,
	ListBundleSkills,
	ListSkillBundles,
	PinSkill,
	PurgeSkill,
	PurgeSkillBundle,
	RefreshSkillBundle,
	ResolveArtifactSkill,
	RetireSkillBundle,
	RuntimeCatalogIDForCollection,
	SetSkillEnabled,
	UnadoptSkill,
	UpdateSkillBundle,
} from '@/apis/wailsjs/go/main/SkillStoreWrapper';
import type {
	artifact as wailsArtifact,
	bundle as wailsSkillBundle,
	store as wailsSkillStore,
} from '@/apis/wailsjs/go/models';

function skillArtifactFromWails(artifactValue: wailsArtifact.Artifact, field = 'skillArtifact'): SkillArtifactView {
	const artifact = requireWailsBody(artifactValue, field);
	const ref: ArtifactRef = {
		rootID: artifact.rootID,
		artifactID: artifact.id,
	};

	const address: ArtifactAddress = {
		...ref,
		collectionID: artifact.collectionID,
		kind: artifact.kind,
	};

	return {
		artifact: ref,
		address,
		revision: artifact.revision,
		name: artifact.name,
		kind: artifact.kind,
		enabled: artifact.enabled,
		adoption: enumFromWails(artifact.adoption, ArtifactAdoptionMode, 'skillArtifact.adoption'),
		state: enumFromWails(artifact.state, ArtifactState, 'skillArtifact.state'),
		binding: requireWailsBody(artifact.binding, `${field}.binding`),
		definitionDigest: artifact.resolvedDefinition ?? undefined,
		diagnostics:
			artifact.diagnostics === null || artifact.diagnostics === undefined
				? undefined
				: wailsObjectArrayOrEmpty<ArtifactDiagnostic>(artifact.diagnostics, `${field}.diagnostics`),
		createdAt: artifact.createdAt,
		modifiedAt: artifact.modifiedAt,
	};
}

function skillBundleFromWails(bundleValue: wailsSkillBundle.Bundle, field = 'skillBundle'): SkillBundleView {
	const bundle = requireWailsBody(bundleValue, field);
	const collection = requireWailsBody(bundle.collection, `${field}.collection`);
	const data = requireWailsBody(bundle.data, `${field}.data`);
	const sources = wailsObjectArrayOrEmpty<wailsSkillBundle.Bundle['sources'][number]>(
		bundle.sources,
		`${field}.sources`
	);
	const attachments = wailsObjectArrayOrEmpty<wailsSkillBundle.Bundle['attachments'][number]>(
		bundle.attachments,
		`${field}.attachments`
	);
	const sourcesByID = new Map(sources.map(source => [source.id, source]));

	return {
		bundle: {
			rootID: collection.rootID,
			collectionID: collection.id,
		},
		revision: collection.revision,
		displayName: collection.displayName,
		description: collection.description ?? undefined,
		enabled: collection.enabled,
		retiredAt: collection.retiredAt ?? undefined,
		logicalName: data.logicalName,
		logicalVersion: data.logicalVersion ?? undefined,
		labels: data.labels ?? undefined,
		managedSourceID: data.managedSourceID ?? undefined,
		attachments: attachments.map(attachment => {
			const source = sourcesByID.get(attachment.sourceID);

			return {
				sourceID: attachment.sourceID,
				revision: attachment.revision,
				role: enumFromWails(attachment.role, SkillBundleAttachmentRole, 'skillBundle.attachment.role'),
				enabled: attachment.enabled,
				sourceDisplayName: source?.displayName,
				sourceKind: source?.kind,
			};
		}),
		createdAt: collection.createdAt,
		modifiedAt: collection.modifiedAt,
	};
}

export class WailsSkillStoreAPI implements ISkillStoreAPI {
	async createSkillBundle(rootID: ArtifactRootID, body: CreateSkillBundleBody): Promise<SkillBundleView> {
		const managedSourceID = body.managedSourceID?.trim() || undefined;
		const managedSourceStorageKey = body.managedSourceStorageKey?.trim() || undefined;

		if (Boolean(managedSourceID) !== Boolean(managedSourceStorageKey)) {
			throw new Error('Managed Source ID and managed Source storage key must be supplied together.');
		}

		const bundle = await CreateSkillBundle({
			RootID: rootID,
			CollectionID: body.collectionID,
			DisplayName: body.displayName,
			Description: body.description ?? '',
			Enabled: body.enabled,
			LogicalName: body.logicalName,
			LogicalVersion: body.logicalVersion ?? '',
			Labels: body.labels ?? {},
			ManagedSourceID: managedSourceID ?? '',
			ManagedSourceStorageKey: managedSourceStorageKey ?? '',
			Attachments: (body.attachments ?? []).map(attachment => ({
				SourceID: attachment.sourceID,
				Role: attachment.role,
				Enabled: attachment.enabled,
				DiscoveryRoot: attachment.discoveryRoot,
				ExpectedMemberDigests: attachment.expectedMemberDigests ?? {},
			})),
		} as Parameters<typeof CreateSkillBundle>[0]);

		return skillBundleFromWails(bundle);
	}

	async getSkillBundle(bundle: SkillBundleRef): Promise<SkillBundleView> {
		return skillBundleFromWails(await GetSkillBundle(bundle as Parameters<typeof GetSkillBundle>[0]));
	}

	async listSkillBundles(rootID: ArtifactRootID): Promise<SkillBundleView[]> {
		const bundles = await ListSkillBundles(rootID as Parameters<typeof ListSkillBundles>[0]);
		return wailsObjectArrayOrEmpty<wailsSkillBundle.Bundle>(bundles, 'ListSkillBundles').map((bundle, index) => {
			return skillBundleFromWails(bundle, `ListSkillBundles[${index}]`);
		});
	}

	async updateSkillBundle(bundle: SkillBundleRef, body: UpdateSkillBundleBody): Promise<SkillBundleView> {
		const updated = await UpdateSkillBundle({
			Bundle: bundle,
			ExpectedRevision: body.expectedRevision,
			DisplayName: body.displayName,
			Description: body.description ?? '',
			Enabled: body.enabled,
		} as Parameters<typeof UpdateSkillBundle>[0]);

		return skillBundleFromWails(updated);
	}

	async retireSkillBundle(bundle: SkillBundleRef, expectedRevision: number): Promise<RetireSkillBundleResult> {
		const retired = await RetireSkillBundle(bundle as Parameters<typeof RetireSkillBundle>[0], expectedRevision);
		return { bundle, revision: retired.revision };
	}

	async purgeSkillBundle(bundle: SkillBundleRef, expectedRevision: number): Promise<SkillBundleRef> {
		await PurgeSkillBundle(bundle as Parameters<typeof PurgeSkillBundle>[0], expectedRevision);
		return bundle;
	}

	async attachSkillBundleSource(bundle: SkillBundleRef, body: AttachSkillBundleSourceBody): Promise<SkillBundleView> {
		const updated = await AttachSkillSource(
			bundle as Parameters<typeof AttachSkillSource>[0],
			body.expectedCollectionRevision,
			{
				SourceID: body.sourceID,
				Role: body.role,
				Enabled: body.enabled,
				DiscoveryRoot: body.discoveryRoot,
				ExpectedMemberDigests: body.expectedMemberDigests ?? {},
			} as Parameters<typeof AttachSkillSource>[2]
		);

		return skillBundleFromWails(updated);
	}

	async refreshSkillBundle(bundle: SkillBundleRef): Promise<void> {
		await RefreshSkillBundle(bundle as Parameters<typeof RefreshSkillBundle>[0]);
	}

	async listSkillBundleArtifacts(bundle: SkillBundleRef): Promise<SkillArtifactView[]> {
		const artifacts = await ListBundleSkills(bundle as Parameters<typeof ListBundleSkills>[0]);
		return wailsObjectArrayOrEmpty<wailsArtifact.Artifact>(artifacts, 'ListBundleSkills').map((artifact, index) => {
			return skillArtifactFromWails(artifact, `ListBundleSkills[${index}]`);
		});
	}

	async createManagedSkill(bundle: SkillBundleRef, body: CreateManagedSkillBody): Promise<CreateManagedSkillResult> {
		const semanticAuthoringInput =
			body.skillMD !== undefined
				? {
						SKILLMD: byteArrayToWails(body.skillMD),
					}
				: {
						Document: body.document,
					};

		const result = await CreateManagedSkill({
			Bundle: bundle,
			ExpectedCollectionRevision: body.expectedCollectionRevision,
			ExpectedArtifactRevision: body.expectedArtifactRevision ?? 0,
			ArtifactID: body.artifactID,
			SkillName: body.skillName,
			Files: (body.files ?? []).map(file => ({
				locator: file.locator,
				content: byteArrayToWails(file.content),
			})),
			Enabled: body.enabled,
			...semanticAuthoringInput,
		} as Parameters<typeof CreateManagedSkill>[0]);

		const respBody = requireWailsBody(result, 'CreateManagedSkill');
		return {
			artifact: skillArtifactFromWails(respBody.Artifact, 'CreateManagedSkill.Artifact'),
			address: requireWailsBody(respBody.Address, 'CreateManagedSkill.Address'),
		};
	}

	async getManagedSkillDocument(artifact: ArtifactRef): Promise<ManagedSkillDocumentView> {
		const result = await GetManagedSkillDocument(artifact as Parameters<typeof GetManagedSkillDocument>[0]);
		const body = requireWailsBody(result, 'GetManagedSkillDocument');

		return {
			artifact: skillArtifactFromWails(body.Artifact, 'GetManagedSkillDocument.Artifact'),
			document: requireWailsBody(body.Document, 'GetManagedSkillDocument.Document') as SkillDocumentInput,
		};
	}

	async adoptSkill(bundle: SkillBundleRef, body: AdoptSkillBody): Promise<SkillArtifactView> {
		const artifact = await AdoptSkill({
			Bundle: bundle,
			Occurrence: body.occurrence,
			ArtifactID: body.artifactID,
			ExpectedCatalogRevision: body.expectedCatalogRevision,
			Name: body.name,
			Enabled: body.enabled,
		} as Parameters<typeof AdoptSkill>[0]);

		return skillArtifactFromWails(artifact);
	}

	async pinSkill(bundle: SkillBundleRef, body: PinSkillBody): Promise<SkillArtifactView> {
		const artifact = await PinSkill({
			Bundle: bundle,
			ExpectedCollectionRevision: body.expectedCollectionRevision,
			ArtifactID: body.artifactID,
			Binding: body.binding,
			Name: body.name,
			Enabled: body.enabled,
		} as Parameters<typeof PinSkill>[0]);

		return skillArtifactFromWails(artifact);
	}

	async setSkillEnabled(artifact: ArtifactRef, body: SetSkillEnabledBody): Promise<SkillArtifactView> {
		const updated = await SetSkillEnabled(
			artifact as Parameters<typeof SetSkillEnabled>[0],
			body.expectedRevision,
			body.enabled
		);

		return skillArtifactFromWails(updated);
	}

	async unadoptSkill(artifact: ArtifactRef, expectedRevision: number, suppress: boolean): Promise<ArtifactRef> {
		await UnadoptSkill(artifact as Parameters<typeof UnadoptSkill>[0], expectedRevision, suppress);
		return artifact;
	}

	async purgeSkill(artifact: ArtifactRef, expectedRevision: number): Promise<ArtifactRef> {
		await PurgeSkill(artifact as Parameters<typeof PurgeSkill>[0], expectedRevision);
		return artifact;
	}

	async runtimeCatalogIDForCollection(bundle: SkillBundleRef): Promise<SkillRuntimeCatalogID> {
		const catalogID = await RuntimeCatalogIDForCollection(
			bundle as Parameters<typeof RuntimeCatalogIDForCollection>[0]
		);
		return requireNonBlankString(catalogID, 'RuntimeCatalogIDForCollection');
	}

	async resolveArtifactSkill(artifact: ArtifactRef): Promise<ResolvedSkillRuntime> {
		const value = requireWailsBody(
			await ResolveArtifactSkill(artifact as Parameters<typeof ResolveArtifactSkill>[0]),
			'ResolveArtifactSkill'
		) as wailsSkillStore.ResolvedArtifactSkill;

		const resolvedArtifact = requireWailsBody(value.artifact, 'ResolveArtifactSkill.artifact');
		const collection = requireWailsBody(value.collection, 'ResolveArtifactSkill.collection');
		const definition = requireWailsBody(value.definition, 'ResolveArtifactSkill.definition');

		return {
			artifact: {
				rootID: requireWailsString(resolvedArtifact.rootID, 'ResolveArtifactSkill.artifact.rootID'),
				artifactID: requireWailsString(resolvedArtifact.artifactID, 'ResolveArtifactSkill.artifact.artifactID'),
			},
			collection: {
				rootID: requireWailsString(collection.rootID, 'ResolveArtifactSkill.collection.rootID'),
				collectionID: requireWailsString(collection.collectionID, 'ResolveArtifactSkill.collection.collectionID'),
			},
			definition: {
				type: requireWailsString(definition.type, 'ResolveArtifactSkill.definition.type'),
				name: requireWailsString(definition.name, 'ResolveArtifactSkill.definition.name'),
				location: requireWailsString(definition.location, 'ResolveArtifactSkill.definition.location'),
			},
			version: requireWailsString(value.version, 'ResolveArtifactSkill.version'),
		};
	}
}
