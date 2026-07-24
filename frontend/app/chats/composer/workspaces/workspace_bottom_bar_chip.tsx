import type { Dispatch, SetStateAction } from 'react';
import { useEffect, useMemo, useState } from 'react';

import {
	FiAlertCircle,
	FiBriefcase,
	FiCheck,
	FiFolderPlus,
	FiRefreshCw,
	FiSearch,
	FiSettings,
	FiX,
} from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { SkillRef } from '@/spec/skill';

import {
	actionTriggerChipClearButtonClasses,
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip, HoverTipContent } from '@/components/hover_tip';

import type { ComposerWorkspaceController } from '@/chats/composer/workspaces/use_composer_workspace';
import { WorkspaceSelectionModal } from '@/chats/composer/workspaces/workspace_selection_modal';
import type { WorkspaceSetupSubmission } from '@/workspaces/workspace_setup_modal';
import { WorkspaceSetupModal } from '@/workspaces/workspace_setup_modal';

interface WorkspaceBottomBarChipProps {
	store: MenuStore;
	state: ComposerWorkspaceController;
	activeSkillRefs: SkillRef[];
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	isInputLocked?: boolean;
}

export function WorkspaceBottomBarChip({
	store,
	state,
	activeSkillRefs,
	setActiveSkillRefs,
	isInputLocked = false,
}: WorkspaceBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const [search, setSearch] = useState('');
	const [isCreateOpen, setIsCreateOpen] = useState(false);
	const [isSelectionOpen, setIsSelectionOpen] = useState(false);
	const [actionError, setActionError] = useState<string | null>(null);

	useEffect(() => {
		if (!isInputLocked) {
			return;
		}
		store.hide();
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setIsCreateOpen(false);
		// oxlint-disable-next-line react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setIsSelectionOpen(false);
	}, [isInputLocked, store]);

	useEffect(() => {
		if (open) {
			void state.refreshWorkspaces();
		}
	}, [open, state]);

	const visibleWorkspaces = useMemo(() => {
		const query = search.trim().toLowerCase();
		if (!query) {
			return state.workspaces;
		}
		return state.workspaces.filter(workspace =>
			[workspace.displayName, workspace.description, workspace.primaryPath]
				.filter(Boolean)
				.join('\n')
				.toLowerCase()
				.includes(query)
		);
	}, [search, state.workspaces]);

	const selectedName = state.workspace?.displayName ?? state.selection?.displayName ?? 'Workspace';

	const selectedContextCount = state.selection?.contextRefs?.length ?? 0;
	const selectedSkillCount = state.selection?.skillRefs?.length ?? 0;
	const selected = Boolean(state.selection);

	const hoverContent = (
		<HoverTipContent
			title="Workspace"
			description="Attach project Context and Workspace Skills to this conversation without changing the configured Workspace."
			sections={[
				{
					id: 'current',
					title: 'Current selection',
					items: selected
						? [
								selectedName,
								`${selectedContextCount} Context file${selectedContextCount === 1 ? '' : 's'} selected`,
								`${selectedSkillCount} Workspace Skill${selectedSkillCount === 1 ? '' : 's'} selected`,
								state.attentionCount > 0
									? `${state.attentionCount} item${state.attentionCount === 1 ? '' : 's'} need attention`
									: 'All selected items currently resolve',
							]
						: ['No Workspace is attached.'],
				},
				{
					id: 'behavior',
					title: 'Behavior',
					items: [
						'Workspace Context is refetched when sending or editing.',
						'Workspace Skills use the normal conversation Skill session.',
						'Detaching affects future turns only.',
					],
				},
			]}
		/>
	);

	return (
		<div className="relative shrink-0" data-bottom-bar-workspace>
			<HoverTip content={hoverContent} placement="top" tooltipClassName="max-w-sm">
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
							icon={<FiBriefcase size={14} />}
							label={selected ? selectedName : 'Workspace'}
							count={
								selectedContextCount > 0 ? (
									<span className="badge badge-secondary badge-xs">Context {selectedContextCount}</span>
								) : undefined
							}
							suffix={
								<span className="flex items-center gap-1">
									{selectedSkillCount > 0 ? (
										<span className="badge badge-info badge-xs">Skills {selectedSkillCount}</span>
									) : null}
									{state.attentionCount > 0 ? (
										<FiAlertCircle size={13} className="text-warning" />
									) : selected && selectedSkillCount === 0 ? (
										<FiCheck size={13} />
									) : null}
								</span>
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
								void state.detachWorkspace();
								store.hide();
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
				className={actionTriggerMenuWideClasses}
				autoFocusOnShow={false}
			>
				<div className="mb-2 flex items-center justify-between gap-2 px-1">
					<div className="text-base-content/70 text-xs font-semibold">Workspace</div>
					<button
						type="button"
						className="btn btn-ghost btn-xs rounded-lg"
						disabled={state.workspacesLoading}
						onClick={() => {
							void state.refreshWorkspaces();
						}}
						title="Refresh Workspace list"
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
							{selectedContextCount} Context · {selectedSkillCount} Skills
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
									void state.detachWorkspace().finally(() => {
										store.hide();
									});
								}}
							>
								<FiX size={12} />
								Detach
							</button>
						</div>
					</div>
				) : null}

				<label className="input input-sm mb-2 flex items-center gap-2 rounded-xl">
					<FiSearch size={13} />
					<input
						type="search"
						className="grow"
						value={search}
						onChange={event => {
							setSearch(event.currentTarget.value);
						}}
						placeholder="Search Workspaces..."
						spellCheck="false"
					/>
				</label>

				{state.workspacesLoading && state.workspaces.length === 0 ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
						Loading Workspaces...
					</div>
				) : visibleWorkspaces.length === 0 ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
						No Workspaces match.
					</div>
				) : (
					<div className="space-y-1">
						{visibleWorkspaces.map(workspace => {
							const isCurrent = workspace.rootID === state.selection?.rootID;
							return (
								<MenuItem
									key={workspace.rootID}
									hideOnClick={false}
									disabled={!workspace.enabled || isInputLocked}
									className={`${actionTriggerMenuItemClasses} items-start`}
									onClick={() => {
										if (isCurrent || !workspace.enabled) {
											return;
										}
										setActionError(null);
										void state
											.attachWorkspace(workspace)
											.then(() => {
												store.hide();
											})
											.catch((error: unknown) => {
												setActionError(error instanceof Error ? error.message : 'Workspace could not be selected.');
											});
									}}
								>
									<FiBriefcase size={14} className="mt-0.5 shrink-0" />
									<div className="min-w-0 flex-1">
										<div className="truncate text-xs font-medium">{workspace.displayName}</div>
										{workspace.primaryPath ? (
											<div className="text-base-content/60 truncate font-mono text-[10px]">{workspace.primaryPath}</div>
										) : null}
									</div>
									{isCurrent ? (
										<span className="badge badge-secondary badge-xs">Selected</span>
									) : !workspace.enabled ? (
										<span className="badge badge-warning badge-xs">Disabled</span>
									) : null}
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
							setIsCreateOpen(true);
						}}
					>
						<FiFolderPlus size={14} />
						Add Workspace from folder or path
					</button>
				</div>
			</Menu>

			<WorkspaceSetupModal
				isOpen={isCreateOpen}
				onClose={() => {
					setIsCreateOpen(false);
				}}
				onSubmit={async (submission: WorkspaceSetupSubmission) => {
					if (submission.kind !== 'filesystem') {
						throw new Error('Expected a filesystem Workspace.');
					}
					await state.createFilesystemWorkspace(submission.payload);
				}}
				existingDisplayNames={state.workspaces.map(workspace => workspace.displayName)}
				presentation="composer"
			/>

			<WorkspaceSelectionModal
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
