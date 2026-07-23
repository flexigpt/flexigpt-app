import type { SubmitEventHandler } from 'react';
import { useId, useMemo, useState } from 'react';

import { FiAlertCircle, FiFolder, FiPlus, FiTrash2, FiUpload } from 'react-icons/fi';

import type {
	CreateFilesystemWorkspacePayload,
	UpdateWorkspacePayload,
	WorkspaceDiscovery,
	WorkspaceView,
} from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { backendAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import {
	getErrorMessage,
	WORKSPACE_DEFAULT_CONTEXT_FILES,
	WORKSPACE_DEFAULT_SKILL_ROOTS,
	workspaceLocatorToPath,
	workspacePathToLocator,
} from '@/workspaces/lib/workspace_utils';

export type WorkspaceSetupSubmission =
	| {
			kind: 'filesystem';
			payload: CreateFilesystemWorkspacePayload;
	  }
	| {
			kind: 'update';
			payload: UpdateWorkspacePayload;
	  };

interface WorkspaceSetupModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (submission: WorkspaceSetupSubmission) => Promise<void>;
	workspace?: WorkspaceView;
	existingDisplayNames: readonly string[];
}

interface WorkspaceSetupForm {
	displayName: string;
	description: string;
	rootPath: string;
	enabled: boolean;
	includeReadme: boolean;
	contextFiles: string[];
	skillRoots: string[];
}

interface FormErrors {
	displayName?: string;
	rootPath?: string;
	discovery?: string;
}

function normalizeIdentity(value: string): string {
	return value.trim().toLowerCase();
}

function unique(values: string[]): string[] {
	return [...new Set(values.map(value => value.trim()).filter(Boolean))];
}

function displayNameFromPath(path: string): string {
	const segments = path.replaceAll('\\', '/').split('/').filter(Boolean);
	return segments.at(-1) ?? '';
}

function initialForm(workspace?: WorkspaceView): WorkspaceSetupForm {
	return {
		displayName: workspace?.displayName ?? '',
		description: workspace?.description ?? '',
		rootPath: workspace?.primaryPath ?? '',
		enabled: workspace?.enabled ?? true,
		includeReadme: workspace?.discovery.includeReadme ?? false,
		contextFiles: [...(workspace?.discovery.additionalLocators ?? [])],
		skillRoots: (workspace?.discovery.additionalRoots ?? []).map(root => root.root),
	};
}

function buildDiscovery(form: WorkspaceSetupForm): WorkspaceDiscovery {
	const contextFiles = unique(form.contextFiles)
		.map(value => workspacePathToLocator(form.rootPath, value))
		.toSorted();

	const skillRoots = unique(form.skillRoots)
		.map(value => workspacePathToLocator(form.rootPath, value, true))
		.toSorted();

	return {
		includeReadme: form.includeReadme,
		additionalLocators: contextFiles.length > 0 ? contextFiles : undefined,
		additionalRoots:
			skillRoots.length > 0
				? skillRoots.map(root => ({
						root,
						recursive: true,
						includePatterns: ['SKILL.md'],
					}))
				: undefined,
	};
}

function validateForm(
	form: WorkspaceSetupForm,
	workspace: WorkspaceView | undefined,
	existingDisplayNames: readonly string[]
): FormErrors {
	const errors: FormErrors = {};
	const displayName = form.displayName.trim();
	const rootPath = form.rootPath.trim();

	if (!displayName) {
		errors.displayName = 'Display name is required.';
	} else if (displayName.length > 256) {
		errors.displayName = 'Display name must be 256 characters or fewer.';
	} else if (existingDisplayNames.some(name => normalizeIdentity(name) === normalizeIdentity(displayName))) {
		errors.displayName = 'Another workspace already uses this display name.';
	}

	if (!workspace && !rootPath) {
		errors.rootPath = 'Choose or paste a project folder.';
	}

	if (rootPath.length > 4096) {
		errors.rootPath = 'Project path is too long.';
	}

	if ((form.contextFiles.some(value => value.trim()) || form.skillRoots.some(value => value.trim())) && !rootPath) {
		errors.discovery = 'Choose a project folder before adding Context files or Skill folders.';
	} else if (rootPath) {
		try {
			buildDiscovery(form);
		} catch (error) {
			errors.discovery = getErrorMessage(error, 'Discovery paths are invalid.');
		}
	}

	return errors;
}

function WorkspaceSetupModalContent({
	onSubmit,
	workspace,
	existingDisplayNames,
}: Omit<WorkspaceSetupModalProps, 'isOpen' | 'onClose'>) {
	const [form, setForm] = useState<WorkspaceSetupForm>(() => initialForm(workspace));
	const [submitted, setSubmitted] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [pickerError, setPickerError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const displayNameID = useId();
	const descriptionID = useId();
	const rootPathID = useId();

	const { requestClose, unmountingRef } = useModalDialogController();
	const errors = useMemo(
		() => validateForm(form, workspace, existingDisplayNames),
		[existingDisplayNames, form, workspace]
	);

	const updateForm = <K extends keyof WorkspaceSetupForm>(key: K, value: WorkspaceSetupForm[K]) => {
		setForm(previous => ({ ...previous, [key]: value }));
		setSubmitError('');
		setPickerError('');
	};

	const chooseRootPath = async () => {
		setPickerError('');
		try {
			const path = await backendAPI.pickDirectoryPath();
			if (!path) {
				return;
			}

			setForm(previous => ({
				...previous,
				rootPath: path,
				displayName: previous.displayName.trim() || displayNameFromPath(path),
			}));
		} catch (error) {
			setPickerError(getErrorMessage(error, 'Could not open the folder picker.'));
		}
	};

	const chooseContextFiles = async () => {
		const rootPath = form.rootPath.trim();
		if (!rootPath) {
			setPickerError('Choose a project folder before selecting Context files.');
			return;
		}

		setPickerError('');
		try {
			const paths = await backendAPI.pickFilePaths(true);
			const locators = paths.map(path => workspacePathToLocator(rootPath, path));
			setForm(previous => ({
				...previous,
				contextFiles: unique([...previous.contextFiles, ...locators]),
			}));
		} catch (error) {
			setPickerError(getErrorMessage(error, 'Selected files must be inside the project folder.'));
		}
	};

	const chooseSkillRoot = async () => {
		const rootPath = form.rootPath.trim();
		if (!rootPath) {
			setPickerError('Choose a project folder before selecting a Skill folder.');
			return;
		}

		setPickerError('');
		try {
			const path = await backendAPI.pickDirectoryPath();
			if (!path) {
				return;
			}
			const locator = workspacePathToLocator(rootPath, path, true);
			setForm(previous => ({
				...previous,
				skillRoots: unique([...previous.skillRoots, locator]),
			}));
		} catch (error) {
			setPickerError(getErrorMessage(error, 'Selected folders must be inside the project folder.'));
		}
	};

	const updateContextFile = (index: number, value: string) => {
		setForm(previous => ({
			...previous,
			contextFiles: previous.contextFiles.map((item, itemIndex) => (itemIndex === index ? value : item)),
		}));
	};

	const updateSkillRoot = (index: number, value: string) => {
		setForm(previous => ({
			...previous,
			skillRoots: previous.skillRoots.map((item, itemIndex) => (itemIndex === index ? value : item)),
		}));
	};

	const removeContextFile = (index: number) => {
		setForm(previous => ({
			...previous,
			contextFiles: previous.contextFiles.filter((_, itemIndex) => itemIndex !== index),
		}));
	};

	const removeSkillRoot = (index: number) => {
		setForm(previous => ({
			...previous,
			skillRoots: previous.skillRoots.filter((_, itemIndex) => itemIndex !== index),
		}));
	};

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

		let discovery: WorkspaceDiscovery;
		try {
			discovery = buildDiscovery(form);
		} catch (error) {
			setSubmitError(getErrorMessage(error, 'Discovery paths are invalid.'));
			return;
		}

		const description = form.description.trim() || undefined;
		const submission: WorkspaceSetupSubmission = workspace
			? {
					kind: 'update',
					payload: {
						expectedRevision: workspace.revision,
						displayName: form.displayName.trim(),
						description,
						enabled: form.enabled,
						discovery,
					},
				}
			: {
					kind: 'filesystem',
					payload: {
						displayName: form.displayName.trim(),
						description,
						rootPath: form.rootPath.trim(),
						discovery,
					},
				};

		setIsSubmitting(true);
		try {
			await onSubmit(submission);
			if (!unmountingRef.current) {
				requestClose(true);
			}
		} catch (error) {
			if (!unmountingRef.current) {
				setSubmitError(getErrorMessage(error, 'Failed to save workspace.'));
			}
		} finally {
			if (!unmountingRef.current) {
				setIsSubmitting(false);
			}
		}
	};

	const hasProjectFolder = Boolean(form.rootPath.trim());

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={workspace ? 'Edit Workspace' : 'Add Workspace'}
				description={
					workspace
						? 'Manage the project folder, discovery paths, Context files, and Skill folders.'
						: 'Choose a project folder. Workspace discovery finds standard Context files and Skills automatically.'
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

				{pickerError ? (
					<div className="alert alert-warning rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={14} />
						<span>{pickerError}</span>
					</div>
				) : null}

				<ModalSection
					title="Project folder"
					description="Workspace discovery reads this folder but does not execute Skills, scripts, tools, or external processes."
				>
					<ModalField
						label="Folder path"
						htmlFor={rootPathID}
						required={!workspace}
						error={submitted ? errors.rootPath : undefined}
					>
						<div className="flex flex-col gap-2 sm:flex-row">
							<input
								id={rootPathID}
								type="text"
								className={`input min-w-0 grow rounded-xl font-mono text-xs ${
									submitted && errors.rootPath ? 'input-error' : ''
								}`}
								value={form.rootPath}
								onChange={event => {
									updateForm('rootPath', event.currentTarget.value);
								}}
								readOnly={Boolean(workspace)}
								placeholder="/path/to/project"
								spellCheck="false"
								autoComplete="off"
								disabled={isSubmitting}
							/>
							{!workspace ? (
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										void chooseRootPath();
									}}
									disabled={isSubmitting}
								>
									<FiFolder size={14} />
									<span>Choose Folder</span>
								</button>
							) : null}
						</div>
					</ModalField>

					{workspace?.primaryPath ? (
						<div className="text-base-content/60 text-xs">
							The primary workspace folder cannot be changed after creation. Create a new workspace if the project
							moves.
						</div>
					) : null}
				</ModalSection>

				<ModalSection title="Identity">
					<ModalField
						label="Display name"
						htmlFor={displayNameID}
						required
						error={submitted ? errors.displayName : undefined}
					>
						<input
							id={displayNameID}
							type="text"
							className={`input w-full rounded-xl ${submitted && errors.displayName ? 'input-error' : ''}`}
							value={form.displayName}
							onChange={event => {
								updateForm('displayName', event.currentTarget.value);
							}}
							maxLength={256}
							spellCheck="false"
							autoComplete="off"
							disabled={isSubmitting}
						/>
					</ModalField>

					<ModalField label="Description" htmlFor={descriptionID} align="start">
						<textarea
							id={descriptionID}
							className="textarea min-h-24 w-full rounded-xl"
							value={form.description}
							onChange={event => {
								updateForm('description', event.currentTarget.value);
							}}
							maxLength={2000}
							disabled={isSubmitting}
						/>
					</ModalField>

					{workspace ? (
						<ModalField label="Enabled" htmlFor={`${displayNameID}-enabled`}>
							<input
								id={`${displayNameID}-enabled`}
								type="checkbox"
								className="toggle toggle-accent"
								checked={form.enabled}
								onChange={event => {
									updateForm('enabled', event.currentTarget.checked);
								}}
								disabled={isSubmitting}
							/>
						</ModalField>
					) : null}
				</ModalSection>

				<ModalSection
					title="Discovery"
					description="Use the defaults for normal projects. Add paths only when a project keeps Context or Skills in non-standard locations."
				>
					<div className="border-base-content/10 bg-base-100 rounded-2xl border p-3 text-xs">
						<div className="font-semibold">Default discovery</div>
						<ul className="text-base-content/70 mt-2 list-disc space-y-1 pl-5">
							{WORKSPACE_DEFAULT_CONTEXT_FILES.map(locator => (
								<li key={locator}>{locator}</li>
							))}
							{WORKSPACE_DEFAULT_SKILL_ROOTS.map(locator => (
								<li key={locator}>{locator}</li>
							))}
						</ul>
					</div>

					<ModalField label="Include README" htmlFor={`${displayNameID}-readme`}>
						<label className="flex items-center gap-3">
							<input
								id={`${displayNameID}-readme`}
								type="checkbox"
								className="toggle toggle-accent"
								checked={form.includeReadme}
								onChange={event => {
									updateForm('includeReadme', event.currentTarget.checked);
								}}
								disabled={isSubmitting}
							/>
							<span className="text-base-content/70 text-xs">
								Discover README.md as project Context when it exists.
							</span>
						</label>
					</ModalField>

					{submitted && errors.discovery ? (
						<div className="text-error text-xs" role="alert">
							{errors.discovery}
						</div>
					) : null}

					<div className="space-y-3">
						<div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
							<div>
								<div className="text-sm font-medium">Additional Context files</div>
								<div className="text-base-content/70 mt-1 text-xs">
									Add Markdown files that should become project Context.
								</div>
							</div>
							<div className="flex flex-wrap gap-2">
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										updateForm('contextFiles', [...form.contextFiles, '']);
									}}
									disabled={isSubmitting || !hasProjectFolder}
								>
									<FiPlus size={14} />
									<span>Add Path</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										void chooseContextFiles();
									}}
									disabled={isSubmitting || !hasProjectFolder}
								>
									<FiUpload size={14} />
									<span>Choose Files</span>
								</button>
							</div>
						</div>

						{form.contextFiles.length === 0 ? (
							<div className="border-base-content/10 text-base-content/60 rounded-2xl border p-4 text-sm">
								No additional Context files.
							</div>
						) : null}

						{form.contextFiles.map((locator, index) => (
							<div key={`${index}-${locator}`} className="border-base-content/10 rounded-2xl border p-3">
								<div className="flex gap-2">
									<input
										type="text"
										className="input input-sm min-w-0 grow rounded-xl font-mono text-xs"
										value={locator}
										onChange={event => {
											updateContextFile(index, event.currentTarget.value);
										}}
										placeholder="docs/project-context.md"
										spellCheck="false"
										disabled={isSubmitting}
									/>
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										onClick={() => {
											removeContextFile(index);
										}}
										disabled={isSubmitting}
										aria-label="Remove Context file"
									>
										<FiTrash2 size={14} />
									</button>
								</div>
								{locator.trim() && hasProjectFolder ? (
									<div className="text-base-content/60 mt-2 font-mono text-xs break-all">
										{workspaceLocatorToPath(form.rootPath, locator)}
									</div>
								) : null}
							</div>
						))}
					</div>

					<div className="space-y-3">
						<div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
							<div>
								<div className="text-sm font-medium">Additional Skill folders</div>
								<div className="text-base-content/70 mt-1 text-xs">
									Each selected folder is scanned recursively for SKILL.md files.
								</div>
							</div>
							<div className="flex flex-wrap gap-2">
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										updateForm('skillRoots', [...form.skillRoots, '']);
									}}
									disabled={isSubmitting || !hasProjectFolder}
								>
									<FiPlus size={14} />
									<span>Add Path</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										void chooseSkillRoot();
									}}
									disabled={isSubmitting || !hasProjectFolder}
								>
									<FiFolder size={14} />
									<span>Choose Folder</span>
								</button>
							</div>
						</div>

						{form.skillRoots.length === 0 ? (
							<div className="border-base-content/10 text-base-content/60 rounded-2xl border p-4 text-sm">
								No additional Skill folders.
							</div>
						) : null}

						{form.skillRoots.map((locator, index) => (
							<div key={`${index}-${locator}`} className="border-base-content/10 rounded-2xl border p-3">
								<div className="flex gap-2">
									<input
										type="text"
										className="input input-sm min-w-0 grow rounded-xl font-mono text-xs"
										value={locator}
										onChange={event => {
											updateSkillRoot(index, event.currentTarget.value);
										}}
										placeholder=".agent-skills"
										spellCheck="false"
										disabled={isSubmitting}
									/>
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										onClick={() => {
											removeSkillRoot(index);
										}}
										disabled={isSubmitting}
										aria-label="Remove Skill folder"
									>
										<FiTrash2 size={14} />
									</button>
								</div>
								{locator.trim() && hasProjectFolder ? (
									<div className="text-base-content/60 mt-2 font-mono text-xs break-all">
										{workspaceLocatorToPath(form.rootPath, locator)}
									</div>
								) : null}
							</div>
						))}
					</div>
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
						{isSubmitting ? (
							<>
								<span className="loading loading-spinner loading-xs" />
								Saving...
							</>
						) : workspace ? (
							'Save Changes'
						) : (
							'Create Workspace'
						)}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

export function WorkspaceSetupModal(props: WorkspaceSetupModalProps) {
	if (!props.isOpen) {
		return null;
	}

	const modalKey = props.workspace
		? `workspace:${props.workspace.rootID}:${props.workspace.revision}`
		: 'workspace:new';

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<WorkspaceSetupModalContent key={modalKey} {...props} />
		</ModalDialog>
	);
}
