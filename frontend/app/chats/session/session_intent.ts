import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { Attachment } from '@/spec/attachment';
import type { UIModelOption } from '@/spec/model';
import type {
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
import { CapabilityTargetForm } from '@/spec/artifact';
import { AttachmentKind } from '@/spec/attachment';
import { emptySessionConfiguration, SessionModelDefaultField } from '@/spec/session';
import {
	SessionActivityOverrideMode,
	SessionAttachmentReferenceKind,
	SessionCollectionOverrideMode,
} from '@/spec/session_base';
import { MCPToolExposure } from '@/spec/session_mcp';

import { mcpManagementAPI } from '@/apis/baseapi';

import type { EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

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

function artifactKey(ref: ArtifactRef): string {
	return `artifact:${ref.rootID}:${ref.artifactID}`;
}

function sessionCapabilityKey(target: CapabilityTarget): string {
	if (target.form === CapabilityTargetForm.Artifact && target.artifact) {
		return artifactKey(target.artifact);
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
	const selectedByKey = new Map((selectedRefs ?? []).map(value => [artifactKey(value.artifact), value.artifact]));
	const baselineByKey = new Map(baselineRefs.map(ref => [artifactKey(ref), ref]));
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
	const baselineByKey = new Map(baselineRefs.map(ref => [artifactKey(ref), ref] as const));
	const selectedByKey = new Map(
		(selectedRefs ?? []).map(selection => [artifactKey(selection.artifact), selection] as const)
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
	const text = payload.sessionInstructionText?.trim();

	if (!text) {
		return [];
	}

	return [
		{
			key: 'inline:composer-system-instructions',
			mode: SessionCollectionOverrideMode.Add,
			value: {
				key: 'inline:composer-system-instructions',
				label: 'Composer system instructions',
				text,
			},
		},
	];
}

function skillOverridesFromPayload(payload: EditorSubmitPayload) {
	const seenEnabled = new Set<string>();
	const skills = [];

	for (const selection of payload.enabledSkillSelections ?? payload.enabledSkillRefs ?? []) {
		const ref = artifactSelection(
			selection,
			'expectedRevision' in selection ? (selection.expectedRevision as number | undefined) : undefined,
			'expectedDigest' in selection ? (selection.expectedDigest as string | undefined) : undefined
		);
		const key = artifactKey(ref);

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

	const enabled = new Set(skills.map(value => value.key));
	const active = [];

	for (const selection of payload.activeSkillSelections ?? payload.activeSkillRefs ?? []) {
		const ref = artifactSelection(
			selection,
			'expectedRevision' in selection ? (selection.expectedRevision as number | undefined) : undefined,
			'expectedDigest' in selection ? (selection.expectedDigest as string | undefined) : undefined
		);
		const key = artifactKey(ref);

		if (!enabled.has(key)) {
			continue;
		}

		active.push({
			key,
			mode: SessionActivityOverrideMode.Activate,
			value: ref,
		});
	}

	return {
		skills,
		active,
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
			key: artifactKey(ref),
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
			})),
			structuredContent: update.structuredContent,
			updatedAt: update.updatedAt,
		});
	}

	return output;
}

export async function buildSessionSubmissionIntent(
	payload: EditorSubmitPayload,
	model: UIModelOption
): Promise<SessionSubmissionIntent> {
	const configuration = emptySessionConfiguration();
	const skillOverrides = skillOverridesFromPayload(payload);

	configuration.agent = payload.agentSelection ?? (payload.agentRef ? artifactSelection(payload.agentRef) : undefined);
	configuration.workspace = workspaceSelectionFromPayload(payload);
	configuration.model =
		payload.modelSelectionSource === 'agent' && payload.agentRef
			? undefined
			: model.model
				? artifactSelection(model.model, model.modelArtifactRevision, model.modelDefinitionDigest)
				: undefined;
	configuration.modelOverrides = modelOverridesFromComposer(model, payload.includeModelDefault);
	configuration.instructions = instructionOverridesFromPayload(payload);
	configuration.tools = toolOverridesFromChoices(payload);
	configuration.skills = skillOverrides.skills;
	configuration.activeSkills = skillOverrides.active;
	configuration.mcp = await mcpOverridesFromPayload(payload);
	configuration.history = {
		all: model.includePreviousMessages === 'all',
		...(typeof model.includePreviousMessages === 'number'
			? {
					includePreviousUserTurns: Math.max(0, model.includePreviousMessages),
				}
			: {}),
	};

	return {
		configuration,
		input: {
			text: payload.text,
			attachments: payload.attachments.map(sessionAttachmentFromComposer),
			mcpAppContext: appContextUpdatesFromPayload(payload),
		},
	};
}
