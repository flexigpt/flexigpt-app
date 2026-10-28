import type { UIToolOutput } from '@/spec/inference';
import type { MCPContent } from '@/spec/session_mcp';
import { MCPContentType } from '@/spec/session_mcp';

import { isJSONObject } from '@/lib/jsonschema_utils';

function textFromUIToolOutputItem(item: unknown): string {
	const maybe = item as { textItem?: { text?: unknown } } | null | undefined;
	const text = maybe?.textItem?.text;
	return typeof text === 'string' ? text : '';
}

export function getMCPAppToolResultContent(output: UIToolOutput): MCPContent[] | undefined {
	if (Array.isArray(output.mcpApp?.content)) {
		return output.mcpApp.content;
	}

	// Legacy fallback only: if an older output has an app URI but did not
	// preserve MCP content, convert FlexiGPT text outputs back into MCP text
	// content. Never use this as structuredContent.
	const text = (output.toolOutputs ?? [])
		.map(t => textFromUIToolOutputItem(t))
		.filter(Boolean)
		.join('\n\n');
	if (!text) {
		return undefined;
	}

	return [
		{
			type: MCPContentType.Text,
			text,
		},
	];
}

export function getMCPAppToolResultStructuredContent(output: UIToolOutput): Record<string, unknown> | undefined {
	return isJSONObject(output.mcpApp?.structuredContent) ? output.mcpApp.structuredContent : undefined;
}
