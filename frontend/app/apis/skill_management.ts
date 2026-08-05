import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import { ArtifactAdoptionMode, ArtifactState } from '@/spec/artifact';
import type {
	ManagedSkillDocumentView,
	RenderSkillResponse,
	RuntimeSkillListItem,
	Skill,
	SkillArtifactCreateInput,
	SkillArtifactView,
	SkillBundle,
	SkillBundleView,
	SkillDocumentInput,
	SkillListItem,
	SkillRef,
} from '@/spec/skill';
import { SkillBundleAttachmentRole, SkillInsert, SkillPresenceStatus, SkillType } from '@/spec/skill';

import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IArtifactStoreAPI, ISkillBundleAPI } from '@/apis/interface';

const DEFAULT_SKILL_ROOT_DISPLAY_NAME = 'FlexiGPT Skills';
const DEFAULT_SKILL_ROOT_DESCRIPTION = 'Artifact Store namespace for user-managed Skill Bundles.';
const MANAGED_SOURCE_KIND = 'managed-directory';
const FILESYSTEM_SOURCE_KIND = 'fs-directory';

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
	const managed = bundle.attachments.some(
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
		digest: runtime?.definitionDigest ?? artifact.definitionDigest,
		rawFrontmatter: runtime?.rawFrontmatter,
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

export class SkillManagementAPI {
	private rootCreationPromise?: Promise<ArtifactRootID>;

	constructor(
		// oxlint-disable-next-line typescript/parameter-properties
		private readonly skills: ISkillBundleAPI,
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

	async listSkills(bundleIDs?: string[], includeDisabled = true): Promise<SkillListItem[]> {
		const views = await this.listBundleViews(bundleIDs);
		const output: SkillListItem[] = [];

		for (const bundle of views) {
			const artifacts = await this.skills.listSkillBundleArtifacts(bundle.bundle);
			const runtimeByArtifact = new Map<string, RuntimeSkillListItem>();
			const runtimeErrors = new Map<string, string>();

			await Promise.all(
				artifacts.map(async artifact => {
					if (!bundle.enabled || !artifact.enabled || artifact.state !== ArtifactState.Available) {
						return;
					}

					try {
						const runtime = await this.skills.listRuntimeSkills({
							allowArtifacts: [artifact.artifact],
						});
						const item = runtime.find(value => artifactRefKey(value.artifact) === artifactRefKey(artifact.artifact));
						if (item) {
							runtimeByArtifact.set(artifactRefKey(artifact.artifact), item);
						}
					} catch (error) {
						runtimeErrors.set(
							artifactRefKey(artifact.artifact),
							getErrorMessage(error, 'Runtime metadata is unavailable.')
						);
					}
				})
			);

			for (const artifact of artifacts) {
				const skill = toSkill(
					bundle,
					artifact,
					runtimeByArtifact.get(artifactRefKey(artifact.artifact)),
					runtimeErrors.get(artifactRefKey(artifact.artifact))
				);

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
		const sourceID = getUUIDv7();
		let sourceRevision: number | undefined;

		try {
			const source = await this.artifacts.createArtifactSource(rootID, {
				id: sourceID,
				kind: MANAGED_SOURCE_KIND,
				displayName: `${displayName} managed Skills`,
				enabled: true,
				config: '{}',
			});
			sourceRevision = source.revision;

			await this.skills.createSkillBundle(rootID, {
				collectionID: bundleID,
				displayName,
				description,
				enabled: isEnabled,
				logicalName: slug,
				attachments: [
					{
						sourceID,
						role: SkillBundleAttachmentRole.Managed,
						enabled: true,
						discoveryRoot: '.',
					},
				],
			});
		} catch (error) {
			if (sourceRevision !== undefined) {
				await this.cleanupSource(rootID, sourceID, sourceRevision);
			}
			throw error;
		}
	}

	async patchSkillBundle(bundleID: string, enabled: boolean): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		await this.skills.updateSkillBundle(bundle.bundle, {
			expectedRevision: bundle.revision,
			displayName: bundle.displayName,
			description: bundle.description,
			enabled,
		});
	}

	async updateSkillBundleMetadata(bundleID: string, displayName: string, description?: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		await this.skills.updateSkillBundle(bundle.bundle, {
			expectedRevision: bundle.revision,
			displayName,
			description,
			enabled: bundle.enabled,
		});
	}

	async refreshSkillBundle(bundleID: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		await this.skills.refreshSkillBundle(bundle.bundle);
	}

	async deleteSkillBundle(bundleID: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const members = await this.skills.listSkillBundleArtifacts(bundle.bundle);
		if (members.length > 0) {
			throw new Error('Remove all Skills before deleting the Skill Bundle.');
		}

		const managedSourceIDs = bundle.attachments
			.filter(attachment => attachment.role === SkillBundleAttachmentRole.Managed)
			.map(attachment => attachment.sourceID);

		const retired = await this.skills.retireSkillBundle(bundle.bundle, bundle.revision);
		await this.skills.purgeSkillBundle(bundle.bundle, retired.revision);

		for (const sourceID of managedSourceIDs) {
			try {
				const source = await this.artifacts.getArtifactSource(bundle.bundle.rootID, sourceID);
				const retiredSource = await this.artifacts.retireArtifactSource(
					bundle.bundle.rootID,
					sourceID,
					source.revision
				);
				await this.artifacts.purgeArtifactSource(bundle.bundle.rootID, sourceID, retiredSource.revision);
			} catch (error) {
				console.error('Skill Bundle was deleted but managed Source cleanup failed:', error);
			}
		}
	}

	async putSkillArtifact(bundleID: string, artifactID: string, input: SkillArtifactCreateInput): Promise<Skill> {
		const bundle = await this.ensureManagedAttachment(await this.resolveBundle(bundleID));
		const resp = await this.skills.createManagedSkill(bundle.bundle, {
			expectedCollectionRevision: bundle.revision,
			artifactID,
			skillName: input.name,
			document: this.documentFromInput(input),
			enabled: input.isEnabled,
		});
		return toSkill(bundle, resp.artifact, {
			artifact: resp.artifact.artifact,
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
	}

	async getManagedSkillDocument(bundleID: string, artifactID: string): Promise<ManagedSkillDocumentView> {
		const bundle = await this.resolveBundle(bundleID);
		const artifact = await this.resolveArtifact(bundle, artifactID);
		return this.skills.getManagedSkillDocument(artifact.artifact);
	}

	async registerFilesystemSkills(bundleID: string, rootPath: string, displayName: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const sourceID = getUUIDv7();
		let sourceRevision: number | undefined;

		try {
			const source = await this.artifacts.createArtifactSource(bundle.bundle.rootID, {
				id: sourceID,
				kind: FILESYSTEM_SOURCE_KIND,
				displayName,
				enabled: true,
				config: JSON.stringify({ rootPath }),
			});
			sourceRevision = source.revision;

			const updated = await this.skills.attachSkillBundleSource(bundle.bundle, {
				expectedCollectionRevision: bundle.revision,
				sourceID,
				role: SkillBundleAttachmentRole.External,
				enabled: true,
				discoveryRoot: '.',
			});
			await this.skills.refreshSkillBundle(updated.bundle);
		} catch (error) {
			if (sourceRevision !== undefined) {
				await this.cleanupSource(bundle.bundle.rootID, sourceID, sourceRevision);
			}
			throw error;
		}
	}

	async patchSkill(
		bundleID: string,
		skillSlug: string,
		isEnabled?: boolean,
		_location?: string,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const artifact = await this.resolveSkillArtifact(bundle, skillSlug);
		const hasDocumentChanges = displayName !== undefined || description !== undefined || tags !== undefined;

		if (!hasDocumentChanges) {
			await this.skills.setSkillEnabled(artifact.artifact, {
				expectedRevision: artifact.revision,
				enabled: isEnabled ?? artifact.enabled,
			});
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
	}

	async deleteSkill(bundleID: string, skillSlug: string): Promise<void> {
		const bundle = await this.resolveBundle(bundleID);
		const artifact = await this.resolveSkillArtifact(bundle, skillSlug);

		if (artifact.adoption === ArtifactAdoptionMode.Observed) {
			await this.skills.unadoptSkill(artifact.artifact, artifact.revision, true);
			return;
		}

		await this.skills.purgeSkill(artifact.artifact, artifact.revision);
	}

	async renderSkill(ref: SkillRef, args?: Record<string, string>): Promise<RenderSkillResponse> {
		return this.skills.renderSkill(ref, args);
	}

	private documentFromInput(input: SkillArtifactCreateInput): SkillDocumentInput {
		const description = input.description.trim();
		if (!description) {
			throw new Error('Skill description is required.');
		}
		if (!input.markdownBody.trim()) {
			throw new Error('SKILL.md body is required.');
		}

		return {
			name: input.name.trim(),
			displayName: input.displayName?.trim() || undefined,
			description,
			insert: input.insert,
			arguments: input.arguments,
			tags: input.tags,
			markdownBody: input.markdownBody,
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

	private async resolveSkillArtifact(bundle: SkillBundleView, skillSlug: string): Promise<SkillArtifactView> {
		const artifacts = await this.skills.listSkillBundleArtifacts(bundle.bundle);

		for (const artifact of artifacts) {
			if (artifact.artifact.artifactID === skillSlug || skillNameFromLocator(artifact.binding.locator) === skillSlug) {
				return artifact;
			}

			try {
				const runtime = await this.skills.listRuntimeSkills({
					allowArtifacts: [artifact.artifact],
				});
				if (runtime.some(value => value.name === skillSlug)) {
					return artifact;
				}
			} catch {
				// An unavailable runtime projection can still be managed by Artifact ID.
			}
		}

		throw new Error('Skill not found.');
	}

	private async ensureManagedAttachment(bundle: SkillBundleView): Promise<SkillBundleView> {
		if (bundle.attachments.some(attachment => attachment.role === SkillBundleAttachmentRole.Managed)) {
			return bundle;
		}

		const sourceID = getUUIDv7();
		let sourceRevision: number | undefined;
		try {
			const source = await this.artifacts.createArtifactSource(bundle.bundle.rootID, {
				id: sourceID,
				kind: MANAGED_SOURCE_KIND,
				displayName: `${bundle.displayName} managed Skills`,
				enabled: true,
				config: '{}',
			});
			sourceRevision = source.revision;

			return await this.skills.attachSkillBundleSource(bundle.bundle, {
				expectedCollectionRevision: bundle.revision,
				sourceID,
				role: SkillBundleAttachmentRole.Managed,
				enabled: true,
				discoveryRoot: '.',
			});
		} catch (error) {
			if (sourceRevision !== undefined) {
				await this.cleanupSource(bundle.bundle.rootID, sourceID, sourceRevision);
			}
			throw error;
		}
	}

	private async resolveCreationRoot(): Promise<ArtifactRootID> {
		if (!this.rootCreationPromise) {
			this.rootCreationPromise = this.resolveOrCreateRoot();
		}

		try {
			return await this.rootCreationPromise;
		} finally {
			this.rootCreationPromise = undefined;
		}
	}

	private async resolveOrCreateRoot(): Promise<ArtifactRootID> {
		const roots = await this.artifacts.listArtifactRoots();
		const existing = roots.find(
			root => root.displayName.trim().toLowerCase() === DEFAULT_SKILL_ROOT_DISPLAY_NAME.toLowerCase()
		);
		if (existing) {
			return existing.id;
		}

		const created = await this.artifacts.createArtifactRoot({
			id: getUUIDv7(),
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
