import { FiChevronRight, FiServer } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore } from '@ariakit/react';

import type { UIToolOutput } from '@/spec/inference';
import {
	type MCPContent,
	MCPContentType,
	type MCPConversationContext,
	type MCPServerSelection,
	MCPToolExposure,
} from '@/spec/mcp';

import { isJSONObject } from '@/lib/jsonschema_utils';

function toolExposureLabel(server: MCPServerSelection): string {
	if (server.toolExposure === MCPToolExposure.MCPToolExposureAll) {
		const count = server.selectedTools?.length ?? 0;
		return count > 0 ? `All tools (${count})` : 'All tools';
	}
	if (server.toolExposure === MCPToolExposure.MCPToolExposureSelected)
		return `${server.selectedTools?.length ?? 0} tools`;
	return 'No tools';
}

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
	if (!text) return undefined;

	return [
		{
			type: MCPContentType.MCPContentTypeText,
			text,
		},
	];
}

export function getMCPAppToolResultStructuredContent(output: UIToolOutput): Record<string, unknown> | undefined {
	return isJSONObject(output.mcpApp?.structuredContent) ? output.mcpApp.structuredContent : undefined;
}

export function MCPMessageContextChip({ context }: { context?: MCPConversationContext }) {
	const count = context?.servers?.length ?? 0;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	if (!context || count === 0) return null;

	const resourceCount = (context.resources?.length ?? 0) + (context.resourceTemplates?.length ?? 0);
	const promptCount = context.prompts?.length ?? 0;

	return (
		<div
			className="bg-secondary/10 text-base-content border-secondary/40 flex min-h-6 shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={`MCP\n${count} server${count === 1 ? '' : 's'}`}
			data-message-chip="mcp-context"
		>
			<FiServer size={14} />
			<span className="max-w-24 truncate">MCP</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs h-5 min-h-0 p-0 shadow-none"
				aria-label="Show MCP context for this message"
				title="Show MCP context for this message"
			>
				<FiChevronRight size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={8}
				overflowPadding={8}
				portal
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 max-w-lg min-w-72 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-2 text-xs font-semibold">MCP context</div>

				{context.servers.map(server => (
					<div key={`${server.bundleID}:${server.serverID}`} className="bg-base-200 mb-1 rounded-xl px-2 py-1">
						<div className="truncate text-xs font-semibold">{server.serverID}</div>
						<div className="text-base-content/60 truncate text-xs">{server.bundleID}</div>
						<div className="mt-1 flex flex-wrap gap-1">
							<span className="badge badge-ghost badge-xs">{toolExposureLabel(server)}</span>
							{server.includeServerInstructions ? (
								<span className="badge badge-ghost badge-xs">instructions</span>
							) : null}
						</div>
					</div>
				))}

				{resourceCount > 0 ? <div className="text-base-content/70 mt-2 text-xs">Resources: {resourceCount}</div> : null}
				{promptCount > 0 ? <div className="text-base-content/70 text-xs">Prompts: {promptCount}</div> : null}
			</Menu>
		</div>
	);
}
