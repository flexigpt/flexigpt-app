import type { AssistantPreset } from '@/spec/assistantpreset';
import {
	BASE_ASSISTANT_PRESET_BUNDLEID,
	BASE_ASSISTANT_PRESET_SLUG,
	BASE_ASSISTANT_PRESET_VERSION,
} from '@/spec/assistantpreset';
import type { MCPConversationContext } from '@/spec/mcp';
import type { UIChatOption } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';
import { ToolStoreChoiceType } from '@/spec/tool';

import { areComparableValuesEqual } from '@/lib/obj_utils';

import type { SystemInstructionSource } from '@/chats/composer/skills/prompt_utils';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import { areSkillRefListsEqual } from '@/skills/lib/skill_identity_utils';
import { areToolChoiceListsEqual } from '@/tools/lib/tool_choice_utils';

export interface AssistantPresetOptionItem {
	key: string;
	bundleID: string;
	bundleSlug: string;
	bundleDisplayName: string;
	displayName: string;
	description?: string;
	preset: AssistantPreset;
	label: string;

	isSelectable: boolean;
	availabilityReason?: string;
}

interface AssistantPresetPreparedRuntimeSelections {
	hasToolsSelection: boolean;
	conversationToolChoices: ToolStoreChoice[];
	webSearchChoices: ToolStoreChoice[];
	hasSkillsSelection: boolean;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	hasMCPSelection: boolean;
	mcpContext?: MCPConversationContext;
}

interface AssistantPresetComparisonState {
	model?: Record<string, unknown>;
	instructions?: string[];
	tools?: {
		conversationToolChoices: AssistantPresetNormalizedToolChoice[];
		webSearchChoices: AssistantPresetNormalizedToolChoice[];
	};
	skills?: string[];
	activeSkills?: string[];
	mcp?: MCPConversationContext;
}

export interface AssistantPresetPreparedApplication {
	presetKey: string;
	option: AssistantPresetOptionItem;
	preset: AssistantPreset;

	hasStartingTextSelection: boolean;
	nextStartingText: string;

	hasModelSelection: boolean;
	nextSelectedModel: UIChatOption;

	hasIncludeModelSystemPromptSelection: boolean;
	nextIncludeModelSystemPrompt: boolean;

	hasInstructionSourceSelection: boolean;
	nextSelectedInstructionSourceKeys: string[];
	preparedInstructionSources: SystemInstructionSource[];

	runtimeSelections: AssistantPresetPreparedRuntimeSelections;
	comparisonState: AssistantPresetComparisonState;
}

export interface AssistantPresetRuntimeSnapshot {
	conversationToolChoices: ToolStoreChoice[];
	webSearchChoices: ToolStoreChoice[];
	enabledSkillRefs: SkillRef[];
	activeSkillRefs?: SkillRef[];
	mcpContext?: MCPConversationContext;
}

interface AssistantPresetNormalizedToolChoice {
	toolType: ToolStoreChoiceType;
	bundleID: string;
	toolSlug: string;
	toolVersion: string;
	autoExecute: boolean;
	userArgSchemaInstance?: string;
}

export interface AssistantPresetModificationSummary {
	model: boolean;
	instructions: boolean;
	tools: boolean;
	skills: boolean;
	mcp: boolean;
	any: boolean;
	modifiedLabels: string[];
}

export const EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT: AssistantPresetRuntimeSnapshot = {
	conversationToolChoices: [],
	webSearchChoices: [],
	enabledSkillRefs: [],
	activeSkillRefs: [],
	mcpContext: undefined,
};

export const EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY: AssistantPresetModificationSummary = {
	model: false,
	instructions: false,
	tools: false,
	skills: false,
	mcp: false,
	any: false,
	modifiedLabels: [],
};

export function areAssistantRuntimeSnapshotsEqual(
	a: AssistantPresetRuntimeSnapshot,
	b: AssistantPresetRuntimeSnapshot
): boolean {
	return (
		areToolChoiceListsEqual(a.conversationToolChoices, b.conversationToolChoices) &&
		areToolChoiceListsEqual(a.webSearchChoices, b.webSearchChoices) &&
		areSkillRefListsEqual(a.enabledSkillRefs, b.enabledSkillRefs) &&
		areSkillRefListsEqual(a.activeSkillRefs, b.activeSkillRefs) &&
		areComparableValuesEqual(
			normalizeAssistantPresetMCPContext(a.mcpContext),
			normalizeAssistantPresetMCPContext(b.mcpContext)
		)
	);
}

export function buildAssistantPresetIdentityKey(
	bundleID: string,
	assistantPresetSlug: string,
	version: string
): string {
	return `${bundleID}/${assistantPresetSlug}@${version}`;
}

export function buildAssistantPresetModelComparisonState(
	preset: AssistantPreset,
	selectedModel: UIChatOption,
	includeModelSystemPrompt: boolean
): Record<string, unknown> | undefined {
	const modelState: Record<string, unknown> = {};

	if (preset.startingModelPresetRef) {
		modelState.modelRef = {
			providerName: selectedModel.providerName,
			modelPresetID: selectedModel.modelPresetID,
		};
	}

	if (preset.startingIncludeModelSystemPrompt !== undefined) {
		modelState.includeModelSystemPrompt = includeModelSystemPrompt;
	}

	return Object.keys(modelState).length > 0 ? modelState : undefined;
}

export function normalizeAssistantPresetToolChoices(
	choices: Array<
		Pick<
			ToolStoreChoice,
			'bundleID' | 'toolSlug' | 'toolVersion' | 'toolType' | 'autoExecute' | 'userArgSchemaInstance'
		>
	>
): AssistantPresetNormalizedToolChoice[] {
	return choices.map(choice => ({
		toolType: choice.toolType,
		bundleID: choice.bundleID,
		toolSlug: choice.toolSlug,
		toolVersion: choice.toolVersion,
		autoExecute: choice.autoExecute,
		userArgSchemaInstance: choice.userArgSchemaInstance?.trim() || undefined,
	}));
}

export function normalizeAssistantPresetSkillRefs(refs: SkillRef[]): string[] {
	return [...new Set(refs.map(ref => `${ref.bundleID}/${ref.skillSlug}#${ref.skillID}`))].toSorted();
}

export function mapAssistantPresetWebSearchTemplatesToChoices(templates: WebSearchChoiceTemplate[]): ToolStoreChoice[] {
	return templates.map((template, index) => ({
		choiceID: `assistant-preset-web-search:${index}:${template.bundleID}:${template.toolSlug}:${template.toolVersion}`,
		bundleID: template.bundleID,
		bundleSlug: template.bundleSlug,
		toolID: template.toolID,
		toolSlug: template.toolSlug,
		toolVersion: template.toolVersion,
		toolType: template.toolType ?? ToolStoreChoiceType.WebSearch,
		displayName: template.displayName,
		description: template.description,
		autoExecute: template.autoExecute,
		userArgSchemaInstance: template.userArgSchemaInstance,
	}));
}

function normalizeComparableStringList(values?: string[]): string[] | undefined {
	const out = [...new Set((values ?? []).map(value => value.trim()).filter(Boolean))].toSorted();

	return out.length > 0 ? out : undefined;
}

function normalizeComparableArgumentValues(values?: Record<string, string>): Record<string, string> | undefined {
	const entries = Object.entries(values ?? {})
		.map(([key, value]) => [key.trim(), value] as const)
		.filter(([key, value]) => key.length > 0 && typeof value === 'string' && value.trim().length > 0)
		.toSorted(([a], [b]) => a.localeCompare(b));

	return entries.length > 0 ? Object.fromEntries(entries) : undefined;
}

function compareStrings(a: string, b: string): number {
	return a.localeCompare(b);
}

/**
 * Normalizes MCP context for assistant preset comparison.
 *
 * The MCP picker can hydrate selections from discovery in a different order
 * from the saved preset. For sync/modified checks, order is not meaningful,
 * so compare stable identity fields plus arguments and policy-relevant tool
 * metadata.
 */
export function normalizeAssistantPresetMCPContext(
	context?: MCPConversationContext
): MCPConversationContext | undefined {
	if (!context) {
		return undefined;
	}

	type MCPServer = NonNullable<MCPConversationContext['servers']>[number];
	type MCPSelectedTool = NonNullable<MCPServer['selectedTools']>[number];
	type MCPResource = NonNullable<MCPConversationContext['resources']>[number];
	type MCPResourceTemplate = NonNullable<MCPConversationContext['resourceTemplates']>[number];
	type MCPPrompt = NonNullable<MCPConversationContext['prompts']>[number];

	const servers: MCPConversationContext['servers'] = (context.servers ?? [])
		.filter(server => server.bundleID?.trim() && server.serverID?.trim())
		.map(server => {
			const selectedTools: MCPSelectedTool[] = (server.selectedTools ?? [])
				.filter(tool => tool.toolName?.trim())
				.map(tool => {
					const visibility = normalizeComparableStringList(tool.visibility);

					const selectedTool: MCPSelectedTool = {
						bundleID: tool.bundleID || server.bundleID,
						serverID: tool.serverID || server.serverID,
						toolName: tool.toolName,
					};

					if (tool.providerToolName) {
						selectedTool.providerToolName = tool.providerToolName;
					}
					if (tool.choiceID) {
						selectedTool.choiceID = tool.choiceID;
					}
					if (tool.digest) {
						selectedTool.digest = tool.digest;
					}
					if (tool.approvalRule) {
						selectedTool.approvalRule = tool.approvalRule;
					}
					if (tool.executionMode) {
						selectedTool.executionMode = tool.executionMode;
					}
					if (tool.appResourceUri) {
						selectedTool.appResourceUri = tool.appResourceUri;
					}
					if (visibility) {
						selectedTool.visibility = visibility;
					}

					return selectedTool;
				})
				.toSorted((a, b) =>
					compareStrings(`${a.bundleID}/${a.serverID}/${a.toolName}`, `${b.bundleID}/${b.serverID}/${b.toolName}`)
				);

			const normalizedServer: MCPServer = {
				bundleID: server.bundleID,
				serverID: server.serverID,
				toolExposure: server.toolExposure,
			};

			if (server.snapshotDigest) {
				normalizedServer.snapshotDigest = server.snapshotDigest;
			}
			if (selectedTools.length > 0) {
				normalizedServer.selectedTools = selectedTools;
			}
			if (server.includeServerInstructions) {
				normalizedServer.includeServerInstructions = true;
			}

			return normalizedServer;
		})
		.toSorted((a, b) => compareStrings(`${a.bundleID}/${a.serverID}`, `${b.bundleID}/${b.serverID}`));

	const resources: NonNullable<MCPConversationContext['resources']> = (context.resources ?? [])
		.filter(resource => resource.bundleID?.trim() && resource.serverID?.trim() && resource.uri?.trim())
		.map(resource => {
			const normalizedResource: MCPResource = {
				bundleID: resource.bundleID,
				serverID: resource.serverID,
				uri: resource.uri,
				displayName: resource.uri,
			};

			if (resource.digest) {
				normalizedResource.digest = resource.digest;
			}

			return normalizedResource;
		})
		.toSorted((a, b) => compareStrings(`${a.bundleID}/${a.serverID}/${a.uri}`, `${b.bundleID}/${b.serverID}/${b.uri}`));

	const resourceTemplates: NonNullable<MCPConversationContext['resourceTemplates']> = (context.resourceTemplates ?? [])
		.filter(template => template.bundleID?.trim() && template.serverID?.trim() && template.uriTemplate?.trim())
		.map(template => {
			const argumentValues = normalizeComparableArgumentValues(template.argumentValues);

			const normalizedTemplate: MCPResourceTemplate = {
				bundleID: template.bundleID,
				serverID: template.serverID,
				uriTemplate: template.uriTemplate,
				displayName: template.uriTemplate,
			};

			if (argumentValues) {
				normalizedTemplate.argumentValues = argumentValues;
			}
			if (template.digest) {
				normalizedTemplate.digest = template.digest;
			}

			return normalizedTemplate;
		})
		.toSorted((a, b) =>
			compareStrings(`${a.bundleID}/${a.serverID}/${a.uriTemplate}`, `${b.bundleID}/${b.serverID}/${b.uriTemplate}`)
		);

	const prompts: NonNullable<MCPConversationContext['prompts']> = (context.prompts ?? [])
		.filter(prompt => prompt.bundleID?.trim() && prompt.serverID?.trim() && prompt.promptName?.trim())
		.map(prompt => {
			const argumentValues = normalizeComparableArgumentValues(prompt.argumentValues);

			const normalizedPrompt: MCPPrompt = {
				bundleID: prompt.bundleID,
				serverID: prompt.serverID,
				promptName: prompt.promptName,
				displayName: prompt.promptName,
			};

			if (argumentValues) {
				normalizedPrompt.argumentValues = argumentValues;
			}
			if (prompt.digest) {
				normalizedPrompt.digest = prompt.digest;
			}

			return normalizedPrompt;
		})
		.toSorted((a, b) =>
			compareStrings(`${a.bundleID}/${a.serverID}/${a.promptName}`, `${b.bundleID}/${b.serverID}/${b.promptName}`)
		);
	if (servers.length === 0 && resources.length === 0 && resourceTemplates.length === 0 && prompts.length === 0) {
		return undefined;
	}

	return {
		servers,
		...(resources.length > 0 ? { resources } : {}),
		...(resourceTemplates.length > 0 ? { resourceTemplates } : {}),
		...(prompts.length > 0 ? { prompts } : {}),
	};
}

export function getAssistantPresetModificationSummary(args: {
	preparedApplication: AssistantPresetPreparedApplication | null;
	currentSelectedModel: UIChatOption;
	currentIncludeModelSystemPrompt: boolean;
	currentSelectedInstructionSourceKeys: string[];
	currentRuntimeSnapshot: AssistantPresetRuntimeSnapshot;
}): AssistantPresetModificationSummary {
	const { preparedApplication } = args;
	if (!preparedApplication) {
		return EMPTY_ASSISTANT_PRESET_MODIFICATION_SUMMARY;
	}

	const currentModelState = buildAssistantPresetModelComparisonState(
		preparedApplication.preset,
		args.currentSelectedModel,
		args.currentIncludeModelSystemPrompt
	);

	const currentToolsState = {
		conversationToolChoices: normalizeAssistantPresetToolChoices(args.currentRuntimeSnapshot.conversationToolChoices),
		webSearchChoices: normalizeAssistantPresetToolChoices(args.currentRuntimeSnapshot.webSearchChoices),
	};

	const currentSkillsState = normalizeAssistantPresetSkillRefs(args.currentRuntimeSnapshot.enabledSkillRefs);
	const currentActiveSkillsState = normalizeAssistantPresetSkillRefs(args.currentRuntimeSnapshot.activeSkillRefs ?? []);

	const currentMCPState = normalizeAssistantPresetMCPContext(args.currentRuntimeSnapshot.mcpContext);

	const model = preparedApplication.comparisonState.model
		? !areComparableValuesEqual(preparedApplication.comparisonState.model, currentModelState)
		: false;

	const instructions = preparedApplication.comparisonState.instructions
		? !areComparableValuesEqual(preparedApplication.comparisonState.instructions, [
				...args.currentSelectedInstructionSourceKeys,
			])
		: false;

	const tools = preparedApplication.comparisonState.tools
		? !areComparableValuesEqual(preparedApplication.comparisonState.tools, currentToolsState)
		: false;

	const skills =
		preparedApplication.comparisonState.skills || preparedApplication.comparisonState.activeSkills
			? !areComparableValuesEqual(preparedApplication.comparisonState.skills ?? [], currentSkillsState) ||
				!areComparableValuesEqual(preparedApplication.comparisonState.activeSkills ?? [], currentActiveSkillsState)
			: false;

	const mcp = preparedApplication.comparisonState.mcp
		? !areComparableValuesEqual(preparedApplication.comparisonState.mcp, currentMCPState)
		: false;

	const modifiedLabels: string[] = [];
	if (model) {
		modifiedLabels.push('Model');
	}
	if (instructions) {
		modifiedLabels.push('Instructions');
	}
	if (tools) {
		modifiedLabels.push('Tools');
	}
	if (skills) {
		modifiedLabels.push('Skills');
	}
	if (mcp) {
		modifiedLabels.push('MCP');
	}

	return {
		model,
		instructions,
		tools,
		skills,
		mcp,
		any: modifiedLabels.length > 0,
		modifiedLabels,
	};
}

export function findBaseAssistantPresetOption(
	options: AssistantPresetOptionItem[],
	baseBundleID: string,
	baseSlug: string,
	baseVersion: string
): AssistantPresetOptionItem | null {
	return (
		options.find(
			option =>
				option.bundleID === baseBundleID && option.preset.slug === baseSlug && option.preset.version === baseVersion
		) ?? null
	);
}

export function findDefaultAssistantPresetOption(
	options: AssistantPresetOptionItem[]
): AssistantPresetOptionItem | null {
	const o = findBaseAssistantPresetOption(
		options,
		BASE_ASSISTANT_PRESET_BUNDLEID,
		BASE_ASSISTANT_PRESET_SLUG,
		BASE_ASSISTANT_PRESET_VERSION
	);
	if (o && o.isSelectable) {
		return o;
	}

	return options.find(option => option.isSelectable) ?? o;
}
