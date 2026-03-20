import {
	InputKind,
	type InputUnion,
	OutputKind,
	type OutputUnion,
	type ToolCall,
	type UIToolCall,
} from '@/spec/inference';

import { getPrettyToolName } from '@/tools/lib/tool_identity_utils';

export function collectToolCallsFromInputs(
	inputs: InputUnion[] | undefined,
	existing?: Map<string, ToolCall>
): Map<string, ToolCall> {
	const map = existing ?? new Map<string, ToolCall>();

	const addCall = (call?: ToolCall) => {
		if (call?.callID) map.set(call.callID, call);
	};

	if (!inputs) return map;

	for (const iu of inputs) {
		switch (iu.kind) {
			case InputKind.FunctionToolCall:
				addCall(iu.functionToolCall);
				break;
			case InputKind.CustomToolCall:
				addCall(iu.customToolCall);
				break;
			case InputKind.WebSearchToolCall:
				addCall(iu.webSearchToolCall);
				break;
			default:
				break;
		}
	}

	return map;
}

export function collectToolCallsFromOutputs(
	outputs: OutputUnion[] | undefined,
	existing?: Map<string, ToolCall>
): Map<string, ToolCall> {
	const map = existing ?? new Map<string, ToolCall>();

	const addCall = (call?: ToolCall) => {
		if (call?.callID) map.set(call.callID, call);
	};

	if (!outputs) return map;

	for (const o of outputs) {
		switch (o.kind) {
			case OutputKind.FunctionToolCall:
				addCall(o.functionToolCall);
				break;
			case OutputKind.CustomToolCall:
				addCall(o.customToolCall);
				break;
			case OutputKind.WebSearchToolCall:
				addCall(o.webSearchToolCall);
				break;
			default:
				break;
		}
	}

	return map;
}

/**
 * Best-effort short summary of tool-call arguments for chip labels.
 */
function summarizeToolCallArguments(args: string): string | undefined {
	if (!args) return undefined;
	try {
		const parsed = JSON.parse(args);
		if (parsed == null || typeof parsed !== 'object') {
			return typeof parsed === 'string' ? parsed : undefined;
		}
		const obj = parsed as Record<string, unknown>;
		const primaryKeys = ['file', 'path', 'url', 'query', 'id', 'name'];
		const parts: string[] = [];

		for (const key of primaryKeys) {
			if (obj[key] != null) {
				parts.push(obj[key] as string);
			}
		}

		if (parts.length === 0) {
			const keys = Object.keys(obj);
			for (const key of keys.slice(0, 2)) {
				parts.push(`${key}=${String(obj[key])}`);
			}
		}

		return parts.length ? parts.join(', ') : undefined;
	} catch {
		return undefined;
	}
}

/**
 * Label used for tool-call chips in composer / history.
 */
export function formatToolCallLabel(call: UIToolCall): string {
	const pretty = getPrettyToolName(call.name);
	const argSummary = summarizeToolCallArguments(call.arguments ?? '');
	return argSummary ? `${pretty}: ${argSummary}` : pretty;
}
