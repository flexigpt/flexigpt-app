import { FiChevronUp, FiServer, FiX } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore } from '@ariakit/react';

import { type MCPConversationContext, type MCPServerSelection, MCPToolExposure } from '@/spec/mcp';

import type { UseComposerMCPResult } from '@/chats/composer/mcp/mcp_composer_types';

function toolExposureLabel(server: MCPServerSelection): string {
	if (server.toolExposure === MCPToolExposure.MCPToolExposureAll) {
		const count = server.selectedTools?.length ?? 0;
		return count > 0 ? `All tools (${count})` : 'All tools';
	}
	if (server.toolExposure === MCPToolExposure.MCPToolExposureSelected)
		return `${server.selectedTools?.length ?? 0} tools`;
	return 'No tools';
}

export function MCPComposerChips({ state, isBusy = false }: { state: UseComposerMCPResult; isBusy?: boolean }) {
	const count = state.selectedServerCount;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	if (count === 0) return null;

	return (
		<div
			className="bg-secondary/10 text-base-content border-secondary/40 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={`MCP\n${count} selected server${count === 1 ? '' : 's'}`}
			data-attachment-chip="mcp-context"
		>
			<FiServer size={14} />
			<span className="max-w-24 truncate">MCP</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show MCP context"
				title="Show MCP context"
			>
				<FiChevronUp size={14} />
			</MenuButton>

			<button
				type="button"
				className="btn btn-ghost btn-xs text-error shrink-0 px-0 py-0 shadow-none"
				onClick={() => {
					state.clear();
					menu.hide();
				}}
				disabled={isBusy}
				title="Clear MCP context"
				aria-label="Clear MCP context"
			>
				<FiX size={14} />
			</button>

			<Menu
				store={menu}
				gutter={8}
				overflowPadding={8}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 max-w-lg min-w-72 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-2 text-xs font-semibold">MCP context</div>

				{Object.values(state.selectedByServerKey).map(selection => (
					<div key={`${selection.bundleID}:${selection.serverID}`} className="bg-base-200 mb-1 rounded-xl px-2 py-1">
						<div className="truncate text-xs font-semibold">{selection.serverID}</div>
						<div className="text-base-content/60 truncate text-xs">{selection.bundleID}</div>
						<div className="mt-1 flex flex-wrap gap-1">
							<span className="badge badge-ghost badge-xs">{toolExposureLabel(selection)}</span>
							{selection.selectedResources.length > 0 ? (
								<span className="badge badge-ghost badge-xs">{selection.selectedResources.length} resources</span>
							) : null}
							{selection.selectedResourceTemplates.length > 0 ? (
								<span className="badge badge-ghost badge-xs">
									{selection.selectedResourceTemplates.length} resource templates
								</span>
							) : null}
							{selection.selectedPrompts.length > 0 ? (
								<span className="badge badge-ghost badge-xs">{selection.selectedPrompts.length} prompts</span>
							) : null}
							{selection.includeServerInstructions ? (
								<span className="badge badge-ghost badge-xs">instructions</span>
							) : null}
						</div>
					</div>
				))}
			</Menu>
		</div>
	);
}

export function MCPMessageContextChip({ context }: { context?: MCPConversationContext }) {
	const count = context?.servers?.length ?? 0;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	if (!context || count === 0) return null;

	const resourceCount = (context.resources?.length ?? 0) + (context.resourceTemplates?.length ?? 0);
	const promptCount = context.prompts?.length ?? 0;

	return (
		<div
			className="bg-secondary/10 text-base-content border-secondary/40 flex shrink-0 items-center gap-1 rounded-2xl border px-2 py-0"
			title={`MCP\n${count} server${count === 1 ? '' : 's'}`}
			data-message-chip="mcp-context"
		>
			<FiServer size={14} />
			<span className="max-w-24 truncate">MCP</span>
			<span className="text-base-content/60 whitespace-nowrap">{count}</span>

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label="Show MCP context"
				title="Show MCP context"
			>
				<FiChevronUp size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={8}
				overflowPadding={8}
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
