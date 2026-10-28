import { FiChevronRight, FiFileText, FiFolderPlus, FiServer, FiZap } from 'react-icons/fi';

import { Menu, MenuButton, useMenuStore, useStoreState } from '@ariakit/react';

import type { SessionOperationResolutionSummary, SessionResolutionItem } from '@/spec/session_base';
import type { MCPAppContextUpdate } from '@/spec/session_mcp';
import { SessionConfigurationOwner, SessionResolutionState, SessionWorkspaceContextOutcome } from '@/spec/session_base';

import { getMessageBarChipClassName } from '@/chats/messages/message_bar_chip_utils';

function itemLabel(item: SessionResolutionItem | undefined, unavailable: string): string {
	return item?.used?.displayName?.trim() || unavailable;
}

function sourceLocations(item: SessionResolutionItem): string[] {
	const values = item.used?.sourceLocations?.filter(Boolean) ?? [];
	if (values.length > 0) {
		return values;
	}
	return item.used?.sourceLocation?.trim() ? [item.used.sourceLocation] : [];
}

function contextOutcomeLabel(value: SessionWorkspaceContextOutcome | undefined): string {
	if (value === undefined) {
		return 'Unavailable';
	}
	switch (value) {
		case SessionWorkspaceContextOutcome.Included:
			return 'Included';
		case SessionWorkspaceContextOutcome.PartiallyIncluded:
			return 'Partially included';
		case SessionWorkspaceContextOutcome.NotIncluded:
			return 'Not included';
		case SessionWorkspaceContextOutcome.NotAllowed:
			return 'Not allowed';
		default:
			return 'Unavailable';
	}
}

function isWorkspaceContext(item: SessionResolutionItem): boolean {
	return item.owner === SessionConfigurationOwner.Workspace && item.kind === 'workspaceContext';
}

function isWorkspaceSkill(item: SessionResolutionItem): boolean {
	return item.owner === SessionConfigurationOwner.Workspace && item.kind === 'skill';
}

function ContextLocations({ item }: { item: SessionResolutionItem }) {
	const locations = sourceLocations(item);
	if (locations.length === 0) {
		return null;
	}

	return (
		<div className="text-base-content/60 mt-1 space-y-0.5 text-[10px]">
			{locations.map((location, index) => (
				<div key={`${location}:${index}`} className="truncate">
					{location}
				</div>
			))}
		</div>
	);
}

export function MessageSkillsResolutionChip({ resolution }: { resolution?: SessionOperationResolutionSummary }) {
	const skills = resolution?.skills ?? [];
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const open = useStoreState(menu, 'open');

	if (skills.length === 0) {
		return null;
	}

	const activeCount = skills.filter(item => item.skill?.active).length;
	const enabledCount = skills.filter(item => item.skill?.enabled).length;

	return (
		<div className="shrink-0">
			<MenuButton
				store={menu}
				className={getMessageBarChipClassName('secondary', false, true)}
				title={`Skills recorded for this message\n${enabledCount} enabled\n${activeCount} active`}
				data-message-chip="skills-context"
				aria-label="Show Skills used for this message"
			>
				<FiZap size={14} />
				<span className="max-w-24 truncate">Skills</span>
				<span className="text-base-content/60 whitespace-nowrap">{skills.length}</span>
				<FiChevronRight className={`shrink-0 transition-transform ${open ? 'rotate-90' : ''}`} size={14} />
			</MenuButton>

			{open ? (
				<Menu
					store={menu}
					gutter={8}
					overflowPadding={8}
					className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-96 w-lg max-w-[calc(100vw-1rem)] overflow-y-auto border p-3 shadow-xl"
				>
					<div className="text-sm font-semibold">Skills recorded for this message</div>
					<div className="text-base-content/60 mt-1 text-xs">Recorded at send time.</div>

					<div className="mt-3 space-y-1">
						{skills.map((item, index) => {
							const state = item.skill;
							const available = item.state === SessionResolutionState.Included && Boolean(state?.sessionAvailable);

							return (
								<div key={`${itemLabel(item, 'Unavailable Skill')}:${index}`} className="bg-base-100 rounded-xl p-2">
									<div className="flex min-w-0 items-start gap-2">
										<FiZap className="mt-0.5 shrink-0" size={13} />
										<div className="min-w-0 flex-1">
											<div className="truncate text-xs font-medium">{itemLabel(item, 'Unavailable Skill')}</div>
											{item.used?.description ? (
												<div className="text-base-content/60 mt-1 text-[10px]">{item.used.description}</div>
											) : null}
											<ContextLocations item={item} />
										</div>
										<div className="flex shrink-0 flex-wrap justify-end gap-1">
											<span className={`badge badge-xs ${state?.enabled ? 'badge-info' : 'badge-ghost'}`}>
												{state?.enabled ? 'Enabled' : 'Disabled'}
											</span>
											<span className={`badge badge-xs ${state?.active ? 'badge-success' : 'badge-ghost'}`}>
												{state?.active ? 'Active' : 'Inactive'}
											</span>
											<span className={`badge badge-xs ${available ? 'badge-info' : 'badge-warning'}`}>
												{available ? 'Session available' : 'Session unavailable'}
											</span>
										</div>
									</div>
								</div>
							);
						})}
					</div>
				</Menu>
			) : null}
		</div>
	);
}

export function MessageWorkspaceResolutionChip({ resolution }: { resolution?: SessionOperationResolutionSummary }) {
	const workspace = resolution?.workspace;
	const contexts = (resolution?.instructions ?? []).filter(i => {
		return isWorkspaceContext(i);
	});
	const skills = (resolution?.skills ?? []).filter(s => {
		return isWorkspaceSkill(s);
	});
	const servers = (resolution?.mcp ?? []).filter(item => item.owner === SessionConfigurationOwner.Workspace);
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const open = useStoreState(menu, 'open');

	if (!workspace) {
		return null;
	}

	const workspaceName = itemLabel(workspace, 'Workspace');
	const unavailable = workspace.state !== SessionResolutionState.Included;

	return (
		<div className="shrink-0">
			<MenuButton
				store={menu}
				className={getMessageBarChipClassName(unavailable ? 'secondary' : 'info', false, true)}
				title={`${workspaceName}\nWorkspace state recorded for this message`}
				data-message-chip="workspace-context"
				aria-label="Show Workspace state for this message"
			>
				<FiFolderPlus size={14} />
				<span className="max-w-32 truncate">{workspaceName}</span>
				<FiChevronRight className={`shrink-0 transition-transform ${open ? 'rotate-90' : ''}`} size={14} />
			</MenuButton>

			{open ? (
				<Menu
					store={menu}
					gutter={8}
					overflowPadding={8}
					className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-96 w-lg max-w-[calc(100vw-1rem)] overflow-y-auto border p-3 shadow-xl"
				>
					<div className="text-sm font-semibold">{workspaceName}</div>
					<div className="text-base-content/60 mt-1 text-xs">Selected and recorded at send time.</div>

					{contexts.length > 0 ? (
						<section className="mt-3">
							<div className="text-xs font-semibold">Context outcome</div>
							<div className="mt-1 space-y-1">
								{contexts.map((item, index) => (
									<div
										key={`${itemLabel(item, 'Unavailable Context')}:${index}`}
										className="bg-base-100 flex gap-2 rounded-xl p-2"
									>
										<FiFileText className="mt-0.5 shrink-0" size={13} />
										<div className="min-w-0 flex-1">
											<div className="truncate text-xs">{itemLabel(item, 'Unavailable Context')}</div>
											<ContextLocations item={item} />
											{item.workspaceContext?.diagnosticSummary ? (
												<div className="text-base-content/60 mt-1 text-[10px]">
													{item.workspaceContext.diagnosticSummary}
												</div>
											) : null}
										</div>
										<span className="badge badge-ghost badge-xs">
											{contextOutcomeLabel(item.workspaceContext?.outcome)}
										</span>
									</div>
								))}
							</div>
						</section>
					) : null}

					{skills.length > 0 ? (
						<section className="mt-3">
							<div className="text-xs font-semibold">Workspace Skill outcome</div>
							<div className="mt-1 space-y-1">
								{skills.map((item, index) => (
									<div
										key={`${itemLabel(item, 'Unavailable Workspace Skill')}:${index}`}
										className="bg-base-100 flex gap-2 rounded-xl p-2"
									>
										<FiZap className="mt-0.5 shrink-0" size={13} />
										<div className="min-w-0 flex-1">
											<div className="truncate text-xs">{itemLabel(item, 'Unavailable Workspace Skill')}</div>
											<ContextLocations item={item} />
										</div>
										<div className="flex shrink-0 flex-wrap justify-end gap-1">
											<span className={`badge badge-xs ${item.skill?.enabled ? 'badge-info' : 'badge-ghost'}`}>
												{item.skill?.enabled ? 'Enabled' : 'Disabled'}
											</span>
											<span className={`badge badge-xs ${item.skill?.active ? 'badge-success' : 'badge-ghost'}`}>
												{item.skill?.active ? 'Active' : 'Inactive'}
											</span>
											<span
												className={`badge badge-xs ${item.skill?.sessionAvailable ? 'badge-info' : 'badge-warning'}`}
											>
												{item.skill?.sessionAvailable ? 'Session available' : 'Session unavailable'}
											</span>
										</div>
									</div>
								))}
							</div>
						</section>
					) : null}

					{servers.length > 0 ? (
						<section className="mt-3">
							<div className="text-xs font-semibold">Workspace MCP servers</div>
							<div className="mt-1 space-y-1">
								{servers.map((item, index) => (
									<div
										key={`${itemLabel(item, 'Unavailable MCP server')}:${index}`}
										className="bg-base-100 flex gap-2 rounded-xl p-2"
									>
										<FiServer className="mt-0.5 shrink-0" size={13} />
										<div className="min-w-0 flex-1">
											<div className="truncate text-xs">{itemLabel(item, 'Unavailable MCP server')}</div>
											<ContextLocations item={item} />
										</div>
										<span
											className={`badge badge-xs ${item.state === SessionResolutionState.Included ? 'badge-info' : 'badge-warning'}`}
										>
											{item.state === SessionResolutionState.Included ? 'Available' : 'Unavailable'}
										</span>
									</div>
								))}
							</div>
						</section>
					) : null}
				</Menu>
			) : null}
		</div>
	);
}

export function MCPAppContextResolutionChip({ updates }: { updates: MCPAppContextUpdate[] }) {
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const open = useStoreState(menu, 'open');

	if (updates.length === 0) {
		return null;
	}

	return (
		<div className="shrink-0">
			<MenuButton
				store={menu}
				className={getMessageBarChipClassName('secondary', false, true)}
				title={`MCP App context\n${updates.length} update${updates.length === 1 ? '' : 's'}`}
				data-message-chip="mcp-app-context"
				aria-label="Show MCP App context for this message"
			>
				<FiServer size={14} />
				<span className="max-w-28 truncate">App context</span>
				<span className="text-base-content/60 whitespace-nowrap">{updates.length}</span>
				<FiChevronRight className={`shrink-0 transition-transform ${open ? 'rotate-90' : ''}`} size={14} />
			</MenuButton>

			{open ? (
				<Menu
					store={menu}
					gutter={8}
					overflowPadding={8}
					className="rounded-box bg-base-200 text-base-content border-base-content z-50 max-h-96 w-lg max-w-[calc(100vw-1rem)] overflow-y-auto border p-3 shadow-xl"
				>
					<div className="text-sm font-semibold">MCP App context</div>
					<div className="text-base-content/60 mt-1 text-xs">Recorded at send time.</div>

					<div className="mt-3 space-y-2">
						{updates.map((update, index) => (
							<div
								key={`${update.contextLabel || 'App context update'}:${index}`}
								className="bg-base-100 rounded-xl p-2"
							>
								<div className="text-xs font-medium">{update.contextLabel || 'App context update'}</div>
								<div className="text-base-content/60 mt-1 text-[10px]">
									{update.appDisplayName ? `${update.appDisplayName} · ` : ''}
									{update.serverDisplayName || 'MCP server'}
								</div>

								{update.content?.length ? (
									<div className="mt-2 space-y-1">
										{update.content.map((item, itemIndex) => (
											<div
												key={`${item.displayName || 'App context item'}:${itemIndex}`}
												className="border-base-content/10 rounded-lg border px-2 py-1"
											>
												<div className="text-[11px]">{item.displayName || 'App context item'}</div>
												{item.description ? (
													<div className="text-base-content/60 mt-0.5 text-[10px]">{item.description}</div>
												) : null}
											</div>
										))}
									</div>
								) : null}

								{update.structuredContentLabel ? (
									<div className="border-base-content/10 mt-2 rounded-lg border px-2 py-1 text-[11px]">
										{update.structuredContentLabel}
									</div>
								) : null}
							</div>
						))}
					</div>
				</Menu>
			) : null}
		</div>
	);
}
