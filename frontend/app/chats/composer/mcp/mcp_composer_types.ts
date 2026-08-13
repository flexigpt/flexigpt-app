import type { ArtifactRef } from '@/spec/artifact';
import type {
	MCPArgumentDefinition,
	MCPAuthHealth,
	MCPConversationContext,
	MCPPromptRef,
	MCPPromptSelection,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPResourceTemplateSelection,
	MCPServerRuntimeSnapshot,
	MCPServerSelection,
	MCPToolCapability,
	MCPToolSelection,
	MCPTransportType,
} from '@/spec/mcp_artifact';
import { MCPToolExposure } from '@/spec/mcp_artifact';

import type { MCPBundleView, MCPServerView } from '@/mcpservers/lib/mcp_management';

export interface MCPComposerServerOption {
	bundle: MCPBundleView;
	server: MCPServerView;
	transport: MCPTransportType;
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
	server: ArtifactRef;
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
	refreshServer: (server: ArtifactRef) => Promise<void>;
	ensureDiscoveryLoaded: (server: ArtifactRef) => Promise<void>;
	prepareForSubmit: () => Promise<MCPConversationContext | undefined>;

	connectServer: (server: ArtifactRef) => Promise<void>;
	disconnectServer: (server: ArtifactRef) => Promise<void>;
	cancelOAuth: (server: ArtifactRef) => Promise<void>;
	openAuthURL: (url: string) => void;

	setServerSelected: (option: MCPComposerServerOption, selected: boolean) => void;
	setToolExposure: (server: ArtifactRef, exposure: MCPToolExposure) => void;
	setIncludeServerInstructions: (server: ArtifactRef, include: boolean) => void;
	toggleTool: (tool: MCPToolCapability, selected: boolean) => void;
	toggleResource: (resource: MCPResourceRef, selected: boolean) => void;
	toggleResourceTemplate: (template: MCPResourceTemplateRef, selected: boolean) => void;
	togglePrompt: (prompt: MCPPromptRef, selected: boolean) => void;
	setResourceTemplateArgumentValue: (
		server: ArtifactRef,
		uriTemplate: string,
		argumentName: string,
		value: string
	) => void;
	setPromptArgumentValue: (server: ArtifactRef, promptName: string, argumentName: string, value: string) => void;
	clear: () => void;
	restoreContext: (context?: MCPConversationContext) => void;
}

export function mcpServerKey(server: ArtifactRef): string {
	return `${server.rootID}::${server.artifactID}`;
}

export function mcpToolKey(tool: Pick<MCPToolCapability | MCPToolSelection, 'server' | 'toolName'>): string {
	return `${mcpServerKey(tool.server)}::${tool.toolName}`;
}

export function mcpResourceKey(resource: Pick<MCPResourceRef, 'server' | 'uri'>): string {
	return `${mcpServerKey(resource.server)}::${resource.uri}`;
}

export function mcpResourceTemplateKey(template: Pick<MCPResourceTemplateRef, 'server' | 'uriTemplate'>): string {
	return `${mcpServerKey(template.server)}::${template.uriTemplate}`;
}

export function mcpPromptKey(prompt: Pick<MCPPromptRef, 'server' | 'promptName'>): string {
	return `${mcpServerKey(prompt.server)}::${prompt.promptName}`;
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
		server: selection.server,
		snapshotDigest: selection.snapshotDigest,
		toolExposure: selection.toolExposure,
		selectedTools:
			selection.toolExposure !== MCPToolExposure.None && selection.selectedTools.length > 0
				? selection.selectedTools.map(tool =>
						Object.assign({}, tool, {
							server: tool.server ?? selection.server,
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
		const key = mcpServerKey(server.server);
		out[key] = {
			server: server.server,
			snapshotDigest: server.snapshotDigest,
			toolExposure: server.toolExposure,
			selectedTools: (server.selectedTools ?? []).map(tool =>
				Object.assign({}, tool, {
					server: tool.server ?? server.server,
				})
			),
			selectedResources: [],
			selectedResourceTemplates: [],
			selectedPrompts: [],
			includeServerInstructions: server.includeServerInstructions,
		};
	}

	for (const resource of context.resources ?? []) {
		const key = mcpServerKey(resource.server);

		if (!out[key]) {
			out[key] = {
				server: resource.server,
				toolExposure: MCPToolExposure.None,
				selectedTools: [],
				selectedResources: [],
				selectedResourceTemplates: [],
				selectedPrompts: [],
			};
		}
		out[key].selectedResources.push(resource);
	}

	for (const template of context.resourceTemplates ?? []) {
		const key = mcpServerKey(template.server);
		if (!out[key]) {
			out[key] = {
				server: template.server,
				toolExposure: MCPToolExposure.None,
				selectedTools: [],
				selectedResources: [],
				selectedResourceTemplates: [],
				selectedPrompts: [],
			};
		}
		out[key].selectedResourceTemplates.push(template);
	}

	for (const prompt of context.prompts ?? []) {
		const key = mcpServerKey(prompt.server);
		if (!out[key]) {
			out[key] = {
				server: prompt.server,
				toolExposure: MCPToolExposure.None,
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
