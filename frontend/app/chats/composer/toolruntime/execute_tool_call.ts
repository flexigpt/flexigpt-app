import type { UIToolCall, UIToolOutput } from '@/spec/inference';

import { skillStoreAPI, toolRuntimeAPI } from '@/apis/baseapi';

import { isRunnableComposerToolCall, isSkillsToolName } from '@/chats/composer/toolruntime/tool_runtime_utils';
import { formatToolOutputSummary } from '@/tools/lib/tool_output_utils';

const TOOL_CALL_TIMEOUT_MS = 90_000;

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

interface ExecuteComposerToolCallArgs {
	toolCall: UIToolCall;
	ensureSkillSession: () => Promise<string | null>;
	getCurrentSkillSessionID: () => string | null;
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
