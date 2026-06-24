import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import {
	type InvokeMCPToolRequestBody,
	MCPApprovalDecision,
	MCPApprovalResolution,
	type MCPContent,
	MCPContentType,
	MCPInvocationSource,
	type MCPToolAppRenderInfo,
	type MCPToolSelection,
} from '@/spec/mcp';
import { ToolOutputKind } from '@/spec/tool';

import { isJSONObject } from '@/lib/jsonschema_utils';

import { mcpAPI, skillStoreAPI, toolRuntimeAPI } from '@/apis/baseapi';

import type { RequestMCPApproval } from '@/chats/composer/mcp/use_mcp_approval';
import { isSkillsToolName } from '@/skills/lib/skill_identity_utils';
import { isRunnableComposerToolCall } from '@/tools/lib/tool_call_utils';
import { formatToolOutputSummary } from '@/tools/lib/tool_output_utils';

const TOOL_CALL_TIMEOUT_MS = 600_000;

function withTimeout<T>(promise: Promise<T>, ms: number, timeoutMessage: string): Promise<T> {
	return new Promise<T>((resolve, reject) => {
		const timer = window.setTimeout(() => {
			reject(new Error(timeoutMessage));
		}, ms);

		promise.then(
			value => {
				window.clearTimeout(timer);
				resolve(value);
			},
			(error: unknown) => {
				window.clearTimeout(timer);
				reject(error);
			}
		);
	});
}

function parseToolArguments(raw?: string): Record<string, any> | undefined {
	if (!raw || raw.trim().length === 0) return undefined;

	const parsed = JSON.parse(raw) as unknown;
	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error('Tool arguments must be a JSON object.');
	}

	return parsed as Record<string, any>;
}

function mcpContentToText(content: MCPContent): string {
	switch (content.type) {
		case MCPContentType.MCPContentTypeText:
			return content.text ?? '';

		case MCPContentType.MCPContentTypeResource:
			if (content.resource?.text) return content.resource.text;
			return JSON.stringify(content.resource ?? content, null, 2);

		case MCPContentType.MCPContentTypeResourceLink:
			return [content.title || content.name || content.uri, content.description, content.uri]
				.filter(Boolean)
				.join('\n');

		case MCPContentType.MCPContentTypeImage:
			return `[MCP image content${content.mimeType ? `: ${content.mimeType}` : ''}]`;

		case MCPContentType.MCPContentTypeAudio:
			return `[MCP audio content${content.mimeType ? `: ${content.mimeType}` : ''}]`;

		default:
			return JSON.stringify(content, null, 2);
	}
}

function mcpToolLabel(selection: MCPToolSelection, fallbackName: string): string {
	return selection.toolName || selection.providerToolName || fallbackName;
}

function buildMCPToolOutput(args: {
	toolCall: UIToolCall;
	selection: MCPToolSelection;
	text: string;
	isError?: boolean;
	errorMessage?: string;
	mcpApp?: MCPToolAppRenderInfo;
}): UIToolOutput {
	const name = mcpToolLabel(args.selection, args.toolCall.name);
	const firstLine = args.text
		.split('\n')
		.find(line => line.trim().length > 0)
		?.trim();

	return {
		id: args.toolCall.id,
		callID: args.toolCall.callID,
		name,
		choiceID: args.toolCall.choiceID,
		type: args.toolCall.type,
		summary: args.isError
			? `MCP error: ${firstLine?.slice(0, 80) || name}`
			: `MCP result: ${firstLine?.slice(0, 80) || name}`,
		toolOutputs: [
			{
				kind: ToolOutputKind.Text,
				textItem: {
					text: args.text,
				},
			},
		],
		isError: !!args.isError,
		errorMessage: args.errorMessage,
		arguments: args.toolCall.arguments,
		webSearchToolCallItems: args.toolCall.webSearchToolCallItems,
		toolStoreChoice: args.toolCall.toolStoreChoice,
		mcpToolSelection: args.selection,
		mcpApp: args.mcpApp,
	};
}

async function executeMCPToolCall(
	toolCall: UIToolCall,
	selection: MCPToolSelection,
	requestMCPApproval?: RequestMCPApproval
): Promise<ExecuteComposerToolCallResult> {
	const bundleID = selection.bundleID;
	if (!bundleID || !selection.serverID || !selection.toolName) {
		return {
			ok: false,
			errorMessage: 'Cannot resolve MCP tool identity for this call.',
		};
	}

	let parsedArgs: Record<string, any> | undefined;
	try {
		parsedArgs = parseToolArguments(toolCall.arguments);
	} catch (err) {
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: (err as Error)?.message || 'Invalid MCP tool arguments.',
				isError: true,
				errorMessage: (err as Error)?.message || 'Invalid MCP tool arguments.',
			}),
		};
	}

	const req: InvokeMCPToolRequestBody = {
		source: MCPInvocationSource.MCPInvocationSourceModel,
		serverID: selection.serverID,
		toolName: selection.toolName,
		providerToolName: selection.providerToolName || toolCall.name,
		toolDigest: selection.digest,
		arguments: parsedArgs,
		toolUseID: toolCall.callID || toolCall.id,
	};

	const evaluation = await mcpAPI.evaluateMCPToolCall(bundleID, req);

	if (!evaluation) {
		const message = 'MCP approval evaluation did not return a decision.';
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: message,
				isError: true,
				errorMessage: message,
			}),
		};
	}

	if (evaluation.decision === MCPApprovalDecision.MCPApprovalDecisionDenied) {
		const message = evaluation.reason || 'MCP policy denied this tool call.';
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: message,
				isError: true,
				errorMessage: message,
			}),
		};
	}

	if (evaluation.decision === MCPApprovalDecision.MCPApprovalDecisionApprovalRequired) {
		if (!evaluation.approvalID) {
			const message = 'MCP approval was required but no approval ID was returned.';
			return {
				ok: true,
				output: buildMCPToolOutput({
					toolCall,
					selection,
					text: message,
					isError: true,
					errorMessage: message,
				}),
			};
		}

		let resolution: MCPApprovalResolution;

		try {
			resolution =
				requestMCPApproval && evaluation.summary
					? await requestMCPApproval({
							approvalID: evaluation.approvalID,
							summary: evaluation.summary,
							reason: evaluation.reason,
						})
					: MCPApprovalResolution.MCPApprovalResolutionDenyOnce;
		} catch {
			resolution = MCPApprovalResolution.MCPApprovalResolutionDenyOnce;
		}

		const token = await mcpAPI.resolveMCPApproval(evaluation.approvalID, resolution);

		if (
			resolution !== MCPApprovalResolution.MCPApprovalResolutionAllowOnce &&
			resolution !== MCPApprovalResolution.MCPApprovalResolutionAllowAlways
		) {
			const message = evaluation.reason
				? `MCP tool call denied by user. ${evaluation.reason}`
				: 'MCP tool call denied by user.';
			return {
				ok: true,
				output: buildMCPToolOutput({
					toolCall,
					selection,
					text: message,
					isError: true,
					errorMessage: message,
				}),
			};
		}

		if (!token?.token) {
			const message = 'MCP approval did not return a usable token.';
			return {
				ok: true,
				output: buildMCPToolOutput({
					toolCall,
					selection,
					text: message,
					isError: true,
					errorMessage: message,
				}),
			};
		}

		req.approvalID = token.approvalID;
		req.approvalToken = token.token;
	} else if (evaluation.decision !== MCPApprovalDecision.MCPApprovalDecisionAllowed) {
		const message = `Unsupported MCP approval decision: ${String(evaluation.decision)}`;
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: message,
				isError: true,
				errorMessage: message,
			}),
		};
	}

	try {
		const resp = await withTimeout(
			mcpAPI.invokeMCPTool(bundleID, req),
			TOOL_CALL_TIMEOUT_MS,
			`MCP tool call "${selection.toolName}" timed out after ${Math.round(TOOL_CALL_TIMEOUT_MS / 1000)} seconds.`
		);

		const toolContent = Array.isArray(resp?.content) ? resp.content : undefined;
		const structuredContent = isJSONObject(resp?.structuredContent) ? resp.structuredContent : undefined;

		const contentText = (toolContent ?? []).map(mcpContentToText).filter(Boolean).join('\n\n');
		const structuredText = structuredContent !== undefined ? JSON.stringify(structuredContent, null, 2) : '';

		const text = [contentText, structuredText].filter(Boolean).join('\n\n') || 'MCP tool returned no content.';
		const isError = !!resp?.isError;
		const appRenderInfo =
			resp?.app && resp.app.resourceUri
				? {
						resourceUri: resp.app.resourceUri,
						mimeType: resp.app.mimeType,
						content: toolContent,
						...(structuredContent !== undefined ? { structuredContent } : {}),
						isError,
					}
				: undefined;

		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text,
				isError,
				errorMessage: isError ? text.split('\n')[0] : undefined,
				mcpApp: appRenderInfo,
			}),
		};
	} catch (err) {
		const message = (err as Error)?.message || 'MCP tool invocation failed.';
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: message,
				isError: true,
				errorMessage: message,
			}),
		};
	}
}

interface ExecuteComposerToolCallArgs {
	toolCall: UIToolCall;
	ensureSkillSession: () => Promise<string | null>;
	getCurrentSkillSessionID: () => string | null;
	requestMCPApproval?: RequestMCPApproval;
}

type ExecuteComposerToolCallResult =
	| {
			ok: true;
			output: UIToolOutput;
			refreshActiveSkillRefsForSessionID?: string;
	  }
	| {
			ok: false;
			errorMessage: string;
	  };

export async function executeComposerToolCall({
	toolCall,
	ensureSkillSession,
	getCurrentSkillSessionID,
	requestMCPApproval,
}: ExecuteComposerToolCallArgs): Promise<ExecuteComposerToolCallResult> {
	if (!isRunnableComposerToolCall(toolCall)) {
		return {
			ok: false,
			errorMessage: 'This tool call type cannot be executed from the composer.',
		};
	}

	const args = toolCall.arguments && toolCall.arguments.trim().length > 0 ? toolCall.arguments : undefined;

	if (isSkillsToolName(toolCall.name)) {
		let sid = getCurrentSkillSessionID();

		if (!sid) {
			try {
				sid = await ensureSkillSession();
			} catch {
				sid = null;
			}
		}

		if (!sid) {
			return {
				ok: false,
				errorMessage: 'No active skills session. Enable skills and resend, or run again after a session is created.',
			};
		}

		try {
			const resp = await withTimeout(
				skillStoreAPI.invokeSkillTool(sid, toolCall.name, args),
				TOOL_CALL_TIMEOUT_MS,
				`Tool call "${toolCall.name}" timed out after ${Math.round(TOOL_CALL_TIMEOUT_MS / 1000)} seconds.`
			);

			const isError = !!resp.isError;
			const errorMessage =
				resp.errorMessage || (isError ? 'Skill tool reported an error. Inspect the output for details.' : undefined);

			return {
				ok: true,
				refreshActiveSkillRefsForSessionID: sid,
				output: {
					id: toolCall.id,
					callID: toolCall.callID,
					name: toolCall.name,
					choiceID: toolCall.choiceID,
					type: toolCall.type,
					summary: isError
						? `Tool error: ${formatToolOutputSummary(toolCall.name)}`
						: formatToolOutputSummary(toolCall.name),
					toolOutputs: resp.outputs,
					isError,
					errorMessage,
					arguments: toolCall.arguments,
					webSearchToolCallItems: toolCall.webSearchToolCallItems,
					toolStoreChoice: toolCall.toolStoreChoice,
				},
			};
		} catch (err) {
			return {
				ok: false,
				errorMessage: (err as Error)?.message || 'Skill tool invocation failed.',
			};
		}
	}

	if (toolCall.mcpToolSelection) {
		return executeMCPToolCall(toolCall, toolCall.mcpToolSelection, requestMCPApproval);
	}

	const bundleID = toolCall.toolStoreChoice?.bundleID;
	const toolSlug = toolCall.toolStoreChoice?.toolSlug;
	const toolVersion = toolCall.toolStoreChoice?.toolVersion;

	if (!bundleID || !toolSlug || !toolVersion) {
		return {
			ok: false,
			errorMessage: 'Cannot resolve tool identity for this call.',
		};
	}

	try {
		const resp = await withTimeout(
			toolRuntimeAPI.invokeTool(bundleID, toolSlug, toolVersion, args),
			TOOL_CALL_TIMEOUT_MS,
			`Tool call "${toolCall.name}" timed out after ${Math.round(TOOL_CALL_TIMEOUT_MS / 1000)} seconds.`
		);

		const isError = !!resp.isError;
		const errorMessage =
			resp.errorMessage || (isError ? 'Tool reported an error. Inspect the output for details.' : undefined);

		return {
			ok: true,
			output: {
				id: toolCall.id,
				callID: toolCall.callID,
				name: toolCall.name,
				choiceID: toolCall.choiceID,
				type: toolCall.type,
				summary: isError
					? `Tool error: ${formatToolOutputSummary(toolCall.name)}`
					: formatToolOutputSummary(toolCall.name),
				toolOutputs: resp.outputs,
				isError,
				errorMessage,
				arguments: toolCall.arguments,
				webSearchToolCallItems: toolCall.webSearchToolCallItems,
				toolStoreChoice: toolCall.toolStoreChoice,
			},
		};
	} catch (err) {
		return {
			ok: false,
			errorMessage: (err as Error)?.message || 'Tool invocation failed.',
		};
	}
}
