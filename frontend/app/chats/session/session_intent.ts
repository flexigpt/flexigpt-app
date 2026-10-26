import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { Attachment } from '@/spec/attachment';
import type { UIModelOption } from '@/spec/model';
import type {
	SessionActiveSkillOverride,
	SessionArtifactOverride,
	SessionConfiguration,
	SessionInstructionOverride,
	SessionModelDefaultsOverride,
	SessionToolOverride,
	SessionUserInput,
} from '@/spec/session';
import type { SessionArtifactSelection } from '@/spec/session_base';
import type { MCPServerOverride, MCPServerSelection } from '@/spec/session_mcp';
import type { WorkspaceConversationResourceSelectionRef, WorkspaceRuntimePlan } from '@/spec/workspace';
import { artifactRefKey, CapabilityTargetForm } from '@/spec/artifact';
import { AttachmentKind } from '@/spec/attachment';
import { emptySessionConfiguration, SessionModelDefaultField } from '@/spec/session';
import {
	SessionActivityOverrideMode,
	SessionAttachmentReferenceKind,
	SessionCollectionOverrideMode,
} from '@/spec/session_base';
import { MCPToolExposure } from '@/spec/session_mcp';

import { mcpManagementAPI } from '@/apis/baseapi';
import { toolIdentityKey } from '@/apis/tool_management';

import type { AgentCapabilityExclusion, EditorSubmitPayload } from '@/chats/composer/editor/editor_types';

export interface SessionSubmissionIntent {
	configuration: SessionConfiguration;
	input: SessionUserInput;
}

function artifactSelection(ref: ArtifactRef, revision?: number, digest?: string): SessionArtifactSelection {
	return {
		rootID: ref.rootID,
		artifactID: ref.artifactID,
		...(revision && revision > 0 ? { expectedRevision: revision } : {}),
		...(digest?.trim() ? { expectedDigest: digest } : {}),
	};
}

function cloneConfigurationValue<T>(value: T): T {
	return structuredClone(value);
}

function sameArtifactSelection(
	left: SessionArtifactSelection | undefined,
	right: SessionArtifactSelection | undefined
): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

function preserveArtifactSelectionPins(
	previous: SessionArtifactSelection | undefined,
	next: SessionArtifactSelection
): SessionArtifactSelection {
	if (!sameArtifactSelection(previous, next)) {
		return cloneConfigurationValue(next);
	}

	return {
		...cloneConfigurationValue(next),
		...(previous?.expectedRevision === undefined ? {} : { expectedRevision: previous.expectedRevision }),
		...(previous?.expectedDigest ? { expectedDigest: previous.expectedDigest } : {}),
	};
}

// oxlint-disable-next-line typescript/consistent-type-definitions
type CollectionOverride<T> = {
	key: string;
	mode: SessionCollectionOverrideMode;
	value?: T;
};

function mergeCollectionOverrides<T, O extends CollectionOverride<T>>(
	baseline: O[] | undefined,
	generated: O[] | undefined,
	mergeValue: (previous: T | undefined, next: T) => T,
	releasedRemoveKeys: ReadonlySet<string>
): O[] | undefined {
	if (!baseline?.length) {
		return generated?.length ? cloneConfigurationValue(generated) : undefined;
	}

	const generatedByKey = new Map<string, O>();
	for (const override of generated ?? []) {
		generatedByKey.set(override.key, override);
	}

	const emittedGeneratedKeys = new Set<string>();
	const output: O[] = [];

	for (const previous of baseline) {
		const next = generatedByKey.get(previous.key);

		if (previous.mode === SessionCollectionOverrideMode.Inherit) {
			output.push(cloneConfigurationValue(previous));
			continue;
		}

		if (previous.mode === SessionCollectionOverrideMode.Remove) {
			if (releasedRemoveKeys.has(previous.key) && !next) {
				continue;
			}

			if (next?.mode === SessionCollectionOverrideMode.Remove) {
				output.push(cloneConfigurationValue(next));
				emittedGeneratedKeys.add(next.key);
			} else {
				output.push(cloneConfigurationValue(previous));
			}
			continue;
		}

		if (previous.mode === SessionCollectionOverrideMode.Replace && previous.value === undefined) {
			output.push(cloneConfigurationValue(previous));
			continue;
		}

		if (!previous.value) {
			continue;
		}

		if (!next) {
			if (previous.mode === SessionCollectionOverrideMode.Replace) {
				output.push({
					key: previous.key,
					mode: SessionCollectionOverrideMode.Replace,
				} as O);
			}
			continue;
		}

		if (next.mode === SessionCollectionOverrideMode.Remove) {
			output.push(cloneConfigurationValue(next));
			emittedGeneratedKeys.add(next.key);
			continue;
		}

		if (next.value) {
			output.push({
				...cloneConfigurationValue(next),
				mode: previous.mode,
				value: mergeValue(previous.value, next.value),
			} as O);
			emittedGeneratedKeys.add(next.key);
			continue;
		}

		output.push(cloneConfigurationValue(next));
		emittedGeneratedKeys.add(next.key);
	}

	for (const next of generated ?? []) {
		if (!emittedGeneratedKeys.has(next.key)) {
			output.push(cloneConfigurationValue(next));
		}
	}

	return output.length > 0 ? output : undefined;
}

function mergeToolOverrideValue(
	previous: NonNullable<SessionToolOverride['value']> | undefined,
	next: NonNullable<SessionToolOverride['value']>
): NonNullable<SessionToolOverride['value']> {
	const previousArtifact = previous?.capability.artifact;
	const nextArtifact = next.capability.artifact;

	return {
		...cloneConfigurationValue(next),
		capability: {
			...cloneConfigurationValue(next.capability),
			...(nextArtifact
				? {
						artifact: preserveArtifactSelectionPins(previousArtifact, nextArtifact),
					}
				: {}),
		},
	};
}

function preserveExpectedDigest<T extends { expectedDigest?: string }>(previous: T | undefined, next: T): T {
	if (previous?.expectedDigest === undefined) {
		return cloneConfigurationValue(next);
	}

	return {
		...cloneConfigurationValue(next),
		expectedDigest: previous.expectedDigest,
	};
}

function mergeMCPServerSelection(
	previous: MCPServerSelection | undefined,
	next: MCPServerSelection
): MCPServerSelection {
	const previousTools = new Map((previous?.selectedTools ?? []).map(value => [value.toolName, value] as const));
	const previousResources = new Map((previous?.resources ?? []).map(value => [value.uri, value] as const));
	const previousTemplates = new Map(
		(previous?.resourceTemplates ?? []).map(value => [value.uriTemplate, value] as const)
	);
	const previousPrompts = new Map((previous?.prompts ?? []).map(value => [value.promptName, value] as const));

	return {
		...cloneConfigurationValue(next),
		...(previous?.key ? { key: previous.key } : {}),
		server: preserveArtifactSelectionPins(previous?.server, next.server),
		selectedTools: next.selectedTools?.map(value => preserveExpectedDigest(previousTools.get(value.toolName), value)),
		resources: next.resources?.map(value => preserveExpectedDigest(previousResources.get(value.uri), value)),
		resourceTemplates: next.resourceTemplates?.map(value =>
			preserveExpectedDigest(previousTemplates.get(value.uriTemplate), value)
		),
		prompts: next.prompts?.map(value => preserveExpectedDigest(previousPrompts.get(value.promptName), value)),
	};
}

function mergeActiveSkillOverrides(
	baseline: SessionActiveSkillOverride[] | undefined,
	generated: SessionActiveSkillOverride[] | undefined,
	releasedRemoveKeys: ReadonlySet<string>
): SessionActiveSkillOverride[] | undefined {
	if (!baseline?.length) {
		return generated?.length ? cloneConfigurationValue(generated) : undefined;
	}

	const generatedByKey = new Map<string, SessionActiveSkillOverride>();
	for (const override of generated ?? []) {
		generatedByKey.set(override.key, override);
	}

	const emittedGeneratedKeys = new Set<string>();
	const output: SessionActiveSkillOverride[] = [];

	for (const previous of baseline) {
		const next = generatedByKey.get(previous.key);

		if (previous.mode === SessionActivityOverrideMode.Inherit) {
			output.push(cloneConfigurationValue(previous));
			continue;
		}

		if (previous.mode === SessionActivityOverrideMode.Deactivate) {
			if (releasedRemoveKeys.has(previous.key) && !next) {
				continue;
			}

			if (next?.mode === SessionActivityOverrideMode.Deactivate) {
				output.push(cloneConfigurationValue(next));
				emittedGeneratedKeys.add(next.key);
			} else {
				output.push(cloneConfigurationValue(previous));
			}
			continue;
		}

		if (!next) {
			continue;
		}

		if (next.mode === SessionActivityOverrideMode.Activate && next.value) {
			output.push({
				...cloneConfigurationValue(next),
				value: preserveArtifactSelectionPins(previous.value, next.value),
			});
			emittedGeneratedKeys.add(next.key);
			continue;
		}

		output.push(cloneConfigurationValue(next));
		emittedGeneratedKeys.add(next.key);
	}

	for (const next of generated ?? []) {
		if (!emittedGeneratedKeys.has(next.key)) {
			output.push(cloneConfigurationValue(next));
		}
	}

	return output.length > 0 ? output : undefined;
}

function mergeInstructionOverrides(
	baseline: SessionInstructionOverride[] | undefined,
	generated: SessionInstructionOverride[] | undefined,
	releasedRemoveKeys: ReadonlySet<string>
): SessionInstructionOverride[] | undefined {
	const next = [...(generated ?? [])];
	const generatedKeys = new Set(next.map(override => override.key));

	for (const previous of baseline ?? []) {
		if (previous.value?.artifact && !generatedKeys.has(previous.key)) {
			next.push(cloneConfigurationValue(previous));
		}
	}

	return mergeCollectionOverrides<NonNullable<SessionInstructionOverride['value']>, SessionInstructionOverride>(
		baseline,
		next,
		(previous, current) => ({
			...cloneConfigurationValue(current),
			...(current.artifact
				? {
						artifact: preserveArtifactSelectionPins(previous?.artifact, current.artifact),
					}
				: {}),
		}),
		releasedRemoveKeys
	);
}

function mergeWorkspaceIntent(
	baseline: SessionConfiguration['workspace'],
	generated: SessionConfiguration['workspace'],
	releasedRemoveKeys: ReadonlySet<string>
): SessionConfiguration['workspace'] {
	if (!generated) {
		return undefined;
	}
	if (!baseline) {
		return cloneConfigurationValue(generated);
	}

	return {
		...cloneConfigurationValue(generated),
		workspace: preserveArtifactSelectionPins(baseline.workspace, generated.workspace),
		context: mergeCollectionOverrides<SessionArtifactSelection, SessionArtifactOverride>(
			baseline.context,
			generated.context,
			(previous, current) => preserveArtifactSelectionPins(previous, current),
			releasedRemoveKeys
		),
		skills: mergeCollectionOverrides<SessionArtifactSelection, SessionArtifactOverride>(
			baseline.skills,
			generated.skills,
			(previous, current) => preserveArtifactSelectionPins(previous, current),
			releasedRemoveKeys
		),
		mcp: mergeCollectionOverrides(baseline.mcp, generated.mcp, mergeMCPServerSelection, releasedRemoveKeys),
	};
}

function mergeSessionConfiguration(
	baseline: SessionConfiguration,
	generated: SessionConfiguration,
	payload: EditorSubmitPayload
): SessionConfiguration {
	const releasedRemoveKeys = new Set(payload.agentCapabilityInclusionKeys);
	const unresolvedToolKeys = new Set(
		(payload.unresolvedToolSelections ?? []).map(issue => sessionCapabilityKey(issue.selection.target))
	);
	const generatedTools = [...(generated.tools ?? [])];
	const generatedToolKeys = new Set(generatedTools.map(override => override.key));

	for (const previous of baseline.tools ?? []) {
		if (previous.value && unresolvedToolKeys.has(previous.key) && !generatedToolKeys.has(previous.key)) {
			generatedTools.push(cloneConfigurationValue(previous));
		}
	}

	return {
		...cloneConfigurationValue(baseline),
		agent: generated.agent
			? payload.agentSelectionRefreshRequested
				? cloneConfigurationValue(generated.agent)
				: preserveArtifactSelectionPins(baseline.agent, generated.agent)
			: cloneConfigurationValue(baseline.agent),
		workspace:
			payload.workspaceSelection && !payload.workspacePlan && baseline.workspace
				? cloneConfigurationValue(baseline.workspace)
				: mergeWorkspaceIntent(baseline.workspace, generated.workspace, releasedRemoveKeys),
		model: generated.model
			? preserveArtifactSelectionPins(baseline.model, generated.model)
			: payload.modelSelectionSource === 'agent' && payload.agentRef
				? undefined
				: cloneConfigurationValue(baseline.model),
		modelOverrides: cloneConfigurationValue(generated.modelOverrides),
		instructions: mergeInstructionOverrides(baseline.instructions, generated.instructions, releasedRemoveKeys),
		tools: mergeCollectionOverrides(baseline.tools, generatedTools, mergeToolOverrideValue, releasedRemoveKeys),
		skills: mergeCollectionOverrides<SessionArtifactSelection, SessionArtifactOverride>(
			baseline.skills,
			generated.skills,
			(previous, current) => preserveArtifactSelectionPins(previous, current),
			releasedRemoveKeys
		),
		activeSkills: mergeActiveSkillOverrides(baseline.activeSkills, generated.activeSkills, releasedRemoveKeys),
		mcp: mergeCollectionOverrides(baseline.mcp, generated.mcp, mergeMCPServerSelection, releasedRemoveKeys),
		history: cloneConfigurationValue(generated.history),
	};
}

function nonSessionSkillKeysFromPayload(payload: EditorSubmitPayload): Set<string> {
	const keys = new Set<string>();

	for (const ref of payload.nonSessionSkillRefs ?? []) {
		if (!ref.rootID || !ref.artifactID) {
			continue;
		}
		keys.add(artifactRefKey(ref));
	}

	return keys;
}

function nonSessionSkillRemovalOverrides(keys: ReadonlySet<string>): SessionArtifactOverride[] {
	return [...keys].map(key => ({
		key,
		mode: SessionCollectionOverrideMode.Remove,
	}));
}

function sessionCapabilityKey(target: CapabilityTarget): string {
	if (target.form === CapabilityTargetForm.Artifact && target.artifact) {
		return artifactRefKey(target.artifact);
	}

	return ['direct', target.providerIdentity ?? '', target.providerLocalID ?? '', target.type, target.name].join(':');
}

function sessionCapabilitySelection(target: CapabilityTarget) {
	if (target.form === CapabilityTargetForm.Artifact && target.artifact) {
		return {
			artifact: artifactSelection(target.artifact),
		};
	}

	if (
		target.form === CapabilityTargetForm.Direct &&
		target.providerIdentity &&
		target.providerLocalID &&
		target.name &&
		target.type
	) {
		return {
			direct: {
				providerIdentity: target.providerIdentity,
				providerLocalID: target.providerLocalID,
				name: target.name,
				type: target.type,
			},
		};
	}

	throw new Error(`Tool target "${target.name}" has no usable Session capability identity.`);
}

function parseOptionalJSON(raw: string | undefined, field: string): unknown {
	if (!raw?.trim()) {
		return undefined;
	}

	try {
		return JSON.parse(raw);
	} catch {
		throw new Error(`${field} must contain valid JSON.`);
	}
}

function modelOverridesFromComposer(
	model: UIModelOption,
	includeModelDefault: boolean | undefined
): SessionModelDefaultsOverride {
	const defaults = model.requestPatch?.defaults;
	const clear = new Set(model.requestPatch?.clear);

	if (includeModelDefault === false) {
		clear.add(SessionModelDefaultField.SystemPrompt);
	} else if (includeModelDefault === true) {
		clear.delete(SessionModelDefaultField.SystemPrompt);
	}

	return {
		...(defaults?.stream === undefined ? {} : { stream: defaults.stream }),
		...(defaults?.maxPromptTokens === undefined ? {} : { maxPromptTokens: defaults.maxPromptTokens }),
		...(defaults?.maxOutputTokens === undefined ? {} : { maxOutputTokens: defaults.maxOutputTokens }),
		...(defaults?.temperature === undefined ? {} : { temperature: defaults.temperature }),
		...(defaults?.systemPrompt === undefined ? {} : { systemPrompt: defaults.systemPrompt }),
		...(defaults?.timeoutMS === undefined ? {} : { timeoutMS: defaults.timeoutMS }),
		...(defaults?.reasoning ? { reasoning: structuredClone(defaults.reasoning) } : {}),
		...(defaults?.cacheControl ? { cacheControl: structuredClone(defaults.cacheControl) } : {}),
		...(defaults?.output ? { output: structuredClone(defaults.output) } : {}),
		...(defaults?.stopSequences ? { stopSequences: [...defaults.stopSequences] } : {}),
		...(defaults?.adapterParameters
			? {
					adapterParameters: structuredClone(defaults.adapterParameters),
				}
			: {}),
		...(clear.size > 0 ? { clear: [...clear] } : {}),
	};
}

function sessionAttachmentFromComposer(value: Attachment) {
	const label = value.label?.trim() || 'Attachment';

	switch (value.kind) {
		case AttachmentKind.file:
			return {
				kind: SessionAttachmentReferenceKind.File,
				label,
				mode: value.mode,
				availableModes: value.availableContentBlockModes,
				file: value.fileRef
					? {
							path: value.fileRef.path,
							originalPath: value.fileRef.origPath,
							name: value.fileRef.name,
							exists: value.fileRef.exists,
							isDirectory: value.fileRef.isDir,
							size: value.fileRef.size,
							originalSize: value.fileRef.origSize,
							modifiedAt: value.fileRef.modTime,
							originalModifiedAt: value.fileRef.origModTime,
						}
					: undefined,
			};

		case AttachmentKind.image:
			return {
				kind: SessionAttachmentReferenceKind.Image,
				label,
				mode: value.mode,
				availableModes: value.availableContentBlockModes,
				image: value.imageRef
					? {
							path: value.imageRef.path,
							originalPath: value.imageRef.origPath,
							name: value.imageRef.name,
							exists: value.imageRef.exists,
							isDirectory: value.imageRef.isDir,
							size: value.imageRef.size,
							originalSize: value.imageRef.origSize,
							modifiedAt: value.imageRef.modTime,
							originalModifiedAt: value.imageRef.origModTime,
							mimeType: value.imageRef.mimeType,
							width: value.imageRef.width,
							height: value.imageRef.height,
							format: value.imageRef.format,
						}
					: undefined,
			};

		case AttachmentKind.url:
			return {
				kind: SessionAttachmentReferenceKind.URL,
				label,
				mode: value.mode,
				availableModes: value.availableContentBlockModes,
				url: value.urlRef
					? {
							url: value.urlRef.url,
							normalized: value.urlRef.normalized,
							original: value.urlRef.origNormalized,
						}
					: undefined,
			};

		default:
			return {
				kind: SessionAttachmentReferenceKind.Generic,
				label,
				mode: value.mode,
				availableModes: value.availableContentBlockModes,
				generic: {
					handle: value.genericRef?.handle || label,
					originalHandle: value.genericRef?.origHandle,
				},
			};
	}
}

function toolOverridesFromChoices(payload: EditorSubmitPayload): SessionToolOverride[] {
	const seen = new Set<string>();
	const overrides: SessionToolOverride[] = [];

	for (const choice of payload.finalToolChoices) {
		const key = toolIdentityKey(choice.target);

		if (seen.has(key)) {
			continue;
		}
		seen.add(key);

		overrides.push({
			key: sessionCapabilityKey(choice.target),
			mode: SessionCollectionOverrideMode.Add,
			value: {
				key,
				capability:
					choice.target.form === CapabilityTargetForm.Artifact && choice.target.artifact
						? {
								artifact: artifactSelection(choice.target.artifact, choice.artifactRevision, choice.definitionDigest),
							}
						: sessionCapabilitySelection(choice.target),
				autoExecute: choice.autoExecute,
				userArguments: parseOptionalJSON(choice.userArgSchemaInstance, `Tool "${choice.target.name}" options`),
			},
		});
	}

	return overrides;
}

function artifactOverridesForWorkspace(
	baselineRefs: ArtifactRef[],
	selectedRefs: Array<{ artifact: ArtifactRef }> | undefined
): SessionArtifactOverride[] {
	const selectedByKey = new Map((selectedRefs ?? []).map(value => [artifactRefKey(value.artifact), value.artifact]));
	const baselineByKey = new Map(baselineRefs.map(ref => [artifactRefKey(ref), ref]));
	const output: SessionArtifactOverride[] = [];

	for (const [key] of baselineByKey) {
		if (!selectedByKey.has(key)) {
			output.push({
				key,
				mode: SessionCollectionOverrideMode.Remove,
			});
		}
	}

	for (const [key, ref] of selectedByKey) {
		if (baselineByKey.has(key)) {
			continue;
		}

		output.push({
			key,
			mode: SessionCollectionOverrideMode.Add,
			value: artifactSelection(ref),
		});
	}

	return output;
}

function mcpOverridesForWorkspace(
	baselineRefs: ArtifactRef[],
	selectedRefs: WorkspaceConversationResourceSelectionRef[] | undefined
): MCPServerOverride[] {
	const baselineByKey = new Map(baselineRefs.map(ref => [artifactRefKey(ref), ref] as const));
	const selectedByKey = new Map(
		(selectedRefs ?? []).map(selection => [artifactRefKey(selection.artifact), selection] as const)
	);
	const output: MCPServerOverride[] = [];

	for (const [key] of baselineByKey) {
		if (selectedByKey.has(key)) {
			continue;
		}

		output.push({
			key,
			mode: SessionCollectionOverrideMode.Remove,
		});
	}

	for (const [key, selection] of selectedByKey) {
		if (baselineByKey.has(key)) {
			continue;
		}

		output.push({
			key,
			mode: SessionCollectionOverrideMode.Add,
			value: {
				server: artifactSelection(selection.artifact, selection.artifactRevision, selection.definitionDigest),
				toolExposure: MCPToolExposure.None,
			},
		});
	}

	return output;
}

function workspaceSelectionFromPayload(payload: EditorSubmitPayload): SessionConfiguration['workspace'] {
	const selection = payload.workspaceSelection;

	if (!selection) {
		return undefined;
	}

	const plan: WorkspaceRuntimePlan | undefined = payload.workspacePlan;
	const baselineContexts = plan?.prompt.contributions.map(value => value.artifact) ?? [];
	const baselineSkills = plan?.skills.skills.map(value => value.artifact) ?? [];
	const baselineMCPServers = plan?.mcpServers.servers.map(value => value.artifact) ?? [];

	return {
		workspace: artifactSelection(selection.workspace, selection.workspaceRevision),
		context: artifactOverridesForWorkspace(baselineContexts, selection.contextRefs),
		skills: artifactOverridesForWorkspace(baselineSkills, selection.skillRefs),
		mcp: mcpOverridesForWorkspace(baselineMCPServers, selection.mcpRefs),
	};
}

function instructionOverridesFromPayload(payload: EditorSubmitPayload): SessionInstructionOverride[] {
	const output = (payload.selectedSessionInstructionOverrides ?? []).map(override => structuredClone(override));
	const text = payload.additionalSessionInstructionText?.trim();

	if (!text) {
		return output;
	}

	const key = 'inline:composer-system-instructions';
	const inlineOverride: SessionInstructionOverride = {
		key,
		mode: SessionCollectionOverrideMode.Add,
		value: {
			key,
			label: 'Composer system instructions',
			text,
		},
	};
	const existingIndex = output.findIndex(override => override.key === key);

	if (existingIndex >= 0) {
		output[existingIndex] = inlineOverride;
	} else {
		output.push(inlineOverride);
	}

	return output;
}

function buildSessionConfiguration(
	payload: EditorSubmitPayload,
	model: UIModelOption,
	directToolOverrides: SessionToolOverride[],
	skillOverrides: ReturnType<typeof skillOverridesFromPayload>,
	agentExclusions: ReturnType<typeof agentCapabilityExclusionOverrides>,
	directMCPOverrides: MCPServerOverride[]
): SessionConfiguration {
	const configuration = emptySessionConfiguration();
	const nonSessionSkillKeys = nonSessionSkillKeysFromPayload(payload);
	const directToolKeys = new Set(directToolOverrides.map(override => override.key));
	const directSkillKeys = new Set(skillOverrides.skills.map(override => override.key));
	const nonSessionOnlySkillKeys = new Set([...nonSessionSkillKeys].filter(key => !directSkillKeys.has(key)));
	const nonSessionSkillRemovals = nonSessionSkillRemovalOverrides(nonSessionOnlySkillKeys);
	const instructionOverrides = instructionOverridesFromPayload(payload);
	const directMCPKeys = new Set(directMCPOverrides.map(override => override.key));

	configuration.agent = payload.agentSelection ?? (payload.agentRef ? artifactSelection(payload.agentRef) : undefined);
	configuration.workspace = workspaceSelectionFromPayload(payload);
	configuration.model =
		payload.modelSelectionSource === 'agent' && payload.agentRef
			? undefined
			: model.model
				? artifactSelection(model.model, model.modelArtifactRevision, model.modelDefinitionDigest)
				: undefined;
	configuration.modelOverrides = modelOverridesFromComposer(model, payload.includeModelDefault);
	configuration.instructions = [...instructionOverrides, ...agentExclusions.instructions];
	configuration.tools = [
		...directToolOverrides,
		...agentExclusions.tools.filter(override => !directToolKeys.has(override.key)),
	];
	configuration.skills = [
		...skillOverrides.skills,
		...agentExclusions.skills.filter(
			override => !directSkillKeys.has(override.key) && !nonSessionOnlySkillKeys.has(override.key)
		),
		...nonSessionSkillRemovals,
	];
	configuration.activeSkills = skillOverrides.active;
	configuration.mcp = [
		...directMCPOverrides,
		...agentExclusions.mcp.filter(override => !directMCPKeys.has(override.key)),
	];
	configuration.history = {
		all: model.includePreviousMessages === 'all',
		...(typeof model.includePreviousMessages === 'number'
			? {
					includePreviousUserTurns: Math.max(0, model.includePreviousMessages),
				}
			: {}),
	};

	return configuration;
}

export async function buildSessionSubmissionIntent(
	payload: EditorSubmitPayload,
	model: UIModelOption
): Promise<SessionSubmissionIntent> {
	const skillOverrides = skillOverridesFromPayload(payload);
	const agentExclusions = agentCapabilityExclusionOverrides(payload.agentCapabilityExclusions);
	const directToolOverrides = toolOverridesFromChoices(payload);
	const directMCPOverrides = await mcpOverridesFromPayload(payload);
	const generatedConfiguration = buildSessionConfiguration(
		payload,
		model,
		directToolOverrides,
		skillOverrides,
		agentExclusions,
		directMCPOverrides
	);
	const configuration = payload.sessionConfigurationBaseline
		? mergeSessionConfiguration(payload.sessionConfigurationBaseline, generatedConfiguration, payload)
		: generatedConfiguration;

	return {
		configuration,
		input: {
			text: payload.text,
			attachments: payload.attachments.map(sessionAttachmentFromComposer),
			mcpAppContext: appContextUpdatesFromPayload(payload),
		},
	};
}

function agentCapabilityExclusionOverrides(exclusions: AgentCapabilityExclusion[] | undefined): {
	instructions: SessionInstructionOverride[];
	tools: SessionToolOverride[];
	skills: SessionArtifactOverride[];
	mcp: MCPServerOverride[];
} {
	const instructions: SessionInstructionOverride[] = [];
	const tools: SessionToolOverride[] = [];
	const skills: SessionArtifactOverride[] = [];
	const mcp: MCPServerOverride[] = [];

	for (const exclusion of exclusions ?? []) {
		if (!exclusion.key.trim()) {
			continue;
		}

		switch (exclusion.kind) {
			case 'instruction':
				if (exclusion.target.form === CapabilityTargetForm.Artifact && exclusion.target.artifact) {
					instructions.push({
						key: exclusion.key,
						mode: SessionCollectionOverrideMode.Remove,
					});
				}
				break;

			case 'tool':
				tools.push({
					key: exclusion.key,
					mode: SessionCollectionOverrideMode.Remove,
				});
				break;

			case 'skill':
				if (exclusion.target.form === CapabilityTargetForm.Artifact && exclusion.target.artifact) {
					skills.push({
						key: exclusion.key,
						mode: SessionCollectionOverrideMode.Remove,
					});
				}
				break;

			case 'mcp':
				if (exclusion.target.form === CapabilityTargetForm.Artifact && exclusion.target.artifact) {
					mcp.push({
						key: exclusion.key,
						mode: SessionCollectionOverrideMode.Remove,
					});
				}
				break;
		}
	}

	return { instructions, tools, skills, mcp };
}

function skillOverridesFromPayload(payload: EditorSubmitPayload) {
	const seenEnabled = new Set<string>();
	const skills: SessionArtifactOverride[] = [];

	for (const selection of payload.enabledSkillSelections ?? payload.enabledSkillRefs ?? []) {
		const ref = artifactSelection(
			selection,
			'expectedRevision' in selection ? (selection.expectedRevision as number | undefined) : undefined,
			'expectedDigest' in selection ? (selection.expectedDigest as string | undefined) : undefined
		);
		const key = artifactRefKey(ref);

		if (seenEnabled.has(key)) {
			continue;
		}
		seenEnabled.add(key);

		skills.push({
			key,
			mode: SessionCollectionOverrideMode.Add,
			value: ref,
		});
	}

	const activeByKey = new Map<string, SessionActiveSkillOverride>();

	for (const selection of payload.activeSkillSelections ?? payload.activeSkillRefs ?? []) {
		const ref = artifactSelection(
			selection,
			'expectedRevision' in selection ? (selection.expectedRevision as number | undefined) : undefined,
			'expectedDigest' in selection ? (selection.expectedDigest as string | undefined) : undefined
		);
		const key = artifactRefKey(ref);

		if (activeByKey.has(key)) {
			continue;
		}

		activeByKey.set(key, {
			key,
			mode: SessionActivityOverrideMode.Activate,
			value: ref,
		});
	}

	for (const override of payload.agentSkillActivityOverrides ?? []) {
		if (!override.key || activeByKey.has(override.key)) {
			continue;
		}

		activeByKey.set(override.key, structuredClone(override));
	}

	return {
		skills,
		active: [...activeByKey.values()],
	};
}

async function mcpOverridesFromPayload(payload: EditorSubmitPayload): Promise<MCPServerOverride[]> {
	const context = payload.mcpContext;

	if (!context?.servers.length) {
		return [];
	}

	const resourcesByServer = new Map<string, typeof context.resources>();
	for (const resource of context.resources ?? []) {
		const values = resourcesByServer.get(resource.server) ?? [];
		resourcesByServer.set(resource.server, [...values, resource]);
	}

	const templatesByServer = new Map<string, typeof context.resourceTemplates>();
	for (const template of context.resourceTemplates ?? []) {
		const values = templatesByServer.get(template.server) ?? [];
		templatesByServer.set(template.server, [...values, template]);
	}

	const promptsByServer = new Map<string, typeof context.prompts>();
	for (const prompt of context.prompts ?? []) {
		const values = promptsByServer.get(prompt.server) ?? [];
		promptsByServer.set(prompt.server, [...values, prompt]);
	}

	const overrides: MCPServerOverride[] = [];

	for (const serverSelection of context.servers) {
		const server = await mcpManagementAPI.findComposerServerByRuntimeID(serverSelection.server);

		if (!server) {
			throw new Error(`Selected MCP server "${serverSelection.server}" is no longer available.`);
		}

		const ref: ArtifactRef = server.ref;
		const selection: MCPServerSelection = {
			server: artifactSelection(ref, server.artifact.revision, server.artifact.resolvedDefinition),
			includeServerInstructions: serverSelection.includeServerInstructions,
			toolExposure: serverSelection.toolExposure,
			selectedTools: (serverSelection.selectedTools ?? []).map(tool => ({
				toolName: tool.toolName,
				expectedDigest: tool.digest,
				approvalRule: tool.approvalRule,
				executionMode: tool.executionMode,
			})),
			resources: (resourcesByServer.get(serverSelection.server) ?? []).map(resource => ({
				uri: resource.uri,
				expectedDigest: resource.digest,
			})),
			resourceTemplates: (templatesByServer.get(serverSelection.server) ?? []).map(template => ({
				uriTemplate: template.uriTemplate,
				expectedDigest: template.digest,
				arguments: template.argumentValues,
			})),
			prompts: (promptsByServer.get(serverSelection.server) ?? []).map(prompt => ({
				promptName: prompt.promptName,
				expectedDigest: prompt.digest,
				arguments: prompt.argumentValues,
			})),
		};

		overrides.push({
			key: artifactRefKey(ref),
			mode: SessionCollectionOverrideMode.Add,
			value: selection,
		});
	}

	return overrides;
}

function appContextUpdatesFromPayload(payload: EditorSubmitPayload) {
	const output = [];

	for (const update of payload.mcpAppContextUpdates ?? []) {
		if (
			typeof update.instanceID !== 'string' ||
			!update.instanceID.trim() ||
			typeof update.server !== 'string' ||
			!update.server.trim() ||
			typeof update.viewID !== 'string' ||
			!update.viewID.trim()
		) {
			continue;
		}

		output.push({
			appInstanceID: update.instanceID,
			viewID: update.viewID,
			serverID: update.server,
			resourceURI: update.resourceUri,
			content: update.content?.map(content => ({
				type: content.type,
				text: content.text ?? content.resource?.text,
				uri: content.uri ?? content.resource?.uri,
				name: content.name,
				description: content.description,
				mimeType: content.mimeType ?? content.resource?.mimeType,
				data: content.data ?? content.resource?.blob,
			})),
			structuredContent: update.structuredContent,
			updatedAt: update.updatedAt,
		});
	}

	return output;
}
