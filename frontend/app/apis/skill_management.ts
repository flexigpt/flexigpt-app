// oxlint-disable typescript/parameter-properties
import type {
	ArtifactAdoptionMode,
	ArtifactRef,
	ArtifactSourceBinding,
	MappedTarget,
	StoreArtifact,
} from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type {
	ArtifactRuntimeSkillListItem,
	ArtifactSkillFilter,
	ArtifactSkillSession,
	ArtifactSkillSessionOptions,
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
	SkillBundle,
	SkillCollectionManagementView,
	SkillDocumentInput,
	SkillListItem,
	SkillManagementView,
	SkillPresenceStatus,
} from '@/spec/skill';
import type { ToolRef } from '@/spec/tool';
import { ArtifactAdoptionMode as ArtifactAdoptionModeValue, ArtifactState } from '@/spec/artifact';
import {
	RuntimeSkillActivity,
	SkillBundleAttachmentRole as SkillBundleAttachmentRoleValue,
	SkillInsert,
	SkillPresenceStatus as SkillPresenceStatusValue,
	SkillType,
} from '@/spec/skill';

import type { JSONRawString } from '@/lib/jsonschema_utils';
import { getErrorMessage } from '@/lib/error_utils';

import type {
	IModelPresetStoreAPI,
	ISkillAggregateAPI,
	ISkillRuntimeAPI,
	ISkillStoreAPI,
	IToolStoreAPI,
} from '@/apis/interface';

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

function presenceStatusFor(artifact: StoreArtifact): SkillPresenceStatus {
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

function isReadOnlyCollection(collection: CollectionView): boolean {
	return !collection.editable && !collection.deletable;
}

function collectionRef(collection: CollectionView): ArtifactRef {
	return {
		rootID: collection.artifact.rootID,
		artifactID: collection.artifact.id,
	};
}

function toBundleAttachment(collection: CollectionView, readOnly: boolean): SkillBundle['attachments'][number] {
	return {
		sourceID: collection.artifact.binding.sourceID,
		revision: collection.artifact.revision,
		role: readOnly ? SkillBundleAttachmentRoleValue.BuiltIn : SkillBundleAttachmentRoleValue.Managed,
		enabled: true,
	};
}

function toSkillBundle(collection: CollectionView): SkillBundle {
	const isBuiltIn = isReadOnlyCollection(collection);

	return {
		schemaVersion: 'collection/v1',
		id: collection.artifact.id,
		rootID: collection.artifact.rootID,
		ref: {
			rootID: collection.artifact.rootID,
			collectionID: collection.artifact.id,
		},
		revision: collection.artifact.revision,
		slug: collection.name,
		logicalVersion: collection.artifact.logicalVersion,
		displayName: collection.displayName || collection.name,
		description: collection.description || undefined,
		managedSourceID: collection.artifact.binding.sourceID,
		isEnabled: collection.artifact.enabled,
		isBuiltIn,
		isEditable: isUserMutableCollection(collection),
		isDeletable: isUserMutableCollection(collection) && collection.deletable,
		isBaseline: collection.baseline,
		sourceID: collection.artifact.binding.sourceID,
		attachments: [toBundleAttachment(collection, isBuiltIn)],
		createdAt: collection.artifact.createdAt,
		modifiedAt: collection.artifact.modifiedAt,
	};
}

function toLegacyBinding(artifact: StoreArtifact): ArtifactSourceBinding {
	return {
		sourceID: artifact.binding.sourceID,
		locator: artifact.binding.locator,
		subresourceLocator: artifact.binding.subresourceLocator,
		expectedKind: artifact.kind,
	};
}

function toLegacyArtifactView(bundle: SkillBundle, artifact: StoreArtifact, isManaged: boolean): SkillArtifactView {
	const adoption: ArtifactAdoptionMode = isManaged
		? ArtifactAdoptionModeValue.Pinned
		: ArtifactAdoptionModeValue.Observed;

	return {
		artifact: {
			rootID: artifact.rootID,
			artifactID: artifact.id,
		},
		address: {
			rootID: artifact.rootID,
			artifactID: artifact.id,
			collectionID: bundle.id,
			kind: artifact.kind,
		},
		revision: artifact.revision,
		name: artifact.logicalName,
		kind: artifact.kind,
		enabled: artifact.enabled,
		adoption,
		state: artifact.state,
		binding: toLegacyBinding(artifact),
		definitionDigest: artifact.resolvedDefinition,
		diagnostics: artifact.diagnostics,
		createdAt: artifact.createdAt,
		modifiedAt: artifact.modifiedAt,
	};
}

interface ResolvedRuntimeArtifactSet {
	definitions: RuntimeSkillDefinition[];
	byArtifactKey: Map<string, ResolvedArtifactSkill>;
	artifactByDefinitionKey: Map<string, ArtifactRef>;
}

function isUserMutableCollection(collection: CollectionView): boolean {
	return collection.editable && !collection.baseline;
}

export class SkillManagementAPI {
	constructor(
		private readonly runtime: ISkillRuntimeAPI,
		private readonly store: ISkillStoreAPI,
		private readonly aggregate: ISkillAggregateAPI,
		private readonly toolStore: IToolStoreAPI,
		private readonly modelPresetStore: IModelPresetStoreAPI
	) {}

	async createArtifactSkillSession(options: ArtifactSkillSessionOptions): Promise<ArtifactSkillSession> {
		const allowConfigured = options.allowArtifacts !== undefined;
		const allowedRefs = options.allowArtifacts ?? [];
		const allowedKeys = new Set(
			allowedRefs.map(a => {
				return artifactRefKey(a);
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
		} as RuntimeSkillSessionOptions);

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

	async renderArtifactSkill(skill: ArtifactRef, args?: Record<string, string>): Promise<RuntimeSkillRenderResult> {
		const resolved = await this.aggregate.resolveArtifactSkill(skill);
		return this.runtime.renderSkill(resolved.Definition, args);
	}

	async getSkillCollectionManagementView(collection: ArtifactRef): Promise<SkillCollectionManagementView> {
		const [view, capabilities] = await Promise.all([
			this.store.getSkillCollection(collection),
			this.store.resolveSkillCollection(collection),
		]);

		return {
			collection: view,
			capabilities,
		};
	}

	async getSkillManagementView(skill: ArtifactRef): Promise<SkillManagementView> {
		const [artifact, memberships, capabilities] = await Promise.all([
			this.store.getSkill(skill),
			this.store.listSkillCollectionMemberships(skill),
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

	async listSkillBundles(bundleIDs?: string[], includeDisabled = true): Promise<SkillBundle[]> {
		const requested = bundleIDs ? new Set(bundleIDs) : undefined;
		const collections = await this.store.listSkillCollectionsForManagement();

		return collections
			.filter(collection => requested === undefined || requested.has(collection.artifact.id))
			.filter(collection => includeDisabled || collection.artifact.enabled)
			.map(a => {
				return toSkillBundle(a);
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

	async listSkills(
		bundleIDs?: string[],
		includeDisabled = true,
		includeRuntimeMetadata = true
	): Promise<SkillListItem[]> {
		const bundles = await this.listSkillBundles(bundleIDs, true);
		const output: SkillListItem[] = [];

		for (const bundle of bundles) {
			const collection = await this.resolveBundleCollection(bundle.id);
			const artifacts = await this.listCollectionSkillArtifacts(collection);

			for (const artifact of artifacts) {
				const skill = await this.projectSkill(bundle, artifact, includeRuntimeMetadata);

				if (!includeDisabled && !skill.isEnabled) {
					continue;
				}

				output.push({
					bundleID: bundle.id,
					bundleSlug: bundle.slug,
					skillSlug: skill.slug,
					skillDefinition: skill,
				});
			}
		}

		return output.toSorted((left, right) => {
			const leftName = left.skillDefinition.displayName || left.skillDefinition.name || left.skillDefinition.slug;
			const rightName = right.skillDefinition.displayName || right.skillDefinition.name || right.skillDefinition.slug;

			return leftName.localeCompare(rightName, undefined, {
				sensitivity: 'base',
			});
		});
	}

	async putSkillBundle(
		_bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const collections = await this.store.listSkillCollectionsForManagement();
		const rootID = this.userSkillRootID(collections);

		let created = await this.store.createSkillCollection({
			rootID,
			name: slug,
			displayName,
			description,
		});

		if (!isEnabled) {
			created = await this.store.setSkillCollectionEnabled(collectionRef(created), created.artifact.revision, false);
		}

		void created;
	}

	async patchSkillBundle(bundleID: string, enabled: boolean): Promise<void> {
		const collection = await this.resolveBundleCollection(bundleID);

		await this.store.setSkillCollectionEnabled(collectionRef(collection), collection.artifact.revision, enabled);
	}

	async updateSkillBundleMetadata(bundleID: string, displayName: string, description?: string): Promise<void> {
		const collection = await this.requireUserMutableBundle(bundleID);

		await this.store.updateSkillCollection({
			collection: collectionRef(collection),
			expectedRevision: collection.artifact.revision,
			displayName,
			description,
		});
	}

	async refreshSkillBundle(bundleID: string): Promise<void> {
		const collection = await this.resolveBundleCollection(bundleID);

		await this.store.refreshSkillSource(collection.artifact.rootID, collection.artifact.binding.sourceID);
	}

	async deleteSkillBundle(bundleID: string): Promise<void> {
		const collection = await this.resolveBundleCollection(bundleID);

		if (!collection.deletable || collection.baseline) {
			throw new Error('This Skill Bundle cannot be deleted.');
		}

		await this.store.deleteSkillCollection({
			collection: collectionRef(collection),
			expectedRevision: collection.artifact.revision,
		});
	}

	async putSkillArtifact(bundleID: string, _artifactID: string, input: SkillArtifactCreateInput): Promise<Skill> {
		const collection = await this.requireUserMutableBundle(bundleID);
		const skillMD = this.encodeManagedSkillMarkdown(input);

		const result = await this.store.createManagedSkill({
			collection: collectionRef(collection),
			expectedCollectionRevision: collection.artifact.revision,
			skillName: input.name,
			skillMD,
			enabled: input.isEnabled,
		});

		const bundle = toSkillBundle(result.collection);
		return this.projectSkill(bundle, result.artifact, true);
	}

	async replaceManagedSkill(bundleID: string, artifactID: string, input: SkillArtifactCreateInput): Promise<void> {
		const collection = await this.requireUserMutableBundle(bundleID);
		const artifact = await this.findCollectionSkillArtifact(collection, artifactID);
		const ref: ArtifactRef = {
			rootID: artifact.rootID,
			artifactID: artifact.id,
		};
		const bundle = toSkillBundle(collection);
		const current = await this.projectSkill(bundle, artifact, true);

		if (!current.isManaged) {
			throw new Error('Only managed Skills can be edited.');
		}
		if (current.resources.hasResources || current.resources.totalCount > 0) {
			throw new Error(
				'Skills with package resources cannot be edited yet. Fork the Skill to create a new managed Skill.'
			);
		}

		const skillMD = this.encodeManagedSkillMarkdown(input);
		const request: ManagedSkillReplaceRequest = {
			collection: collectionRef(collection),
			expectedCollectionRevision: collection.artifact.revision,
			artifact: ref,
			expectedArtifactRevision: artifact.revision,
			skillName: input.name,
			skillMD,
			enabled: input.isEnabled,
		};

		await this.store.replaceManagedSkill(request);
	}

	async getManagedSkillDocument(bundleID: string, artifactID: string): Promise<ManagedSkillDocumentView> {
		const collection = await this.resolveBundleCollection(bundleID);
		const artifact = await this.findCollectionSkillArtifact(collection, artifactID);

		const managed = await this.store.getManagedSkillDocument({
			rootID: artifact.rootID,
			artifactID: artifact.id,
		});

		const bundle = toSkillBundle(collection);
		const isManaged = true;

		return {
			artifact: toLegacyArtifactView(bundle, managed.artifact, isManaged),
			document: managed.document,
		};
	}

	async registerFilesystemSkills(bundleID: string, rootPath: string, sourceDisplayName: string): Promise<void> {
		let collection = await this.requireUserMutableBundle(bundleID);

		const source = await this.store.registerSkillDirectory({
			rootID: collection.artifact.rootID,
			rootPath,
			sourceDisplayName,
		});

		const discovered = await this.store.listSkills(collection.artifact.rootID);

		const sourceSkills = discovered.filter(artifact => artifact.binding.sourceID === source.id);

		for (const artifact of sourceSkills) {
			collection = await this.store.attachSkillArtifactToCollection({
				collection: collectionRef(collection),
				expectedRevision: collection.artifact.revision,
				artifact: {
					rootID: artifact.rootID,
					artifactID: artifact.id,
				},
			});
		}
	}

	async patchSkill(
		bundleID: string,
		artifactID: string,
		isEnabled?: boolean,
		_location?: string,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void> {
		if (displayName !== undefined || description !== undefined || tags !== undefined) {
			const collection = await this.requireUserMutableBundle(bundleID);
			const artifact = await this.findCollectionSkillArtifact(collection, artifactID);
			const ref: ArtifactRef = {
				rootID: artifact.rootID,
				artifactID: artifact.id,
			};
			const managed = await this.store.getManagedSkillDocument(ref);
			const bundle = toSkillBundle(collection);
			const current = await this.projectSkill(bundle, artifact, true);

			if (!current.isManaged) {
				throw new Error('Only managed Skills can be edited.');
			}
			if (current.resources.hasResources || current.resources.totalCount > 0) {
				throw new Error(
					'Skills with package resources cannot be edited yet. Fork the Skill to create a new managed Skill.'
				);
			}

			await this.replaceManagedSkill(bundleID, artifactID, {
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

		const collection = await this.resolveBundleCollection(bundleID);
		const artifact = await this.findCollectionSkillArtifact(collection, artifactID);

		await this.store.setSkillEnabled(
			{
				rootID: artifact.rootID,
				artifactID: artifact.id,
			},
			artifact.revision,
			isEnabled
		);
	}

	async deleteSkill(bundleID: string, artifactID: string): Promise<void> {
		const collection = await this.requireUserMutableBundle(bundleID);
		const artifact = await this.findCollectionSkillArtifact(collection, artifactID);
		const ref: ArtifactRef = {
			rootID: artifact.rootID,
			artifactID: artifact.id,
		};

		const memberships = await this.store.listSkillCollectionMemberships(ref);
		const membership = memberships.find(
			value =>
				value.collection.rootID === collection.artifact.rootID && value.collection.artifactID === collection.artifact.id
		);

		if (!membership) {
			throw new Error('Skill is not a direct member of this Skill Bundle.');
		}

		await this.store.removeSkillCollectionMember({
			collection: membership.collection,
			expectedRevision: membership.collectionRevision,
			index: membership.memberIndex,
		});

		let isManaged: boolean;
		try {
			await this.store.getManagedSkillDocument(ref);
			isManaged = true;
		} catch {
			isManaged = false;
		}

		if (!isManaged) {
			return;
		}

		const remainingMemberships = await this.store.listSkillCollectionMemberships(ref);

		if (remainingMemberships.length > 0) {
			return;
		}

		await this.store.purgeSkill(ref, artifact.revision);
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

	resolveMappedToolTarget(target: MappedTarget): Promise<ToolRef> {
		return this.toolStore.resolveMappedToolTarget(target);
	}

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelPresetRef> {
		return this.modelPresetStore.resolveMappedModelTarget(target);
	}

	private async resolveBundleCollection(bundleID: string): Promise<CollectionView> {
		const collections = await this.store.listSkillCollectionsForManagement();
		const collection = collections.find(value => value.artifact.id === bundleID);

		if (!collection) {
			throw new Error('Skill Bundle not found.');
		}

		return collection;
	}

	private async requireUserMutableBundle(bundleID: string): Promise<CollectionView> {
		const collection = await this.resolveBundleCollection(bundleID);

		if (!isUserMutableCollection(collection)) {
			throw new Error('This Skill Bundle only supports enable and disable actions.');
		}

		return collection;
	}

	private userSkillRootID(collections: CollectionView[]): string {
		const roots = new Set(
			collections
				.filter(collection => collection.baseline && collection.editable && !collection.deletable)
				.map(collection => collection.artifact.rootID)
		);

		if (roots.size !== 1) {
			throw new Error(
				'The user Skill root could not be identified. Expected exactly one editable baseline Skill Bundle.'
			);
		}

		return [...roots][0];
	}

	private async listCollectionSkillArtifacts(collection: CollectionView): Promise<StoreArtifact[]> {
		const plan = await this.store.resolveSkillCollection(collectionRef(collection));
		const refs = new Map<string, ArtifactRef>();

		for (const occurrence of plan.occurrences) {
			if (occurrence.type !== 'skill' || occurrence.status !== 'available' || !occurrence.artifact) {
				continue;
			}

			refs.set(artifactRefKey(occurrence.artifact), occurrence.artifact);
		}

		return Promise.all([...refs.values()].map(ref => this.store.getSkill(ref)));
	}

	private async findCollectionSkillArtifact(collection: CollectionView, artifactID: string): Promise<StoreArtifact> {
		const artifacts = await this.listCollectionSkillArtifacts(collection);
		const artifact = artifacts.find(value => value.id === artifactID);

		if (!artifact) {
			throw new Error('Skill is not a resolved member of this Skill Bundle.');
		}

		return artifact;
	}

	private async projectSkill(
		bundle: SkillBundle,
		artifact: StoreArtifact,
		includeRuntimeMetadata: boolean
	): Promise<Skill> {
		const ref: ArtifactRef = {
			rootID: artifact.rootID,
			artifactID: artifact.id,
		};

		let managedDocument: SkillDocumentInput | undefined;
		let isManaged: boolean;

		try {
			const result = await this.store.getManagedSkillDocument(ref);
			managedDocument = result.document;
			isManaged = true;
		} catch {
			isManaged = false;
		}

		let runtimeRecord: RuntimeSkillRecord | undefined;
		let runtimeError: string | undefined;

		if (includeRuntimeMetadata) {
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

		const insert = runtimeRecord?.insert ?? managedDocument?.insert ?? SkillInsert.Instructions;

		return {
			schemaVersion: 'collection/v1',
			id: artifact.id,
			ref,
			revision: artifact.revision,
			slug: artifact.logicalName,
			name: runtimeRecord?.name || managedDocument?.name || artifact.logicalName,
			displayName:
				runtimeRecord?.displayName || managedDocument?.displayName || artifact.displayName || artifact.logicalName,
			description: runtimeRecord?.description || managedDocument?.description,
			type: bundle.isBuiltIn ? SkillType.EmbeddedFS : SkillType.FS,
			location: runtimeRecord?.def.location || artifact.binding.locator,
			insert,
			arguments: runtimeRecord?.arguments ?? managedDocument?.arguments,
			tags: runtimeRecord?.tags ?? managedDocument?.tags,
			resources: runtimeRecord?.resources ?? emptyResources(),
			digest: runtimeRecord?.digest ?? artifact.resolvedDefinition,
			rawFrontmatter: runtimeRecord?.rawFrontmatter ?? managedDocument?.rawFrontmatter,
			runtimeWarnings: runtimeRecord?.warnings,
			runtimeError,
			presence: {
				status: presenceStatusFor(artifact),
				lastCheckedAt: artifact.modifiedAt,
				lastSeenAt: artifact.state === ArtifactState.Available ? artifact.modifiedAt : undefined,
				missingSince: artifact.state === ArtifactState.Missing ? artifact.modifiedAt : undefined,
				lastCheckError:
					artifact.state === ArtifactState.Invalid || artifact.state === ArtifactState.Incompatible
						? artifact.diagnostics?.map(diagnostic => diagnostic.message).join('\n')
						: runtimeError,
			},
			isEnabled: artifact.enabled,
			isBuiltIn: bundle.isBuiltIn,
			isManaged,
			adoption: isManaged ? ArtifactAdoptionModeValue.Pinned : ArtifactAdoptionModeValue.Observed,
			state: artifact.state,
			diagnostics: artifact.diagnostics,
			createdAt: artifact.createdAt,
			modifiedAt: artifact.modifiedAt,
		};
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

		const values = await Promise.all([...uniqueRefs.values()].map(ref => this.aggregate.resolveArtifactSkill(ref)));

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
}
