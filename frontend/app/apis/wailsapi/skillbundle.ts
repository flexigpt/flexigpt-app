import type { ArtifactAddress, ArtifactDiagnostic, ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import { ArtifactAdoptionMode, ArtifactState } from '@/spec/artifact';
import type {
	AdoptSkillBody,
	AttachSkillBundleSourceBody,
	CreateManagedSkillBody,
	CreateManagedSkillResult,
	CreateSkillBundleBody,
	CreateSkillSessionOptions,
	InvokeSkillToolResponse,
	ManagedSkillDocumentView,
	PinSkillBody,
	RenderSkillResponse,
	RetireSkillBundleResult,
	RuntimeSkillFilter,
	RuntimeSkillListItem,
	SetSkillEnabledBody,
	SkillArtifactView,
	SkillBundleRef,
	SkillBundleView,
	SkillDocumentInput,
	SkillSession,
	UpdateSkillBundleBody,
} from '@/spec/skill';
import { SkillBundleAttachmentRole } from '@/spec/skill';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { ISkillBundleAPI } from '@/apis/interface';
import { byteArrayToWails, enumFromWails, rawJSONToWails, requireWailsBody } from '@/apis/wailsapi/transport';
import {
	AdoptSkill,
	AttachSkillSource,
	CloseSkillSession,
	CreateManagedSkill,
	CreateSkillBundle,
	CreateSkillSession,
	GetManagedSkillDocument,
	GetSkillBundle,
	GetSkillsPrompt,
	InvokeSkillTool,
	ListBundleSkills,
	ListRuntimeSkills,
	ListSkillBundles,
	PinSkill,
	PurgeSkill,
	PurgeSkillBundle,
	RefreshSkillBundle,
	RenderSkill,
	RetireSkillBundle,
	SetSkillEnabled,
	UnadoptSkill,
	UpdateSkillBundle,
} from '@/apis/wailsjs/go/main/SkillBundleWrapper';
import type { artifact as wailsArtifact, bundle as wailsSkillBundle } from '@/apis/wailsjs/go/models';

function skillArtifactFromWails(artifact: wailsArtifact.Artifact): SkillArtifactView {
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
		binding: artifact.binding,
		definitionDigest: artifact.resolvedDefinition ?? undefined,
		diagnostics: artifact.diagnostics as ArtifactDiagnostic[],
		createdAt: artifact.createdAt,
		modifiedAt: artifact.modifiedAt,
	};
}

function skillBundleFromWails(bundle: wailsSkillBundle.Bundle): SkillBundleView {
	const collection = bundle.collection;
	const sourcesByID = new Map(bundle.sources.map(source => [source.id, source]));

	return {
		bundle: {
			rootID: collection.rootID,
			collectionID: collection.id,
		},
		revision: collection.revision,
		displayName: collection.displayName,
		description: collection.description ?? undefined,
		enabled: collection.enabled,
		retiredAt: collection.retiredAt,
		logicalName: bundle.data.logicalName,
		logicalVersion: bundle.data.logicalVersion ?? undefined,
		labels: bundle.data.labels ?? undefined,
		managedSourceID: bundle.data.managedSourceID ?? undefined,
		attachments: bundle.attachments.map(attachment => {
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

export class WailsSkillBundleAPI implements ISkillBundleAPI {
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
		return bundles.map(bundle => skillBundleFromWails(bundle));
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
		return artifacts.map(artifact => skillArtifactFromWails(artifact));
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

		return {
			artifact: skillArtifactFromWails(result.Artifact),
			address: result.Address,
		};
	}

	async getManagedSkillDocument(artifact: ArtifactRef): Promise<ManagedSkillDocumentView> {
		const result = await GetManagedSkillDocument(artifact as Parameters<typeof GetManagedSkillDocument>[0]);

		return {
			artifact: skillArtifactFromWails(result.Artifact),
			document: result.Document as SkillDocumentInput,
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

	async getSkillsPrompt(filter: RuntimeSkillFilter): Promise<string> {
		const response = await GetSkillsPrompt({
			Body: {
				filter: filter,
			},
		} as Parameters<typeof GetSkillsPrompt>[0]);

		return requireWailsBody(response.Body, 'GetSkillsPrompt').prompt;
	}

	async createSkillSession(options: CreateSkillSessionOptions): Promise<SkillSession> {
		const response = await CreateSkillSession({
			Body: {
				closeSessionID: options.closeSessionID,
				maxActivePerSession: options.maxActivePerSession,
				allowArtifacts: options.allowArtifacts,
				activeArtifacts: options.activeArtifacts,
			},
		} as Parameters<typeof CreateSkillSession>[0]);
		const body = requireWailsBody(response.Body, 'CreateSkillSession');

		return {
			sessionID: body.sessionID,
			activeArtifacts: body.activeArtifacts ?? [],
		};
	}

	async closeSkillSession(sessionID: string): Promise<void> {
		await CloseSkillSession({
			SessionID: sessionID,
		} as Parameters<typeof CloseSkillSession>[0]);
	}

	async listRuntimeSkills(filter: RuntimeSkillFilter): Promise<RuntimeSkillListItem[]> {
		const response = await ListRuntimeSkills({
			Body: {
				filter: filter,
			},
		} as Parameters<typeof ListRuntimeSkills>[0]);

		return (requireWailsBody(response.Body, 'ListRuntimeSkills').skills as RuntimeSkillListItem[]) ?? [];
	}

	async invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse> {
		const response = await InvokeSkillTool({
			Body: {
				sessionID,
				toolName,
				args: args === undefined ? undefined : rawJSONToWails(args, 'skill tool arguments'),
			},
		} as Parameters<typeof InvokeSkillTool>[0]);

		return requireWailsBody(response.Body, 'InvokeSkillTool') as InvokeSkillToolResponse;
	}

	async renderSkill(artifact: ArtifactRef, args?: Record<string, string>): Promise<RenderSkillResponse> {
		const response = await RenderSkill({
			Body: {
				artifact,
				arguments: args,
			},
		} as Parameters<typeof RenderSkill>[0]);

		return requireWailsBody(response.Body, 'RenderSkill') as RenderSkillResponse;
	}
}
