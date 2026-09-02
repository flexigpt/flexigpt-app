import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import { ArtifactAdoptionMode, ArtifactState, newArtifactStorageKey } from '@/spec/artifact';
import type {
	CreateSkillSessionOptions,
	InvokeSkillToolResponse,
	ManagedSkillDocumentView,
	RenderSkillResponse,
	ResolvedSkillRuntime,
	RuntimeSkillDefinition,
	RuntimeSkillFilter,
	RuntimeSkillListItem,
	RuntimeSkillQuery,
	RuntimeSkillRecord,
	RuntimeSkillSessionOptions,
	Skill,
	SkillArtifactCreateInput,
	SkillArtifactView,
	SkillBundle,
	SkillBundleRef,
	SkillBundleView,
	SkillDocumentInput,
	SkillListItem,
	SkillRef,
	SkillSession,
} from '@/spec/skill';
import {
	RuntimeSkillActivity,
	SkillBundleAttachmentRole,
	SkillInsert,
	SkillPresenceStatus,
	SkillType,
} from '@/spec/skill';

import type { JSONRawString } from '@/lib/jsonschema_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IArtifactStoreAPI, ISkillRuntimeAPI, ISkillStoreAPI } from '@/apis/interface';

const DEFAULT_SKILL_ROOT_DISPLAY_NAME = 'FlexiGPT Skills';
const DEFAULT_SKILL_ROOT_DESCRIPTION = 'Artifact Store namespace for user-managed Skill Bundles.';
const FILESYSTEM_SOURCE_KIND = 'fs-directory';
const DEFAULT_SKILL_ROOT_STORAGE_KEY = 'skills';
const DEFAULT_SKILL_ROOT_ID: ArtifactRootID = '0198f097-0d5c-7000-8000-000000000001';

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim()) {
		return error.message;
	}
	return fallback;
}

function bundleIsBuiltIn(bundle: SkillBundleView): boolean {
	return bundle.attachments.some(attachment => attachment.role === SkillBundleAttachmentRole.BuiltIn);
}

function artifactRefKey(ref: ArtifactRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

function runtimeDefinitionKey(definition: RuntimeSkillDefinition): string {
	return `${definition.type}\u0000${definition.name}\u0000${definition.location}`;
}

interface ResolvedRuntimeArtifactSet {
	definitions: RuntimeSkillDefinition[];
	byArtifactKey: Map<string, ResolvedSkillRuntime>;
	artifactByDefinitionKey: Map<string, ArtifactRef>;
}

function skillNameFromLocator(locator: string): string {
	const parts = locator.split('/').filter(Boolean);
	const definitionIndex = parts.at(-1)?.toLowerCase() === 'skill.md' ? -2 : -1;
	return parts.at(definitionIndex) || 'skill';
}

function emptyResources(): RuntimeSkillListItem['resources'] {
	return {
		hasResources: false,
		totalCount: 0,
		moreLocations: false,
	};
}

function presenceStatusFor(artifact: SkillArtifactView): SkillPresenceStatus {
	switch (artifact.state) {
		case ArtifactState.Available:
			return SkillPresenceStatus.Present;
		case ArtifactState.Missing:
			return SkillPresenceStatus.Missing;
		case ArtifactState.Invalid:
		case ArtifactState.Incompatible:
			return SkillPresenceStatus.Error;
		default:
			return SkillPresenceStatus.Unknown;
	}
}

function toSkillBundle(bundle: SkillBundleView): SkillBundle {
	return {
		schemaVersion: 'artifact-store/v1',
		id: bundle.bundle.collectionID,
		rootID: bundle.bundle.rootID,
		ref: bundle.bundle,
		revision: bundle.revision,
		slug: bundle.logicalName,
		logicalVersion: bundle.logicalVersion,
		labels: bundle.labels,
		displayName: bundle.displayName,
		managedSourceID: bundle.managedSourceID,
		description: bundle.description,
		isEnabled: bundle.enabled,
		isBuiltIn: bundleIsBuiltIn(bundle),
		attachments: bundle.attachments,
		createdAt: bundle.createdAt,
		modifiedAt: bundle.modifiedAt,
	};
}

function toSkill(
	bundle: SkillBundleView,
	artifact: SkillArtifactView,
	runtime?: RuntimeSkillListItem,
	runtimeError?: string
): Skill {
	const builtIn = bundleIsBuiltIn(bundle);
	const managed =
		artifact.adoption === ArtifactAdoptionMode.Pinned &&
		bundle.managedSourceID === artifact.binding.sourceID &&
		bundle.attachments.some(
			attachment =>
				attachment.sourceID === artifact.binding.sourceID && attachment.role === SkillBundleAttachmentRole.Managed
		);
	const fallbackName = skillNameFromLocator(artifact.binding.locator);
	const name = runtime?.name?.trim() || fallbackName;
	const displayName = runtime?.displayName?.trim() || artifact.name || name;
	const status = presenceStatusFor(artifact);

	return {
		schemaVersion: 'artifact-store/v1',
		id: artifact.artifact.artifactID,
		ref: artifact.artifact,
		revision: artifact.revision,
		slug: name,
		name,
		displayName,
		description: runtime?.description,
		type: builtIn ? SkillType.EmbeddedFS : SkillType.FS,
		location: artifact.binding.locator,
		insert: runtime?.insert ?? SkillInsert.Instructions,
		arguments: runtime?.arguments,
		tags: runtime?.sourceTags,
		resources: runtime?.resources ?? emptyResources(),
		digest: runtime?.digest ?? artifact.definitionDigest,
		rawFrontmatter: runtime?.rawFrontmatter,
		runtimeError,
		runtimeWarnings: [
			...(runtime?.warnings ?? []),
			...(runtime?.errorMessage ? [runtime.errorMessage] : []),
			...(runtimeError ? [runtimeError] : []),
		],
		presence: {
			status,
			lastCheckedAt: artifact.modifiedAt,
			lastSeenAt: status === SkillPresenceStatus.Present ? artifact.modifiedAt : undefined,
			missingSince: status === SkillPresenceStatus.Missing ? artifact.modifiedAt : undefined,
			lastCheckError:
				status === SkillPresenceStatus.Error
					? runtimeError ||
						runtime?.errorMessage ||
						artifact.diagnostics?.map(diagnostic => diagnostic.message).join('\n')
					: undefined,
		},
		isEnabled: artifact.enabled,
		isBuiltIn: builtIn,
		isManaged: managed,
		adoption: artifact.adoption,
		state: artifact.state,
		diagnostics: artifact.diagnostics,
		createdAt: artifact.createdAt,
		modifiedAt: artifact.modifiedAt,
	};
}

function runtimeListItemFromRecord(
	record: RuntimeSkillRecord,
	skillRef: ArtifactRef,
	isActive: boolean
): RuntimeSkillListItem {
	return {
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
		isActive,
	};
}

export class SkillManagementAPI {
	private rootCreationPromise?: Promise<ArtifactRootID>;

	constructor(
		// oxlint-disable-next-line typescript/parameter-properties
		private readonly skills: ISkillStoreAPI,
		// oxlint-disable-next-line typescript/parameter-properties
		private readonly runtime: ISkillRuntimeAPI,
		// oxlint-disable-next-line typescript/parameter-properties
		private readonly artifacts: IArtifactStoreAPI
	) {}

	async listSkillBundles(bundleIDs?: string[], includeDisabled = true): Promise<SkillBundle[]> {
		const roots = await this.artifacts.listArtifactRoots();
		const groups = await Promise.all(roots.map(root => this.skills.listSkillBundles(root.id)));
		const requested = bundleIDs ? new Set(bundleIDs) : undefined;

		return groups
			.flat()
			.filter(bundle => requested === undefined || requested.has(bundle.bundle.collectionID))
			.filter(bundle => includeDisabled || bundle.enabled)
			.map(s => {
				return toSkillBundle(s);
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
		const views = await this.listBundleViews(bundleIDs);
		const output: SkillListItem[] = [];

		for (const bundle of views) {
			const artifacts = await this.skills.listSkillBundleArtifacts(bundle.bundle);
			const runtimeByArtifact = new Map<string, RuntimeSkillListItem>();
			const runtimeErrors = new Map<string, string>();

			const runtimeCandidates = includeRuntimeMetadata
				? artifacts.filter(artifact => bundle.enabled && artifact.enabled && artifact.state === ArtifactState.Available)
				: [];

			if (runtimeCandidates.length > 0) {
				try {
					const runtimeItems = await this.listRuntimeSkills({
						allowArtifacts: runtimeCandidates.map(artifact => artifact.artifact),
					});
					const requestedKeys = new Set(runtimeCandidates.map(artifact => artifactRefKey(artifact.artifact)));

					for (const item of runtimeItems) {
						const key = artifactRefKey(item.skillRef);
						if (requestedKeys.has(key)) {
							runtimeByArtifact.set(key, item);
						}
					}

					for (const artifact of runtimeCandidates) {
						const key = artifactRefKey(artifact.artifact);
						if (!runtimeByArtifact.has(key)) {
							runtimeErrors.set(key, 'Runtime metadata was not returned for this Skill.');
						}
					}
				} catch (error) {
					const message = getErrorMessage(error, 'Runtime metadata is unavailable.');
					for (const artifact of runtimeCandidates) {
						runtimeErrors.set(artifactRefKey(artifact.artifact), message);
					}
				}
			}

			for (const artifact of artifacts) {
				const key = artifactRefKey(artifact.artifact);
				const skill = toSkill(bundle, artifact, runtimeByArtifact.get(key), runtimeErrors.get(key));

				if (!includeDisabled && !skill.isEnabled) {
					continue;
				}

				output.push({
					bundleID: bundle.bundle.collectionID,
					bundleSlug: bundle.logicalName,
					skillSlug: skill.slug,
					skillDefinition: skill,
				});
			}
		}

		return output.toSorted((left, right) => {
			const leftName = left.skillDefinition.displayName || left.skillDefinition.name || left.skillDefinition.slug;
			const rightName = right.skillDefinition.displayName || right.skillDefinition.name || right.skillDefinition.slug;
			return leftName.localeCompare(rightName, undefined, { sensitivity: 'base' });
		});
	}

	async putSkillBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const rootID = await this.resolveCreationRoot();
		const managedSourceID = getUUIDv7();

		const created = await this.skills.createSkillBundle(rootID, {
			collectionID: bundleID,
			displayName,
			description,
			enabled: isEnabled,
			logicalName: slug,
			managedSourceID,
			managedSourceStorageKey: newArtifactStorageKey(),
		});
		await this.syncRuntimeCollection(created.bundle, created.enabled);
	}

	async patchSkillBundle(bundleID: string, enabled: boolean): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const updated = await this.skills.updateSkillBundle(bundle.bundle, {
			expectedRevision: bundle.revision,
			displayName: bundle.displayName,
			description: bundle.description,
			enabled,
		});
		await this.syncRuntimeCollection(updated.bundle, updated.enabled);
	}

	async updateSkillBundleMetadata(bundleID: string, displayName: string, description?: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const updated = await this.skills.updateSkillBundle(bundle.bundle, {
			expectedRevision: bundle.revision,
			displayName,
			description,
			enabled: bundle.enabled,
		});
		await this.syncRuntimeCollection(updated.bundle, updated.enabled);
	}

	async refreshSkillBundle(bundleID: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		await this.skills.refreshSkillBundle(bundle.bundle);
		await this.syncRuntimeCollection(bundle.bundle, bundle.enabled);
	}

	async deleteSkillBundle(bundleID: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const members = await this.skills.listSkillBundleArtifacts(bundle.bundle);
		if (members.length > 0) {
			throw new Error('Remove all Skills before deleting the Skill Bundle.');
		}

		const retired = await this.skills.retireSkillBundle(bundle.bundle, bundle.revision);
		await this.skills.purgeSkillBundle(bundle.bundle, retired.revision);
		await this.syncRuntimeCollection(bundle.bundle, false);
	}

	async putSkillArtifact(bundleID: string, artifactID: string, input: SkillArtifactCreateInput): Promise<Skill> {
		const bundle = await this.requireManagedAttachment(await this.resolveBundle(bundleID));
		const resp = await this.skills.createManagedSkill(bundle.bundle, {
			expectedCollectionRevision: bundle.revision,
			artifactID,
			skillName: input.name,
			document: this.documentFromInput(input),
			enabled: input.isEnabled,
		});

		await this.syncRuntimeCollection(bundle.bundle, bundle.enabled);
		return toSkill(bundle, resp.artifact, {
			skillRef: resp.artifact.artifact,
			name: input.name,
			displayName: input.displayName || input.name,
			description: input.description,
			insert: input.insert,
			arguments: input.arguments,
			sourceTags: input.tags,
			resources: emptyResources(),
		});
	}

	async updateManagedSkill(bundleID: string, artifactID: string, input: SkillArtifactCreateInput): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const artifact = await this.resolveArtifact(bundle, artifactID);
		if (
			artifact.adoption !== ArtifactAdoptionMode.Pinned ||
			!bundle.attachments.some(
				attachment =>
					attachment.sourceID === artifact.binding.sourceID && attachment.role === SkillBundleAttachmentRole.Managed
			)
		) {
			throw new Error('Only managed Skills can be edited.');
		}

		await this.skills.createManagedSkill(bundle.bundle, {
			expectedCollectionRevision: bundle.revision,
			expectedArtifactRevision: artifact.revision,
			artifactID: artifact.artifact.artifactID,
			skillName: input.name,
			document: this.documentFromInput(input),
			enabled: input.isEnabled,
		});
		await this.syncRuntimeCollection(bundle.bundle, bundle.enabled);
	}

	async getManagedSkillDocument(bundleID: string, artifactID: string): Promise<ManagedSkillDocumentView> {
		const bundle = await this.resolveBundle(bundleID);
		const artifact = await this.resolveArtifact(bundle, artifactID);
		return this.skills.getManagedSkillDocument(artifact.artifact);
	}

	async registerFilesystemSkills(bundleID: string, rootPath: string, sourceDisplayName: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const sourceID = getUUIDv7();
		let sourceRevision: number | undefined;
		let attachedBundle: SkillBundleView | undefined;

		try {
			const source = await this.artifacts.createArtifactSource(bundle.bundle.rootID, {
				id: sourceID,
				kind: FILESYSTEM_SOURCE_KIND,
				storageKey: newArtifactStorageKey(),
				displayName: sourceDisplayName,
				enabled: true,
				config: JSON.stringify({ rootPath }),
			});
			sourceRevision = source.revision;

			attachedBundle = await this.skills.attachSkillBundleSource(bundle.bundle, {
				expectedCollectionRevision: bundle.revision,
				sourceID,
				role: SkillBundleAttachmentRole.External,
				enabled: true,
				discoveryRoot: '.',
			});
		} catch (error) {
			if (sourceRevision !== undefined) {
				const latest = await this.resolveBundle(bundleID).catch(() => undefined);
				if (latest?.attachments.some(attachment => attachment.sourceID === sourceID)) {
					// The wrapper may have committed attachment metadata and
					// failed only during its follow-up refresh. The caller's
					// normal bundle refresh can now converge safely.
					return;
				}
				await this.cleanupSource(bundle.bundle.rootID, sourceID, sourceRevision);
			}
			throw error;
		}

		if (attachedBundle) {
			await this.syncRuntimeCollection(attachedBundle.bundle, attachedBundle.enabled);
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
		const bundle = await this.resolveBundle(bundleID);
		const artifact = await this.resolveArtifact(bundle, artifactID);
		const hasDocumentChanges = displayName !== undefined || description !== undefined || tags !== undefined;

		if (!hasDocumentChanges && isEnabled === undefined) {
			return;
		}

		if (!hasDocumentChanges) {
			await this.skills.setSkillEnabled(artifact.artifact, {
				expectedRevision: artifact.revision,
				enabled: isEnabled ?? artifact.enabled,
			});
			await this.syncRuntimeCollection(bundle.bundle, bundle.enabled);
			return;
		}

		const managed = await this.skills.getManagedSkillDocument(artifact.artifact);
		await this.skills.createManagedSkill(bundle.bundle, {
			expectedCollectionRevision: bundle.revision,
			expectedArtifactRevision: artifact.revision,
			artifactID: artifact.artifact.artifactID,
			skillName: managed.document.name,
			document: {
				...managed.document,
				displayName: displayName === undefined ? managed.document.displayName : displayName || undefined,
				description: description === undefined ? managed.document.description : description,
				tags: tags === undefined ? managed.document.tags : tags,
			},
			enabled: isEnabled ?? artifact.enabled,
		});
		await this.syncRuntimeCollection(bundle.bundle, bundle.enabled);
	}

	async deleteSkill(bundleID: string, artifactID: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const artifact = await this.resolveArtifact(bundle, artifactID);

		if (artifact.adoption === ArtifactAdoptionMode.Observed) {
			await this.skills.unadoptSkill(artifact.artifact, artifact.revision, true);
			await this.syncRuntimeCollection(bundle.bundle, bundle.enabled);
			return;
		}

		await this.skills.purgeSkill(artifact.artifact, artifact.revision);
		await this.syncRuntimeCollection(bundle.bundle, bundle.enabled);
	}

	async getSkillsPrompt(filter: RuntimeSkillFilter): Promise<string> {
		const resolved = await this.resolveRuntimeArtifacts(filter.allowArtifacts);

		if (filter.inserts?.length && !filter.inserts.includes(SkillInsert.Instructions)) {
			return '';
		}

		return this.runtime.getSkillsPrompt(
			this.runtimeQuery(filter, filter.allowArtifacts === undefined ? undefined : resolved.definitions)
		);
	}

	async createSkillSession(options: CreateSkillSessionOptions): Promise<SkillSession> {
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

				const key = runtimeDefinitionKey(value.definition);
				if (seen.has(key)) {
					continue;
				}
				seen.add(key);
				definitions.push(value.definition);
			}

			return definitions;
		};

		const runtimeOptions: RuntimeSkillSessionOptions = {
			closeSessionID: options.closeSessionID,
			maxActivePerSession: options.maxActivePerSession,
			allowedSkills: allowConfigured ? definitionsFor(allowedRefs) : undefined,
			activeSkills: definitionsFor(activeRefs),
		};
		const session = await this.runtime.createSkillSession(runtimeOptions);

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

	async closeSkillSession(sessionID: string): Promise<void> {
		await this.runtime.closeSkillSession(sessionID);
	}

	async listRuntimeSkills(filter: RuntimeSkillFilter): Promise<RuntimeSkillListItem[]> {
		if (!filter.allowArtifacts?.length) {
			throw new Error('Artifact skill selection is required.');
		}

		const resolved = await this.resolveRuntimeArtifacts(filter.allowArtifacts);
		const activity = filter.activity ?? RuntimeSkillActivity.Any;
		const query = this.runtimeQuery(filter, resolved.definitions);
		const records = await this.runtime.listSkills(query);

		const activeDefinitionKeys = new Set<string>();
		if (filter.sessionID && activity === RuntimeSkillActivity.Any) {
			const activeRecords = await this.runtime.listSkills({
				...query,
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
				runtimeListItemFromRecord(
					record,
					skillRef,
					activity === RuntimeSkillActivity.Active ||
						(activity === RuntimeSkillActivity.Any && activeDefinitionKeys.has(runtimeDefinitionKey(record.def)))
				),
			];
		});
	}

	async invokeSkillTool(sessionID: string, toolName: string, args?: JSONRawString): Promise<InvokeSkillToolResponse> {
		return this.runtime.invokeSkillTool(sessionID, toolName, args);
	}

	async renderSkill(ref: SkillRef, args?: Record<string, string>): Promise<RenderSkillResponse> {
		const resolved = await this.resolveRuntimeArtifacts([ref]);
		const value = resolved.byArtifactKey.get(artifactRefKey(ref));
		if (!value) {
			throw new Error(`Runtime definition was not resolved for Skill ${ref.artifactID}.`);
		}

		const rendered = await this.runtime.renderSkill(value.definition, args);
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

	private documentFromInput(input: SkillArtifactCreateInput): SkillDocumentInput {
		const name = input.name.trim();
		if (!name) {
			throw new Error('Skill name is required.');
		}

		const description = input.description.trim();
		if (!description) {
			throw new Error('Skill description is required.');
		}
		if (!input.markdownBody.trim()) {
			throw new Error('SKILL.md body is required.');
		}

		return {
			name,
			displayName: input.displayName?.trim() || undefined,
			description,
			insert: input.insert,
			arguments: input.arguments,
			tags: input.tags,
			markdownBody: input.markdownBody,
		};
	}

	private runtimeQuery(
		filter: RuntimeSkillFilter,
		allowSkills: RuntimeSkillDefinition[] | undefined
	): RuntimeSkillQuery {
		return {
			types: filter.types,
			inserts: filter.inserts,
			namePrefix: filter.namePrefix,
			locationPrefix: filter.locationPrefix,
			allowSkills,
			sessionID: filter.sessionID,
			activity: filter.activity,
		};
	}

	private async syncRuntimeCollection(collection: SkillBundleRef, enabled = true): Promise<void> {
		const catalogID = await this.skills.runtimeCatalogIDForCollection(collection);

		if (enabled) {
			await this.runtime.syncSkillCatalog(catalogID);
			return;
		}

		await this.runtime.removeSkillCatalog(catalogID);
	}

	private async resolveRuntimeArtifacts(refs: ArtifactRef[] | undefined): Promise<ResolvedRuntimeArtifactSet> {
		const uniqueRefs = new Map<string, ArtifactRef>();
		for (const ref of refs ?? []) {
			uniqueRefs.set(artifactRefKey(ref), ref);
		}

		if (uniqueRefs.size === 0) {
			return {
				definitions: [],
				byArtifactKey: new Map(),
				artifactByDefinitionKey: new Map(),
			};
		}

		const values = await Promise.all([...uniqueRefs.values()].map(ref => this.skills.resolveArtifactSkill(ref)));

		const collections = new Map<string, SkillBundleRef>();
		for (const value of values) {
			collections.set(`${value.collection.rootID}:${value.collection.collectionID}`, value.collection);
		}

		await Promise.all([...collections.values()].map(collection => this.syncRuntimeCollection(collection)));

		const definitions: RuntimeSkillDefinition[] = [];
		const byArtifactKey = new Map<string, ResolvedSkillRuntime>();
		const artifactByDefinitionKey = new Map<string, ArtifactRef>();

		for (const value of values) {
			const definitionKey = runtimeDefinitionKey(value.definition);
			const previous = artifactByDefinitionKey.get(definitionKey);
			if (previous && artifactRefKey(previous) !== artifactRefKey(value.artifact)) {
				throw new Error(
					`Artifact Skills ${previous.artifactID} and ${value.artifact.artifactID} resolve to the same runtime definition.`
				);
			}

			byArtifactKey.set(artifactRefKey(value.artifact), value);
			artifactByDefinitionKey.set(definitionKey, value.artifact);
			definitions.push(value.definition);
		}

		return {
			definitions,
			byArtifactKey,
			artifactByDefinitionKey,
		};
	}

	private async listBundleViews(bundleIDs?: string[]): Promise<SkillBundleView[]> {
		const roots = await this.artifacts.listArtifactRoots();
		const groups = await Promise.all(roots.map(root => this.skills.listSkillBundles(root.id)));
		const requested = bundleIDs ? new Set(bundleIDs) : undefined;
		return groups.flat().filter(bundle => requested === undefined || requested.has(bundle.bundle.collectionID));
	}

	private async resolveBundle(bundleID: string): Promise<SkillBundleView> {
		const bundles = await this.listBundleViews([bundleID]);
		const bundle = bundles.find(value => value.bundle.collectionID === bundleID);
		if (!bundle) {
			throw new Error('Skill Bundle not found.');
		}
		return bundle;
	}

	private async resolveArtifact(bundle: SkillBundleView, artifactID: string): Promise<SkillArtifactView> {
		const artifacts = await this.skills.listSkillBundleArtifacts(bundle.bundle);
		const artifact = artifacts.find(value => value.artifact.artifactID === artifactID);
		if (!artifact) {
			throw new Error('Skill Artifact not found.');
		}
		return artifact;
	}

	private async requireManagedAttachment(bundle: SkillBundleView): Promise<SkillBundleView> {
		if (!bundle.managedSourceID) {
			throw new Error(
				'This Skill Bundle has no bundle-owned managed Source. It is a legacy or incomplete bundle and must be repaired before adding managed Skills.'
			);
		}

		const managedAttachments = bundle.attachments.filter(
			attachment => attachment.role === SkillBundleAttachmentRole.Managed
		);
		if (managedAttachments.length !== 1 || managedAttachments[0].sourceID !== bundle.managedSourceID) {
			throw new Error(
				'The Skill Bundle managed Source ownership record does not match its managed attachment. Repair the bundle before adding managed Skills.'
			);
		}

		return bundle;
	}

	private async resolveCreationRoot(): Promise<ArtifactRootID> {
		const promise = this.rootCreationPromise ?? this.resolveOrCreateRoot();
		this.rootCreationPromise = promise;

		try {
			return await promise;
		} finally {
			if (this.rootCreationPromise === promise) {
				this.rootCreationPromise = undefined;
			}
		}
	}

	private async resolveOrCreateRoot(): Promise<ArtifactRootID> {
		const roots = await this.artifacts.listArtifactRoots();
		const existing = roots.find(root => root.id === DEFAULT_SKILL_ROOT_ID);
		if (existing) {
			return existing.id;
		}

		const created = await this.artifacts.createArtifactRoot({
			id: DEFAULT_SKILL_ROOT_ID,
			storageKey: DEFAULT_SKILL_ROOT_STORAGE_KEY,
			displayName: DEFAULT_SKILL_ROOT_DISPLAY_NAME,
			description: DEFAULT_SKILL_ROOT_DESCRIPTION,
		});
		return created.id;
	}

	private async cleanupSource(rootID: ArtifactRootID, sourceID: string, revision: number): Promise<void> {
		try {
			const retired = await this.artifacts.retireArtifactSource(rootID, sourceID, revision);
			await this.artifacts.purgeArtifactSource(rootID, sourceID, retired.revision);
		} catch (error) {
			console.error('Failed to compensate orphaned Skill Source:', error);
		}
	}
}
