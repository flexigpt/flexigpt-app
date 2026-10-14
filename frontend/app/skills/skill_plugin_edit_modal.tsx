import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';

import { skillManagementAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

interface SkillPluginEditModalProps {
	isOpen: boolean;
	plugin: PluginListItem | null;
	onClose: () => void;
	onSaved: () => Promise<void> | void;
}

function SkillPluginEditModalContent({
	plugin,
	onClose,
	onSaved,
}: Omit<SkillPluginEditModalProps, 'isOpen' | 'plugin'> & { plugin: PluginListItem }) {
	const [displayName, setDisplayName] = useState(plugin.displayName);
	const [description, setDescription] = useState(plugin.description ?? '');
	const [error, setError] = useState('');
	const [saving, setSaving] = useState(false);

	const submit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		if (!displayName.trim()) {
			setError('Display name is required.');
			return;
		}

		setSaving(true);
		setError('');

		void skillManagementAPI
			.updateSkillPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
				displayName: displayName.trim(),
				description: description.trim() || undefined,
			})
			.then(async () => {
				await onSaved();
				onClose();
			})
			.catch((cause: unknown) => {
				setError(cause instanceof Error ? cause.message : 'Skill Plugin could not be updated.');
			})
			.finally(() => {
				setSaving(false);
			});
	};

	return (
		<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
			<ModalHeader title="Edit Skill Plugin" onClose={onClose} closeDisabled={saving} />

			<form className="space-y-4 p-4 sm:p-6" onSubmit={submit}>
				{error ? (
					<div className="alert alert-error rounded-2xl text-sm">
						<FiAlertCircle size={14} />
						<span>{error}</span>
					</div>
				) : null}

				<ModalSection title="Plugin details">
					<ModalField label="Logical name">
						<div className="input bg-base-300 flex items-center rounded-xl font-mono text-sm">{plugin.name}</div>
					</ModalField>

					<ModalField label="Display name" htmlFor="skill-plugin-display-name" required>
						<input
							id="skill-plugin-display-name"
							className="input w-full rounded-xl"
							value={displayName}
							disabled={saving}
							onChange={event => {
								setDisplayName(event.currentTarget.value);
								setError('');
							}}
						/>
					</ModalField>

					<ModalField label="Description" htmlFor="skill-plugin-description">
						<textarea
							id="skill-plugin-description"
							className="textarea min-h-24 w-full rounded-xl"
							value={description}
							disabled={saving}
							onChange={event => {
								setDescription(event.currentTarget.value);
							}}
						/>
					</ModalField>
				</ModalSection>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" disabled={saving} onClick={onClose}>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={saving}>
						{saving ? 'Saving...' : 'Save Plugin'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function SkillPluginEditModal(props: SkillPluginEditModalProps) {
	if (!props.isOpen || !props.plugin) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<SkillPluginEditModalContent
				key={`${props.plugin.ref.rootID}:${props.plugin.ref.artifactID}:${props.plugin.revision}`}
				plugin={props.plugin}
				onClose={props.onClose}
				onSaved={props.onSaved}
			/>
		</ModalDialog>
	);
}
