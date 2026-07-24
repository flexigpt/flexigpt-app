import type { Dispatch, SetStateAction } from 'react';

import { FiAlertCircle, FiCheck, FiFileText, FiRefreshCw, FiZap } from 'react-icons/fi';

import type { SkillRef } from '@/spec/skill';
import { WorkspaceRecordState, WorkspaceSkillInsert } from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import type { ComposerWorkspaceController } from '@/chats/composer/workspaces/use_composer_workspace';
import { isWorkspaceSkillSessionEligible } from '@/chats/composer/workspaces/use_composer_workspace';
import { createWorkspaceSkillRef, isWorkspaceSkillRef, skillRefKey } from '@/skills/lib/skill_identity_utils';

interface WorkspaceSelectionModalProps {
	isOpen: boolean;
	onClose: () => void;
	state: ComposerWorkspaceController;
	activeSkillRefs: SkillRef[];
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	isInputLocked?: boolean;
}

const diagnosticTitle = (diagnostics?: Array<{ code: string; message: string }>) =>
	(diagnostics ?? []).map(diagnostic => `${diagnostic.code}: ${diagnostic.message}`).join('\n');

function WorkspaceSelectionModalContent({
	state,
	activeSkillRefs,
	setActiveSkillRefs,
	isInputLocked = false,
}: Omit<WorkspaceSelectionModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose } = useModalDialogController();

	const activeKeys = new Set(
		activeSkillRefs
			.filter(r => {
				return isWorkspaceSkillRef(r);
			})
			.map(r => {
				return skillRefKey(r);
			})
	);

	const workspaceName = state.workspace?.displayName ?? state.selection?.displayName ?? 'Unavailable Workspace';

	const selectedCatalogState = state.catalogKnown
		? 'Current catalog state loaded.'
		: 'Catalog state could not be fully loaded. Stored selections are preserved without being marked missing.';

	return (
		<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title="Edit Workspace for This Conversation"
				description={`Choose which Context files and Skills from ${workspaceName} are available to future turns. This does not modify the configured Workspace.`}
				onClose={requestClose}
			/>

			<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
				{state.selectionError ? (
					<div className="alert alert-warning rounded-2xl text-sm">
						<FiAlertCircle size={14} />
						<span>{state.selectionError}</span>
					</div>
				) : null}

				{!state.catalogKnown ? (
					<div className="alert alert-warning rounded-2xl text-sm">
						<FiAlertCircle size={14} />
						<span>{selectedCatalogState}</span>
					</div>
				) : null}

				{state.changedCount > 0 ? (
					<div className="alert alert-info rounded-2xl text-sm">
						<FiRefreshCw size={14} />
						<span>
							{state.changedCount} selected resource
							{state.changedCount === 1 ? ' has' : 's have'} changed. The current content will be loaded when the
							message is sent.
						</span>
					</div>
				) : null}

				<ModalSection
					title="Context files"
					description="Selected files are composed by the backend for each send. Unselecting a file affects future turns only."
				>
					<div className="space-y-2">
						{state.contexts.map(context => {
							const selected = state.selectedContextIDs.has(context.recordID);
							const usable =
								context.enabled &&
								context.state === WorkspaceRecordState.Available &&
								context.catalogCurrent &&
								!context.runtimeDisabled;

							return (
								<label
									key={context.recordID}
									className={`border-base-content/10 bg-base-100 flex cursor-pointer items-start gap-3 rounded-2xl border p-3 ${
										selected ? 'border-secondary/50 bg-secondary/10' : ''
									}`}
								>
									<input
										type="checkbox"
										className="checkbox checkbox-sm mt-0.5"
										checked={selected}
										disabled={isInputLocked || (!selected && !usable)}
										onChange={event => {
											state.toggleContext(context, event.currentTarget.checked);
										}}
									/>

									<FiFileText size={15} className="mt-0.5 shrink-0" />

									<span className="min-w-0 flex-1">
										<span className="block truncate text-sm font-medium">{context.name}</span>
										<span className="text-base-content/60 block truncate font-mono text-xs">{context.locator}</span>
									</span>

									<span className="flex shrink-0 flex-wrap justify-end gap-1">
										{selected ? (
											<span className="badge badge-secondary badge-xs">
												<FiCheck size={10} />
												Selected
											</span>
										) : null}
										<span className={`badge badge-xs ${usable ? 'badge-success' : 'badge-warning'}`}>
											{usable ? 'Available' : context.runtimeDisabled ? 'Runtime disabled' : context.state}
										</span>
									</span>

									{context.diagnostics?.length ? (
										<span className="sr-only">{diagnosticTitle(context.diagnostics)}</span>
									) : null}
								</label>
							);
						})}

						{state.contexts.length === 0 ? (
							<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
								No current Context records were found.
							</div>
						) : null}

						{state.missingContextRefs.map(ref => (
							<div
								key={ref.recordID}
								className="border-warning/40 bg-warning/10 flex items-start gap-3 rounded-2xl border p-3"
							>
								<FiAlertCircle size={15} className="text-warning mt-0.5" />
								<div className="min-w-0 flex-1">
									<div className="text-sm font-medium">{ref.name || ref.recordID}</div>
									<div className="text-base-content/60 font-mono text-xs break-all">{ref.locator || ref.recordID}</div>
								</div>
								<span className="badge badge-warning badge-xs">Missing</span>
								<button
									type="button"
									className="btn btn-ghost btn-xs text-error rounded-lg"
									disabled={isInputLocked}
									onClick={() => {
										state.removeContextRef(ref.recordID);
									}}
								>
									Remove
								</button>
							</div>
						))}
					</div>
				</ModalSection>

				<ModalSection
					title="Workspace Skills"
					description="Selected Skills enter the normal conversation Skill session. Selection does not execute them."
				>
					<div className="space-y-2">
						{state.skills.map(skill => {
							const selected = state.selectedSkillIDs.has(skill.recordID);
							const ref = createWorkspaceSkillRef(skill.rootID, skill.recordID);
							if (!ref) {
								return null;
							}
							const active = activeKeys.has(skillRefKey(ref));
							const provider = state.workspaceSkillProvidersByRecordID.get(skill.recordID);
							const instruction = skill.skill.insert === WorkspaceSkillInsert.Instructions;
							const usable = isWorkspaceSkillSessionEligible(skill, provider);
							const canActivate = instruction && (skill.skill.arguments?.length ?? 0) === 0;
							const unavailableLabel = !instruction
								? 'Template'
								: !provider
									? 'Runtime unavailable'
									: provider.shadowed
										? 'Shadowed'
										: !provider.runtimeAllowed
											? 'Runtime denied'
											: skill.runtimeDisabled
												? 'Runtime disabled'
												: skill.state;

							return (
								<div
									key={skill.recordID}
									className={`border-base-content/10 bg-base-100 rounded-2xl border p-3 ${
										selected ? 'border-secondary/50 bg-secondary/10' : ''
									}`}
								>
									<div className="flex items-start gap-3">
										<label className="flex min-w-0 flex-1 cursor-pointer items-start gap-3">
											<input
												type="checkbox"
												className="checkbox checkbox-sm mt-0.5"
												checked={selected}
												disabled={isInputLocked || (!selected && !usable)}
												onChange={event => {
													void state.toggleSkill(skill, event.currentTarget.checked);
												}}
											/>
											<FiZap size={15} className="mt-0.5 shrink-0" />
											<span className="min-w-0 flex-1">
												<span className="block truncate text-sm font-medium">
													{skill.skill.displayName || skill.skill.name}
												</span>
												<span className="text-base-content/60 block truncate font-mono text-xs">{skill.locator}</span>
											</span>
										</label>

										<span className="flex shrink-0 flex-wrap justify-end gap-1">
											<span className="badge badge-ghost badge-xs">{skill.skill.insert}</span>
											<span className={`badge badge-xs ${usable ? 'badge-success' : 'badge-warning'}`}>
												{usable ? 'Available' : unavailableLabel}
											</span>
											{provider?.diagnostics?.length ? (
												<span className="badge badge-warning badge-xs" title={diagnosticTitle(provider.diagnostics)}>
													Details
												</span>
											) : null}
										</span>
									</div>

									{selected && instruction ? (
										<div className="mt-3 flex justify-end">
											<label
												className="flex items-center gap-2 text-xs"
												title={
													!usable
														? `${unavailableLabel}. Resolve this Workspace Skill before activating it.`
														: canActivate
															? 'Load this instruction Skill as active session context.'
															: 'Argument-backed Skills cannot be preloaded as active.'
												}
											>
												<span>Active</span>
												<input
													type="checkbox"
													className="toggle toggle-accent toggle-sm"
													checked={active}
													disabled={isInputLocked || !usable || !canActivate}
													onChange={event => {
														const checked = event.currentTarget.checked;
														setActiveSkillRefs(previous => {
															const byKey = new Map(previous.map(item => [skillRefKey(item), item]));
															if (checked) {
																byKey.set(skillRefKey(ref), ref);
															} else {
																byKey.delete(skillRefKey(ref));
															}
															return [...byKey.values()];
														});
													}}
												/>
											</label>
										</div>
									) : null}
								</div>
							);
						})}

						{state.skills.length === 0 ? (
							<div className="text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
								No current Workspace Skills were found.
							</div>
						) : null}

						{state.missingSkillRefs.map(ref => (
							<div
								key={ref.recordID}
								className="border-warning/40 bg-warning/10 flex items-start gap-3 rounded-2xl border p-3"
							>
								<FiAlertCircle size={15} className="text-warning mt-0.5" />
								<div className="min-w-0 flex-1">
									<div className="text-sm font-medium">{ref.displayName || ref.name || ref.recordID}</div>
									<div className="text-base-content/60 font-mono text-xs break-all">{ref.locator || ref.identity}</div>
								</div>
								<span className="badge badge-warning badge-xs">Missing</span>
								<button
									type="button"
									className="btn btn-ghost btn-xs text-error rounded-lg"
									disabled={isInputLocked}
									onClick={() => {
										void state.removeSkillRef(ref.recordID);
									}}
								>
									Remove
								</button>
							</div>
						))}
					</div>
				</ModalSection>
			</div>

			<ModalActions>
				<button
					type="button"
					className="btn btn-ghost rounded-xl"
					disabled={isInputLocked || state.selectionLoading}
					onClick={() => {
						void state.refreshSelectedWorkspace();
					}}
				>
					<FiRefreshCw size={14} />
					Refresh status
				</button>
				<button
					type="button"
					className="btn btn-ghost rounded-xl"
					disabled={isInputLocked || state.selectionLoading || !state.workspace}
					onClick={() => {
						void state.updateSelectionFromCurrentContents();
					}}
				>
					Use current contents
				</button>
				<button
					type="button"
					className="btn btn-primary rounded-xl"
					onClick={() => {
						requestClose();
					}}
				>
					Done
				</button>
			</ModalActions>
		</div>
	);
}

export function WorkspaceSelectionModal(props: WorkspaceSelectionModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<WorkspaceSelectionModalContent {...props} />
		</ModalDialog>
	);
}
