import type { ArtifactRef, ArtifactRootID, ArtifactSourceID } from '@/spec/artifact';
import type { StoreArtifact, StoreArtifactSourceSummary } from '@/spec/artifact_store';
import type {
	AddArtifactMemberRequest,
	AddMemberRequest,
	ArtifactMembershipView,
	CollectionCapabilityPlan,
	CollectionView,
	CreateCollectionRequest,
	DeleteCollectionRequest,
	RemoveMemberRequest,
	UpdateCollectionRequest,
} from '@/spec/collection';
import type { CapabilityPlan } from '@/spec/resolution';
import type {
	ManagedSkillCreateRequest,
	ManagedSkillCreateResult,
	ManagedSkillReplaceRequest,
	ManagedSkillReplaceResult,
	SkillDirectoryRegistration,
	SkillPathRegistration,
	SkillPathRegistrationResult,
	StoreManagedSkillDocument,
} from '@/spec/skill_store';

import type { ISkillStoreAPI } from '@/apis/interface';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	AddSkillCollectionMember,
	AddSkillPath,
	AttachSkillArtifactToCollection,
	CreateManagedSkill,
	CreateSkillCollection,
	DeleteSkillCollection,
	GetManagedSkillDocument,
	GetSkill,
	GetSkillCollection,
	ListSkillCollectionMemberships,
	ListSkillCollections,
	ListSkillCollectionsForManagement,
	ListSkills,
	ListSkillsForManagement,
	PurgeSkill,
	RefreshSkillSource,
	RegisterSkillDirectory,
	RemoveSkillCollectionMember,
	ReplaceManagedSkill,
	ResolveSkillArtifactCapabilities,
	ResolveSkillCapabilities,
	ResolveSkillCollection,
	SetSkillCollectionEnabled,
	SetSkillEnabled,
	UpdateSkillCollection,
} from '@/apis/wailsjs/go/main/SkillStoreWrapper';

export class WailsSkillStoreAPI implements ISkillStoreAPI {
	async addSkillCollectionMember(request: AddMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await AddSkillCollectionMember(request as Parameters<typeof AddSkillCollectionMember>[0]),
			'AddSkillCollectionMember'
		);
	}

	async addSkillPath(request: SkillPathRegistration): Promise<SkillPathRegistrationResult> {
		return requiredObject<SkillPathRegistrationResult>(
			await AddSkillPath(request as Parameters<typeof AddSkillPath>[0]),
			'AddSkillPath'
		);
	}

	async attachSkillArtifactToCollection(request: AddArtifactMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await AttachSkillArtifactToCollection(request as Parameters<typeof AttachSkillArtifactToCollection>[0]),
			'AttachSkillArtifactToCollection'
		);
	}

	async createManagedSkill(request: ManagedSkillCreateRequest): Promise<ManagedSkillCreateResult> {
		return requiredObject<ManagedSkillCreateResult>(
			await CreateManagedSkill(request as Parameters<typeof CreateManagedSkill>[0]),
			'CreateManagedSkill'
		);
	}

	async replaceManagedSkill(request: ManagedSkillReplaceRequest): Promise<ManagedSkillReplaceResult> {
		const response = await ReplaceManagedSkill(request as Parameters<typeof ReplaceManagedSkill>[0]);

		return requiredObject<ManagedSkillReplaceResult>(response, 'ReplaceManagedSkill');
	}

	async createSkillCollection(request: CreateCollectionRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await CreateSkillCollection(request as Parameters<typeof CreateSkillCollection>[0]),
			'CreateSkillCollection'
		);
	}

	async deleteSkillCollection(request: DeleteCollectionRequest): Promise<void> {
		await DeleteSkillCollection(request as Parameters<typeof DeleteSkillCollection>[0]);
	}

	async getManagedSkillDocument(skill: ArtifactRef): Promise<StoreManagedSkillDocument> {
		return requiredObject<StoreManagedSkillDocument>(
			await GetManagedSkillDocument(skill as Parameters<typeof GetManagedSkillDocument>[0]),
			'GetManagedSkillDocument'
		);
	}

	async getSkill(skill: ArtifactRef): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(await GetSkill(skill as Parameters<typeof GetSkill>[0]), 'GetSkill');
	}

	async getSkillCollection(collection: ArtifactRef): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await GetSkillCollection(collection as Parameters<typeof GetSkillCollection>[0]),
			'GetSkillCollection'
		);
	}

	async listSkillCollectionMemberships(skill: ArtifactRef): Promise<ArtifactMembershipView[]> {
		return wailsObjectArrayOrEmpty<ArtifactMembershipView>(
			await ListSkillCollectionMemberships(skill as Parameters<typeof ListSkillCollectionMemberships>[0]),
			'ListSkillCollectionMemberships'
		);
	}

	async listSkillCollections(rootID: ArtifactRootID): Promise<CollectionView[]> {
		return wailsObjectArrayOrEmpty<CollectionView>(
			await ListSkillCollections(rootID as Parameters<typeof ListSkillCollections>[0]),
			'ListSkillCollections'
		);
	}

	async listSkillCollectionsForManagement(): Promise<CollectionView[]> {
		return wailsObjectArrayOrEmpty<CollectionView>(
			await ListSkillCollectionsForManagement(),
			'ListSkillCollectionsForManagement'
		);
	}

	async listSkills(rootID: ArtifactRootID): Promise<StoreArtifact[]> {
		return wailsObjectArrayOrEmpty<StoreArtifact>(
			await ListSkills(rootID as Parameters<typeof ListSkills>[0]),
			'ListSkills'
		);
	}

	async listSkillsForManagement(): Promise<StoreArtifact[]> {
		return wailsObjectArrayOrEmpty<StoreArtifact>(await ListSkillsForManagement(), 'ListSkillsForManagement');
	}

	async purgeSkill(skill: ArtifactRef, expectedRevision: number): Promise<void> {
		await PurgeSkill(skill as Parameters<typeof PurgeSkill>[0], expectedRevision);
	}

	async refreshSkillSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<void> {
		await RefreshSkillSource(
			rootID as Parameters<typeof RefreshSkillSource>[0],
			sourceID as Parameters<typeof RefreshSkillSource>[1]
		);
	}

	async registerSkillDirectory(request: SkillDirectoryRegistration): Promise<StoreArtifactSourceSummary> {
		return requiredObject<StoreArtifactSourceSummary>(
			await RegisterSkillDirectory(request as Parameters<typeof RegisterSkillDirectory>[0]),
			'RegisterSkillDirectory'
		);
	}

	async removeSkillCollectionMember(request: RemoveMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await RemoveSkillCollectionMember(request as Parameters<typeof RemoveSkillCollectionMember>[0]),
			'RemoveSkillCollectionMember'
		);
	}

	async resolveSkillArtifactCapabilities(skill: ArtifactRef): Promise<CapabilityPlan> {
		return requiredObject<CapabilityPlan>(
			await ResolveSkillArtifactCapabilities(skill as Parameters<typeof ResolveSkillArtifactCapabilities>[0]),
			'ResolveSkillArtifactCapabilities'
		);
	}

	async resolveSkillCapabilities(skill: ArtifactRef): Promise<CapabilityPlan> {
		return requiredObject<CapabilityPlan>(
			await ResolveSkillCapabilities(skill as Parameters<typeof ResolveSkillCapabilities>[0]),
			'ResolveSkillCapabilities'
		);
	}

	async resolveSkillCollection(collection: ArtifactRef): Promise<CollectionCapabilityPlan> {
		return requiredObject<CollectionCapabilityPlan>(
			await ResolveSkillCollection(collection as Parameters<typeof ResolveSkillCollection>[0]),
			'ResolveSkillCollection'
		);
	}

	async setSkillCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await SetSkillCollectionEnabled(
				collection as Parameters<typeof SetSkillCollectionEnabled>[0],
				expectedRevision,
				enabled
			),
			'SetSkillCollectionEnabled'
		);
	}

	async setSkillEnabled(skill: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await SetSkillEnabled(skill as Parameters<typeof SetSkillEnabled>[0], expectedRevision, enabled),
			'SetSkillEnabled'
		);
	}

	async updateSkillCollection(request: UpdateCollectionRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await UpdateSkillCollection(request as Parameters<typeof UpdateSkillCollection>[0]),
			'UpdateSkillCollection'
		);
	}
}
