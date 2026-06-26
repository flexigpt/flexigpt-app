import type {
	MCPArgumentDefinition,
	MCPAuthHealth,
	MCPBundle,
	MCPConversationContext,
	MCPPromptRef,
	MCPPromptSelection,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPResourceTemplateSelection,
	MCPServerConfig,
	MCPServerRuntimeSnapshot,
	MCPServerSelection,
	MCPToolCapability,
	MCPToolSelection,
} from '@/spec/mcp';
import { MCPToolExposure } from '@/spec/mcp';

export interface MCPComposerServerOption {
	bundle: MCPBundle;
	server: MCPServerConfig;
	runtime?: MCPServerRuntimeSnapshot;
	authHealth?: MCPAuthHealth;

	tools: MCPToolCapability[];
	resources: MCPResourceRef[];
	resourceTemplates: MCPResourceTemplateRef[];
	prompts: MCPPromptRef[];

	discoveryLoaded: boolean;
	discoveryLoading: boolean;
	discoveryError?: string;
}

export interface MCPComposerServerSelection {
	bundleID: string;
	serverID: string;
	snapshotDigest?: string;
	toolExposure: MCPToolExposure;
	selectedTools: MCPToolSelection[];
	selectedResources: MCPResourceRef[];
	selectedResourceTemplates: MCPResourceTemplateSelection[];
	selectedPrompts: MCPPromptSelection[];
	includeServerInstructions?: boolean;
}

export interface UseComposerMCPResult {
	options: MCPComposerServerOption[];
	loading: boolean;
	error?: string;
	selectedByServerKey: Record<string, MCPComposerServerSelection>;
	mcpContext?: MCPConversationContext;
	selectedServerCount: number;
	selectedToolCount: number;
	selectedResourceCount: number;
	selectedPromptCount: number;
	requiredArgumentMissingCount: number;
	argumentsBlocked: boolean;

	refreshAll: () => Promise<void>;
	refreshServer: (bundleID: string, serverID: string) => Promise<void>;
	ensureDiscoveryLoaded: (bundleID: string, serverID: string) => Promise<void>;
	prepareForSubmit: () => Promise<MCPConversationContext | undefined>;

	connectServer: (bundleID: string, serverID: string) => Promise<void>;
	disconnectServer: (bundleID: string, serverID: string) => Promise<void>;
	cancelOAuth: (bundleID: string, serverID: string) => Promise<void>;
	openAuthURL: (url: string) => void;

	setServerSelected: (option: MCPComposerServerOption, selected: boolean) => void;
	setToolExposure: (bundleID: string, serverID: string, exposure: MCPToolExposure) => void;
	setIncludeServerInstructions: (bundleID: string, serverID: string, include: boolean) => void;
	toggleTool: (tool: MCPToolCapability, selected: boolean) => void;
	toggleResource: (resource: MCPResourceRef, selected: boolean) => void;
	toggleResourceTemplate: (template: MCPResourceTemplateRef, selected: boolean) => void;
	togglePrompt: (prompt: MCPPromptRef, selected: boolean) => void;
	setResourceTemplateArgumentValue: (
		bundleID: string,
		serverID: string,
		uriTemplate: string,
		argumentName: string,
		value: string
	) => void;
	setPromptArgumentValue: (
		bundleID: string,
		serverID: string,
		promptName: string,
		argumentName: string,
		value: string
	) => void;
	clear: () => void;
	restoreContext: (context?: MCPConversationContext) => void;
}

export function mcpServerKey(bundleID: string, serverID: string): string {
	return `${bundleID}::${serverID}`;
}

export function mcpToolKey(
	tool: Pick<MCPToolCapability | MCPToolSelection, 'serverID' | 'toolName'> & { bundleID?: string }
): string {
	return `${tool.bundleID ?? ''}::${tool.serverID}::${tool.toolName}`;
}

export function mcpResourceKey(resource: Pick<MCPResourceRef, 'bundleID' | 'serverID' | 'uri'>): string {
	return `${resource.bundleID}::${resource.serverID}::${resource.uri}`;
}

export function mcpResourceTemplateKey(
	template: Pick<MCPResourceTemplateRef, 'bundleID' | 'serverID' | 'uriTemplate'>
): string {
	return `${template.bundleID}::${template.serverID}::${template.uriTemplate}`;
}

export function mcpPromptKey(prompt: Pick<MCPPromptRef, 'bundleID' | 'serverID' | 'promptName'>): string {
	return `${prompt.bundleID}::${prompt.serverID}::${prompt.promptName}`;
}

export function normalizeMCPArgumentDefinitions(
	args?: Record<string, MCPArgumentDefinition | string>
): MCPArgumentDefinition[] {
	if (!args) {
		return [];
	}

	return Object.entries(args)
		.map(([key, value]) => {
			if (typeof value === 'string') {
				return {
					name: key,
					description: value,
					required: false,
				} satisfies MCPArgumentDefinition;
			}

			return Object.assign({}, value, {
				name: value.name || key,
				required: Boolean(value.required),
			}) satisfies MCPArgumentDefinition;
		})
		.filter(arg => arg.name.trim().length > 0)
		.toSorted((a, b) => a.name.localeCompare(b.name));
}

export function countMissingRequiredMCPArguments(
	items: Array<{
		arguments?: Record<string, MCPArgumentDefinition | string>;
		argumentValues?: Record<string, string>;
	}>
): number {
	let count = 0;

	for (const item of items) {
		for (const arg of normalizeMCPArgumentDefinitions(item.arguments)) {
			if (!arg.required) {
				continue;
			}
			if (!item.argumentValues?.[arg.name]?.trim()) {
				count++;
			}
		}
	}

	return count;
}

export function mcpSelectionToContext(
	selectedByServerKey: Record<string, MCPComposerServerSelection>
): MCPConversationContext | undefined {
	const selections = Object.values(selectedByServerKey);
	if (selections.length === 0) {
		return undefined;
	}

	const servers: MCPServerSelection[] = selections.map(selection => ({
		bundleID: selection.bundleID,
		serverID: selection.serverID,
		snapshotDigest: selection.snapshotDigest,
		toolExposure: selection.toolExposure,
		selectedTools:
			selection.toolExposure !== MCPToolExposure.MCPToolExposureNone && selection.selectedTools.length > 0
				? selection.selectedTools.map(tool =>
						Object.assign({}, tool, {
							bundleID: tool.bundleID ?? selection.bundleID,
							serverID: tool.serverID || selection.serverID,
						})
					)
				: undefined,
		includeServerInstructions: selection.includeServerInstructions,
	}));

	const resources = selections.flatMap(selection => selection.selectedResources);
	const resourceTemplates = selections.flatMap(selection => selection.selectedResourceTemplates);
	const prompts = selections.flatMap(selection => selection.selectedPrompts);

	return {
		servers,
		resources: resources.length > 0 ? resources : undefined,
		resourceTemplates: resourceTemplates.length > 0 ? resourceTemplates : undefined,
		prompts: prompts.length > 0 ? prompts : undefined,
	};
}

export function mcpContextToSelectionMap(context?: MCPConversationContext): Record<string, MCPComposerServerSelection> {
	if (!context) {
		return {};
	}

	const out: Record<string, MCPComposerServerSelection> = {};

	for (const server of context.servers ?? []) {
		const key = mcpServerKey(server.bundleID, server.serverID);
		out[key] = {
			bundleID: server.bundleID,
			serverID: server.serverID,
			snapshotDigest: server.snapshotDigest,
			toolExposure: server.toolExposure,
			selectedTools: (server.selectedTools ?? []).map(tool =>
				Object.assign({}, tool, {
					bundleID: tool.bundleID ?? server.bundleID,
					serverID: tool.serverID || server.serverID,
				})
			),
			selectedResources: [],
			selectedResourceTemplates: [],
			selectedPrompts: [],
			includeServerInstructions: server.includeServerInstructions,
		};
	}

	for (const resource of context.resources ?? []) {
		const key = mcpServerKey(resource.bundleID, resource.serverID);
		if (!out[key]) {
			out[key] = {
				bundleID: resource.bundleID,
				serverID: resource.serverID,
				toolExposure: MCPToolExposure.MCPToolExposureNone,
				selectedTools: [],
				selectedResources: [],
				selectedResourceTemplates: [],
				selectedPrompts: [],
			};
		}
		out[key].selectedResources.push(resource);
	}

	for (const template of context.resourceTemplates ?? []) {
		const key = mcpServerKey(template.bundleID, template.serverID);
		if (!out[key]) {
			out[key] = {
				bundleID: template.bundleID,
				serverID: template.serverID,
				toolExposure: MCPToolExposure.MCPToolExposureNone,
				selectedTools: [],
				selectedResources: [],
				selectedResourceTemplates: [],
				selectedPrompts: [],
			};
		}
		out[key].selectedResourceTemplates.push(template);
	}

	for (const prompt of context.prompts ?? []) {
		const key = mcpServerKey(prompt.bundleID, prompt.serverID);
		if (!out[key]) {
			out[key] = {
				bundleID: prompt.bundleID,
				serverID: prompt.serverID,
				toolExposure: MCPToolExposure.MCPToolExposureNone,
				selectedTools: [],
				selectedResources: [],
				selectedResourceTemplates: [],
				selectedPrompts: [],
			};
		}
		out[key].selectedPrompts.push(prompt);
	}

	return out;
}
