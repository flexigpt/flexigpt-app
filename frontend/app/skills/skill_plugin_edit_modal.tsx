import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';

import { skillManagementAPI } from '@/apis/baseapi';

import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
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
		<ManagementModalPanel width="compact">
			<ModalHeader title="Edit Skill Plugin" onClose={onClose} closeDisabled={saving} />

			<form className="flex min-h-0 flex-1 flex-col" onSubmit={submit}>
				<ManagementModalBody>
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
				</ManagementModalBody>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" disabled={saving} onClick={onClose}>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={saving}>
						{saving ? 'Saving...' : 'Save Plugin'}
					</button>
				</ModalActions>
			</form>
		</ManagementModalPanel>
	);
}

export function SkillPluginEditModal(props: SkillPluginEditModalProps) {
	if (!props.isOpen || !props.plugin) {
		return null;
	}

	return (
		<ManagementModalDialog isOpen onClose={props.onClose} blockCancel>
			<SkillPluginEditModalContent
				key={`${props.plugin.ref.rootID}:${props.plugin.ref.artifactID}:${props.plugin.revision}`}
				plugin={props.plugin}
				onClose={props.onClose}
				onSaved={props.onSaved}
			/>
		</ManagementModalDialog>
	);
}
