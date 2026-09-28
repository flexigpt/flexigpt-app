import { FiChevronRight, FiServer } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore, useStoreState } from '@ariakit/react';

import type { MCPConversationContext } from '@/spec/mcp';

import { toolExposureLabel } from '@/chats/messages/mcp_message_context_utils';
import { getMessageBarChipClassName } from '@/chats/messages/message_bar_chip_utils';

export function MCPMessageContextChip({ context }: { context?: MCPConversationContext }) {
	const count = context?.servers?.length ?? 0;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const open = useStoreState(menu, 'open');

	if (!context || count === 0) {
		return null;
	}

	const resourceCount = (context.resources?.length ?? 0) + (context.resourceTemplates?.length ?? 0);
	const promptCount = context.prompts?.length ?? 0;

	return (
		<div className="shrink-0">
			<MenuButton
				store={menu}
				className={getMessageBarChipClassName('secondary', false, true)}
				aria-label="Show MCP context for this message"
				title={`MCP\n${count} server${count === 1 ? '' : 's'}`}
				data-message-chip="mcp-context"
			>
				<FiServer size={14} />
				<span className="max-w-24 truncate">MCP</span>
				<span className="text-base-content/60 whitespace-nowrap">{count}</span>
				<FiChevronRight
					className={`shrink-0 transition-transform ${open ? 'rotate-90' : ''}`}
					aria-hidden="true"
					size={14}
				/>
			</MenuButton>

			{open ? (
				<Menu
					store={menu}
					gutter={8}
					overflowPadding={8}
					portal
					className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 max-w-lg min-w-72 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
					autoFocusOnShow
				>
					<div className="text-base-content/70 mb-2 text-xs font-semibold">MCP context</div>

					{context.servers.map((server, index) => (
						<div key={server.server} className="bg-base-200 mb-1 rounded-xl px-2 py-1">
							<div className="text-xs font-medium">MCP server {index + 1}</div>
							<div className="mt-1 flex flex-wrap gap-1">
								<span className="badge badge-ghost badge-xs">{toolExposureLabel(server)}</span>
								{server.includeServerInstructions ? (
									<span className="badge badge-ghost badge-xs">instructions</span>
								) : null}
							</div>
						</div>
					))}

					{resourceCount > 0 ? (
						<div className="text-base-content/70 mt-2 text-xs">Resources: {resourceCount}</div>
					) : null}
					{promptCount > 0 ? <div className="text-base-content/70 text-xs">Prompts: {promptCount}</div> : null}
				</Menu>
			) : null}
		</div>
	);
}
