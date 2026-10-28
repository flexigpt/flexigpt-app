import { FiChevronRight, FiServer } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore, useStoreState } from '@ariakit/react';

import type { SessionOperationResolutionSummary } from '@/spec/session_base';
import { SessionResolutionState } from '@/spec/session_base';

import { getMessageBarChipClassName } from '@/chats/messages/message_bar_chip_utils';

function namedItems(
	items:
		| Array<{
				displayName: string;
				description?: string;
				availability: 'available' | 'unavailable';
		  }>
		| undefined
) {
	if (!items?.length) {
		return null;
	}

	return (
		<div className="mt-1 space-y-1">
			{items.map((item, index) => (
				<div key={`${item.displayName}:${index}`} className="bg-base-100 flex items-start gap-2 rounded-lg px-2 py-1">
					<div className="min-w-0 flex-1">
						<div className="truncate text-[11px]">{item.displayName}</div>
						{item.description ? <div className="text-base-content/60 text-[10px]">{item.description}</div> : null}
					</div>
					<span className={`badge badge-xs ${item.availability === 'available' ? 'badge-info' : 'badge-warning'}`}>
						{item.availability === 'available' ? 'Available' : 'Unavailable'}
					</span>
				</div>
			))}
		</div>
	);
}

export function MCPMessageContextChip({ resolution }: { resolution?: SessionOperationResolutionSummary }) {
	const servers = resolution?.mcp ?? [];
	const count = servers.length;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const open = useStoreState(menu, 'open');

	if (count === 0) {
		return null;
	}

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

					{servers.map((server, index) => (
						<div
							key={`${server.used?.displayName || 'Unavailable MCP server'}:${index}`}
							className="bg-base-200 mb-2 rounded-xl p-2 last:mb-0"
						>
							<div className="text-xs font-medium">{server.used?.displayName || 'Unavailable MCP server'}</div>
							<div className="mt-1 flex flex-wrap gap-1">
								<span
									className={`badge badge-xs ${server.state === SessionResolutionState.Included ? 'badge-info' : 'badge-warning'}`}
								>
									{server.state === SessionResolutionState.Included ? 'Available' : 'Unavailable'}
								</span>
								<span className="badge badge-ghost badge-xs">
									{server.mcp?.instructionsIncluded ? 'Instructions included' : 'Instructions not included'}
								</span>
							</div>

							{server.mcp?.tools?.length ? (
								<div className="text-base-content/70 mt-2 text-[10px]">Selected Tools</div>
							) : null}
							{namedItems(server.mcp?.tools)}

							{server.mcp?.resources?.length ? (
								<div className="text-base-content/70 mt-2 text-[10px]">Selected Resources</div>
							) : null}
							{namedItems(server.mcp?.resources)}

							{server.mcp?.resourceTemplates?.length ? (
								<div className="text-base-content/70 mt-2 text-[10px]">Selected Resource templates</div>
							) : null}
							{namedItems(server.mcp?.resourceTemplates)}

							{server.mcp?.prompts?.length ? (
								<div className="text-base-content/70 mt-2 text-[10px]">Selected Prompts</div>
							) : null}
							{namedItems(server.mcp?.prompts)}
						</div>
					))}
				</Menu>
			) : null}
		</div>
	);
}
