import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type {
	InvokeMCPToolRequestBody,
	MCPApprovalEvaluation,
	MCPApprovalResolutionResult,
	MCPContent,
	MCPToolAppRenderInfo,
	MCPToolSelection,
} from '@/spec/mcp_artifact';
import { MCPApprovalDecision, MCPApprovalResolution, MCPContentType, MCPInvocationSource } from '@/spec/mcp_artifact';
import { ToolOutputKind } from '@/spec/tool';

import { isJSONObject } from '@/lib/jsonschema_utils';

import { mcpAPI, skillManagementAPI, toolRuntimeAPI } from '@/apis/baseapi';

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

		promise
			.then(value => {
				window.clearTimeout(timer);
				resolve(value);
			})
			.catch((error: unknown) => {
				window.clearTimeout(timer);
				reject(error);
			});
	});
}

function parseToolArguments(raw?: string): Record<string, any> | undefined {
	if (!raw || raw.trim().length === 0) {
		return undefined;
	}

	const parsed = JSON.parse(raw) as unknown;
	if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
		throw new Error('Tool arguments must be a JSON object.');
	}

	return parsed as Record<string, any>;
}

function mcpContentToText(content: MCPContent): string {
	switch (content.type) {
		case MCPContentType.Text:
			return content.text ?? '';

		case MCPContentType.Resource:
			if (content.resource?.text) {
				return content.resource.text;
			}
			return JSON.stringify(content.resource ?? content, null, 2);

		case MCPContentType.ResourceLink:
			return [content.title || content.name || content.uri, content.description, content.uri]
				.filter(Boolean)
				.join('\n');

		case MCPContentType.Image:
			return `[MCP image content${content.mimeType ? `: ${content.mimeType}` : ''}]`;

		case MCPContentType.Audio:
			return `[MCP audio content${content.mimeType ? `: ${content.mimeType}` : ''}]`;

		default:
			return JSON.stringify(content, null, 2);
	}
}

function mcpToolLabel(selection: MCPToolSelection, fallbackName: string): string {
	return selection.toolName || selection.providerToolName || fallbackName;
}

function mcpFailureText(message: string): string {
	const normalized = message.trim() || 'MCP tool invocation failed.';
	return [
		`MCP tool error: ${normalized}`,
		'This is a tool error result for the requested call.',
		'The model may correct the arguments or retry the tool call.',
	].join('\n\n');
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
	if (!selection.server || !selection.toolName) {
		const message = 'Cannot resolve MCP tool identity for this call.';
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: mcpFailureText(message),
				isError: true,
				errorMessage: message,
			}),
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
				text: mcpFailureText((err as Error)?.message || 'Invalid MCP tool arguments.'),
				isError: true,
				errorMessage: (err as Error)?.message || 'Invalid MCP tool arguments.',
			}),
		};
	}

	const req: InvokeMCPToolRequestBody = {
		source: MCPInvocationSource.Model,
		toolName: selection.toolName,
		providerToolName: selection.providerToolName || toolCall.name,
		choiceID: selection.choiceID,
		toolDigest: selection.digest,
		arguments: parsedArgs,
		toolUseID: toolCall.callID || toolCall.id,
	};

	let evaluation: MCPApprovalEvaluation;
	try {
		evaluation = await mcpAPI.evaluateMCPToolCall(selection.server, req);
	} catch (error) {
		const message = error instanceof Error && error.message.trim() ? error.message : 'MCP approval evaluation failed.';
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: mcpFailureText(message),
				isError: true,
				errorMessage: message,
			}),
		};
	}
	if (!evaluation) {
		const message = 'MCP approval evaluation did not return a decision.';
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: mcpFailureText(message),
				isError: true,
				errorMessage: message,
			}),
		};
	}

	if (evaluation.decision === MCPApprovalDecision.Denied) {
		const message = evaluation.reason || 'MCP policy denied this tool call.';
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: mcpFailureText(message),
				isError: true,
				errorMessage: message,
			}),
		};
	}

	if (evaluation.decision === MCPApprovalDecision.ApprovalRequired) {
		if (!evaluation.approvalID) {
			const message = 'MCP approval was required but no approval ID was returned.';
			return {
				ok: true,
				output: buildMCPToolOutput({
					toolCall,
					selection,
					text: mcpFailureText(message),
					isError: true,
					errorMessage: message,
				}),
			};
		}

		let approval: MCPApprovalResolutionResult;

		try {
			approval =
				requestMCPApproval && evaluation.summary
					? await requestMCPApproval({
							approvalID: evaluation.approvalID,
							summary: evaluation.summary,
							reason: evaluation.reason,
						})
					: await mcpAPI.resolveMCPApproval(evaluation.approvalID, MCPApprovalResolution.DenyOnce);
		} catch (error) {
			const message =
				error instanceof Error && error.message.trim() ? error.message : 'MCP approval could not be resolved.';
			return {
				ok: true,
				output: buildMCPToolOutput({
					toolCall,
					selection,
					text: mcpFailureText(message),
					isError: true,
					errorMessage: message,
				}),
			};
		}

		if (approval.decision !== MCPApprovalDecision.Allowed) {
			const message = evaluation.reason
				? `MCP tool call denied by user. ${evaluation.reason}`
				: 'MCP tool call denied by user.';
			return {
				ok: true,
				output: buildMCPToolOutput({
					toolCall,
					selection,
					text: mcpFailureText(message),
					isError: true,
					errorMessage: message,
				}),
			};
		}

		if (approval.resolution === MCPApprovalResolution.AllowOnce && !approval.token) {
			const message = 'Allow-once approval did not return a usable token.';
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
		req.approvalID = approval.approvalID;
		if (approval.token) {
			req.approvalToken = approval.token;
		}
	} else if (evaluation.decision !== MCPApprovalDecision.Allowed) {
		const message = `Unsupported MCP approval decision: ${String(evaluation.decision)}`;
		return {
			ok: true,
			output: buildMCPToolOutput({
				toolCall,
				selection,
				text: mcpFailureText(message),
				isError: true,
				errorMessage: message,
			}),
		};
	}

	try {
		const resp = await withTimeout(
			mcpAPI.invokeMCPTool(selection.server, req),
			TOOL_CALL_TIMEOUT_MS,
			`MCP tool call "${selection.toolName}" timed out after ${Math.round(TOOL_CALL_TIMEOUT_MS / 1000)} seconds.`
		);

		const toolContent = Array.isArray(resp?.content) ? resp.content : undefined;
		const structuredContent = isJSONObject(resp?.structuredContent) ? resp.structuredContent : undefined;

		const contentText = (toolContent ?? [])
			.map(t => mcpContentToText(t))
			.filter(Boolean)
			.join('\n\n');
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
				text: mcpFailureText(message),
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
				skillManagementAPI.invokeSkillTool(sid, toolCall.name, args),
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
					skillRuntimeMeta: resp.meta,
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
