import type { SubmitEventHandler } from 'react';
import { useId, useMemo, useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type {
	AttachWorkspaceSourcePayload,
	UpdateWorkspaceAttachmentPayload,
	WorkspaceAttachmentView,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceAttachmentRole } from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { getErrorMessage } from '@/workspaces/lib/workspace_utils';

interface WorkspaceAttachmentModalProps {
	isOpen: boolean;
	onClose: () => void;
	workspace: WorkspaceView;
	attachment?: WorkspaceAttachmentView;
	onAttach: (sourceID: string, payload: AttachWorkspaceSourcePayload) => Promise<void>;
	onUpdate: (sourceID: string, payload: UpdateWorkspaceAttachmentPayload) => Promise<void>;
}

interface AttachmentForm {
	sourceID: string;
	role: WorkspaceAttachmentRole;
	priority: string;
	enabled: boolean;
	recursive: boolean;
	authoritative: boolean;
}

const ATTACHABLE_ROLES = [
	WorkspaceAttachmentRole.Library,
	WorkspaceAttachmentRole.AttachedPackage,
	WorkspaceAttachmentRole.Overlay,
] as const;

function WorkspaceAttachmentModalContent({
	workspace,
	attachment,
	onAttach,
	onUpdate,
}: Omit<WorkspaceAttachmentModalProps, 'isOpen' | 'onClose'>) {
	const [form, setForm] = useState<AttachmentForm>({
		sourceID: attachment?.sourceID ?? '',
		role: attachment?.role ?? WorkspaceAttachmentRole.Library,
		priority: String(attachment?.priority ?? 100),
		enabled: attachment?.enabled ?? true,
		recursive: attachment?.settings.recursive ?? true,
		authoritative: attachment?.settings.authoritative ?? false,
	});
	const [submitted, setSubmitted] = useState(false);
	const [isSubmitting, setIsSubmitting] = useState(false);
	const [submitError, setSubmitError] = useState('');

	const sourceIDInput = useId();
	const priorityInput = useId();
	const { requestClose, unmountingRef } = useModalDialogController();

	const errors = useMemo(() => {
		const next: { sourceID?: string; priority?: string } = {};
		const priority = Number(form.priority);

		if (!attachment && !form.sourceID.trim()) {
			next.sourceID = 'Source ID is required.';
		} else if (!attachment && workspace.attachments.some(existing => existing.sourceID === form.sourceID.trim())) {
			next.sourceID = 'This source is already attached.';
		}

		if (!Number.isSafeInteger(priority)) {
			next.priority = 'Priority must be a whole number.';
		}

		return next;
	}, [attachment, form.priority, form.sourceID, workspace.attachments]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		event.stopPropagation();

		if (isSubmitting) {
			return;
		}

		setSubmitted(true);
		setSubmitError('');

		if (Object.keys(errors).length > 0) {
			return;
		}

		const common = {
			role: form.role,
			priority: Number(form.priority),
			enabled: form.enabled,
			settings: {
				recursive: form.recursive,
				authoritative: form.authoritative,
			},
		};

		setIsSubmitting(true);

		try {
			if (attachment) {
				await onUpdate(attachment.sourceID, {
					...common,
					expectedRootRevision: workspace.revision,
					expectedAttachmentRevision: attachment.revision,
				});
			} else {
				await onAttach(form.sourceID.trim(), {
					...common,
					expectedRootRevision: workspace.revision,
				});
			}

			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(getErrorMessage(error, 'Failed to save attached source.'));
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	const roleOptions =
		attachment?.role === WorkspaceAttachmentRole.Primary ? [WorkspaceAttachmentRole.Primary] : ATTACHABLE_ROLES;

	return (
		<div className="modal-box bg-base-200 flex max-h-[85vh] w-[calc(100%-1rem)] max-w-3xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={attachment ? 'Edit Attached Source' : 'Attach Existing Source'}
				description={
					attachment
						? 'Update source precedence and discovery settings.'
						: 'Attach a source already registered by the backend.'
				}
				onClose={() => {
					requestClose();
				}}
				closeDisabled={isSubmitting}
			/>

			<form
				noValidate
				onSubmit={handleSubmit}
				className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6"
			>
				{submitError ? (
					<div className="alert alert-error rounded-2xl text-sm">
						<FiAlertCircle size={14} />
						<span>{submitError}</span>
					</div>
				) : null}

				<div className="alert alert-info rounded-2xl text-sm">
					This API attaches a registered source by source ID. It does not create a source from a filesystem path. Use
					workspace discovery paths for ordinary context files and skill folders.
				</div>

				<ModalSection title="Source">
					<ModalField
						label="Source ID"
						htmlFor={sourceIDInput}
						required
						error={submitted ? errors.sourceID : undefined}
					>
						<input
							id={sourceIDInput}
							type="text"
							className={`input w-full rounded-xl ${submitted && errors.sourceID ? 'input-error' : ''}`}
							value={form.sourceID}
							onChange={event => {
								setForm(previous => ({ ...previous, sourceID: event.currentTarget.value }));
							}}
							readOnly={Boolean(attachment)}
							spellCheck="false"
							autoComplete="off"
						/>
					</ModalField>

					<ModalField label="Role">
						<select
							className="select w-full rounded-xl"
							value={form.role}
							onChange={event => {
								setForm(previous => ({
									...previous,
									role: event.currentTarget.value as WorkspaceAttachmentRole,
								}));
							}}
							disabled={attachment?.role === WorkspaceAttachmentRole.Primary || isSubmitting}
						>
							{roleOptions.map(role => (
								<option key={role} value={role}>
									{role}
								</option>
							))}
						</select>
					</ModalField>

					<ModalField
						label="Priority"
						htmlFor={priorityInput}
						required
						hint="Lower or higher precedence is interpreted by the backend workspace policy."
						error={submitted ? errors.priority : undefined}
					>
						<input
							id={priorityInput}
							type="number"
							step={1}
							className={`input w-full rounded-xl ${submitted && errors.priority ? 'input-error' : ''}`}
							value={form.priority}
							onChange={event => {
								setForm(previous => ({ ...previous, priority: event.currentTarget.value }));
							}}
							disabled={isSubmitting}
						/>
					</ModalField>
				</ModalSection>

				<ModalSection title="Behavior">
					<ModalField label="Enabled">
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={form.enabled}
							onChange={event => {
								setForm(previous => ({ ...previous, enabled: event.currentTarget.checked }));
							}}
							disabled={isSubmitting}
						/>
					</ModalField>

					<ModalField label="Recursive">
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={form.recursive}
							onChange={event => {
								setForm(previous => ({ ...previous, recursive: event.currentTarget.checked }));
							}}
							disabled={isSubmitting}
						/>
					</ModalField>

					<ModalField
						label="Authoritative"
						hint="An authoritative source may take precedence according to backend workspace rules."
					>
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={form.authoritative}
							onChange={event => {
								setForm(previous => ({ ...previous, authoritative: event.currentTarget.checked }));
							}}
							disabled={isSubmitting}
						/>
					</ModalField>
				</ModalSection>

				<ModalActions className="-mx-4 -mb-4 sm:-mx-6 sm:-mb-6">
					<button
						type="button"
						className="btn bg-base-300 rounded-xl"
						onClick={() => {
							requestClose();
						}}
						disabled={isSubmitting}
					>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting}>
						{isSubmitting ? 'Saving...' : attachment ? 'Save Changes' : 'Attach Source'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function WorkspaceAttachmentModal(props: WorkspaceAttachmentModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<WorkspaceAttachmentModalContent
				key={
					props.attachment
						? `${props.attachment.sourceID}:${props.attachment.revision}`
						: `${props.workspace.rootID}:new-source`
				}
				{...props}
			/>
		</ModalDialog>
	);
}
