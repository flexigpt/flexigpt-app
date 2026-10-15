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
	SkillArgument,
	SkillDirectoryRegistration,
	SkillDocumentInput,
	SkillPathRegistration,
	SkillPathRegistrationResult,
	StoreManagedSkillDocument,
	StoreSkillListItem,
} from '@/spec/skill';
import type { ResolvedToolView } from '@/spec/tool';
import { ArtifactState, CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';
import { SkillInsert } from '@/spec/skill';

import { mapWithConcurrency } from '@/lib/async_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type { ISkillAggregateAPI, ISkillStoreAPI, IToolTargetResolver } from '@/apis/interface';
import type { ModelManagementAPI, ModelManagementItem } from '@/apis/model_management';

import {
	canManageSkillPlugin,
	canWriteSkillPlugin,
	sameSkillArtifact,
	skillArtifactKey,
} from '@/skills/lib/skill_management_utils';

// oxlint-disable typescript/parameter-properties

export interface SkillManagementPresentation {
	summary?: ArtifactSkillSummary;
	document?: SkillDocumentInput;
	memberships?: ArtifactPluginMembershipView[];
	location?: string;
	errors: string[];
}

export interface SkillPluginSkills {
	plugin: PluginView;
	skills: StoreSkillListItem[];
	diagnostics: string[];
}

export interface SkillRegistrationOutcome {
	source: StoreArtifactSourceSummary;
	attachedCount: number;
	warnings: string[];
}

export interface SkillRemovalOutcome {
	detached: boolean;
	purged: boolean;
	warning?: string;
}

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
	insert: SkillInsert | undefined;
	arguments?: SkillArgument[];
	tags?: string[];
	document?: SkillDocumentInput;
	metadataError?: string;
	hasArguments: boolean;
	hasResources: boolean;
	runtimeResolved: boolean;
	runtimeError?: string;
}

export interface SkillManagementPageData {
	plugins: PluginListItem[];
	skills: StoreSkillListItem[];
	presentations: Record<string, SkillManagementPresentation>;
}

const errorText = (error: unknown, fallback: string) =>
	error instanceof Error && error.message.trim() ? error.message : fallback;

export class SkillManagementAPI {
	constructor(
		private readonly store: ISkillStoreAPI,
		private readonly aggregate: ISkillAggregateAPI,
		private readonly tools: IToolTargetResolver,
		private readonly models: ModelManagementAPI
	) {}

	private readonly presentationLoads = new Map<string, Promise<SkillManagementPresentation>>();

	private readonly composerSkillCatalog = createSharedAsyncCatalog<ComposerSkillCatalogItem[]>(async () => {
		const { skills, plugins, presentations } = await this.loadManagementPageData();
		const enabledPlans = await mapWithConcurrency(
			plugins.filter(plugin => plugin.enabled && plugin.state === ArtifactState.Available),
			8,
			plugin => this.store.resolveSkillPlugin(plugin.ref)
		);
		const eligible = new Set<string>();
		for (const plan of enabledPlans) {
			for (const occurrence of plan.occurrences) {
				if (
					occurrence.type === 'skill' &&
					occurrence.status === CapabilityResolutionStatus.Available &&
					occurrence.target?.form === CapabilityTargetForm.Artifact &&
					occurrence.target.artifact
				) {
					eligible.add(skillArtifactKey(occurrence.target.artifact));
				}
			}
		}

		const values = await mapWithConcurrency(skills, 8, async skill => {
			const presentation = presentations[skillArtifactKey(skill.ref)];
			const summary = presentation?.summary;
			const document = presentation?.document;
			return {
				ref: skill.ref,
				name: skill.name,
				displayName: skill.displayName || skill.name,
				description: skill.description,
				state: skill.state,
				enabled: skill.enabled && Boolean(summary?.IsEnabled) && eligible.has(skillArtifactKey(skill.ref)),
				builtIn: skill.builtIn,
				managed: skill.managed,
				revision: skill.revision,
				definitionDigest: skill.definitionDigest,
				insert: summary?.Insert ?? document?.insert,
				hasArguments: summary?.HasArguments ?? Boolean(document?.arguments?.length),
				hasResources: summary?.HasResources ?? false,
				runtimeResolved: Boolean(summary),
				runtimeError: !summary
					? presentation?.errors.join('\n') || 'Skill interpretation is unavailable.'
					: !eligible.has(skillArtifactKey(skill.ref))
						? 'No enabled Plugin currently resolves this Skill.'
						: undefined,
				arguments: document?.arguments,
				tags: document?.tags,
				document,
				metadataError: presentation?.errors.length ? presentation.errors.join('\n') : undefined,
			} satisfies ComposerSkillCatalogItem;
		});

		return values.toSorted((left, right) => {
			const display = left.displayName.localeCompare(right.displayName, undefined, {
				numeric: true,
				sensitivity: 'base',
			});

			return display !== 0 ? display : left.name.localeCompare(right.name, undefined, { sensitivity: 'base' });
		});
	});

	invalidateComposerSkillsCatalog(): void {
		this.composerSkillCatalog.invalidate();
		this.presentationLoads.clear();
	}

	listComposerSkillCatalog(force = false): Promise<ComposerSkillCatalogItem[]> {
		return this.composerSkillCatalog.load(force);
	}

	private async mutate<T>(operation: () => Promise<T>): Promise<T> {
		try {
			return await operation();
		} finally {
			// A multi-step operation can fail after a durable mutation.
			this.invalidateComposerSkillsCatalog();
		}
	}

	async loadManagementPageData(): Promise<SkillManagementPageData> {
		const [plugins, skills] = await Promise.all([
			this.store.listSkillPluginsForManagement(),
			this.store.listSkillsForManagement(),
		]);

		return {
			plugins,
			skills,
			presentations: await this.loadSkillPresentations(skills),
		};
	}

	async loadSkillPresentations(skills: StoreSkillListItem[]): Promise<Record<string, SkillManagementPresentation>> {
		const values = await mapWithConcurrency(skills, 8, async skill => ({
			key: skillArtifactKey(skill.ref),
			value: await this.getSkillPresentation(skill),
		}));
		return Object.fromEntries(values.map(value => [value.key, value.value]));
	}

	getSkillPresentation(skill: StoreSkillListItem): Promise<SkillManagementPresentation> {
		const key = JSON.stringify([
			skill.ref.rootID,
			skill.ref.artifactID,
			skill.revision,
			skill.definitionDigest,
			skill.enabled,
			skill.state,
		]);
		const existing = this.presentationLoads.get(key);
		if (existing) {
			return existing;
		}

		const load = (async (): Promise<SkillManagementPresentation> => {
			const [summary, document, memberships, resolved] = await Promise.allSettled([
				this.aggregate.describeArtifactSkill(skill.ref),
				skill.managed ? this.store.getManagedSkillDocument(skill.ref) : Promise.resolve(null),
				this.store.listSkillPluginMemberships(skill.ref),
				this.aggregate.resolveArtifactSkill(skill.ref),
			]);
			const result: SkillManagementPresentation = { errors: [] };

			if (summary.status === 'fulfilled') {
				result.summary = summary.value;
			} else {
				result.errors.push(errorText(summary.reason, 'Skill interpretation could not be loaded.'));
			}
			if (document.status === 'fulfilled' && document.value) {
				if (document.value.artifact.revision === skill.revision) {
					result.document = document.value.document;
				} else {
					result.errors.push('The Skill changed while its document was loading. Refresh to inspect it.');
				}
			} else if (document.status === 'rejected') {
				result.errors.push(errorText(document.reason, 'Managed document could not be loaded.'));
			}
			if (memberships.status === 'fulfilled') {
				result.memberships = memberships.value;
			} else {
				result.errors.push(errorText(memberships.reason, 'Plugin memberships could not be loaded.'));
			}
			if (resolved.status === 'fulfilled') {
				result.location = resolved.value.Definition.location;
			}
			return result;
		})();

		this.presentationLoads.set(key, load);
		return load;
	}

	async getSkillPluginSkills(plugin: ArtifactRef, listedSkills?: StoreSkillListItem[]): Promise<SkillPluginSkills> {
		const [view, plan, allSkills] = await Promise.all([
			this.store.getSkillPlugin(plugin),
			this.store.resolveSkillPlugin(plugin),
			listedSkills ? Promise.resolve(listedSkills) : this.store.listSkillsForManagement(),
		]);
		if (view.artifact.revision !== plan.plugin.artifact.revision) {
			throw new Error('The Plugin changed while its membership was loading. Refresh and try again.');
		}

		const selected = new Set<string>();
		const diagnostics: string[] = [];
		for (const occurrence of plan.occurrences) {
			if (
				occurrence.type === 'skill' &&
				occurrence.target?.form === CapabilityTargetForm.Artifact &&
				occurrence.target.artifact
			) {
				selected.add(skillArtifactKey(occurrence.target.artifact));
			}
			if (occurrence.status !== CapabilityResolutionStatus.Available) {
				diagnostics.push(occurrence.message || `${occurrence.name || occurrence.path}: ${occurrence.status}`);
			}
		}
		if (!plan.complete && diagnostics.length === 0) {
			diagnostics.push('Plugin capability resolution is incomplete.');
		}

		// Failures here must not turn into an empty, apparently deletable Plugin.
		const memberships = await mapWithConcurrency(allSkills, 8, async skill => ({
			skill,
			values: await this.store.listSkillPluginMemberships(skill.ref),
		}));
		for (const membership of memberships) {
			if (membership.values.some(value => sameSkillArtifact(value.plugin, plugin))) {
				selected.add(skillArtifactKey(membership.skill.ref));
			}
		}

		return {
			plugin: view,
			skills: allSkills
				.filter(skill => selected.has(skillArtifactKey(skill.ref)))
				.toSorted((left, right) =>
					(left.displayName || left.name).localeCompare(right.displayName || right.name, undefined, {
						numeric: true,
						sensitivity: 'base',
					})
				),
			diagnostics,
		};
	}

	private async getMutablePlugin(
		plugin: ArtifactRef,
		expectedRevision: number,
		requireEnabled: boolean
	): Promise<PluginListItem> {
		const plugins = await this.store.listSkillPluginsForManagement();
		const current = plugins.find(value => sameSkillArtifact(value.ref, plugin));
		if (!current) {
			throw new Error('Skill Plugin is unavailable.');
		}
		if (current.revision !== expectedRevision) {
			throw new Error('Skill Plugin changed. Refresh before applying this change.');
		}
		if (requireEnabled ? !canWriteSkillPlugin(current) : !canManageSkillPlugin(current)) {
			throw new Error(
				requireEnabled
					? 'An available, enabled, editable Skill Plugin is required.'
					: 'This Skill Plugin does not allow membership changes.'
			);
		}
		return current;
	}

	async refreshSkillPluginSources(plugin: ArtifactRef): Promise<void> {
		const data = await this.getSkillPluginSkills(plugin);
		const listedPlugins = await this.store.listSkillPluginsForManagement();
		const item = listedPlugins.find(value => sameSkillArtifact(value.ref, plugin));
		const sources = new Map<string, { rootID: string; sourceID: string }>();
		if (item && !item.builtIn) {
			sources.set(JSON.stringify([item.ref.rootID, item.sourceID]), {
				rootID: item.ref.rootID,
				sourceID: item.sourceID,
			});
		}
		const artifacts = await mapWithConcurrency(
			data.skills.filter(skill => !skill.builtIn),
			8,
			skill => this.store.getSkill(skill.ref)
		);
		for (const artifact of artifacts) {
			sources.set(JSON.stringify([artifact.rootID, artifact.binding.sourceID]), {
				rootID: artifact.rootID,
				sourceID: artifact.binding.sourceID,
			});
		}
		await this.mutate(async () => {
			for (const source of sources.values()) {
				await this.store.refreshSkillSource(source.rootID, source.sourceID);
			}
		});
	}

	async refreshAllSkillSources(): Promise<void> {
		const [plugins, skills] = await Promise.all([
			this.store.listSkillPluginsForManagement(),
			this.store.listSkillsForManagement(),
		]);
		const sources = new Map<string, { rootID: string; sourceID: string }>();
		for (const plugin of plugins.filter(value => !value.builtIn)) {
			sources.set(JSON.stringify([plugin.ref.rootID, plugin.sourceID]), {
				rootID: plugin.ref.rootID,
				sourceID: plugin.sourceID,
			});
		}
		const artifacts = await mapWithConcurrency(
			skills.filter(skill => !skill.builtIn),
			8,
			skill => this.store.getSkill(skill.ref)
		);
		for (const artifact of artifacts) {
			sources.set(JSON.stringify([artifact.rootID, artifact.binding.sourceID]), {
				rootID: artifact.rootID,
				sourceID: artifact.binding.sourceID,
			});
		}
		await this.mutate(async () => {
			for (const source of sources.values()) {
				await this.store.refreshSkillSource(source.rootID, source.sourceID);
			}
		});
	}

	async registerSkillDirectoryAndAttach(
		request: SkillDirectoryRegistration,
		destination?: PluginListItem
	): Promise<SkillRegistrationOutcome> {
		if (destination) {
			await this.getMutablePlugin(destination.ref, destination.revision, true);
			if (destination.ref.rootID !== request.rootID) {
				throw new Error('Directory and destination Plugin must use the same Root.');
			}
		}

		return this.mutate(async () => {
			const source = await this.store.registerSkillDirectory(request);
			const outcome: SkillRegistrationOutcome = { source, attachedCount: 0, warnings: [] };
			try {
				await this.store.refreshSkillSource(request.rootID, source.id);
				if (!destination) {
					return outcome;
				}
				const skills = await this.store.listSkills(request.rootID);
				const artifacts = await mapWithConcurrency(skills, 8, skill => this.store.getSkill(skill.ref));
				let revision = destination.revision;
				for (const artifact of artifacts.filter(value => value.binding.sourceID === source.id)) {
					const ref = { rootID: artifact.rootID, artifactID: artifact.id };
					const memberships = await this.store.listSkillPluginMemberships(ref);
					if (memberships.some(value => sameSkillArtifact(value.plugin, destination.ref))) {
						continue;
					}
					const updated = await this.store.attachSkillArtifactToPlugin({
						plugin: destination.ref,
						expectedRevision: revision,
						artifact: ref,
					});
					revision = updated.artifact.revision;
					outcome.attachedCount += 1;
				}
			} catch (error) {
				outcome.warnings.push(
					`Directory registered, but discovery or attachment did not finish: ${
						error instanceof Error ? error.message : 'unknown error'
					}. Refresh and use Attach Existing for remaining Artifacts.`
				);
			}
			return outcome;
		});
	}

	async addSkillPathAndAttach(
		plugin: PluginListItem,
		path: string
	): Promise<{ artifact: StoreArtifact; attached: boolean; warning?: string }> {
		await this.getMutablePlugin(plugin.ref, plugin.revision, true);
		return this.mutate(async () => {
			const registered = await this.store.addSkillPath({
				rootID: plugin.ref.rootID,
				path,
				sourceDisplayName: `${plugin.displayName || plugin.name} external Skill`,
				enabled: true,
			});
			try {
				await this.store.attachSkillArtifactToPlugin({
					plugin: plugin.ref,
					expectedRevision: plugin.revision,
					artifact: { rootID: registered.artifact.rootID, artifactID: registered.artifact.id },
				});
				return { artifact: registered.artifact, attached: true };
			} catch (error) {
				return {
					artifact: registered.artifact,
					attached: false,
					warning: `Path registered, but Plugin attachment failed: ${
						error instanceof Error ? error.message : 'unknown error'
					}. Use Attach Existing rather than registering the path again.`,
				};
			}
		});
	}

	async detachSkillFromPlugin(plugin: PluginListItem, skill: StoreSkillListItem): Promise<void> {
		await this.getMutablePlugin(plugin.ref, plugin.revision, false);
		const memberships = await this.store.listSkillPluginMemberships(skill.ref);
		const membership = memberships.find(value => sameSkillArtifact(value.plugin, plugin.ref));
		if (!membership) {
			throw new Error('This Skill is not a direct member. Inspect the Plugin declaration instead.');
		}
		const view = await this.store.getSkillPlugin(plugin.ref);
		if (view.artifact.revision !== plugin.revision || membership.pluginRevision !== plugin.revision) {
			throw new Error('Plugin membership changed. Refresh before detaching.');
		}
		if (view.entries[membership.memberIndex]?.selector) {
			throw new Error('This Skill is selected by a declaration. Remove that declaration in Plugin Details.');
		}
		await this.removeSkillPluginMember({
			plugin: plugin.ref,
			expectedRevision: plugin.revision,
			index: membership.memberIndex,
		});
	}

	async deleteSkillFromPlugin(plugin: PluginListItem, skill: StoreSkillListItem): Promise<SkillRemovalOutcome> {
		const current = (await this.store.listSkills(skill.ref.rootID)).find(value =>
			sameSkillArtifact(value.ref, skill.ref)
		);
		if (!current || current.revision !== skill.revision) {
			throw new Error('Skill changed or disappeared. Refresh before deleting.');
		}
		if (current.builtIn) {
			throw new Error('Built-in Skills cannot be deleted.');
		}
		await this.detachSkillFromPlugin(plugin, current);
		if (!current.managed) {
			return { detached: true, purged: false };
		}
		try {
			const remaining = await this.store.listSkillPluginMemberships(current.ref);
			if (remaining.length > 0) {
				return { detached: true, purged: false };
			}
			await this.purgeSkill(current.ref, current.revision);
			return { detached: true, purged: true };
		} catch (error) {
			return {
				detached: true,
				purged: false,
				warning: `Skill removed from the Plugin, but managed cleanup did not finish: ${
					error instanceof Error ? error.message : 'unknown error'
				}. The Artifact remains available under Unattached Skills.`,
			};
		}
	}

	async readManagedInstructionSource(skill: StoreSkillListItem): Promise<SkillDocumentInput> {
		if (!skill.managed) {
			throw new Error('This API surface does not expose a copyable document for this external or built-in Skill.');
		}
		const [managed, summary] = await Promise.all([
			this.store.getManagedSkillDocument(skill.ref),
			this.aggregate.describeArtifactSkill(skill.ref),
		]);
		if (managed.artifact.revision !== skill.revision) {
			throw new Error('Skill changed. Refresh before copying its instructions.');
		}
		if (summary.Insert !== SkillInsert.Instructions || summary.HasArguments || summary.HasResources) {
			throw new Error('Only instruction Skills without arguments or resources can be copied as plain instructions.');
		}
		return managed.document;
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
		return this.mutate(async () => {
			await this.getMutablePlugin(request.plugin, request.expectedPluginRevision, true);
			const summary = await this.aggregate.describeArtifactSkill(request.artifact);
			if (summary.HasResources) {
				throw new Error(
					'This Skill contains package resources. Document-only replacement is disabled; fork its document instead.'
				);
			}
			// If resource interpretation fails, do not guess that the package is empty.
			return this.store.replaceManagedSkill(request);
		});
	}

	createSkillPlugin(request: CreatePluginRequest): Promise<PluginView> {
		return this.mutate(async () => {
			if (request.rootID) {
				const plugins = await this.store.listSkillPluginsForManagement();
				if (!plugins.some(plugin => plugin.ref.rootID === request.rootID && plugin.baseline && !plugin.builtIn)) {
					throw new Error('The selected Root has no user Skill baseline Plugin.');
				}
			}
			return this.store.createSkillPlugin(request);
		});
	}

	deleteSkillPlugin(request: DeletePluginRequest): Promise<void> {
		return this.mutate(async () => {
			const view = await this.store.getSkillPlugin(request.plugin);
			if (view.artifact.revision !== request.expectedRevision) {
				throw new Error('Plugin changed. Refresh before deleting.');
			}
			if (!view.deletable || view.baseline || view.members.length > 0 || view.entries.length > 0) {
				throw new Error('Only an empty, deletable Skill Plugin can be deleted.');
			}
			await this.store.deleteSkillPlugin(request);
		});
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
		return this.mutate(async () => {
			const current = (await this.store.listSkills(skill.rootID)).find(value => sameSkillArtifact(value.ref, skill));
			if (!current || current.revision !== expectedRevision) {
				throw new Error('Skill changed or disappeared. Refresh before purging.');
			}
			if (!current.managed || current.builtIn) {
				throw new Error('Only a user-managed Skill Artifact can be purged.');
			}
			const memberships = await this.store.listSkillPluginMemberships(skill);
			if (memberships.length > 0) {
				throw new Error('This Skill is still used by a Plugin. Remove its memberships before purging.');
			}
			await this.store.purgeSkill(skill, expectedRevision);
		});
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
