import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { SkillPlugin } from '@/spec/skill';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

interface SkillPluginEditModalProps {
	isOpen: boolean;
	onClose: () => void;
	plugin: SkillPlugin;
	onSubmit: (pluginID: string, displayName: string, description?: string) => Promise<void>;
}

function SkillPluginEditModalContent({ plugin, onSubmit }: Omit<SkillPluginEditModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose, unmountingRef } = useModalDialogController();
	const [displayName, setDisplayName] = useState(plugin.displayName || plugin.slug);
	const [description, setDescription] = useState(plugin.description || '');
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
		void onSubmit(plugin.id, normalizedName, description.trim() || undefined)
			.then(() => {
				if (!unmountingRef.current) {
					requestClose(true);
				}
			})
			.catch((error: unknown) => {
				if (!unmountingRef.current) {
					setSubmitError(error instanceof Error ? error.message : 'Skill Plugin could not be updated.');
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
				title="Edit Skill Plugin"
				description="Update the Plugin display name and description. Its stable name remains unchanged."
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

				<ModalSection title="Plugin details">
					<ModalField label="Name">
						<div className="input bg-base-300 flex items-center rounded-xl font-mono text-sm">{plugin.slug}</div>
					</ModalField>

					<ModalField label="Display name" htmlFor="skill-plugin-display-name" required>
						<input
							id="skill-plugin-display-name"
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

					<ModalField label="Description" htmlFor="skill-plugin-description" align="start">
						<textarea
							id="skill-plugin-description"
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

export function SkillPluginEditModal(props: SkillPluginEditModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<SkillPluginEditModalContent
				key={`${props.plugin.id}:${props.plugin.revision}`}
				plugin={props.plugin}
				onSubmit={props.onSubmit}
			/>
		</ModalDialog>
	);
}
