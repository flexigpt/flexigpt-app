import type {
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeServerID,
	MCPServerRuntimeSnapshot,
	MCPToolCapability,
} from '@/spec/mcp';
import type {
	MCPCompleteArgumentRequestBody,
	MCPCompletionResult,
	MCPContent,
	MCPDiscoveryPage,
	MCPGetPromptResponseBody,
	MCPReadResourceResponseBody,
} from '@/spec/session_mcp';
import {
	MCPApprovalRule,
	MCPAppVisibility,
	MCPExecutionMode,
	MCPServerStatus,
	MCPTaskSupport,
	MCPToolRisk,
} from '@/spec/mcp';
import { MCPContentType } from '@/spec/session_mcp';

import type { IMCPRuntimeAPI } from '@/apis/interface';
import {
	enumFromWails,
	goBytesFromWails,
	optionalWailsString,
	requiredObject,
	requireWailsBoolean,
	wailsArrayOrEmpty,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	CancelMCPServerAuthorization,
	CompleteMCPArgument,
	ConnectMCPServer,
	DisconnectMCPServer,
	GetMCPPrompt,
	ListMCPServerPrompts,
	ListMCPServerResources,
	ListMCPServerResourceTemplates,
	ListMCPServerTools,
	ReadMCPResource,
	RefreshMCPServer,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/mcpruntimewrapper';

function runtimeSnapshotFromWails(value: unknown, operation: string): MCPServerRuntimeSnapshot {
	const snapshot = requiredObject<MCPServerRuntimeSnapshot>(value, operation);

	return {
		...snapshot,
		status: enumFromWails(snapshot.status, MCPServerStatus, `${operation}.status`),
	};
}

function mcpContentFromWails(value: unknown, operation: string): MCPContent {
	const rawContent = requiredObject<Record<string, unknown>>(value, operation);
	const content = requiredObject<MCPContent>(value, operation);
	let resource: MCPContent['resource'];

	if (rawContent.resource !== null && rawContent.resource !== undefined) {
		const rawResource = requiredObject<Record<string, unknown>>(rawContent.resource, `${operation}.resource`);

		resource = {
			...rawResource,
			blob: goBytesFromWails(rawResource.blob, `${operation}.resource.blob`),
		} as MCPContent['resource'];
	}

	return {
		...content,
		type: enumFromWails(rawContent.type, MCPContentType, `${operation}.type`),
		data: goBytesFromWails(rawContent.data, `${operation}.data`),
		resource,
	};
}

function mcpPromptResponseFromWails(value: unknown, operation: string): MCPGetPromptResponseBody {
	const rawResponse = requiredObject<Record<string, unknown>>(value, operation);
	const response = requiredObject<MCPGetPromptResponseBody>(value, operation);

	return {
		...response,
		messages:
			rawResponse.messages === null || rawResponse.messages === undefined
				? undefined
				: (wailsObjectArrayOrEmpty(rawResponse.messages, `${operation}.messages`).map((rawMessage, index) => {
						const message = requiredObject<Record<string, unknown>>(rawMessage, `${operation}.messages[${index}]`);

						return {
							...message,
							content: mcpContentFromWails(message.content, `${operation}.messages[${index}].content`),
						};
					}) as MCPGetPromptResponseBody['messages']),
	};
}

function mcpReadResourceResponseFromWails(value: unknown, operation: string): MCPReadResourceResponseBody {
	const rawResponse = requiredObject<Record<string, unknown>>(value, operation);
	const response = requiredObject<MCPReadResourceResponseBody>(value, operation);

	return {
		...response,
		contents:
			rawResponse.contents === null || rawResponse.contents === undefined
				? undefined
				: (wailsObjectArrayOrEmpty(rawResponse.contents, `${operation}.contents`).map((content, index) =>
						mcpContentFromWails(content, `${operation}.contents[${index}]`)
					) as MCPReadResourceResponseBody['contents']),
	};
}

function mcpToolCapabilityFromWails(value: unknown, operation: string): MCPToolCapability {
	const tool = requiredObject<MCPToolCapability>(value, operation);

	return {
		...tool,
		inferredRisk: enumFromWails(tool.inferredRisk, MCPToolRisk, `${operation}.inferredRisk`),
		approvalRule: enumFromWails(tool.approvalRule, MCPApprovalRule, `${operation}.approvalRule`),
		executionMode: enumFromWails(tool.executionMode, MCPExecutionMode, `${operation}.executionMode`),
		taskSupport: enumFromWails(tool.taskSupport, MCPTaskSupport, `${operation}.taskSupport`),
		app:
			tool.app === null || tool.app === undefined
				? undefined
				: {
						...tool.app,
						visibility: wailsArrayOrEmpty(tool.app.visibility, `${operation}.app.visibility`).map((item, index) =>
							enumFromWails(item, MCPAppVisibility, `${operation}.app.visibility[${index}]`)
						),
					},
	};
}

function discoveryPageFromWails<T extends object>(value: unknown, operation: string): MCPDiscoveryPage<T> {
	const page = requiredObject<MCPDiscoveryPage<T>>(value, operation);

	return {
		...page,
		items: wailsObjectArrayOrEmpty<T>(page.items, `${operation}.items`),
		nextPageToken: optionalWailsString(page.nextPageToken, `${operation}.nextPageToken`) || undefined,
	};
}

export class WailsMCPRuntimeAPI implements IMCPRuntimeAPI {
	async cancelMCPServerAuthorization(server: MCPRuntimeServerID): Promise<boolean> {
		return requireWailsBoolean(
			await CancelMCPServerAuthorization(server as Parameters<typeof CancelMCPServerAuthorization>[0]),
			'CancelMCPServerAuthorization'
		);
	}

	async completeMCPArgument(
		server: MCPRuntimeServerID,
		request: MCPCompleteArgumentRequestBody
	): Promise<MCPCompletionResult> {
		return requiredObject<MCPCompletionResult>(
			await CompleteMCPArgument(
				server as Parameters<typeof CompleteMCPArgument>[0],
				request as Parameters<typeof CompleteMCPArgument>[1]
			),
			'CompleteMCPArgument'
		);
	}

	async connectMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return runtimeSnapshotFromWails(
			await ConnectMCPServer(server as Parameters<typeof ConnectMCPServer>[0]),
			'ConnectMCPServer'
		);
	}

	async disconnectMCPServer(server: MCPRuntimeServerID): Promise<void> {
		await DisconnectMCPServer(server as Parameters<typeof DisconnectMCPServer>[0]);
	}

	async getMCPPrompt(
		server: MCPRuntimeServerID,
		promptName: string,
		promptArguments: Record<string, string>
	): Promise<MCPGetPromptResponseBody> {
		return mcpPromptResponseFromWails(
			await GetMCPPrompt(server as Parameters<typeof GetMCPPrompt>[0], promptName, promptArguments),
			'GetMCPPrompt'
		);
	}

	async listMCPServerPrompts(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPPromptRef>> {
		return discoveryPageFromWails<MCPPromptRef>(
			await ListMCPServerPrompts(server as Parameters<typeof ListMCPServerPrompts>[0], pageSize, pageToken),
			'ListMCPServerPrompts'
		);
	}

	async listMCPServerResources(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPResourceRef>> {
		return discoveryPageFromWails<MCPResourceRef>(
			await ListMCPServerResources(server as Parameters<typeof ListMCPServerResources>[0], pageSize, pageToken),
			'ListMCPServerResources'
		);
	}

	async listMCPServerResourceTemplates(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPResourceTemplateRef>> {
		return discoveryPageFromWails<MCPResourceTemplateRef>(
			await ListMCPServerResourceTemplates(
				server as Parameters<typeof ListMCPServerResourceTemplates>[0],
				pageSize,
				pageToken
			),
			'ListMCPServerResourceTemplates'
		);
	}

	async listMCPServerTools(
		server: MCPRuntimeServerID,
		pageSize: number,
		pageToken = ''
	): Promise<MCPDiscoveryPage<MCPToolCapability>> {
		const page = discoveryPageFromWails<MCPToolCapability>(
			await ListMCPServerTools(server as Parameters<typeof ListMCPServerTools>[0], pageSize, pageToken),
			'ListMCPServerTools'
		);

		return {
			...page,
			items: page.items.map((tool, index) => mcpToolCapabilityFromWails(tool, `ListMCPServerTools.items[${index}]`)),
		};
	}

	async readMCPResource(server: MCPRuntimeServerID, uri: string): Promise<MCPReadResourceResponseBody> {
		return mcpReadResourceResponseFromWails(
			await ReadMCPResource(server as Parameters<typeof ReadMCPResource>[0], uri),
			'ReadMCPResource'
		);
	}

	async refreshMCPServer(server: MCPRuntimeServerID): Promise<MCPServerRuntimeSnapshot> {
		return runtimeSnapshotFromWails(
			await RefreshMCPServer(server as Parameters<typeof RefreshMCPServer>[0]),
			'RefreshMCPServer'
		);
	}
}
