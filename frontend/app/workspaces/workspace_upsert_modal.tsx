import type { SubmitEventHandler } from 'react';
import { useEffect, useId, useMemo, useRef, useState } from 'react';

import { FiAlertCircle, FiPlus, FiTrash2 } from 'react-icons/fi';

import type {
	CreateEmptyWorkspacePayload,
	CreateFilesystemWorkspacePayload,
	UpdateWorkspacePayload,
	WorkspaceDiscovery,
	WorkspaceDiscoveryRoot,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceMode } from '@/spec/workspace';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { cloneWorkspaceDiscovery, getErrorMessage } from '@/workspaces/lib/workspace_utils';

export type WorkspaceUpsertSubmission =
	| {
			kind: 'filesystem';
			payload: CreateFilesystemWorkspacePayload;
	  }
	| {
			kind: 'empty';
			payload: CreateEmptyWorkspacePayload;
	  }
	| {
			kind: 'update';
			payload: UpdateWorkspacePayload;
	  };

interface WorkspaceUpsertModalProps {
	isOpen: boolean;
	onClose: () => void;
	onSubmit: (submission: WorkspaceUpsertSubmission) => Promise<void>;
	workspace?: WorkspaceView;
	existingDisplayNames: readonly string[];
}

interface DiscoveryRootForm {
	key: number;
	root: string;
	recursive: boolean;
	includePatterns: string;
}

interface WorkspaceFormData {
	mode: WorkspaceMode;
	displayName: string;
	description: string;
	rootPath: string;
	trustReference: string;
	enabled: boolean;
	includeReadme: boolean;
	additionalLocators: string;
	additionalRoots: DiscoveryRootForm[];
}

interface FormErrors {
	displayName?: string;
	rootPath?: string;
	additionalLocators?: string;
	additionalRoots?: string;
	trustReference?: string;
}

function normalizeIdentity(value: string): string {
	return value.trim().toLowerCase();
}

function splitNonEmptyLines(value: string): string[] {
	return value
		.split('\n')
		.map(line => line.trim())
		.filter(Boolean);
}

function initializeForm(workspace?: WorkspaceView): WorkspaceFormData {
	const discovery = cloneWorkspaceDiscovery(workspace?.discovery);

	return {
		mode: workspace?.mode ?? WorkspaceMode.Filesystem,
		displayName: workspace?.displayName ?? '',
		description: workspace?.description ?? '',
		rootPath: '',
		trustReference: '',
		enabled: workspace?.enabled ?? true,
		includeReadme: discovery.includeReadme ?? true,
		additionalLocators: (discovery.additionalLocators ?? []).join('\n'),
		additionalRoots: (discovery.additionalRoots ?? []).map((root, index) => ({
			key: index + 1,
			root: root.root,
			recursive: root.recursive,
			includePatterns: (root.includePatterns ?? []).join(', '),
		})),
	};
}

function buildDiscovery(form: WorkspaceFormData): WorkspaceDiscovery {
	const additionalLocators = splitNonEmptyLines(form.additionalLocators);
	const additionalRoots: WorkspaceDiscoveryRoot[] = form.additionalRoots
		.map(root => ({
			root: root.root.trim(),
			recursive: root.recursive,
			includePatterns: root.includePatterns
				.split(',')
				.map(pattern => pattern.trim())
				.filter(Boolean),
		}))
		.filter(root => root.root.length > 0)
		.map(root =>
			Object.assign(root, { includePatterns: root.includePatterns?.length ? root.includePatterns : undefined })
		);

	return {
		includeReadme: form.includeReadme,
		additionalLocators: additionalLocators.length > 0 ? additionalLocators : undefined,
		additionalRoots: additionalRoots.length > 0 ? additionalRoots : undefined,
	};
}

function validateForm(
	form: WorkspaceFormData,
	workspace: WorkspaceView | undefined,
	existingDisplayNames: readonly string[],
	replaceTrustReference: boolean
): FormErrors {
	const errors: FormErrors = {};
	const displayName = form.displayName.trim();

	if (!displayName) {
		errors.displayName = 'Display name is required.';
	} else if (displayName.length > 256) {
		errors.displayName = 'Display name must be 256 characters or fewer.';
	} else if (existingDisplayNames.some(name => normalizeIdentity(name) === normalizeIdentity(displayName))) {
		errors.displayName = 'Another workspace already uses this display name.';
	}

	if (!workspace && form.mode === WorkspaceMode.Filesystem && !form.rootPath.trim()) {
		errors.rootPath = 'Root path is required for a filesystem workspace.';
	}

	if (form.rootPath.length > 4096) {
		errors.rootPath = 'Root path is too long.';
	}

	const locatorLines = splitNonEmptyLines(form.additionalLocators);
	if (new Set(locatorLines).size !== locatorLines.length) {
		errors.additionalLocators = 'Additional context paths must not contain duplicates.';
	}

	const roots = form.additionalRoots.map(root => root.root.trim()).filter(Boolean);
	if (new Set(roots).size !== roots.length) {
		errors.additionalRoots = 'Discovery folders must not contain duplicates.';
	}

	if (replaceTrustReference && form.trustReference.length > 4096) {
		errors.trustReference = 'Trust reference is too long.';
	}

	return errors;
}

function WorkspaceUpsertModalContent({
	onSubmit,
	workspace,
	existingDisplayNames,
}: Omit<WorkspaceUpsertModalProps, 'isOpen' | 'onClose'>) {
	const [form, setForm] = useState<WorkspaceFormData>(() => initializeForm(workspace));
	const [replaceTrustReference, setReplaceTrustReference] = useState(!workspace);
	const [submitted, setSubmitted] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const nextRootKeyRef = useRef(form.additionalRoots.length + 1);
	const displayNameRef = useRef<HTMLInputElement | null>(null);

	const displayNameID = useId();
	const descriptionID = useId();
	const rootPathID = useId();
	const trustReferenceID = useId();
	const locatorsID = useId();

	const { requestClose, unmountingRef } = useModalDialogController();
	const errors = useMemo(
		() => validateForm(form, workspace, existingDisplayNames, replaceTrustReference),
		[existingDisplayNames, form, replaceTrustReference, workspace]
	);

	useEffect(() => {
		const frame = window.requestAnimationFrame(() => {
			displayNameRef.current?.focus({ preventScroll: true });
		});

		return () => {
			window.cancelAnimationFrame(frame);
		};
	}, []);

	const update = <K extends keyof WorkspaceFormData>(field: K, value: WorkspaceFormData[K]) => {
		setForm(previous => ({ ...previous, [field]: value }));
		setSubmitError('');
	};

	const updateRoot = (key: number, patch: Partial<DiscoveryRootForm>) => {
		setForm(previous => ({
			...previous,
			additionalRoots: previous.additionalRoots.map(root => (root.key === key ? { ...root, ...patch } : root)),
		}));
		setSubmitError('');
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

		const discovery = buildDiscovery(form);
		const description = form.description.trim() || undefined;
		const trustReference = form.trustReference.trim() || undefined;

		let submission: WorkspaceUpsertSubmission;

		if (workspace) {
			const payload: UpdateWorkspacePayload = {
				expectedRevision: workspace.revision,
				displayName: form.displayName.trim(),
				description,
				enabled: form.enabled,
				discovery,
			};

			if (replaceTrustReference) {
				payload.trustReference = trustReference ?? '';
			}

			submission = {
				kind: 'update',
				payload,
			};
		} else if (form.mode === WorkspaceMode.Filesystem) {
			submission = {
				kind: 'filesystem',
				payload: {
					displayName: form.displayName.trim(),
					description,
					rootPath: form.rootPath.trim(),
					trustReference,
					discovery,
				},
			};
		} else {
			submission = {
				kind: 'empty',
				payload: {
					displayName: form.displayName.trim(),
					description,
					trustReference,
					discovery,
				},
			};
		}

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

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={workspace ? 'Edit Workspace' : 'Add Workspace'}
				description={
					workspace
						? 'Manage workspace metadata and additional discovery paths.'
						: 'Add a root path and let workspace discovery find context and skills.'
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
				aria-busy={isSubmitting}
			>
				{submitError ? (
					<div className="alert alert-error rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={14} />
						<span>{submitError}</span>
					</div>
				) : null}

				{!workspace ? (
					<ModalSection
						title="Workspace source"
						description="Most workspaces should use one filesystem root. Empty workspaces are useful when all sources will be attached later."
					>
						<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
							<label className="border-base-content/10 hover:bg-base-100 flex cursor-pointer items-start gap-3 rounded-2xl border p-4">
								<input
									type="radio"
									className="radio radio-sm mt-1"
									checked={form.mode === WorkspaceMode.Filesystem}
									onChange={() => {
										update('mode', WorkspaceMode.Filesystem);
									}}
									disabled={isSubmitting}
								/>

								<span className="block font-medium">Filesystem workspace</span>
								<span className="text-base-content/70 mt-1 block text-xs">
									Discover context and skills from one project folder.
								</span>
							</label>

							<label className="border-base-content/10 hover:bg-base-100 flex cursor-pointer items-start gap-3 rounded-2xl border p-4">
								<input
									type="radio"
									className="radio radio-sm mt-1"
									checked={form.mode === WorkspaceMode.Empty}
									onChange={() => {
										update('mode', WorkspaceMode.Empty);
									}}
									disabled={isSubmitting}
								/>

								<span className="block font-medium">Empty workspace</span>
								<span className="text-base-content/70 mt-1 block text-xs">
									Create a workspace shell without a primary filesystem source.
								</span>
							</label>
						</div>

						{form.mode === WorkspaceMode.Filesystem ? (
							<ModalField
								label="Root Path"
								htmlFor={rootPathID}
								required
								hint="Absolute or backend-supported filesystem path for the project root."
								error={submitted ? errors.rootPath : undefined}
							>
								<input
									id={rootPathID}
									type="text"
									className={`input w-full rounded-xl ${submitted && errors.rootPath ? 'input-error' : ''}`}
									value={form.rootPath}
									onChange={event => {
										update('rootPath', event.currentTarget.value);
									}}
									placeholder="/path/to/project"
									spellCheck="false"
									autoComplete="off"
									disabled={isSubmitting}
								/>
							</ModalField>
						) : null}
					</ModalSection>
				) : (
					<div className="alert alert-info rounded-2xl text-sm">
						The primary source path is intentionally hidden by the workspace API after creation. Additional discovery
						paths can still be managed below.
					</div>
				)}

				<ModalSection title="Identity">
					<ModalField
						label="Display Name"
						htmlFor={displayNameID}
						required
						error={submitted ? errors.displayName : undefined}
					>
						<input
							id={displayNameID}
							ref={displayNameRef}
							type="text"
							className={`input w-full rounded-xl ${submitted && errors.displayName ? 'input-error' : ''}`}
							value={form.displayName}
							onChange={event => {
								update('displayName', event.currentTarget.value);
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
								update('description', event.currentTarget.value);
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
									update('enabled', event.currentTarget.checked);
								}}
								disabled={isSubmitting}
							/>
						</ModalField>
					) : null}
				</ModalSection>

				<ModalSection
					title="Discovery"
					description="Default workspace conventions are discovered automatically. Add only project-specific files or folders here."
				>
					<ModalField label="Include README" htmlFor={`${locatorsID}-readme`}>
						<label className="flex items-center gap-3">
							<input
								id={`${locatorsID}-readme`}
								type="checkbox"
								className="toggle toggle-accent"
								checked={form.includeReadme}
								onChange={event => {
									update('includeReadme', event.currentTarget.checked);
								}}
								disabled={isSubmitting}
							/>
							<span className="text-base-content/70 text-xs">
								Discover a project README as workspace context when available.
							</span>
						</label>
					</ModalField>

					<ModalField
						label="Additional Files"
						htmlFor={locatorsID}
						align="start"
						hint="One backend-supported locator or path per line."
						error={submitted ? errors.additionalLocators : undefined}
					>
						<textarea
							id={locatorsID}
							className={`textarea min-h-28 w-full rounded-xl font-mono text-xs ${
								submitted && errors.additionalLocators ? 'textarea-error' : ''
							}`}
							value={form.additionalLocators}
							onChange={event => {
								update('additionalLocators', event.currentTarget.value);
							}}
							placeholder={'docs/project-context.md\nAGENTS.md'}
							spellCheck="false"
							disabled={isSubmitting}
						/>
					</ModalField>

					<div className="space-y-3">
						<div className="flex items-center justify-between gap-3">
							<div>
								<div className="text-sm font-medium">Additional discovery folders</div>
								<div className="text-base-content/70 mt-1 text-xs">
									Use folders when several context files or skill directories should be discovered together.
								</div>
							</div>
							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								onClick={() => {
									const key = nextRootKeyRef.current;
									nextRootKeyRef.current += 1;
									update('additionalRoots', [
										...form.additionalRoots,
										{
											key,
											root: '',
											recursive: true,
											includePatterns: '',
										},
									]);
								}}
								disabled={isSubmitting}
							>
								<FiPlus size={14} />
								<span>Add Folder</span>
							</button>
						</div>

						{submitted && errors.additionalRoots ? (
							<div className="text-error text-xs">{errors.additionalRoots}</div>
						) : null}

						{form.additionalRoots.length === 0 ? (
							<div className="border-base-content/10 text-base-content/70 rounded-2xl border px-4 py-6 text-center text-sm">
								No additional discovery folders.
							</div>
						) : null}

						{form.additionalRoots.map(root => (
							<div
								key={root.key}
								className="border-base-content/10 grid grid-cols-1 gap-3 rounded-2xl border p-3 md:grid-cols-12"
							>
								<div className="md:col-span-5">
									<label className="text-xs font-medium" htmlFor={`${locatorsID}-root-${root.key}`}>
										Folder
									</label>
									<input
										id={`${locatorsID}-root-${root.key}`}
										type="text"
										className="input input-sm mt-1 w-full rounded-xl font-mono text-xs"
										value={root.root}
										onChange={event => {
											updateRoot(root.key, { root: event.currentTarget.value });
										}}
										placeholder="docs/context"
										spellCheck="false"
										disabled={isSubmitting}
									/>
								</div>

								<div className="md:col-span-5">
									<label className="text-xs font-medium" htmlFor={`${locatorsID}-patterns-${root.key}`}>
										Include patterns
									</label>
									<input
										id={`${locatorsID}-patterns-${root.key}`}
										type="text"
										className="input input-sm mt-1 w-full rounded-xl font-mono text-xs"
										value={root.includePatterns}
										onChange={event => {
											updateRoot(root.key, { includePatterns: event.currentTarget.value });
										}}
										placeholder="*.md, **/SKILL.md"
										spellCheck="false"
										disabled={isSubmitting}
									/>
								</div>

								<div className="flex items-end gap-3 md:col-span-2 md:justify-end">
									<label className="flex items-center gap-2 text-xs">
										<input
											type="checkbox"
											className="checkbox checkbox-sm"
											checked={root.recursive}
											onChange={event => {
												updateRoot(root.key, { recursive: event.currentTarget.checked });
											}}
											disabled={isSubmitting}
										/>
										Recursive
									</label>
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										onClick={() => {
											update(
												'additionalRoots',
												form.additionalRoots.filter(existing => existing.key !== root.key)
											);
										}}
										disabled={isSubmitting}
										aria-label="Remove discovery folder"
									>
										<FiTrash2 size={14} />
									</button>
								</div>
							</div>
						))}
					</div>
				</ModalSection>

				<ModalSection
					title="Trust reference"
					description="Trust references are write-only in the frontend projection and are never displayed after saving."
				>
					{workspace ? (
						<label className="flex items-center gap-3 text-sm">
							<input
								type="checkbox"
								className="checkbox checkbox-sm"
								checked={replaceTrustReference}
								onChange={event => {
									setReplaceTrustReference(event.currentTarget.checked);
								}}
								disabled={isSubmitting}
							/>
							<span>
								{workspace.hasTrustReference
									? 'Replace or clear the existing trust reference'
									: 'Set a trust reference'}
							</span>
						</label>
					) : null}

					{replaceTrustReference ? (
						<ModalField
							label="Reference"
							htmlFor={trustReferenceID}
							hint={workspace ? 'Leave empty to clear the current trust reference.' : 'Optional trust reference.'}
							error={submitted ? errors.trustReference : undefined}
						>
							<input
								id={trustReferenceID}
								type="password"
								className={`input w-full rounded-xl ${submitted && errors.trustReference ? 'input-error' : ''}`}
								value={form.trustReference}
								onChange={event => {
									update('trustReference', event.currentTarget.value);
								}}
								autoComplete="off"
								spellCheck="false"
								disabled={isSubmitting}
							/>
						</ModalField>
					) : null}
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

export function WorkspaceUpsertModal(props: WorkspaceUpsertModalProps) {
	if (!props.isOpen) {
		return null;
	}

	const modalKey = props.workspace
		? `workspace:${props.workspace.rootID}:${props.workspace.revision}`
		: 'workspace:new';

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<WorkspaceUpsertModalContent key={modalKey} {...props} />
		</ModalDialog>
	);
}
