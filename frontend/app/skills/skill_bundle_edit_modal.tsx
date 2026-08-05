import type { SubmitEventHandler } from 'react';
import { useState } from 'react';

import { FiAlertCircle } from 'react-icons/fi';

import type { SkillBundle } from '@/spec/skill';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

interface SkillBundleEditModalProps {
	isOpen: boolean;
	onClose: () => void;
	bundle: SkillBundle;
	onSubmit: (bundleID: string, displayName: string, description?: string) => Promise<void>;
}

function SkillBundleEditModalContent({ bundle, onSubmit }: Omit<SkillBundleEditModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose, unmountingRef } = useModalDialogController();
	const [displayName, setDisplayName] = useState(bundle.displayName || bundle.slug);
	const [description, setDescription] = useState(bundle.description || '');
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		const normalizedName = displayName.trim();
		if (!normalizedName) {
			setSubmitError('Display name is required.');
			return;
		}

		setSubmitError('');
		setIsSubmitting(true);
		void onSubmit(bundle.id, normalizedName, description.trim() || undefined)
			.then(() => {
				if (!unmountingRef.current) {
					requestClose(true);
				}
			})
			.catch((error: unknown) => {
				if (!unmountingRef.current) {
					setSubmitError(error instanceof Error ? error.message : 'Skill Bundle could not be updated.');
				}
			})
			.finally(() => {
				if (!unmountingRef.current) {
					setIsSubmitting(false);
				}
			});
	};

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-2xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title="Edit Skill Bundle"
				description="Update local Bundle presentation. Logical identity and Artifact Store ownership remain unchanged."
				onClose={requestClose}
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

				<ModalSection title="Bundle identity">
					<ModalField label="Logical name">
						<div className="input bg-base-300 flex items-center rounded-xl font-mono text-sm">{bundle.slug}</div>
					</ModalField>

					<ModalField label="Display name" htmlFor="skill-bundle-display-name" required>
						<input
							id="skill-bundle-display-name"
							type="text"
							className="input w-full rounded-xl"
							value={displayName}
							disabled={isSubmitting}
							onChange={event => {
								setDisplayName(event.currentTarget.value);
								setSubmitError('');
							}}
							maxLength={256}
						/>
					</ModalField>

					<ModalField label="Description" htmlFor="skill-bundle-description" align="start">
						<textarea
							id="skill-bundle-description"
							className="textarea min-h-28 w-full rounded-xl"
							value={description}
							disabled={isSubmitting}
							onChange={event => {
								setDescription(event.currentTarget.value);
							}}
						/>
					</ModalField>
				</ModalSection>

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
						{isSubmitting ? 'Saving...' : 'Save Changes'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function SkillBundleEditModal(props: SkillBundleEditModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<SkillBundleEditModalContent
				key={`${props.bundle.id}:${props.bundle.revision}`}
				bundle={props.bundle}
				onSubmit={props.onSubmit}
			/>
		</ModalDialog>
	);
}
