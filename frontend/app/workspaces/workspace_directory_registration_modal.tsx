import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import { FiAlertCircle, FiFolder } from 'react-icons/fi';

import { backendAPI } from '@/apis/baseapi';

import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';

interface WorkspaceDirectoryRegistrationModalProps {
	isOpen: boolean;
	onClose: () => void;
	onRegister: (path: string) => Promise<void>;
	title?: string;
	description?: string;
}

function WorkspaceDirectoryRegistrationModalContent({
	onClose,
	onRegister,
	title = 'Add Workspace Directory',
	description = 'Choose a repository. FlexiGPT finds its instruction files, documentation, Skills, MCP servers, and optional Workspace setup.',
}: Omit<WorkspaceDirectoryRegistrationModalProps, 'isOpen'>) {
	const [path, setPath] = useState('');
	const [error, setError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const chooseDirectory = async () => {
		setError('');

		try {
			const selected = await backendAPI.pickDirectoryPath();
			if (selected) {
				setPath(selected);
			}
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : 'Could not open the directory picker.');
		}
	};

	const submit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();

		const selectedPath = path.trim();
		if (!selectedPath) {
			setError('Choose or enter a repository directory.');
			return;
		}

		setError('');
		setIsSubmitting(true);

		void onRegister(selectedPath).then(
			() => {
				setIsSubmitting(false);
				onClose();
			},
			(cause: unknown) => {
				setError(cause instanceof Error ? cause.message : 'Workspace directory registration failed.');
				setIsSubmitting(false);
			}
		);
	};

	return (
		<ManagementModalPanel width="compact">
			<ModalHeader title={title} description={description} onClose={onClose} closeDisabled={isSubmitting} />

			<form className="flex min-h-0 flex-1 flex-col" onSubmit={submit} aria-busy={isSubmitting}>
				<ManagementModalBody>
					{error ? (
						<div className="alert alert-error rounded-2xl text-sm" role="alert">
							<FiAlertCircle size={14} />
							<span>{error}</span>
						</div>
					) : null}

					<ModalField
						label="Repository"
						htmlFor="workspace-directory-path"
						required
						hint="Only repository-relative file paths are shown in Workspace views."
					>
						<div className="flex gap-2">
							<input
								id="workspace-directory-path"
								type="text"
								className="input min-w-0 grow rounded-xl font-mono text-xs"
								value={path}
								onChange={event => {
									setPath(event.currentTarget.value);
								}}
								disabled={isSubmitting}
								placeholder="/path/to/repository"
								autoComplete="off"
								spellCheck="false"
							/>
							<button
								type="button"
								className="btn btn-ghost rounded-xl p-0 text-xs"
								disabled={isSubmitting}
								onClick={() => {
									void chooseDirectory();
								}}
							>
								<FiFolder size={12} />
								Choose directory
							</button>
						</div>
					</ModalField>
				</ManagementModalBody>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" disabled={isSubmitting} onClick={onClose}>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting}>
						{isSubmitting ? 'Adding directory...' : 'Add Workspace Directory'}
					</button>
				</ModalActions>
			</form>
		</ManagementModalPanel>
	);
}

export function WorkspaceDirectoryRegistrationModal(props: WorkspaceDirectoryRegistrationModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ManagementModalDialog isOpen onClose={props.onClose} blockCancel>
			<WorkspaceDirectoryRegistrationModalContent
				onClose={props.onClose}
				onRegister={props.onRegister}
				title={props.title}
				description={props.description}
			/>
		</ManagementModalDialog>
	);
}
