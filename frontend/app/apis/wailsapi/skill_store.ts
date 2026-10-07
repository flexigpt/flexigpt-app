import type {
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	CapabilityPlan,
	StoreArtifact,
	StoreArtifactSourceSummary,
} from '@/spec/artifact';
import type {
	AddPluginArtifactMemberRequest,
	AddPluginMemberRequest,
	ArtifactPluginMembershipView,
	CreatePluginRequest,
	DeletePluginRequest,
	PluginCapabilityPlan,
	PluginListItem,
	PluginView,
	RemovePluginMemberRequest,
	UpdatePluginRequest,
} from '@/spec/plugin';
import type {
	ManagedSkillCreateRequest,
	ManagedSkillCreateResult,
	ManagedSkillReplaceRequest,
	ManagedSkillReplaceResult,
	SkillDirectoryRegistration,
	SkillPathRegistration,
	SkillPathRegistrationResult,
	StoreManagedSkillDocument,
	StoreSkillListItem,
} from '@/spec/skill';

import type { ISkillStoreAPI } from '@/apis/interface';
import {
	artifactPluginMembershipFromWails,
	capabilityPlanFromWails,
	pluginCapabilityPlanFromWails,
	pluginListItemFromWails,
	pluginResultFromWails,
	pluginViewFromWails,
	storeSkillListItemFromWails,
} from '@/apis/wailsapi/list_item_projection';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	AddSkillPath,
	AddSkillPluginMember,
	AttachSkillArtifactToPlugin,
	CreateManagedSkill,
	CreateSkillPlugin,
	DeleteSkillPlugin,
	GetManagedSkillDocument,
	GetSkill,
	GetSkillPlugin,
	ListSkillPluginMemberships,
	ListSkillPlugins,
	ListSkillPluginsForManagement,
	ListSkills,
	ListSkillsForManagement,
	PurgeSkill,
	RefreshSkillSource,
	RegisterSkillDirectory,
	RemoveSkillPluginMember,
	ReplaceManagedSkill,
	ResolveSkillCapabilities,
	ResolveSkillPlugin,
	SetSkillEnabled,
	SetSkillPluginEnabled,
	UpdateSkillPlugin,
} from '@/apis/wailsjs/go/main/SkillStoreWrapper';

export class WailsSkillStoreAPI implements ISkillStoreAPI {
	async addSkillPluginMember(request: AddPluginMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await AddSkillPluginMember(request as Parameters<typeof AddSkillPluginMember>[0]),
			'AddSkillPluginMember'
		);
	}

	async addSkillPath(request: SkillPathRegistration): Promise<SkillPathRegistrationResult> {
		return requiredObject<SkillPathRegistrationResult>(
			await AddSkillPath(request as Parameters<typeof AddSkillPath>[0]),
			'AddSkillPath'
		);
	}

	async attachSkillArtifactToPlugin(request: AddPluginArtifactMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await AttachSkillArtifactToPlugin(request as Parameters<typeof AttachSkillArtifactToPlugin>[0]),
			'AttachSkillArtifactToPlugin'
		);
	}

	async createManagedSkill(request: ManagedSkillCreateRequest): Promise<ManagedSkillCreateResult> {
		return pluginResultFromWails<ManagedSkillCreateResult>(
			await CreateManagedSkill(request as Parameters<typeof CreateManagedSkill>[0]),
			'CreateManagedSkill'
		);
	}

	async replaceManagedSkill(request: ManagedSkillReplaceRequest): Promise<ManagedSkillReplaceResult> {
		return pluginResultFromWails<ManagedSkillReplaceResult>(
			await ReplaceManagedSkill(request as Parameters<typeof ReplaceManagedSkill>[0]),
			'ReplaceManagedSkill'
		);
	}

	async createSkillPlugin(request: CreatePluginRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await CreateSkillPlugin(request as Parameters<typeof CreateSkillPlugin>[0]),
			'CreateSkillPlugin'
		);
	}

	async deleteSkillPlugin(request: DeletePluginRequest): Promise<void> {
		await DeleteSkillPlugin(request as Parameters<typeof DeleteSkillPlugin>[0]);
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

	async getSkillPlugin(plugin: ArtifactRef): Promise<PluginView> {
		return pluginViewFromWails(await GetSkillPlugin(plugin as Parameters<typeof GetSkillPlugin>[0]), 'GetSkillPlugin');
	}

	async listSkillPluginMemberships(skill: ArtifactRef): Promise<ArtifactPluginMembershipView[]> {
		return wailsObjectArrayOrEmpty(
			await ListSkillPluginMemberships(skill as Parameters<typeof ListSkillPluginMemberships>[0]),
			'ListSkillPluginMemberships'
		).map((value, index) => artifactPluginMembershipFromWails(value, `ListSkillPluginMemberships[${index}]`));
	}

	async listSkillPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListSkillPlugins(rootID as Parameters<typeof ListSkillPlugins>[0]),
			'ListSkillPlugins'
		).map((value, index) => pluginListItemFromWails(value, `ListSkillPlugins[${index}]`));
	}

	async listSkillPluginsForManagement(): Promise<PluginListItem[]> {
		return wailsObjectArrayOrEmpty(await ListSkillPluginsForManagement(), 'ListSkillPluginsForManagement').map(
			(value, index) => pluginListItemFromWails(value, `ListSkillPluginsForManagement[${index}]`)
		);
	}

	async listSkills(rootID: ArtifactRootID): Promise<StoreSkillListItem[]> {
		return wailsObjectArrayOrEmpty(await ListSkills(rootID as Parameters<typeof ListSkills>[0]), 'ListSkills').map(
			(value, index) => storeSkillListItemFromWails(value, `ListSkills[${index}]`)
		);
	}

	async listSkillsForManagement(): Promise<StoreSkillListItem[]> {
		return wailsObjectArrayOrEmpty(await ListSkillsForManagement(), 'ListSkillsForManagement').map((value, index) =>
			storeSkillListItemFromWails(value, `ListSkillsForManagement[${index}]`)
		);
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

	async removeSkillPluginMember(request: RemovePluginMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await RemoveSkillPluginMember(request as Parameters<typeof RemoveSkillPluginMember>[0]),
			'RemoveSkillPluginMember'
		);
	}

	async resolveSkillCapabilities(skill: ArtifactRef): Promise<CapabilityPlan> {
		return capabilityPlanFromWails(
			await ResolveSkillCapabilities(skill as Parameters<typeof ResolveSkillCapabilities>[0]),
			'ResolveSkillCapabilities'
		);
	}

	async resolveSkillPlugin(plugin: ArtifactRef): Promise<PluginCapabilityPlan> {
		return pluginCapabilityPlanFromWails(
			await ResolveSkillPlugin(plugin as Parameters<typeof ResolveSkillPlugin>[0]),
			'ResolveSkillPlugin'
		);
	}

	async setSkillPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		return pluginViewFromWails(
			await SetSkillPluginEnabled(plugin as Parameters<typeof SetSkillPluginEnabled>[0], expectedRevision, enabled),
			'SetSkillPluginEnabled'
		);
	}

	async setSkillEnabled(skill: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await SetSkillEnabled(skill as Parameters<typeof SetSkillEnabled>[0], expectedRevision, enabled),
			'SetSkillEnabled'
		);
	}

	async updateSkillPlugin(request: UpdatePluginRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await UpdateSkillPlugin(request as Parameters<typeof UpdateSkillPlugin>[0]),
			'UpdateSkillPlugin'
		);
	}
}
