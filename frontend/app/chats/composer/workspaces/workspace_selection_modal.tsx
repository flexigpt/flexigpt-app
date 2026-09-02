import type { Dispatch, SetStateAction, SubmitEventHandler } from 'react';
import { useId, useState } from 'react';

import { FiAlertCircle, FiCheck, FiFileText, FiRefreshCw, FiZap } from 'react-icons/fi';

import { ArtifactState } from '@/spec/artifact';
import type { SkillRef } from '@/spec/skill';
import type { WorkspaceRef, WorkspaceSkillView } from '@/spec/workspace';
import { WorkspaceSkillInsert } from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { skillManagementAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import type { ComposerWorkspaceController } from '@/chats/composer/workspaces/use_composer_workspace';
import { isWorkspaceSkillConversationAvailable } from '@/chats/composer/workspaces/use_composer_workspace';
import { skillRefKey } from '@/skills/lib/skill_identity_utils';
import { artifactRefKey } from '@/workspaces/lib/workspace_api_utils';

interface WorkspaceSelectionModalProps {
	isOpen: boolean;
	onClose: () => void;
	state: ComposerWorkspaceController;
	activeSkillRefs: SkillRef[];
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	isInputLocked?: boolean;
	onInsertTemplateText: (text: string) => Promise<void> | void;
}

const diagnosticTitle = (diagnostics?: Array<{ code: string; message: string }>) =>
	(diagnostics ?? []).map(diagnostic => `${diagnostic.code}: ${diagnostic.message}`).join('\n');

interface WorkspaceTemplateTarget {
	workspace: WorkspaceRef;
	skill: WorkspaceSkillView;
}

interface WorkspaceTemplateRenderModalProps {
	target: WorkspaceTemplateTarget | null;
	onClose: () => void;
	onInsert: (text: string) => Promise<void> | void;
	onInserted: () => void;
}

function workspaceSkillUnavailableLabel(skill: WorkspaceSkillView): string {
	if (!skill.skill.isEnabled) {
		return 'Disabled';
	}
	if (skill.state !== ArtifactState.Available) {
		return skill.state;
	}
	if (!skill.projectionValid) {
		return 'Invalid';
	}
	if (!skill.catalogCurrent) {
		return 'Catalog stale';
	}
	if (skill.runtimeDisabled) {
		return 'Runtime disabled';
	}
	return 'Unavailable';
}

function WorkspaceTemplateRenderModalContent({
	target,
	onInsert,
	onInserted,
}: Omit<WorkspaceTemplateRenderModalProps, 'onClose'> & { target: WorkspaceTemplateTarget }) {
	const { requestClose, unmountingRef } = useModalDialogController();
	const fieldIDPrefix = useId();
	const argumentsList = target.skill.skill.arguments ?? [];
	const [argumentValues, setArgumentValues] = useState<Record<string, string>>(() =>
		Object.fromEntries(argumentsList.map(argument => [argument.name, argument.default ?? '']))
	);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

		if (isSubmitting) {
			return;
		}

		setSubmitError('');
		setIsSubmitting(true);

		try {
			const rendered = await skillManagementAPI.renderSkill(target.skill.artifact, argumentValues);

			if (rendered.insert !== WorkspaceSkillInsert.UserMessage) {
				throw new Error(
					`Expected a user-message template, but the renderer returned ${rendered.insert ?? 'no insert type'}.`
				);
			}

			const text = rendered.text ?? '';
			if (!text.trim()) {
				throw new Error('The Workspace template rendered an empty message.');
			}

			await onInsert(text);
			if (!unmountingRef.current) {
				requestClose(true);
				onInserted();
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(error instanceof Error && error.message.trim() ? error.message : 'Template rendering failed.');
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	return (
		<div className="modal-box bg-base-200 max-h-[80vh] w-[calc(100%-1rem)] max-w-2xl overflow-y-auto rounded-2xl p-0">
			<ModalHeader
				title="Use Workspace Template"
				description={`${target.skill.skill.displayName || target.skill.skill.name} renders into composer text and is not added to the installed Templates catalog.`}
				onClose={requestClose}
				closeDisabled={isSubmitting}
			/>

			<form className="space-y-4 p-4 sm:p-6" onSubmit={handleSubmit} aria-busy={isSubmitting}>
				{submitError ? (
					<div className="alert alert-error rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={14} aria-hidden="true" />
						<span>{submitError}</span>
					</div>
				) : null}

				{argumentsList.length > 0 ? (
					<ModalSection title="Template arguments">
						<div className="space-y-3">
							{argumentsList.map((argument, index) => {
								const inputID = `${fieldIDPrefix}-argument-${index}`;
								return (
									<ModalField key={argument.name} label={argument.name} htmlFor={inputID} hint={argument.description}>
										<input
											id={inputID}
											type="text"
											className="input w-full rounded-xl"
											value={argumentValues[argument.name] ?? ''}
											onChange={event => {
												const value = event.currentTarget.value;
												setArgumentValues(previous => ({
													...previous,
													[argument.name]: value,
												}));
											}}
											placeholder={argument.default}
											disabled={isSubmitting}
											autoComplete="off"
										/>
									</ModalField>
								);
							})}
						</div>
					</ModalSection>
				) : (
					<div className="text-base-content/70 text-sm">This Workspace template has no arguments.</div>
				)}

				<ModalActions className="-mx-4 -mb-4 sm:-mx-6 sm:-mb-6">
					<button
						type="button"
						className="btn bg-base-300 rounded-xl"
						disabled={isSubmitting}
						onClick={() => {
							requestClose();
						}}
					>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting}>
						{isSubmitting ? 'Rendering...' : 'Insert Template'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

function WorkspaceTemplateRenderModal(props: WorkspaceTemplateRenderModalProps) {
	if (!props.target) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<WorkspaceTemplateRenderModalContent
				key={`${artifactRefKey(props.target.skill.artifact)}:${props.target.skill.recordRevision}`}
				target={props.target}
				onInsert={props.onInsert}
				onInserted={props.onInserted}
			/>
		</ModalDialog>
	);
}

function WorkspaceSelectionModalContent({
	state,
	activeSkillRefs,
	setActiveSkillRefs,
	isInputLocked = false,
	onInsertTemplateText,
}: Omit<WorkspaceSelectionModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose } = useModalDialogController();
	const [templateTarget, setTemplateTarget] = useState<WorkspaceTemplateTarget | null>(null);

	const activeKeys = new Set(
		activeSkillRefs.map(r => {
			return skillRefKey(r);
		})
	);

	const workspaceName = state.workspace?.displayName ?? state.selection?.displayName ?? 'Unavailable Workspace';

	const selectedCatalogState = state.catalogKnown
		? 'Current catalog state loaded.'
		: 'Catalog state could not be fully loaded. Stored selections are preserved without being marked missing.';

	return (
		<>
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
								const contextKey = artifactRefKey(context.artifact);
								const selected = state.selectedContextIDs.has(contextKey);
								const usable =
									context.enabled &&
									context.state === ArtifactState.Available &&
									context.catalogCurrent &&
									context.projectionValid &&
									!context.runtimeDisabled;

								return (
									<label
										key={contextKey}
										className={`border-base-content/10 bg-base-100 flex cursor-pointer items-start gap-3 rounded-2xl border p-3 ${
											selected ? 'border-secondary/50 bg-secondary/10' : ''
										}`}
									>
										<input
											type="checkbox"
											className="checkbox checkbox-sm mt-0.5"
											checked={selected}
											disabled={isInputLocked || (!selected && !usable)}
											aria-label={`Include ${context.name} as Workspace Context`}
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
												{usable
													? 'Available'
													: !context.projectionValid
														? 'Invalid'
														: context.runtimeDisabled
															? 'Runtime disabled'
															: context.state}
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
									key={artifactRefKey(ref.artifact)}
									className="border-warning/40 bg-warning/10 flex items-start gap-3 rounded-2xl border p-3"
								>
									<FiAlertCircle size={15} className="text-warning mt-0.5" />
									<div className="min-w-0 flex-1">
										<div className="text-sm font-medium">{ref.name || ref.artifact.artifactID}</div>
										<div className="text-base-content/60 font-mono text-xs break-all">
											{ref.locator || ref.artifact.artifactID}
										</div>
									</div>
									<span className="badge badge-warning badge-xs">Missing</span>
									<button
										type="button"
										className="btn btn-ghost btn-xs text-error rounded-lg"
										disabled={isInputLocked}
										onClick={() => {
											state.removeContextRef(ref.artifact);
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
						description="Selected instruction Skills enter the normal conversation Skill session. User-message Skills render as transient composer templates and never appear in the installed Skills or Templates menus."
					>
						<div className="space-y-2">
							{state.skills.map(skill => {
								const skillKey = artifactRefKey(skill.artifact);
								const selected = state.selectedSkillIDs.has(skillKey);
								const active = activeKeys.has(skillRefKey(skill.artifact));
								const instruction = skill.skill.insert === WorkspaceSkillInsert.Instructions;
								const usable = isWorkspaceSkillConversationAvailable(skill);
								const canActivate = instruction && (skill.skill.arguments?.length ?? 0) === 0;
								const unavailableLabel = workspaceSkillUnavailableLabel(skill);
								const selectedWorkspaceRef = state.workspace?.workspace ?? state.selection?.workspace;

								return (
									<div
										key={skillKey}
										className={`border-base-content/10 bg-base-100 rounded-2xl border p-3 ${
											selected ? 'border-secondary/50 bg-secondary/10' : ''
										}`}
									>
										<div className="flex items-start gap-3">
											{instruction ? (
												<label className="flex min-w-0 flex-1 cursor-pointer items-start gap-3">
													<input
														type="checkbox"
														className="checkbox checkbox-sm mt-0.5"
														checked={selected}
														disabled={isInputLocked || (!selected && !usable)}
														aria-label={`Include ${skill.skill.displayName || skill.skill.name} as a Workspace instruction Skill`}
														onChange={event => {
															void state.toggleSkill(skill, event.currentTarget.checked);
														}}
													/>
													<FiZap size={15} className="mt-0.5 shrink-0" aria-hidden="true" />
													<span className="min-w-0 flex-1">
														<span className="block truncate text-sm font-medium">
															{skill.skill.displayName || skill.skill.name}
														</span>
														<span className="text-base-content/60 block truncate font-mono text-xs">
															{skill.locator}
														</span>
													</span>
												</label>
											) : (
												<div className="flex min-w-0 flex-1 items-start gap-3">
													<FiZap size={15} className="mt-0.5 shrink-0" />
													<span className="min-w-0 flex-1">
														<span className="block truncate text-sm font-medium">
															{skill.skill.displayName || skill.skill.name}
														</span>
														<span className="text-base-content/60 block truncate font-mono text-xs">
															{skill.locator}
														</span>
													</span>
												</div>
											)}

											<span className="flex shrink-0 flex-wrap justify-end gap-1">
												<span className="badge badge-ghost badge-xs">{skill.skill.insert}</span>
												<span className={`badge badge-xs ${usable ? 'badge-success' : 'badge-warning'}`}>
													{usable ? 'Available' : unavailableLabel}
												</span>
											</span>
										</div>

										{!instruction ? (
											<div className="mt-3 flex justify-end">
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={isInputLocked || !usable || !selectedWorkspaceRef}
													onClick={() => {
														if (!selectedWorkspaceRef) {
															return;
														}
														setTemplateTarget({
															workspace: selectedWorkspaceRef,
															skill,
														});
													}}
												>
													Use Template
												</button>
											</div>
										) : selected ? (
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
														aria-label={`Activate ${skill.skill.displayName || skill.skill.name} in the Skill session`}
														onChange={event => {
															const checked = event.currentTarget.checked;
															setActiveSkillRefs(previous => {
																const byKey = new Map(previous.map(item => [skillRefKey(item), item]));
																const providerKey = skillRefKey(skill.artifact);
																if (checked) {
																	byKey.set(providerKey, skill.artifact);
																} else {
																	byKey.delete(providerKey);
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
									key={artifactRefKey(ref.artifact)}
									className="border-warning/40 bg-warning/10 flex items-start gap-3 rounded-2xl border p-3"
								>
									<FiAlertCircle size={15} className="text-warning mt-0.5" />
									<div className="min-w-0 flex-1">
										<div className="text-sm font-medium">{ref.displayName || ref.name || ref.artifact.artifactID}</div>
										<div className="text-base-content/60 font-mono text-xs break-all">
											{ref.locator || ref.artifact.artifactID}
										</div>
									</div>
									<span className="badge badge-warning badge-xs">Missing</span>
									<button
										type="button"
										className="btn btn-ghost btn-xs text-error rounded-lg"
										disabled={isInputLocked}
										onClick={() => {
											void state.removeSkillRef(ref.artifact);
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

			<WorkspaceTemplateRenderModal
				target={templateTarget}
				onClose={() => {
					setTemplateTarget(null);
				}}
				onInsert={onInsertTemplateText}
				onInserted={() => {
					requestClose(true);
				}}
			/>
		</>
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
