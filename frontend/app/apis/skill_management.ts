// oxlint-disable typescript/parameter-properties
import type {
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactState,
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
	SkillInsert,
	SkillPathRegistration,
	SkillPathRegistrationResult,
	StoreManagedSkillDocument,
	StoreSkillListItem,
} from '@/spec/skill';
import type { ResolvedToolView } from '@/spec/tool';

import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type { ISkillAggregateAPI, ISkillStoreAPI, IToolTargetResolver } from '@/apis/interface';
import type { ModelManagementAPI, ModelManagementItem } from '@/apis/model_management';

export interface ComposerSkillCatalogItem {
	ref: ArtifactRef;
	name: string;
	displayName: string;
	description?: string;
	state: ArtifactState;
	enabled: boolean;
	builtIn: boolean;
	managed: boolean;
	revision: number;
	definitionDigest?: string;
	insert: SkillInsert;
	hasArguments: boolean;
	hasResources: boolean;
}

export interface SkillManagementPageData {
	plugins: PluginListItem[];
	skills: StoreSkillListItem[];
}

export class SkillManagementAPI {
	private readonly composerSkillCatalog = createSharedAsyncCatalog<ComposerSkillCatalogItem[]>(async () => {
		const skills = await this.store.listSkillsForManagement();

		const values = await Promise.all(
			skills.map(async skill => {
				try {
					const summary = await this.aggregate.describeArtifactSkill(skill.ref);

					return {
						ref: skill.ref,
						name: skill.name,
						displayName: skill.displayName || skill.name,
						description: skill.description,
						state: skill.state,
						enabled: skill.enabled,
						builtIn: skill.builtIn,
						managed: skill.managed,
						revision: skill.revision,
						definitionDigest: skill.definitionDigest,
						insert: summary.Insert,
						hasArguments: summary.HasArguments,
						hasResources: summary.HasResources,
					} satisfies ComposerSkillCatalogItem;
				} catch {
					return {
						ref: skill.ref,
						name: skill.name,
						displayName: skill.displayName || skill.name,
						description: skill.description,
						state: skill.state,
						enabled: skill.enabled,
						builtIn: skill.builtIn,
						managed: skill.managed,
						revision: skill.revision,
						definitionDigest: skill.definitionDigest,
						insert: 'instructions' as SkillInsert,
						hasArguments: false,
						hasResources: false,
					} satisfies ComposerSkillCatalogItem;
				}
			})
		);

		return values.toSorted((left, right) => {
			const display = left.displayName.localeCompare(right.displayName, undefined, {
				numeric: true,
				sensitivity: 'base',
			});

			return display !== 0 ? display : left.name.localeCompare(right.name, undefined, { sensitivity: 'base' });
		});
	});

	constructor(
		private readonly store: ISkillStoreAPI,
		private readonly aggregate: ISkillAggregateAPI,
		private readonly tools: IToolTargetResolver,
		private readonly models: ModelManagementAPI
	) {}

	invalidateComposerSkillsCatalog(): void {
		this.composerSkillCatalog.invalidate();
	}

	listComposerSkillCatalog(force = false): Promise<ComposerSkillCatalogItem[]> {
		return this.composerSkillCatalog.load(force);
	}

	private async mutate<T>(operation: () => Promise<T>): Promise<T> {
		const result = await operation();
		this.invalidateComposerSkillsCatalog();
		return result;
	}

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
		return this.mutate(() => this.store.addSkillPluginMember(request));
	}

	addSkillPath(request: SkillPathRegistration): Promise<SkillPathRegistrationResult> {
		return this.mutate(() => this.store.addSkillPath(request));
	}

	attachSkillArtifactToPlugin(request: AddPluginArtifactMemberRequest): Promise<PluginView> {
		return this.mutate(() => this.store.attachSkillArtifactToPlugin(request));
	}

	createManagedSkill(request: ManagedSkillCreateRequest): Promise<ManagedSkillCreateResult> {
		return this.mutate(() => this.store.createManagedSkill(request));
	}

	replaceManagedSkill(request: ManagedSkillReplaceRequest): Promise<ManagedSkillReplaceResult> {
		return this.mutate(() => this.store.replaceManagedSkill(request));
	}

	createSkillPlugin(request: CreatePluginRequest): Promise<PluginView> {
		return this.mutate(() => this.store.createSkillPlugin(request));
	}

	deleteSkillPlugin(request: DeletePluginRequest): Promise<void> {
		return this.mutate(() => this.store.deleteSkillPlugin(request));
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
		return this.mutate(() => this.store.purgeSkill(skill, expectedRevision));
	}

	refreshSkillSource(rootID: ArtifactRootID, sourceID: ArtifactSourceID): Promise<void> {
		return this.mutate(() => this.store.refreshSkillSource(rootID, sourceID));
	}

	registerSkillDirectory(request: SkillDirectoryRegistration): Promise<StoreArtifactSourceSummary> {
		return this.mutate(() => this.store.registerSkillDirectory(request));
	}

	removeSkillPluginMember(request: RemovePluginMemberRequest): Promise<PluginView> {
		return this.mutate(() => this.store.removeSkillPluginMember(request));
	}

	resolveSkillCapabilities(skill: ArtifactRef): Promise<CapabilityPlan> {
		return this.store.resolveSkillCapabilities(skill);
	}

	resolveSkillPlugin(plugin: ArtifactRef): Promise<PluginCapabilityPlan> {
		return this.store.resolveSkillPlugin(plugin);
	}

	setSkillEnabled(skill: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return this.mutate(() => this.store.setSkillEnabled(skill, expectedRevision, enabled));
	}

	setSkillPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		return this.mutate(() => this.store.setSkillPluginEnabled(plugin, expectedRevision, enabled));
	}

	updateSkillPlugin(request: UpdatePluginRequest): Promise<PluginView> {
		return this.mutate(() => this.store.updateSkillPlugin(request));
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
