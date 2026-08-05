import type { SubmitEventHandler } from 'react';
import { useState } from 'react';

import type { AssistantPresetBundle } from '@/spec/assistantpreset';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

interface AssistantPresetBundleEditModalProps {
	isOpen: boolean;
	bundle: AssistantPresetBundle | null;
	onClose: () => void;
	onSubmit: (bundle: AssistantPresetBundle, displayName: string, description?: string) => Promise<void>;
}

function AssistantPresetBundleEditModalContent({
	bundle,
	onSubmit,
}: Omit<AssistantPresetBundleEditModalProps, 'isOpen' | 'onClose'> & { bundle: AssistantPresetBundle }) {
	const { requestClose, unmountingRef } = useModalDialogController();
	const [displayName, setDisplayName] = useState(bundle.displayName);
	const [description, setDescription] = useState(bundle.description ?? '');
	const [error, setError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const submit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		const normalizedName = displayName.trim();
		if (!normalizedName) {
			setError('Bundle name is required.');
			return;
		}
		if (isSubmitting) {
			return;
		}

		setError('');
		setIsSubmitting(true);
		void onSubmit(bundle, normalizedName, description.trim() || undefined)
			.then(() => {
				if (!unmountingRef.current) {
					requestClose(true);
				}
			})
			.catch((submitError: unknown) => {
				if (!unmountingRef.current) {
					setError(submitError instanceof Error ? submitError.message : 'Could not save the preset bundle.');
				}
			})
			.finally(() => {
				if (!unmountingRef.current) {
					setIsSubmitting(false);
				}
			});
	};

	return (
		<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
			<ModalHeader
				title="Edit Assistant Preset Bundle"
				description="Update the library name and description. The stable bundle slug remains unchanged."
				onClose={requestClose}
				closeDisabled={isSubmitting}
			/>
			<form className="space-y-4 p-4 sm:p-6" onSubmit={submit}>
				{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}
				<ModalSection title="Bundle details">
					<ModalField label="Bundle slug" htmlFor="assistant-preset-bundle-slug">
						<input
							id="assistant-preset-bundle-slug"
							className="input w-full rounded-xl font-mono"
							value={bundle.slug}
							readOnly
						/>
					</ModalField>
					<ModalField label="Display name" htmlFor="assistant-preset-bundle-name" required>
						<input
							id="assistant-preset-bundle-name"
							className="input w-full rounded-xl"
							value={displayName}
							onChange={event => {
								setDisplayName(event.currentTarget.value);
							}}
							disabled={isSubmitting}
							maxLength={256}
						/>
					</ModalField>
					<ModalField label="Description" htmlFor="assistant-preset-bundle-description">
						<textarea
							id="assistant-preset-bundle-description"
							className="textarea min-h-24 w-full rounded-xl"
							value={description}
							onChange={event => {
								setDescription(event.currentTarget.value);
							}}
							disabled={isSubmitting}
							maxLength={2000}
						/>
					</ModalField>
				</ModalSection>
				<ModalActions>
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
						{isSubmitting ? 'Saving...' : 'Save Bundle'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function AssistantPresetBundleEditModal(props: AssistantPresetBundleEditModalProps) {
	if (!props.isOpen || !props.bundle) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<AssistantPresetBundleEditModalContent
				key={`${props.bundle.id}:${String(props.bundle.modifiedAt)}`}
				{...props}
				bundle={props.bundle}
			/>
		</ModalDialog>
	);
}
