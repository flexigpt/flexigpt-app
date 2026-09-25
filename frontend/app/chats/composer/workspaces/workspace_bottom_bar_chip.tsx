import type { Dispatch, SetStateAction } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFolderPlus, FiRefreshCw, FiSearch, FiSettings, FiX } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { SkillRef } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';
import { WorkspaceDirectoryOrigin } from '@/spec/workspace';

import {
	actionTriggerChipClearButtonClasses,
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip, HoverTipContent } from '@/components/hover_tip';

import type {
	ComposerWorkspaceCandidate,
	ComposerWorkspaceController,
} from '@/chats/composer/workspaces/use_composer_workspace';
import { WorkspaceDirectorySelectionModal } from '@/chats/composer/workspaces/workspace_selection_modal';
import { WorkspaceDirectoryRegistrationModal } from '@/workspaces/workspace_directory_registration_modal';

interface WorkspaceBottomBarChipProps {
	store: MenuStore;
	state: ComposerWorkspaceController;
	activeSkillRefs: SkillRef[];
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	isInputLocked?: boolean;

	// Kept temporarily so composer callers do not need an unrelated migration.
	// Directory Workspaces no longer use the old Workspace template insertion
	// path. User-message Skills remain regular runtime capabilities.
	onInsertTemplateText: (text: string) => Promise<void> | void;
}

function workspaceKey(candidate: ComposerWorkspaceCandidate): string {
	const artifact = candidate.workspace.workspace.artifact;
	return `${artifact.rootID}:${artifact.id}`;
}

function workspaceDisplayName(candidate: ComposerWorkspaceCandidate): string {
	const artifact = candidate.workspace.workspace.artifact;
	return artifact.displayName || artifact.logicalName;
}

function workspaceSubtitle(candidate: ComposerWorkspaceCandidate): string {
	if (candidate.workspace.origin === WorkspaceDirectoryOrigin.Default) {
		return `${candidate.directory.root.displayName} · Default policy`;
	}

	return candidate.workspace.manifestLocator ?? candidate.workspace.workspace.artifact.binding.locator;
}

function workspaceIsAvailable(candidate: ComposerWorkspaceCandidate): boolean {
	const artifact = candidate.workspace.workspace.artifact;

	return candidate.directory.enabled && artifact.enabled && artifact.state === ArtifactState.Available;
}

function candidateMatchesSearch(candidate: ComposerWorkspaceCandidate, rawQuery: string): boolean {
	const query = rawQuery.trim().toLocaleLowerCase();

	if (!query) {
		return true;
	}

	const artifact = candidate.workspace.workspace.artifact;

	return [
		workspaceDisplayName(candidate),
		candidate.workspace.workspace.description,
		workspaceSubtitle(candidate),
		candidate.directory.root.displayName,
		candidate.directory.root.id,
		artifact.logicalName,
		artifact.binding.locator,
		candidate.workspace.origin,
	]
		.filter(Boolean)
		.join('\n')
		.toLocaleLowerCase()
		.includes(query);
}

function selectedWorkspaceName(state: ComposerWorkspaceController): string {
	if (state.workspace) {
		return workspaceDisplayName(state.workspace);
	}

	return state.selection?.displayName || 'Workspace';
}

export function WorkspaceBottomBarChip({
	store,
	state,
	activeSkillRefs,
	setActiveSkillRefs,
	isInputLocked = false,
}: WorkspaceBottomBarChipProps) {
	useEffect(() => {
		if (isInputLocked) {
			store.hide();
		}
	}, [isInputLocked, store]);

	return (
		<WorkspaceBottomBarChipContent
			store={store}
			state={state}
			activeSkillRefs={activeSkillRefs}
			setActiveSkillRefs={setActiveSkillRefs}
			isInputLocked={isInputLocked}
		/>
	);
}

function WorkspaceBottomBarChipContent({
	store,
	state,
	activeSkillRefs,
	setActiveSkillRefs,
	isInputLocked = false,
}: Omit<WorkspaceBottomBarChipProps, 'onInsertTemplateText'>) {
	const open = useStoreState(store, 'open');
	const [search, setSearch] = useState('');
	const [isDirectoryRegistrationOpen, setIsDirectoryRegistrationOpen] = useState(false);
	const [isSelectionOpen, setIsSelectionOpen] = useState(false);
	const [actionError, setActionError] = useState<string | null>(null);

	const visibleWorkspaces = useMemo(
		() => state.workspaces.filter(candidate => candidateMatchesSearch(candidate, search)),
		[search, state.workspaces]
	);

	const selected = Boolean(state.selection);
	const selectedName = selectedWorkspaceName(state);
	const selectedContextCount = state.selection?.contextRefs?.length ?? 0;
	const selectedSkillCount = state.selection?.skillRefs?.length ?? 0;
	const discoveredMCPCount = state.plan?.mcpServers.servers.length ?? 0;

	const hoverContent = (
		<HoverTipContent
			title="Workspace"
			description="Attach one effective repository Workspace to this conversation."
			sections={[
				{
					id: 'selection',
					title: 'Current selection',
					items: selected
						? [
								selectedName,
								`${selectedContextCount} Context contribution${selectedContextCount === 1 ? '' : 's'} selected`,
								`${selectedSkillCount} instruction Skill${selectedSkillCount === 1 ? '' : 's'} selected`,
								`${discoveredMCPCount} discovered MCP server${discoveredMCPCount === 1 ? '' : 's'} loaded`,
								state.attentionCount > 0
									? `${state.attentionCount} item${state.attentionCount === 1 ? '' : 's'} need attention`
									: 'Current Workspace capabilities resolved',
							]
						: ['No Workspace is attached.'],
				},
				{
					id: 'behavior',
					title: 'Behavior',
					items: [
						'Only effective Workspace declarations can be selected.',
						'Default policy Workspaces disappear when repository manifests exist.',
						'Workspace Context is composed again for each send.',
						'MCP installation inputs and secrets remain managed by MCP setup.',
					],
				},
			]}
		/>
	);

	return (
		<div className="relative shrink-0" data-bottom-bar-workspace>
			<HoverTip content={hoverContent} placement="top" wrapperElement="div" tooltipClassName="max-w-sm">
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${
						selected
							? state.attentionCount > 0
								? 'border-warning/50 bg-warning/10'
								: 'border-secondary/50 bg-secondary/10'
							: open
								? 'border-base-300 bg-base-300/60'
								: 'border-transparent'
					} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						store={store}
						disabled={isInputLocked}
						className="btn btn-xs app-text-neutral h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
						aria-label="Select Workspace"
					>
						<ActionTriggerChipContent
							icon={<FiFolderPlus size={14} />}
							label={selected ? selectedName : 'Workspace'}
							count={
								selectedContextCount > 0 ? (
									<span className="badge badge-secondary badge-xs">Context {selectedContextCount}</span>
								) : undefined
							}
							suffix={
								selected ? (
									<span className="flex items-center gap-1">
										{selectedSkillCount > 0 ? (
											<span className="badge badge-info badge-xs">Skills {selectedSkillCount}</span>
										) : null}
										{state.attentionCount > 0 ? (
											<FiAlertCircle size={14} className="text-warning" />
										) : (
											<FiCheck size={14} className="shrink-0" />
										)}
									</span>
								) : null
							}
							open={open}
						/>
					</MenuButton>

					{selected ? (
						<button
							type="button"
							className={actionTriggerChipClearButtonClasses}
							disabled={isInputLocked}
							onClick={event => {
								event.preventDefault();
								event.stopPropagation();

								void state
									.detachWorkspace()
									.then(() => {
										store.hide();
									})
									.catch((error: unknown) => {
										setActionError(error instanceof Error ? error.message : 'Workspace could not be detached.');
									});
							}}
							title="Detach Workspace from future turns"
							aria-label="Detach Workspace"
						>
							<FiX size={12} />
						</button>
					) : null}
				</div>
			</HoverTip>

			<Menu
				store={store}
				gutter={8}
				overflowPadding={8}
				portal
				autoFocusOnShow={false}
				className={actionTriggerMenuWideClasses}
			>
				<div className="mb-2 flex items-center justify-between gap-2 px-1">
					<div className="text-base-content/70 text-xs font-semibold">Effective Workspaces</div>
					<button
						type="button"
						className="btn btn-ghost btn-xs rounded-lg"
						disabled={state.workspacesLoading}
						onClick={() => {
							void state.refreshWorkspaces();
						}}
						aria-label="Refresh Workspace directory list"
						title="Refresh Workspace directory list"
					>
						<FiRefreshCw size={12} />
					</button>
				</div>

				{state.workspacesLoadError ? (
					<div className="alert alert-warning mb-2 rounded-xl text-xs">
						<FiAlertCircle size={14} />
						<span>{state.workspacesLoadError}</span>
					</div>
				) : null}

				{actionError ? (
					<div className="alert alert-error mb-2 rounded-xl text-xs">
						<FiAlertCircle size={14} />
						<span>{actionError}</span>
					</div>
				) : null}

				{selected ? (
					<div className="border-secondary/30 bg-secondary/10 mb-2 rounded-xl border p-2">
						<div className="truncate text-xs font-semibold">{selectedName}</div>
						<div className="text-base-content/60 mt-1 text-xs">
							{selectedContextCount} Context · {selectedSkillCount} instruction Skills
						</div>
						<div className="mt-2 flex flex-wrap gap-2">
							<button
								type="button"
								className="btn btn-xs rounded-lg"
								onClick={() => {
									store.hide();
									setIsSelectionOpen(true);
								}}
							>
								<FiSettings size={12} />
								Edit selection
							</button>
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={state.selectionLoading}
								onClick={() => {
									void state.refreshSelectedWorkspace();
								}}
							>
								<FiRefreshCw size={12} />
								Refresh status
							</button>
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								disabled={isInputLocked}
								onClick={() => {
									void state
										.detachWorkspace()
										.then(() => {
											store.hide();
										})
										.catch((error: unknown) => {
											setActionError(error instanceof Error ? error.message : 'Workspace could not be detached.');
										});
								}}
							>
								<FiX size={12} />
								Detach
							</button>
						</div>
					</div>
				) : null}

				<div className="input input-sm mb-2 flex items-center gap-2 rounded-xl">
					<label htmlFor="composer-workspace-search" className="sr-only">
						Search Workspaces
					</label>
					<FiSearch size={13} aria-hidden="true" />
					<input
						id="composer-workspace-search"
						type="search"
						className="grow"
						value={search}
						onChange={event => {
							setSearch(event.currentTarget.value);
						}}
						placeholder="Search effective Workspaces..."
						aria-label="Search effective Workspaces"
						spellCheck="false"
					/>
				</div>

				{state.workspacesLoading && state.workspaces.length === 0 ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
						Loading Workspace directories...
					</div>
				) : visibleWorkspaces.length === 0 ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
						No effective Workspaces match.
					</div>
				) : (
					<div className="space-y-1">
						{visibleWorkspaces.map(candidate => {
							const candidateKey = workspaceKey(candidate);
							const current =
								state.selection?.workspace.rootID === candidate.workspace.workspace.artifact.rootID &&
								state.selection?.workspace.artifactID === candidate.workspace.workspace.artifact.id;
							const available = workspaceIsAvailable(candidate);

							return (
								<MenuItem
									key={candidateKey}
									hideOnClick={false}
									disabled={!available || isInputLocked}
									className={`${actionTriggerMenuItemClasses} items-start`}
									onClick={() => {
										if (current || !available) {
											return;
										}

										setActionError(null);

										void state
											.attachWorkspace(candidate)
											.then(() => {
												store.hide();
											})
											.catch((error: unknown) => {
												setActionError(error instanceof Error ? error.message : 'Workspace could not be selected.');
											});
									}}
								>
									<FiFolderPlus size={14} className="mt-0.5 shrink-0" />
									<div className="min-w-0 flex-1">
										<div className="truncate text-xs font-medium">{workspaceDisplayName(candidate)}</div>
										<div className="text-base-content/60 truncate font-mono text-[10px]">
											{workspaceSubtitle(candidate)}
										</div>
									</div>
									{current ? (
										<span className="badge badge-secondary badge-xs">Selected</span>
									) : !available ? (
										<span className="badge badge-warning badge-xs">Unavailable</span>
									) : candidate.workspace.origin === WorkspaceDirectoryOrigin.Default ? (
										<span className="badge badge-info badge-xs">Default</span>
									) : (
										<span className="badge badge-ghost badge-xs">Manifest</span>
									)}
								</MenuItem>
							);
						})}
					</div>
				)}

				<div className="border-base-300 mt-2 border-t pt-2">
					<button
						type="button"
						className="btn btn-ghost btn-sm w-full justify-start rounded-xl"
						disabled={isInputLocked}
						onClick={() => {
							store.hide();
							setIsDirectoryRegistrationOpen(true);
						}}
					>
						<FiFolderPlus size={14} />
						Add Workspace Directory
					</button>
				</div>
			</Menu>

			<WorkspaceDirectoryRegistrationModal
				isOpen={isDirectoryRegistrationOpen}
				onClose={() => {
					setIsDirectoryRegistrationOpen(false);
				}}
				title="Add Workspace Directory to Conversation"
				description="Choose a repository directory. Its effective Workspace will be selected for this conversation after discovery succeeds."
				onRegister={async path => {
					setActionError(null);

					try {
						await state.createWorkspaceDirectory(path);
						store.hide();
					} catch (error) {
						const message = error instanceof Error ? error.message : 'Workspace directory could not be registered.';
						setActionError(message);
						throw error;
					}
				}}
			/>

			<WorkspaceDirectorySelectionModal
				isOpen={isSelectionOpen}
				onClose={() => {
					setIsSelectionOpen(false);
				}}
				state={state}
				activeSkillRefs={activeSkillRefs}
				setActiveSkillRefs={setActiveSkillRefs}
				isInputLocked={isInputLocked}
			/>
		</div>
	);
}
