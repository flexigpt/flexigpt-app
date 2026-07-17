import type { InputUnion, OutputUnion, ToolCall, UIToolCall } from '@/spec/inference';
import { InputKind, OutputKind } from '@/spec/inference';
import { ToolStoreChoiceType } from '@/spec/tool';

export function isRunnableComposerToolCall(toolCall: UIToolCall): boolean {
	return (
		toolCall.type === ToolStoreChoiceType.Function ||
		toolCall.type === ToolStoreChoiceType.Custom ||
		Boolean(toolCall.mcpToolSelection)
	);
}

export function collectToolCallsFromInputs(
	inputs: InputUnion[] | undefined,
	existing?: Map<string, ToolCall>
): Map<string, ToolCall> {
	const map = existing ?? new Map<string, ToolCall>();

	const addCall = (call?: ToolCall) => {
		if (call?.callID) {
			map.set(call.callID, call);
		}
	};

	if (!inputs) {
		return map;
	}

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
		if (call?.callID) {
			map.set(call.callID, call);
		}
	};

	if (!outputs) {
		return map;
	}

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
