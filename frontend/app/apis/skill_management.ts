import type { ArtifactRef, CapabilityTarget, StoreArtifact } from '@/spec/artifact';
import type { PluginListItem, PluginView } from '@/spec/plugin';
import type {
	ArtifactRuntimeSkillListItem,
	ArtifactSkillFilter,
	ArtifactSkillSession,
	ArtifactSkillSessionOptions,
	ArtifactSkillSummary,
	InvokeSkillToolResponse,
	ManagedSkillDocumentView,
	ManagedSkillReplaceRequest,
	RenderSkillResponse,
	ResolvedArtifactSkill,
	RuntimeSkillDefinition,
	RuntimeSkillRecord,
	RuntimeSkillRenderResult,
	RuntimeSkillSessionOptions,
	Skill,
	SkillArtifactCreateInput,
	SkillArtifactView,
	SkillDocumentInput,
	SkillListItem,
	SkillManagementView,
	SkillPlugin,
	SkillPluginManagementView,
	SkillPresenceStatus,
	StoreSkillListItem,
} from '@/spec/skill';
import type { ResolvedToolView } from '@/spec/tool';
import { ArtifactAdoptionMode, ArtifactState, CapabilityResolutionStatus, CapabilityTargetForm } from '@/spec/artifact';
import {
	RuntimeSkillActivity,
	SkillInsert,
	SkillPluginAttachmentRole,
	SkillPresenceStatus as SkillPresenceStatusValue,
	SkillType,
} from '@/spec/skill';

import type { JSONRawString } from '@/lib/jsonschema_utils';
import { mapWithConcurrency } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type { ISkillAggregateAPI, ISkillRuntimeAPI, ISkillStoreAPI, IToolTargetResolver } from '@/apis/interface';
import type { ModelManagementAPI, ModelManagementItem } from '@/apis/model_management';

// oxlint-disable typescript/parameter-properties

const SKILL_READ_CONCURRENCY = 4;

interface SkillPluginRef {
	ref: ArtifactRef;
	revision: number;
	enabled: boolean;
	baseline: boolean;
	editable: boolean;
	deletable: boolean;
	builtIn?: boolean;
	name: string;
	displayName: string;
	description?: string;
	sourceID?: string;
}

interface ResolvedRuntimeArtifactSet {
	definitions: RuntimeSkillDefinition[];
	byArtifactKey: Map<string, ResolvedArtifactSkill>;
	artifactByDefinitionKey: Map<string, ArtifactRef>;
}

interface PrefetchedRuntimeMetadata {
	record?: RuntimeSkillRecord;
	error?: string;
}

export interface SkillManagementPageData {
	skillPlugins: SkillPlugin[];
	skillListItems: SkillListItem[];
}

function artifactRefKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function runtimeDefinitionKey(definition: RuntimeSkillDefinition): string {
	return `${definition.type}\u0000${definition.name}\u0000${definition.location}`;
}

function emptyResources(): Skill['resources'] {
	return {
		hasResources: false,
		totalCount: 0,
		moreLocations: false,
	};
}

function pluginRef(plugin: SkillPluginRef | PluginView): ArtifactRef {
	if ('ref' in plugin) {
		return plugin.ref;
	}

	return {
		rootID: plugin.artifact.rootID,
		artifactID: plugin.artifact.id,
	};
}

function isUserMutablePlugin(plugin: SkillPluginRef | PluginView): boolean {
	return plugin.baseline || plugin.editable;
}

function presenceStatusFor(artifact: Pick<StoreSkillListItem, 'state'>): SkillPresenceStatus {
	switch (artifact.state) {
		case ArtifactState.Available:
			return SkillPresenceStatusValue.Present;
		case ArtifactState.Missing:
			return SkillPresenceStatusValue.Missing;
		case ArtifactState.Invalid:
		case ArtifactState.Incompatible:
			return SkillPresenceStatusValue.Error;
		default:
			return SkillPresenceStatusValue.Unknown;
	}
}

function toSkillPlugin(plugin: PluginListItem): SkillPlugin {
	const builtIn = plugin.builtIn;

	return {
		schemaVersion: 'plugin/v1',
		id: plugin.ref.artifactID,
		rootID: plugin.ref.rootID,
		ref: {
			rootID: plugin.ref.rootID,
			artifactID: plugin.ref.artifactID,
		},
		revision: plugin.revision,
		slug: plugin.name,
		displayName: plugin.displayName || plugin.name,
		description: plugin.description,
		managedSourceID: plugin.sourceID,
		isEnabled: plugin.enabled,
		isBuiltIn: builtIn,
		isEditable: isUserMutablePlugin(plugin),
		isDeletable: isUserMutablePlugin(plugin) && !plugin.baseline && plugin.deletable,
		isBaseline: plugin.baseline,
		sourceID: plugin.sourceID,
		attachments: [
			{
				sourceID: plugin.sourceID,
				revision: plugin.revision,
				role: builtIn ? SkillPluginAttachmentRole.BuiltIn : SkillPluginAttachmentRole.Managed,
				enabled: plugin.enabled,
			},
		],
	};
}

function toManagedSkillArtifactView(artifact: StoreArtifact): SkillArtifactView {
	return {
		artifact: {
			rootID: artifact.rootID,
			artifactID: artifact.id,
		},
		address: {
			rootID: artifact.rootID,
			artifactID: artifact.id,
			kind: artifact.kind,
			logicalName: artifact.logicalName,
		},
		revision: artifact.revision,
		name: artifact.logicalName,
		kind: artifact.kind,
		enabled: artifact.enabled,
		adoption: ArtifactAdoptionMode.Pinned,
		state: artifact.state,
		binding: {
			sourceID: artifact.binding.sourceID,
			locator: artifact.binding.locator,
			subresourceLocator: artifact.binding.subresourceLocator,
		},
		definitionDigest: artifact.resolvedDefinition,
		diagnostics: artifact.diagnostics,
		createdAt: artifact.createdAt,
		modifiedAt: artifact.modifiedAt,
	};
}

function storeSkillListItemFromArtifact(
	artifact: StoreArtifact,
	options: {
		builtIn: boolean;
		managed: boolean;
	}
): StoreSkillListItem {
	return {
		ref: {
			rootID: artifact.rootID,
			artifactID: artifact.id,
		},
		name: artifact.logicalName,
		displayName: artifact.displayName,
		state: artifact.state,
		enabled: artifact.enabled,
		revision: artifact.revision,
		definitionDigest: artifact.resolvedDefinition,
		builtIn: options.builtIn,
		managed: options.managed,
	};
}

export class SkillManagementAPI {
	constructor(
		private readonly runtime: ISkillRuntimeAPI,
		private readonly store: ISkillStoreAPI,
		private readonly aggregate: ISkillAggregateAPI,
		private readonly tools: IToolTargetResolver,
		private readonly models: ModelManagementAPI
	) {}

	private readonly composerSkillsCatalog = createSharedAsyncCatalog<SkillListItem[]>(async () => {
		const skills = await this.listSkills(undefined, false, true);

		// User-message Skills are the source for the Template UI. Runtime
		// metadata is required here so an unavailable Skill is never guessed to
		// be an instruction or template Skill.
		return skills.filter(item => {
			const skill = item.skillDefinition;

			return (
				!skill.runtimeError && (skill.insert === SkillInsert.Instructions || skill.insert === SkillInsert.UserMessage)
			);
		});
	});

	invalidateComposerSkillsCatalog(): void {
		this.composerSkillsCatalog.invalidate();
	}

	async createArtifactSkillSession(options: ArtifactSkillSessionOptions): Promise<ArtifactSkillSession> {
		const allowConfigured = options.allowArtifacts !== undefined;
		const allowedRefs = options.allowArtifacts ?? [];
		const allowedKeys = new Set(
			allowedRefs.map(r => {
				return artifactRefKey(r);
			})
		);
		const activeRefs = (options.activeArtifacts ?? []).filter(
			ref => !allowConfigured || allowedKeys.has(artifactRefKey(ref))
		);

		const resolved = await this.resolveRuntimeArtifacts([...allowedRefs, ...activeRefs]);

		const definitionsFor = (refs: ArtifactRef[]): RuntimeSkillDefinition[] => {
			const definitions: RuntimeSkillDefinition[] = [];
			const seen = new Set<string>();

			for (const ref of refs) {
				const value = resolved.byArtifactKey.get(artifactRefKey(ref));

				if (!value) {
					throw new Error(`Runtime definition was not resolved for Skill ${ref.artifactID}.`);
				}

				const key = runtimeDefinitionKey(value.Definition);
				if (seen.has(key)) {
					continue;
				}

				seen.add(key);
				definitions.push(value.Definition);
			}

			return definitions;
		};

		const session = await this.runtime.createSkillSession({
			closeSessionID: options.closeSessionID,
			maxActivePerSession: options.maxActivePerSession,
			allowedSkills: allowConfigured ? definitionsFor(allowedRefs) : undefined,
			activeSkills: definitionsFor(activeRefs),
		} satisfies RuntimeSkillSessionOptions);

		const activeArtifacts: ArtifactRef[] = [];
		const seen = new Set<string>();

		for (const definition of session.activeSkills) {
			const ref = resolved.artifactByDefinitionKey.get(runtimeDefinitionKey(definition));
			if (!ref || seen.has(artifactRefKey(ref))) {
				continue;
			}

			seen.add(artifactRefKey(ref));
			activeArtifacts.push(ref);
		}

		return {
			sessionID: session.sessionID,
			activeArtifacts,
		};
	}

	invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse> {
		return this.runtime.invokeSkillTool(sessionID, toolName, args);
	}

	closeSkillSession(sessionID: string): Promise<void> {
		return this.runtime.closeSkillSession(sessionID);
	}

	describeArtifactSkill(skill: ArtifactRef): Promise<ArtifactSkillSummary> {
		return this.aggregate.describeArtifactSkill(skill);
	}

	async renderArtifactSkill(skill: ArtifactRef, args?: Record<string, string>): Promise<RuntimeSkillRenderResult> {
		const resolved = await this.aggregate.resolveArtifactSkill(skill);
		return this.runtime.renderSkill(resolved.Definition, args);
	}

	async getSkillPluginManagementView(plugin: ArtifactRef): Promise<SkillPluginManagementView> {
		const [view, capabilities] = await Promise.all([
			this.store.getSkillPlugin(plugin),
			this.store.resolveSkillPlugin(plugin),
		]);

		return {
			plugin: view,
			capabilities,
		};
	}

	async getSkillManagementView(skill: ArtifactRef): Promise<SkillManagementView> {
		const [artifact, memberships, capabilities] = await Promise.all([
			this.store.getSkill(skill),
			this.store.listSkillPluginMemberships(skill),
			this.store.resolveSkillCapabilities(skill),
		]);

		try {
			const runtimeSummary = await this.aggregate.describeArtifactSkill(skill);

			return {
				artifact,
				memberships,
				capabilities,
				runtimeSummary,
			};
		} catch (error) {
			return {
				artifact,
				memberships,
				capabilities,
				runtimeError: getErrorMessage(error, 'Skill runtime metadata is unavailable.'),
			};
		}
	}

	async listArtifactRuntimeSkills(filter: ArtifactSkillFilter): Promise<ArtifactRuntimeSkillListItem[]> {
		if (!filter.allowArtifacts?.length) {
			throw new Error('Artifact Skill selection is required.');
		}

		const resolved = await this.resolveRuntimeArtifacts(filter.allowArtifacts);
		const activity = filter.activity ?? RuntimeSkillActivity.Any;

		const records = await this.runtime.listRuntimeSkills({
			types: filter.types,
			inserts: filter.inserts,
			namePrefix: filter.namePrefix,
			locationPrefix: filter.locationPrefix,
			allowSkills: resolved.definitions,
			sessionID: filter.sessionID,
			activity,
		});

		const activeDefinitionKeys = new Set<string>();

		if (filter.sessionID && activity === RuntimeSkillActivity.Any) {
			const activeRecords = await this.runtime.listRuntimeSkills({
				types: filter.types,
				inserts: filter.inserts,
				namePrefix: filter.namePrefix,
				locationPrefix: filter.locationPrefix,
				allowSkills: resolved.definitions,
				sessionID: filter.sessionID,
				activity: RuntimeSkillActivity.Active,
			});

			for (const record of activeRecords) {
				activeDefinitionKeys.add(runtimeDefinitionKey(record.def));
			}
		}

		return records.flatMap(record => {
			const skillRef = resolved.artifactByDefinitionKey.get(runtimeDefinitionKey(record.def));

			if (!skillRef) {
				return [];
			}

			return [
				{
					skillRef,
					type: record.def.type,
					name: record.name,
					displayName: record.displayName,
					description: record.description,
					digest: record.digest,
					insert: record.insert,
					arguments: record.arguments,
					sourceTags: record.tags,
					resources: record.resources,
					rawFrontmatter: record.rawFrontmatter,
					warnings: record.warnings,
					isActive:
						activity === RuntimeSkillActivity.Active ||
						(activity === RuntimeSkillActivity.Any && activeDefinitionKeys.has(runtimeDefinitionKey(record.def))),
				},
			];
		});
	}

	async loadManagementPageData(
		includeDisabled = true,
		includeRuntimeMetadata = true
	): Promise<SkillManagementPageData> {
		const [plugins, artifacts] = await Promise.all([
			this.store.listSkillPluginsForManagement(),
			this.store.listSkillsForManagement(),
		]);
		const skillPlugins = this.skillPluginsFromPlugins(plugins, undefined, includeDisabled);
		const skillListItems = await this.listSkillsFromPlugins(
			skillPlugins,
			plugins,
			artifacts,
			includeDisabled,
			includeRuntimeMetadata
		);

		return {
			skillPlugins,
			skillListItems,
		};
	}

	listComposerSkills(force = false): Promise<SkillListItem[]> {
		return this.composerSkillsCatalog.load(force);
	}

	async listSkillPlugins(pluginIDs?: string[], includeDisabled = true): Promise<SkillPlugin[]> {
		const plugins = await this.store.listSkillPluginsForManagement();

		return this.skillPluginsFromPlugins(plugins, pluginIDs, includeDisabled);
	}

	async listSkills(
		pluginIDs?: string[],
		includeDisabled = true,
		includeRuntimeMetadata = true
	): Promise<SkillListItem[]> {
		const [pluginItems, artifacts] = await Promise.all([
			this.store.listSkillPluginsForManagement(),
			this.store.listSkillsForManagement(),
		]);
		const plugins = this.skillPluginsFromPlugins(pluginItems, pluginIDs, includeDisabled);

		return this.listSkillsFromPlugins(plugins, pluginItems, artifacts, includeDisabled, includeRuntimeMetadata);
	}

	async putSkillPlugin(
		_pluginID: string,
		rootID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const plugins = await this.store.listSkillPluginsForManagement();
		const hasRequestedBaseline =
			rootID !== '' && plugins.some(plugin => plugin.baseline && plugin.ref.rootID === rootID);

		if (rootID !== '' && !hasRequestedBaseline) {
			throw new Error('The selected Root does not have a user Skill baseline Plugin.');
		}

		let created = await this.store.createSkillPlugin({
			rootID,
			name: slug,
			displayName,
			description,
		});

		if (!isEnabled) {
			created = await this.store.setSkillPluginEnabled(pluginRef(created), created.artifact.revision, false);
		}

		this.invalidateComposerSkillsCatalog();
		void created;
	}

	async patchSkillPlugin(pluginID: string, enabled: boolean): Promise<void> {
		const plugin = await this.resolvePluginPlugin(pluginID);

		await this.store.setSkillPluginEnabled(plugin.ref, plugin.revision, enabled);
		this.invalidateComposerSkillsCatalog();
	}

	async updateSkillPluginMetadata(pluginID: string, displayName: string, description?: string): Promise<void> {
		const plugin = await this.requireUserMutablePlugin(pluginID);

		if (plugin.baseline || !plugin.editable) {
			throw new Error('Baseline Skill Plugin metadata is application-managed.');
		}

		await this.store.updateSkillPlugin({
			plugin: plugin.ref,
			expectedRevision: plugin.revision,
			displayName,
			description,
		});
		this.invalidateComposerSkillsCatalog();
	}

	async refreshSkillPlugin(pluginID: string): Promise<void> {
		const plugin = await this.resolvePluginPlugin(pluginID);
		const listedArtifacts = await this.listPluginSkillItemsForRead(plugin);
		const sourceRefs = new Map<string, { rootID: string; sourceID: string }>();

		// oxlint-disable-next-line unicorn/no-immediate-mutation
		sourceRefs.set(`${plugin.ref.rootID}:${plugin.sourceID}`, {
			rootID: plugin.ref.rootID,
			sourceID: plugin.sourceID,
		});

		const artifacts = await mapWithConcurrency(listedArtifacts, SKILL_READ_CONCURRENCY, item =>
			this.store.getSkill(item.ref)
		);

		for (const artifact of artifacts) {
			sourceRefs.set(`${artifact.rootID}:${artifact.binding.sourceID}`, {
				rootID: artifact.rootID,
				sourceID: artifact.binding.sourceID,
			});
		}

		for (const source of [...sourceRefs.values()].toSorted((left, right) => {
			const leftKey = `${left.rootID}:${left.sourceID}`;
			const rightKey = `${right.rootID}:${right.sourceID}`;

			return leftKey.localeCompare(rightKey);
		})) {
			await this.store.refreshSkillSource(source.rootID, source.sourceID);
		}

		this.invalidateComposerSkillsCatalog();
	}

	async deleteSkillPlugin(pluginID: string): Promise<void> {
		const plugin = await this.resolvePluginPlugin(pluginID);

		if (!plugin.deletable || plugin.baseline) {
			throw new Error('This Skill Plugin cannot be deleted.');
		}

		await this.store.deleteSkillPlugin({
			plugin: plugin.ref,
			expectedRevision: plugin.revision,
		});
		this.invalidateComposerSkillsCatalog();
	}

	async putSkillArtifact(pluginID: string, _artifactID: string, input: SkillArtifactCreateInput): Promise<Skill> {
		const plugin = await this.requireUserMutablePlugin(pluginID);
		const skillMD = this.encodeManagedSkillMarkdown(input);

		const result = await this.store.createManagedSkill({
			plugin: plugin.ref,
			expectedPluginRevision: plugin.revision,
			skillName: input.name,
			skillMD,
			enabled: input.isEnabled,
		});

		this.invalidateComposerSkillsCatalog();

		return this.projectSkill(
			toSkillPlugin(this.pluginListItemFromView(result.plugin, false)),
			storeSkillListItemFromArtifact(result.artifact, {
				builtIn: false,
				managed: true,
			}),
			true,
			undefined,
			true
		);
	}

	async replaceManagedSkill(pluginID: string, artifactID: string, input: SkillArtifactCreateInput): Promise<void> {
		const p = await this.requireUserMutablePlugin(pluginID);
		const artifact = await this.findPluginSkillItem(p, artifactID);
		const plugin = toSkillPlugin(p);
		const current = await this.projectSkill(plugin, artifact, true, undefined, true);

		if (!artifact.managed || !current.isManaged) {
			throw new Error('Only managed Skills can be edited.');
		}
		if (current.resources.hasResources || current.resources.totalCount > 0) {
			throw new Error(
				'Skills with package resources cannot be edited yet. Fork the Skill to create a new managed Skill.'
			);
		}

		const request: ManagedSkillReplaceRequest = {
			plugin: plugin.ref,
			expectedPluginRevision: plugin.revision,
			artifact: artifact.ref,
			expectedArtifactRevision: artifact.revision,
			skillName: input.name,
			skillMD: this.encodeManagedSkillMarkdown(input),
			enabled: input.isEnabled,
		};

		await this.store.replaceManagedSkill(request);
		this.invalidateComposerSkillsCatalog();
	}

	async getManagedSkillDocument(pluginID: string, artifactID: string): Promise<ManagedSkillDocumentView> {
		const plugin = await this.resolvePluginPlugin(pluginID);
		const artifact = await this.findPluginSkillItem(plugin, artifactID);

		if (!artifact.managed) {
			throw new Error('Only managed Skills expose editable Skill documents.');
		}

		const managed = await this.store.getManagedSkillDocument(artifact.ref);

		return {
			artifact: toManagedSkillArtifactView(managed.artifact),
			document: managed.document,
		};
	}

	async registerFilesystemSkills(pluginID: string, rootPath: string, sourceDisplayName: string): Promise<void> {
		let plugin = await this.requireUserMutablePlugin(pluginID);

		const source = await this.store.registerSkillDirectory({
			rootID: plugin.ref.rootID,
			rootPath,
			sourceDisplayName,
		});

		const listed = await this.store.listSkills(plugin.ref.rootID);
		const discovered = await mapWithConcurrency(listed, SKILL_READ_CONCURRENCY, item => this.store.getSkill(item.ref));
		const sourceSkills = discovered.filter(artifact => artifact.binding.sourceID === source.id);

		for (const artifact of sourceSkills) {
			const updated = await this.store.attachSkillArtifactToPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
				artifact: {
					rootID: artifact.rootID,
					artifactID: artifact.id,
				},
			});

			plugin = this.pluginListItemFromView(updated, false);
		}

		this.invalidateComposerSkillsCatalog();
	}

	async patchSkill(
		pluginID: string,
		artifactID: string,
		isEnabled?: boolean,
		_location?: string,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void> {
		if (displayName !== undefined || description !== undefined || tags !== undefined) {
			const plugin = await this.requireUserMutablePlugin(pluginID);
			const artifact = await this.findPluginSkillItem(plugin, artifactID);

			if (!artifact.managed) {
				throw new Error('Only managed Skills can be edited.');
			}

			const managed = await this.store.getManagedSkillDocument(artifact.ref);
			const current = await this.projectSkill(toSkillPlugin(plugin), artifact, true, undefined, true);

			if (current.resources.hasResources || current.resources.totalCount > 0) {
				throw new Error(
					'Skills with package resources cannot be edited yet. Fork the Skill to create a new managed Skill.'
				);
			}

			await this.replaceManagedSkill(pluginID, artifactID, {
				name: managed.document.name,
				displayName: displayName === undefined ? managed.document.displayName : displayName || undefined,
				description: description === undefined ? managed.document.description : description,
				insert: managed.document.insert,
				arguments: managed.document.arguments,
				tags: tags === undefined ? managed.document.tags : tags,
				markdownBody: managed.document.markdownBody,
				isEnabled: isEnabled ?? artifact.enabled,
			});
			return;
		}

		if (isEnabled === undefined) {
			return;
		}

		const plugin = await this.resolvePluginPlugin(pluginID);
		const artifact = await this.findPluginSkillItem(plugin, artifactID);

		await this.store.setSkillEnabled(artifact.ref, artifact.revision, isEnabled);
		this.invalidateComposerSkillsCatalog();
	}

	async deleteSkill(pluginID: string, artifactID: string): Promise<void> {
		const plugin = await this.requireUserMutablePlugin(pluginID);
		const artifact = await this.findPluginSkillItem(plugin, artifactID);
		const memberships = await this.store.listSkillPluginMemberships(artifact.ref);
		const membership = memberships.find(
			value => value.plugin.rootID === plugin.ref.rootID && value.plugin.artifactID === plugin.ref.artifactID
		);

		if (!membership) {
			throw new Error('Skill is not a direct member of this Skill Plugin.');
		}

		await this.store.removeSkillPluginMember({
			plugin: membership.plugin,
			expectedRevision: membership.pluginRevision,
			index: membership.memberIndex,
		});

		this.invalidateComposerSkillsCatalog();

		if (!artifact.managed) {
			return;
		}

		const remainingMemberships = await this.store.listSkillPluginMemberships(artifact.ref);
		if (remainingMemberships.length > 0) {
			return;
		}

		await this.store.purgeSkill(artifact.ref, artifact.revision);
	}

	async renderSkill(skill: ArtifactRef, args?: Record<string, string>): Promise<RenderSkillResponse> {
		const rendered = await this.renderArtifactSkill(skill, args);

		return {
			text: rendered.text,
			insert: rendered.insert,
			name: rendered.name,
			description: rendered.description,
			displayName: rendered.displayName,
			sourceTags: rendered.tags,
			resources: rendered.resources,
			arguments: rendered.arguments,
			appliedArguments: rendered.appliedArguments,
			rawFrontmatter: rendered.rawFrontmatter,
			warnings: rendered.warnings,
		};
	}

	resolveToolTarget(target: CapabilityTarget): Promise<ResolvedToolView> {
		return this.tools.resolveToolTarget(target);
	}

	resolveMappedModelTarget(target: CapabilityTarget): Promise<ModelManagementItem> {
		return this.models.resolveMappedModelTarget(target);
	}

	private skillPluginsFromPlugins(
		plugins: PluginListItem[],
		pluginIDs?: string[],
		includeDisabled = true
	): SkillPlugin[] {
		const requested = pluginIDs ? new Set(pluginIDs) : undefined;

		return plugins
			.filter(plugin => requested === undefined || requested.has(plugin.ref.artifactID))
			.filter(plugin => includeDisabled || plugin.enabled)
			.map(b => {
				return toSkillPlugin(b);
			})
			.toSorted((left, right) => {
				if (left.isBuiltIn !== right.isBuiltIn) {
					return left.isBuiltIn ? -1 : 1;
				}

				return (left.displayName || left.slug).localeCompare(right.displayName || right.slug, undefined, {
					sensitivity: 'base',
				});
			});
	}

	private async listSkillsFromPlugins(
		plugins: SkillPlugin[],
		pluginListItems: PluginListItem[],
		artifacts: StoreSkillListItem[],
		includeDisabled: boolean,
		includeRuntimeMetadata: boolean
	): Promise<SkillListItem[]> {
		const pluginListItemsByID = new Map(pluginListItems.map(p => [p.ref.artifactID, p] as const));
		const artifactsByRef = new Map(artifacts.map(artifact => [artifactRefKey(artifact.ref), artifact] as const));

		const candidateGroups = await mapWithConcurrency(plugins, SKILL_READ_CONCURRENCY, async plugin => {
			const p = pluginListItemsByID.get(plugin.id);

			if (!p) {
				throw new Error(`Skill Plugin ${plugin.id} disappeared while loading Skills.`);
			}

			const members = await this.listPluginSkillItems(p, artifactsByRef);

			return members.map(artifact => ({
				plugin,
				artifact,
			}));
		});
		const candidates = candidateGroups.flat();

		const runtimeMetadata = includeRuntimeMetadata
			? await this.resolveRuntimeMetadata(candidates.map(candidate => candidate.artifact))
			: new Map<string, PrefetchedRuntimeMetadata>();

		const projected = await Promise.all(
			candidates.map(({ plugin, artifact }) =>
				this.projectSkill(plugin, artifact, includeRuntimeMetadata, runtimeMetadata.get(artifactRefKey(artifact.ref)))
			)
		);

		const output: SkillListItem[] = [];

		for (const [index, skill] of projected.entries()) {
			if (!includeDisabled && !skill.isEnabled) {
				continue;
			}

			const plugin = candidates[index].plugin;
			output.push({
				pluginID: plugin.id,
				pluginSlug: plugin.slug,
				skillSlug: skill.slug,
				skillDefinition: skill,
			});
		}

		return output.toSorted((left, right) => {
			const leftName = left.skillDefinition.displayName || left.skillDefinition.name || left.skillDefinition.slug;
			const rightName = right.skillDefinition.displayName || right.skillDefinition.name || right.skillDefinition.slug;

			return leftName.localeCompare(rightName, undefined, {
				sensitivity: 'base',
			});
		});
	}

	private async resolvePluginPlugin(pluginID: string): Promise<PluginListItem> {
		const plugins = await this.store.listSkillPluginsForManagement();
		const plugin = plugins.find(value => value.ref.artifactID === pluginID);

		if (!plugin) {
			throw new Error('Skill Plugin not found.');
		}

		return plugin;
	}

	private async requireUserMutablePlugin(pluginID: string): Promise<PluginListItem> {
		const plugin = await this.resolvePluginPlugin(pluginID);

		if (!isUserMutablePlugin(plugin)) {
			throw new Error('This Skill Plugin only supports enable and disable actions.');
		}

		return plugin;
	}

	private async listPluginSkillItems(
		plugin: PluginListItem,
		availableByRef: ReadonlyMap<string, StoreSkillListItem>
	): Promise<StoreSkillListItem[]> {
		const plan = await this.store.resolveSkillPlugin(plugin.ref);
		const output: StoreSkillListItem[] = [];
		const seen = new Set<string>();

		for (const occurrence of plan.occurrences ?? []) {
			if (
				occurrence.type !== 'skill' ||
				occurrence.status !== CapabilityResolutionStatus.Available ||
				occurrence.target?.form !== CapabilityTargetForm.Artifact ||
				!occurrence.target.artifact
			) {
				continue;
			}

			const key = artifactRefKey(occurrence.target.artifact);
			if (seen.has(key)) {
				continue;
			}

			seen.add(key);

			const item = availableByRef.get(key);
			if (item) {
				output.push(item);
			}
		}

		return output;
	}

	private async listPluginSkillItemsForRead(plugin: PluginListItem): Promise<StoreSkillListItem[]> {
		const artifacts = await this.store.listSkills(plugin.ref.rootID);
		const byRef = new Map(artifacts.map(artifact => [artifactRefKey(artifact.ref), artifact] as const));

		return this.listPluginSkillItems(plugin, byRef);
	}

	private async findPluginSkillItem(plugin: PluginListItem, artifactID: string): Promise<StoreSkillListItem> {
		const artifacts = await this.listPluginSkillItemsForRead(plugin);
		const artifact = artifacts.find(value => value.ref.artifactID === artifactID);

		if (!artifact) {
			throw new Error('Skill is not a resolved member of this Skill Plugin.');
		}

		return artifact;
	}

	private async projectSkill(
		plugin: SkillPlugin,
		artifact: StoreSkillListItem,
		includeRuntimeMetadata: boolean,
		prefetchedRuntime?: PrefetchedRuntimeMetadata,
		loadManagedDocument = false
	): Promise<Skill> {
		const ref = artifact.ref;
		const isManaged = artifact.managed && !plugin.isBuiltIn;
		let managedDocument: SkillDocumentInput | undefined;

		if (isManaged && loadManagedDocument) {
			try {
				managedDocument = (await this.store.getManagedSkillDocument(ref)).document;
			} catch {
				managedDocument = undefined;
			}
		}

		let runtimeRecord: RuntimeSkillRecord | undefined;
		let runtimeError: string | undefined;

		if (includeRuntimeMetadata) {
			if (prefetchedRuntime) {
				runtimeRecord = prefetchedRuntime.record;
				runtimeError = prefetchedRuntime.error;
			} else {
				try {
					const resolved = await this.aggregate.resolveArtifactSkill(ref);
					const records = await this.runtime.listRuntimeSkills({
						allowSkills: [resolved.Definition],
					});

					runtimeRecord = records.find(
						record => runtimeDefinitionKey(record.def) === runtimeDefinitionKey(resolved.Definition)
					);
				} catch (error) {
					runtimeError = getErrorMessage(error, 'Runtime metadata is unavailable.');
				}
			}
		}

		const insert = runtimeRecord?.insert ?? managedDocument?.insert ?? SkillInsert.Instructions;

		return {
			schemaVersion: 'plugin/v1',
			id: artifact.ref.artifactID,
			ref,
			revision: artifact.revision,
			slug: artifact.name,
			name: runtimeRecord?.name || managedDocument?.name || artifact.name,
			displayName: runtimeRecord?.displayName || managedDocument?.displayName || artifact.displayName || artifact.name,
			description: runtimeRecord?.description || managedDocument?.description || artifact.description,
			type: artifact.builtIn ? SkillType.EmbeddedFS : SkillType.FS,
			location: runtimeRecord?.def.location ?? '',
			insert,
			arguments: runtimeRecord?.arguments ?? managedDocument?.arguments,
			tags: runtimeRecord?.tags ?? managedDocument?.tags,
			resources: runtimeRecord?.resources ?? emptyResources(),
			digest: runtimeRecord?.digest ?? artifact.definitionDigest,
			rawFrontmatter: runtimeRecord?.rawFrontmatter ?? managedDocument?.rawFrontmatter,
			runtimeWarnings: runtimeRecord?.warnings,
			runtimeError,
			presence: {
				status: presenceStatusFor(artifact),
				lastCheckError:
					artifact.state === ArtifactState.Invalid || artifact.state === ArtifactState.Incompatible
						? 'The Artifact Store reported an invalid or incompatible Skill.'
						: runtimeError,
			},
			isEnabled: artifact.enabled,
			isBuiltIn: artifact.builtIn,
			isManaged,
			adoption: isManaged ? ArtifactAdoptionMode.Pinned : ArtifactAdoptionMode.Observed,
			state: artifact.state,
		};
	}

	private async resolveRuntimeMetadata(
		artifacts: StoreSkillListItem[]
	): Promise<Map<string, PrefetchedRuntimeMetadata>> {
		const output = new Map<string, PrefetchedRuntimeMetadata>();
		const refsByRoot = new Map<string, Map<string, ArtifactRef>>();

		for (const artifact of artifacts) {
			const ref = artifact.ref;
			const key = artifactRefKey(ref);

			if (artifact.state !== ArtifactState.Available) {
				output.set(key, {
					error: 'Runtime metadata is unavailable because this Skill Artifact is not available.',
				});
				continue;
			}

			if (!artifact.enabled) {
				output.set(key, {
					error: 'Runtime metadata is unavailable because this Skill is disabled.',
				});
				continue;
			}

			const rootRefs = refsByRoot.get(ref.rootID) ?? new Map<string, ArtifactRef>();
			rootRefs.set(key, ref);
			refsByRoot.set(ref.rootID, rootRefs);
		}

		const resolved: ResolvedArtifactSkill[] = [];

		for (const rootRefs of refsByRoot.values()) {
			const refs = [...rootRefs.values()];

			try {
				resolved.push(...(await this.aggregate.resolveArtifactSkills(refs)));
			} catch (error) {
				const message = getErrorMessage(error, 'Runtime metadata is unavailable for this Skill Root.');

				for (const ref of refs) {
					output.set(artifactRefKey(ref), {
						error: message,
					});
				}
			}
		}

		if (resolved.length === 0) {
			return output;
		}

		let records: RuntimeSkillRecord[];

		try {
			records =
				(await this.runtime.listRuntimeSkills({
					allowSkills: resolved.map(value => value.Definition),
				})) ?? [];
		} catch (error) {
			const message = getErrorMessage(error, 'Runtime metadata is unavailable.');

			for (const value of resolved) {
				output.set(artifactRefKey(value.Artifact), {
					error: message,
				});
			}

			return output;
		}

		const recordsByDefinition = new Map(records.map(record => [runtimeDefinitionKey(record.def), record] as const));

		for (const value of resolved) {
			const record = recordsByDefinition.get(runtimeDefinitionKey(value.Definition));

			output.set(
				artifactRefKey(value.Artifact),
				record
					? { record }
					: {
							error: 'The runtime did not index this Skill after catalog synchronization.',
						}
			);
		}

		return output;
	}

	private async resolveRuntimeArtifacts(refs: ArtifactRef[]): Promise<ResolvedRuntimeArtifactSet> {
		const uniqueRefs = new Map<string, ArtifactRef>();

		for (const ref of refs) {
			uniqueRefs.set(artifactRefKey(ref), ref);
		}

		if (uniqueRefs.size === 0) {
			return {
				definitions: [],
				byArtifactKey: new Map(),
				artifactByDefinitionKey: new Map(),
			};
		}

		const values = await this.aggregate.resolveArtifactSkills([...uniqueRefs.values()]);
		const definitions: RuntimeSkillDefinition[] = [];
		const byArtifactKey = new Map<string, ResolvedArtifactSkill>();
		const artifactByDefinitionKey = new Map<string, ArtifactRef>();

		for (const value of values) {
			const definitionKey = runtimeDefinitionKey(value.Definition);
			const previous = artifactByDefinitionKey.get(definitionKey);

			if (previous && artifactRefKey(previous) !== artifactRefKey(value.Artifact)) {
				throw new Error(
					`Artifact Skills ${previous.artifactID} and ${value.Artifact.artifactID} resolve to the same runtime definition.`
				);
			}

			byArtifactKey.set(artifactRefKey(value.Artifact), value);
			artifactByDefinitionKey.set(definitionKey, value.Artifact);
			definitions.push(value.Definition);
		}

		return {
			definitions,
			byArtifactKey,
			artifactByDefinitionKey,
		};
	}

	private yamlString(value: string): string {
		return JSON.stringify(value);
	}

	private encodeManagedSkillMarkdown(input: SkillArtifactCreateInput): number[] {
		const lines = [
			'---',
			`name: ${this.yamlString(input.name.trim())}`,
			`description: ${this.yamlString(input.description.trim())}`,
			`insert: ${input.insert}`,
		];

		if (input.displayName?.trim()) {
			lines.push(`displayName: ${this.yamlString(input.displayName.trim())}`);
		}

		if (input.tags?.length) {
			lines.push('tags:');

			for (const tag of input.tags) {
				lines.push(`  - ${this.yamlString(tag)}`);
			}
		}

		if (input.arguments?.length) {
			lines.push('arguments:');

			for (const argument of input.arguments) {
				lines.push(`  - name: ${this.yamlString(argument.name)}`);

				if (argument.description?.trim()) {
					lines.push(`    description: ${this.yamlString(argument.description.trim())}`);
				}
				if (argument.default !== undefined) {
					lines.push(`    default: ${this.yamlString(argument.default)}`);
				}
			}
		}

		lines.push('---', '', input.markdownBody.trim(), '');

		return [...new TextEncoder().encode(lines.join('\n'))];
	}

	private pluginListItemFromView(plugin: PluginView, builtIn: boolean): PluginListItem {
		return {
			ref: pluginRef(plugin),
			sourceID: plugin.artifact.binding.sourceID,
			name: plugin.name,
			displayName: plugin.displayName,
			description: plugin.description,
			state: plugin.artifact.state,
			enabled: plugin.artifact.enabled,
			revision: plugin.artifact.revision,
			memberCount: plugin.members.length,
			builtIn,
			editable: plugin.editable,
			deletable: plugin.deletable,
			baseline: plugin.baseline,
		};
	}
}
