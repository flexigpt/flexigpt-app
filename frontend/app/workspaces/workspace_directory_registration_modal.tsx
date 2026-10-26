import type { SubmitEventHandler } from 'react';
import { useState } from 'react';
import { FiAlertCircle, FiFolder } from 'react-icons/fi';

import { backendAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
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
		<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-3xl rounded-2xl p-0">
			<ModalHeader title={title} description={description} onClose={onClose} closeDisabled={isSubmitting} />

			<form className="space-y-4 p-4 sm:p-6" onSubmit={submit} aria-busy={isSubmitting}>
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

				<ModalActions className="-mx-4 -mb-4 sm:-mx-6 sm:-mb-6">
					<button type="button" className="btn bg-base-300 rounded-xl" disabled={isSubmitting} onClick={onClose}>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting}>
						{isSubmitting ? 'Adding directory...' : 'Add Workspace Directory'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function WorkspaceDirectoryRegistrationModal(props: WorkspaceDirectoryRegistrationModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen onClose={props.onClose} blockCancel>
			<WorkspaceDirectoryRegistrationModalContent
				onClose={props.onClose}
				onRegister={props.onRegister}
				title={props.title}
				description={props.description}
			/>
		</ModalDialog>
	);
}
