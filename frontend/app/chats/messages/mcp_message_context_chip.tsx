import { FiChevronRight, FiServer } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore } from '@ariakit/react';

import { type MCPConversationContext } from '@/spec/mcp';

import { toolExposureLabel } from '@/chats/messages/mcp_message_context_utils';

export function MCPMessageContextChip({ context }: { context?: MCPConversationContext }) {
	const count = context?.servers?.length ?? 0;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	if (!context || count === 0) {
		return null;
	}

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
