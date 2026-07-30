import type { SubmitEventHandler } from 'react';
import { useMemo, useState } from 'react';

import { FiAlertCircle, FiMapPin, FiPlus } from 'react-icons/fi';

import type { ArtifactSourceBinding } from '@/spec/artifact';
import type { WorkspaceAttachmentView, WorkspaceView } from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { workspaceAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { workspaceRefKey } from '@/workspaces/lib/workspace_api_utils';
import {
	getErrorMessage,
	WORKSPACE_CONTEXT_ARTIFACT_KIND,
	WORKSPACE_SKILL_ARTIFACT_KIND,
} from '@/workspaces/lib/workspace_utils';

type BindingAction = 'pin' | 'suppress';

const ARTIFACT_KIND_OPTIONS = [
	{ value: WORKSPACE_CONTEXT_ARTIFACT_KIND, label: 'Context Artifact' },
	{ value: WORKSPACE_SKILL_ARTIFACT_KIND, label: 'Workspace Skill Artifact' },
] as const;

function sourceLabel(attachment: WorkspaceAttachmentView): string {
	return attachment.path ?? attachment.sourceDisplayName ?? attachment.sourceKind ?? attachment.sourceID;
}

function defaultArtifactName(locator: string): string {
	const normalized = locator.trim().replaceAll('\\', '/').replaceAll(/\/+$/g, '');
	const basename = normalized.slice(normalized.lastIndexOf('/') + 1);
	return basename || 'Pinned Workspace Artifact';
}

function isSafeRelativeLocator(locator: string): boolean {
	const normalized = locator.trim().replaceAll('\\', '/');

	if (!normalized || normalized.startsWith('/') || /^[A-Za-z]:\//.test(normalized)) {
		return false;
	}

	return !normalized.split('/').includes('..');
}

export interface WorkspaceArtifactBindingModalProps {
	isOpen: boolean;
	onClose: () => void;
	workspace: WorkspaceView;
	onChanged: () => Promise<void>;
}

function WorkspaceArtifactBindingModalContent({
	workspace,
	onChanged,
}: Omit<WorkspaceArtifactBindingModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose, unmountingRef } = useModalDialogController();
	const [action, setAction] = useState<BindingAction>('pin');
	const [sourceID, setSourceID] = useState(
		workspace.attachments.find(attachment => attachment.enabled)?.sourceID ?? workspace.attachments[0]?.sourceID ?? ''
	);
	const [locator, setLocator] = useState('');
	const [subresourceLocator, setSubresourceLocator] = useState('');
	const [kind, setKind] = useState<string>(WORKSPACE_CONTEXT_ARTIFACT_KIND);
	const [name, setName] = useState('');
	const [enabled, setEnabled] = useState(true);
	const [runtimeDisabled, setRuntimeDisabled] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const selectedAttachment = useMemo(
		() => workspace.attachments.find(attachment => attachment.sourceID === sourceID),
		[sourceID, workspace.attachments]
	);
	const hasAttachments = workspace.attachments.length > 0;

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (isSubmitting) {
			return;
		}

		const normalizedLocator = locator.trim().replaceAll('\\', '/');
		if (!sourceID) {
			setSubmitError('Select an attached Workspace Source.');
			return;
		}
		if (!isSafeRelativeLocator(normalizedLocator)) {
			setSubmitError('Locator must be a non-empty Source-relative path without parent traversal.');
			return;
		}
		if (!kind.trim()) {
			setSubmitError('Select an expected Artifact kind.');
			return;
		}

		const binding: ArtifactSourceBinding = {
			sourceID,
			locator: normalizedLocator,
			...(subresourceLocator.trim() ? { subresourceLocator: subresourceLocator.trim() } : {}),
			expectedKind: kind,
		};

		setSubmitError('');
		setIsSubmitting(true);

		void (async () => {
			try {
				if (action === 'pin') {
					await workspaceAPI.pinWorkspaceArtifact(workspace.workspace, {
						expectedCollectionRevision: workspace.revision,
						binding,
						name: name.trim() || defaultArtifactName(normalizedLocator),
						enabled,
						settings: {
							runtimeDisabled,
						},
					});
				} else {
					await workspaceAPI.suppressWorkspaceBinding(workspace.workspace, {
						expectedCollectionRevision: workspace.revision,
						binding,
					});
				}

				try {
					await onChanged();
				} catch (reloadError) {
					console.error('Workspace binding changed but catalog reload failed:', reloadError);
				}

				if (!unmountingRef.current) {
					requestClose(true);
				}
			} catch (error) {
				if (!unmountingRef.current) {
					setSubmitError(
						getErrorMessage(
							error,
							action === 'pin'
								? 'The Workspace Artifact could not be pinned.'
								: 'The Workspace Source binding could not be suppressed.'
						)
					);
				}
			} finally {
				if (!unmountingRef.current) {
					setIsSubmitting(false);
				}
			}
		})();
	};

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-2xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={action === 'pin' ? 'Pin Workspace Artifact' : 'Suppress Workspace Binding'}
				description={
					action === 'pin'
						? 'Pin a typed Source binding before content exists, so it remains a durable Workspace Artifact when the source becomes available.'
						: 'Prevent automatic adoption for one typed Source binding. Suppression does not alter source files.'
				}
				onClose={requestClose}
				closeDisabled={isSubmitting}
			/>

			<form
				noValidate
				onSubmit={handleSubmit}
				className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6"
				aria-busy={isSubmitting}
			>
				{submitError ? (
					<div className="alert alert-error rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={14} />
						<span>{submitError}</span>
					</div>
				) : null}

				<ModalSection title="Binding action">
					<div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<label className="border-base-content/10 bg-base-100 flex cursor-pointer items-start gap-3 rounded-2xl border p-3">
							<input
								type="radio"
								className="radio radio-sm mt-0.5"
								checked={action === 'pin'}
								disabled={isSubmitting}
								onChange={() => {
									setAction('pin');
									setSubmitError('');
								}}
							/>
							<FiPlus size={15} className="mt-0.5" />
							<span>
								<span className="block font-medium">Pin Artifact</span>
								<span className="text-base-content/70 mt-1 block text-xs">
									Create a durable Artifact record even when the source item is currently missing.
								</span>
							</span>
						</label>

						<label className="border-base-content/10 bg-base-100 flex cursor-pointer items-start gap-3 rounded-2xl border p-3">
							<input
								type="radio"
								className="radio radio-sm mt-0.5"
								checked={action === 'suppress'}
								disabled={isSubmitting}
								onChange={() => {
									setAction('suppress');
									setSubmitError('');
								}}
							/>

							<span className="block font-medium">Suppress Binding</span>
							<span className="text-base-content/70 mt-1 block text-xs">
								Keep a matching valid observation from being automatically adopted during refresh.
							</span>
						</label>
					</div>
				</ModalSection>

				<ModalSection title="Typed Source binding">
					{!hasAttachments ? (
						<div className="alert alert-warning rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>Attach a Workspace Source before pinning or suppressing a Source binding.</span>
						</div>
					) : null}

					<ModalField label="Attached Source" htmlFor="workspace-binding-source" required>
						<select
							id="workspace-binding-source"
							className="select w-full rounded-xl"
							value={sourceID}
							disabled={isSubmitting || !hasAttachments}
							onChange={event => {
								setSourceID(event.currentTarget.value);
							}}
						>
							<option value="">Select a Source</option>
							{workspace.attachments.map(attachment => (
								<option key={attachment.sourceID} value={attachment.sourceID}>
									{sourceLabel(attachment)}
									{attachment.enabled ? '' : ' [attachment disabled]'}
								</option>
							))}
						</select>
					</ModalField>

					<ModalField
						label="Source-relative locator"
						htmlFor="workspace-binding-locator"
						required
						hint="Use a slash-separated path inside the selected Source. Absolute paths and parent traversal are rejected."
					>
						<div className="relative">
							<FiMapPin size={14} className="text-base-content/50 absolute top-3 left-3" />
							<input
								id="workspace-binding-locator"
								type="text"
								className="input w-full rounded-xl pl-9 font-mono text-sm"
								value={locator}
								disabled={isSubmitting || !hasAttachments}
								onChange={event => {
									setLocator(event.currentTarget.value);
								}}
								placeholder="docs/project-context.md or skills/example/SKILL.md"
								spellCheck="false"
								autoComplete="off"
							/>
						</div>
					</ModalField>

					<ModalField
						label="Subresource locator"
						htmlFor="workspace-binding-subresource"
						hint="Optional. Use this only when one Source file emits multiple Artifact definitions."
					>
						<input
							id="workspace-binding-subresource"
							type="text"
							className="input w-full rounded-xl font-mono text-sm"
							value={subresourceLocator}
							disabled={isSubmitting || !hasAttachments}
							onChange={event => {
								setSubresourceLocator(event.currentTarget.value);
							}}
							spellCheck="false"
							autoComplete="off"
						/>
					</ModalField>

					<ModalField label="Expected Artifact kind" htmlFor="workspace-binding-kind" required>
						<select
							id="workspace-binding-kind"
							className="select w-full rounded-xl"
							value={kind}
							disabled={isSubmitting || !hasAttachments}
							onChange={event => {
								setKind(event.currentTarget.value);
							}}
						>
							{ARTIFACT_KIND_OPTIONS.map(option => (
								<option key={option.value} value={option.value}>
									{option.label} ({option.value})
								</option>
							))}
						</select>
					</ModalField>

					{selectedAttachment ? (
						<div className="text-base-content/60 text-xs">
							Binding Source ID: <span className="font-mono">{selectedAttachment.sourceID}</span>
						</div>
					) : null}
				</ModalSection>

				{action === 'pin' ? (
					<ModalSection title="Pinned Artifact settings">
						<ModalField
							label="Artifact name"
							htmlFor="workspace-binding-name"
							hint="When empty, the final locator path segment becomes the Artifact name."
						>
							<input
								id="workspace-binding-name"
								type="text"
								className="input w-full rounded-xl"
								value={name}
								disabled={isSubmitting || !hasAttachments}
								onChange={event => {
									setName(event.currentTarget.value);
								}}
								placeholder={defaultArtifactName(locator)}
								autoComplete="off"
							/>
						</ModalField>

						<div className="flex flex-wrap gap-5">
							<label className="flex items-center gap-3 text-sm">
								<input
									type="checkbox"
									className="toggle toggle-accent toggle-sm"
									checked={enabled}
									disabled={isSubmitting || !hasAttachments}
									onChange={event => {
										setEnabled(event.currentTarget.checked);
									}}
								/>
								Enable Artifact
							</label>
							<label className="flex items-center gap-3 text-sm">
								<input
									type="checkbox"
									className="toggle toggle-accent toggle-sm"
									checked={!runtimeDisabled}
									disabled={isSubmitting || !hasAttachments}
									onChange={event => {
										setRuntimeDisabled(!event.currentTarget.checked);
									}}
								/>
								Use in conversations when supported
							</label>
						</div>
					</ModalSection>
				) : null}

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
					<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting || !hasAttachments}>
						{isSubmitting ? (
							<>
								<span className="loading loading-spinner loading-xs" />
								Saving...
							</>
						) : action === 'pin' ? (
							'Pin Artifact'
						) : (
							'Suppress Binding'
						)}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function WorkspaceArtifactBindingModal(props: WorkspaceArtifactBindingModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<WorkspaceArtifactBindingModalContent
				key={`${workspaceRefKey(props.workspace.workspace)}:${props.workspace.revision}`}
				workspace={props.workspace}
				onChanged={props.onChanged}
			/>
		</ModalDialog>
	);
}
