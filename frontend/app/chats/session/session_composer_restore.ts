import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type {
	MCPConversationContext,
	MCPPromptRef,
	MCPPromptSelection,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPResourceTemplateSelection,
	MCPToolSelection,
} from '@/spec/mcp';
import type {
	SessionActiveSkillOverride,
	SessionArtifactOverride,
	SessionArtifactSelection,
	SessionMCPServerOverride,
	SessionMCPServerSelection,
	SessionToolOverride,
} from '@/spec/session';
import type { ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import { CapabilityTargetForm, CapabilityTargetProvenance } from '@/spec/artifact';
import { SessionActivityOverrideMode, SessionCollectionOverrideMode } from '@/spec/session';

import { ensureMakeID, getUUIDv7 } from '@/lib/uuid_utils';

import { mcpManagementAPI, toolManagementAPI } from '@/apis/baseapi';
import { toolStoreChoiceFromSelection } from '@/apis/tool_management';

function artifactRef(selection: SessionArtifactSelection): ArtifactRef {
	return {
		rootID: selection.rootID,
		artifactID: selection.artifactID,
	};
}

function artifactKey(selection: SessionArtifactSelection): string {
	return `artifact:${selection.rootID}:${selection.artifactID}`;
}

function commandChoiceID(): string {
	try {
		return getUUIDv7();
	} catch {
		return ensureMakeID();
	}
}

function rawJSONText(value: unknown): string | undefined {
	if (value === undefined || value === null) {
		return undefined;
	}

	if (typeof value === 'string') {
		try {
			JSON.parse(value);
			return value;
		} catch {
			return JSON.stringify(value);
		}
	}

	const encoded = JSON.stringify(value);
	return encoded === undefined ? undefined : encoded;
}

function directCapabilityTarget(
	selection: NonNullable<SessionToolOverride['value']>['capability']['direct']
): CapabilityTarget | undefined {
	if (!selection) {
		return undefined;
	}

	return {
		form: CapabilityTargetForm.Direct,
		type: selection.type,
		name: selection.name,
		provenance: CapabilityTargetProvenance.DirectCapability,
		providerIdentity: selection.providerIdentity,
		providerLocalID: selection.providerLocalID,
	};
}

export function effectiveArtifactSelections(
	overrides: SessionArtifactOverride[] | undefined
): SessionArtifactSelection[] {
	const byKey = new Map<string, SessionArtifactSelection>();
	const order: string[] = [];
	let replacementStarted = false;

	const put = (selection: SessionArtifactSelection) => {
		const key = artifactKey(selection);
		if (!byKey.has(key)) {
			order.push(key);
		}
		byKey.set(key, selection);
	};

	for (const override of overrides ?? []) {
		switch (override.mode) {
			case SessionCollectionOverrideMode.Inherit:
				break;

			case SessionCollectionOverrideMode.Remove:
				byKey.delete(override.key);
				break;

			case SessionCollectionOverrideMode.Replace:
				if (!replacementStarted) {
					byKey.clear();
					order.length = 0;
					replacementStarted = true;
				}
				if (override.value) {
					put(override.value);
				}
				break;

			case SessionCollectionOverrideMode.Add:
				if (override.value) {
					put(override.value);
				}
				break;
		}
	}

	return order.flatMap(key => {
		const selection = byKey.get(key);
		return selection ? [selection] : [];
	});
}

export function effectiveActiveSkillSelections(
	overrides: SessionActiveSkillOverride[] | undefined
): SessionArtifactSelection[] {
	const byKey = new Map<string, SessionArtifactSelection>();

	for (const override of overrides ?? []) {
		switch (override.mode) {
			case SessionActivityOverrideMode.Inherit:
				break;

			case SessionActivityOverrideMode.Deactivate:
				byKey.delete(override.key);
				break;

			case SessionActivityOverrideMode.Activate:
				if (override.value) {
					byKey.set(artifactKey(override.value), override.value);
				}
				break;
		}
	}

	return [...byKey.values()];
}

function effectiveToolOverrides(overrides: SessionToolOverride[] | undefined): SessionToolOverride[] {
	const byKey = new Map<string, SessionToolOverride>();
	const order: string[] = [];
	let replacementStarted = false;

	for (const override of overrides ?? []) {
		switch (override.mode) {
			case SessionCollectionOverrideMode.Inherit:
				continue;

			case SessionCollectionOverrideMode.Remove:
				byKey.delete(override.key);
				continue;

			case SessionCollectionOverrideMode.Replace:
				if (!replacementStarted) {
					byKey.clear();
					order.length = 0;
					replacementStarted = true;
				}
				break;

			case SessionCollectionOverrideMode.Add:
				break;
		}

		if (!override.value) {
			continue;
		}
		if (!byKey.has(override.key)) {
			order.push(override.key);
		}
		byKey.set(override.key, override);
	}

	return order.flatMap(key => {
		const value = byKey.get(key);
		return value ? [value] : [];
	});
}

export async function hydrateSessionToolOverrides(
	overrides: SessionToolOverride[] | undefined
): Promise<{ choices: ToolStoreChoice[]; issues: ToolSelectionIssue[] }> {
	const choices: ToolStoreChoice[] = [];
	const issues: ToolSelectionIssue[] = [];

	for (const override of effectiveToolOverrides(overrides)) {
		const value = override.value;
		if (!value) {
			continue;
		}

		try {
			const target = value.capability.artifact
				? await toolManagementAPI.mapToolTarget(artifactRef(value.capability.artifact))
				: directCapabilityTarget(value.capability.direct);

			if (!target) {
				throw new Error('The saved Tool capability has no usable target.');
			}

			const resolved = await toolManagementAPI.resolveToolTarget(target);
			choices.push(
				toolStoreChoiceFromSelection(
					{
						choiceID: commandChoiceID(),
						target,
						autoExecute: value.autoExecute ?? false,
						userArgSchemaInstance: rawJSONText(value.userArguments),
					},
					resolved
				)
			);
		} catch (error) {
			const target = value.capability.artifact
				? ({
						form: CapabilityTargetForm.Artifact,
						type: 'tool',
						name: value.key || value.capability.artifact.artifactID,
						provenance: CapabilityTargetProvenance.CurrentRoot,
						artifact: artifactRef(value.capability.artifact),
					} satisfies CapabilityTarget)
				: directCapabilityTarget(value.capability.direct);

			if (target) {
				issues.push({
					selection: {
						choiceID: commandChoiceID(),
						target,
						autoExecute: value.autoExecute ?? false,
						userArgSchemaInstance: rawJSONText(value.userArguments),
					},
					message: error instanceof Error ? error.message : 'Saved Tool selection could not be restored.',
				});
			}
		}
	}

	return { choices, issues };
}

function effectiveMCPSelections(overrides: SessionMCPServerOverride[] | undefined): SessionMCPServerSelection[] {
	const byKey = new Map<string, SessionMCPServerSelection>();
	const order: string[] = [];
	let replacementStarted = false;

	for (const override of overrides ?? []) {
		switch (override.mode) {
			case SessionCollectionOverrideMode.Inherit:
				continue;

			case SessionCollectionOverrideMode.Remove:
				byKey.delete(override.key);
				continue;

			case SessionCollectionOverrideMode.Replace:
				if (!replacementStarted) {
					byKey.clear();
					order.length = 0;
					replacementStarted = true;
				}
				break;

			case SessionCollectionOverrideMode.Add:
				break;
		}

		if (!override.value) {
			continue;
		}
		if (!byKey.has(override.key)) {
			order.push(override.key);
		}
		byKey.set(override.key, override.value);
	}

	return order.flatMap(key => {
		const selection = byKey.get(key);
		return selection ? [selection] : [];
	});
}

export async function hydrateSessionMCPOverrides(
	overrides: SessionMCPServerOverride[] | undefined
): Promise<MCPConversationContext | undefined> {
	const selections = effectiveMCPSelections(overrides);
	if (selections.length === 0) {
		return undefined;
	}

	const serverIDs = await mcpManagementAPI.resolveMCPRuntimeServerIDs(selections.map(selection => artifactRef(selection.server)));
	const servers = [];
	const resources: MCPResourceRef[] = [];
	const resourceTemplates: MCPResourceTemplateSelection[] = [];
	const prompts: MCPPromptSelection[] = [];

	for (const selection of selections) {
		const runtimeServerID = serverIDs.get(`${selection.server.rootID}:${selection.server.artifactID}`);
		if (!runtimeServerID) {
			throw new Error(`MCP server "${selection.server.artifactID}" could not be restored.`);
		}

		const selectedTools: MCPToolSelection[] = (selection.selectedTools ?? []).map(tool => ({
			server: runtimeServerID,
			toolName: tool.toolName,
			digest: tool.expectedDigest,
			approvalRule: tool.approvalRule as MCPToolSelection['approvalRule'],
			executionMode: tool.executionMode as MCPToolSelection['executionMode'],
		}));

		servers.push({
			server: runtimeServerID,
			toolExposure: selection.toolExposure,
			selectedTools: selectedTools.length > 0 ? selectedTools : undefined,
			includeServerInstructions: selection.includeServerInstructions,
		});

		for (const resource of selection.resources ?? []) {
			resources.push({
				server: runtimeServerID,
				uri: resource.uri,
				displayName: resource.uri,
				digest: resource.expectedDigest,
			});
		}

		for (const template of selection.resourceTemplates ?? []) {
			resourceTemplates.push({
				server: runtimeServerID,
				uriTemplate: template.uriTemplate,
				displayName: template.uriTemplate,
				digest: template.expectedDigest,
				argumentValues: template.arguments,
			} satisfies MCPResourceTemplateRef & MCPResourceTemplateSelection);
		}

		for (const prompt of selection.prompts ?? []) {
			prompts.push({
				server: runtimeServerID,
				promptName: prompt.promptName,
				displayName: prompt.promptName,
				digest: prompt.expectedDigest,
				argumentValues: prompt.arguments,
			} satisfies MCPPromptRef & MCPPromptSelection);
		}
	}

	return {
		servers,
		resources: resources.length > 0 ? resources : undefined,
		resourceTemplates: resourceTemplates.length > 0 ? resourceTemplates : undefined,
		prompts: prompts.length > 0 ? prompts : undefined,
	};
}
