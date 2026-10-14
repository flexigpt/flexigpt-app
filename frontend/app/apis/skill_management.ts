// oxlint-disable typescript/parameter-properties
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
	ArtifactSkillFilter,
	ArtifactSkillSummary,
	ManagedSkillCreateRequest,
	ManagedSkillCreateResult,
	ManagedSkillReplaceRequest,
	ManagedSkillReplaceResult,
	ResolvedArtifactSkill,
	SkillDirectoryRegistration,
	SkillPathRegistration,
	SkillPathRegistrationResult,
	StoreManagedSkillDocument,
	StoreSkillListItem,
} from '@/spec/skill';
import type { ResolvedToolView } from '@/spec/tool';

import type { ISkillAggregateAPI, ISkillStoreAPI, IToolTargetResolver } from '@/apis/interface';
import type { ModelManagementAPI, ModelManagementItem } from '@/apis/model_management';

export interface SkillManagementPageData {
	plugins: PluginListItem[];
	skills: StoreSkillListItem[];
}

/**
 * Management façade for Skills and Skill Plugins.
 *
 * Session creation, Skill Runtime session ownership, lifecycle tools, and
 * runtime tool invocation are intentionally absent. Those operations are now
 * Run-owned and are accessed only through Session Runtime commands.
 */
export class SkillManagementAPI {
	constructor(
		private readonly store: ISkillStoreAPI,
		private readonly aggregate: ISkillAggregateAPI,
		private readonly tools: IToolTargetResolver,
		private readonly models: ModelManagementAPI
	) {}

	async loadManagementPageData(): Promise<SkillManagementPageData> {
		const [plugins, skills] = await Promise.all([
			this.store.listSkillPluginsForManagement(),
			this.store.listSkillsForManagement(),
		]);

		return {
			plugins,
			skills,
		};
	}

	getArtifactSkillsPrompt(filter: ArtifactSkillFilter): Promise<string> {
		return this.aggregate.getArtifactSkillsPrompt(filter);
	}

	resolveArtifactSkill(skill: ArtifactRef): Promise<ResolvedArtifactSkill> {
		return this.aggregate.resolveArtifactSkill(skill);
	}

	resolveArtifactSkills(skills: ArtifactRef[]): Promise<ResolvedArtifactSkill[]> {
		return this.aggregate.resolveArtifactSkills(skills);
	}

	listArtifactSkillRefs(filter: ArtifactSkillFilter): Promise<ArtifactRef[]> {
		return this.aggregate.listArtifactSkillRefs(filter);
	}

	describeArtifactSkill(skill: ArtifactRef): Promise<ArtifactSkillSummary> {
		return this.aggregate.describeArtifactSkill(skill);
	}

	addSkillPluginMember(request: AddPluginMemberRequest): Promise<PluginView> {
		return this.store.addSkillPluginMember(request);
	}

	addSkillPath(request: SkillPathRegistration): Promise<SkillPathRegistrationResult> {
		return this.store.addSkillPath(request);
	}

	attachSkillArtifactToPlugin(request: AddPluginArtifactMemberRequest): Promise<PluginView> {
		return this.store.attachSkillArtifactToPlugin(request);
	}

	createManagedSkill(request: ManagedSkillCreateRequest): Promise<ManagedSkillCreateResult> {
		return this.store.createManagedSkill(request);
	}

	replaceManagedSkill(request: ManagedSkillReplaceRequest): Promise<ManagedSkillReplaceResult> {
		return this.store.replaceManagedSkill(request);
	}

	createSkillPlugin(request: CreatePluginRequest): Promise<PluginView> {
		return this.store.createSkillPlugin(request);
	}

	deleteSkillPlugin(request: DeletePluginRequest): Promise<void> {
		return this.store.deleteSkillPlugin(request);
	}

	getManagedSkillDocument(skill: ArtifactRef): Promise<StoreManagedSkillDocument> {
		return this.store.getManagedSkillDocument(skill);
	}

	getSkill(skill: ArtifactRef): Promise<StoreArtifact> {
		return this.store.getSkill(skill);
	}

	getSkillPlugin(plugin: ArtifactRef): Promise<PluginView> {
		return this.store.getSkillPlugin(plugin);
	}

	listSkillPluginMemberships(skill: ArtifactRef): Promise<ArtifactPluginMembershipView[]> {
		return this.store.listSkillPluginMemberships(skill);
	}

	listSkillPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]> {
		return this.store.listSkillPlugins(rootID);
	}

	listSkillPluginsForManagement(): Promise<PluginListItem[]> {
		return this.store.listSkillPluginsForManagement();
	}

	listSkills(rootID: ArtifactRootID): Promise<StoreSkillListItem[]> {
		return this.store.listSkills(rootID);
	}

	listSkillsForManagement(): Promise<StoreSkillListItem[]> {
		return this.store.listSkillsForManagement();
	}

	purgeSkill(skill: ArtifactRef, expectedRevision: number): Promise<void> {
		return this.store.purgeSkill(skill, expectedRevision);
	}

	refreshSkillSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<void> {
		return this.store.refreshSkillSource(rootID, sourceID);
	}

	registerSkillDirectory(request: SkillDirectoryRegistration): Promise<StoreArtifactSourceSummary> {
		return this.store.registerSkillDirectory(request);
	}

	removeSkillPluginMember(request: RemovePluginMemberRequest): Promise<PluginView> {
		return this.store.removeSkillPluginMember(request);
	}

	resolveSkillCapabilities(skill: ArtifactRef): Promise<CapabilityPlan> {
		return this.store.resolveSkillCapabilities(skill);
	}

	resolveSkillPlugin(plugin: ArtifactRef): Promise<PluginCapabilityPlan> {
		return this.store.resolveSkillPlugin(plugin);
	}

	setSkillEnabled(skill: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return this.store.setSkillEnabled(skill, expectedRevision, enabled);
	}

	setSkillPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		return this.store.setSkillPluginEnabled(plugin, expectedRevision, enabled);
	}

	updateSkillPlugin(request: UpdatePluginRequest): Promise<PluginView> {
		return this.store.updateSkillPlugin(request);
	}

	resolveToolTarget(target: Parameters<IToolTargetResolver['resolveToolTarget']>[0]): Promise<ResolvedToolView> {
		return this.tools.resolveToolTarget(target);
	}

	resolveMappedModelTarget(
		target: Parameters<ModelManagementAPI['resolveMappedModelTarget']>[0]
	): Promise<ModelManagementItem> {
		return this.models.resolveMappedModelTarget(target);
	}
}
